package router

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gotify/location"
	"github.com/gotify/server/v3/api"
	"github.com/gotify/server/v3/api/stream"
	"github.com/gotify/server/v3/auth"
	"github.com/gotify/server/v3/automation"
	"github.com/gotify/server/v3/config"
	"github.com/gotify/server/v3/connectors"
	"github.com/gotify/server/v3/database"
	"github.com/gotify/server/v3/docs"
	gerror "github.com/gotify/server/v3/error"
	"github.com/gotify/server/v3/model"
	"github.com/gotify/server/v3/operations"
	"github.com/gotify/server/v3/plugin"
	"github.com/gotify/server/v3/security"
	"github.com/gotify/server/v3/ui"
	"github.com/rs/zerolog/log"
)

// Create creates the gin engine with all routes.
func Create(db *database.GormDatabase, vInfo *model.VersionInfo, conf *config.Configuration) (*gin.Engine, func()) {
	g := gin.New()

	g.RemoveExtraSlash = true
	g.RemoteIPHeaders = []string{"X-Forwarded-For"}
	g.SetTrustedProxies(conf.Server.TrustedProxies)
	g.ForwardedByClientIP = true

	g.Use(func(ctx *gin.Context) {
		// Map sockets "@" to 127.0.0.1, because gin-gonic can only trust IPs.
		if ctx.Request.RemoteAddr == "@" {
			ctx.Request.RemoteAddr = "127.0.0.1:65535"
		}
	})

	g.Use(accessLogger(), auditMutations(db), gin.Recovery(), gerror.Handler(), location.Default())
	g.NoRoute(gerror.NotFound())

	if conf.Server.SSL.Enabled && conf.Server.SSL.RedirectToHTTPS {
		g.Use(func(ctx *gin.Context) {
			if ctx.Request.TLS != nil {
				ctx.Next()
				return
			}
			if ctx.Request.Method != http.MethodGet && ctx.Request.Method != http.MethodHead {
				ctx.Data(http.StatusBadRequest, "text/plain; charset=utf-8", []byte("Use HTTPS"))
				ctx.Abort()
				return
			}
			host := ctx.Request.Host
			if idx := strings.LastIndex(host, ":"); idx != -1 {
				host = host[:idx]
			}
			if conf.Server.SSL.Port != 443 {
				host = fmt.Sprintf("%s:%d", host, conf.Server.SSL.Port)
			}
			ctx.Redirect(http.StatusFound, fmt.Sprintf("https://%s%s", host, ctx.Request.RequestURI))
			ctx.Abort()
		})
	}
	attachmentDir := filepath.Join(filepath.Dir(filepath.Clean(conf.UploadedImagesDir)), "attachments")
	maintenanceStop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		run := func(now time.Time) {
			if deleted, err := db.ApplyMessageRetention(now); err != nil {
				log.Error().Err(err).Msg("Could not apply Channel message retention")
			} else if deleted > 0 {
				log.Info().Int("messages", deleted).Msg("Applied Channel message retention")
			}
			policy, err := db.GetSecurityPolicy()
			if err != nil {
				log.Error().Err(err).Msg("Could not load retention policy")
			} else if policy.AuditRetentionDays > 0 {
				before := now.AddDate(0, 0, -policy.AuditRetentionDays)
				if err := db.DeleteAuditEventsBefore(before); err != nil {
					log.Error().Err(err).Msg("Could not apply audit retention")
				}
				if err := db.CleanupAutomationHistory(before); err != nil {
					log.Error().Err(err).Msg("Could not clean automation history")
				}
			}
			if err := cleanupOrphanAttachments(db, attachmentDir); err != nil {
				log.Error().Err(err).Msg("Could not clean orphaned attachments")
			}
		}
		run(time.Now())
		for {
			select {
			case now := <-ticker.C:
				run(now)
			case <-maintenanceStop:
				return
			}
		}
	}()

	streamHandler := stream.New(
		time.Duration(conf.Server.Stream.PingPeriodSeconds)*time.Second, 15*time.Second, conf.Server.Stream.AllowedOrigins,
	)
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-maintenanceStop:
				return
			case now := <-ticker.C:
				connectedTokens := streamHandler.CollectConnectedClientTokens()
				if err := db.UpdateClientTokensLastUsedAndExpiresAt(connectedTokens, &now); err != nil {
					log.Error().Err(err).Msg("Error updating last used")
				}
				if expired, err := db.CleanupExpiredClients(now); err == nil {
					for _, client := range expired {
						streamHandler.NotifyDeletedClient(client.UserID, client.Token)
					}
				} else {
					log.Error().Err(err).Msg("Error cleaning up expired clients")
				}
			}
		}
	}()
	authentication := auth.Auth{
		DB:               db,
		SecureCookie:     conf.Server.SecureCookie,
		LocalAuthEnabled: conf.LocalAuthEnabled,
		CrossOrigin:      http.NewCrossOriginProtection(),
	}
	automationEngine := automation.New(db, streamHandler)
	connectorManager := connectors.New(db, automationEngine)
	automationEngine.AddPostStoreHook(connectorManager.OnMessage)
	messageHandler := api.MessageAPI{Notifier: streamHandler, DB: db, Dispatcher: automationEngine}
	healthHandler := api.HealthAPI{DB: db}
	clientHandler := api.ClientAPI{
		DB:            db,
		ImageDir:      conf.UploadedImagesDir,
		NotifyDeleted: streamHandler.NotifyDeletedClient,
	}
	applicationHandler := api.ApplicationAPI{
		DB:       db,
		ImageDir: conf.UploadedImagesDir,
		OnDelete: func(uint) { automationEngine.ReloadIntegrations() },
	}
	applicationMembershipHandler := api.ApplicationMembershipAPI{
		DB: db,
	}
	sessionHandler := api.SessionAPI{DB: db, NotifyDeleted: streamHandler.NotifyDeletedClient, SecureCookie: conf.Server.SecureCookie, LocalAuthEnabled: conf.LocalAuthEnabled}
	userChangeNotifier := new(api.UserChangeNotifier)
	userHandler := api.UserAPI{DB: db, PasswordStrength: conf.PassStrength, UserChangeNotifier: userChangeNotifier, Registration: conf.Registration}
	mfaHandler := api.MFAAPI{DB: db}
	passkeyHandler := api.PasskeyAPI{DB: db, SecureCookie: conf.Server.SecureCookie}
	var ldapHandler *api.LDAPAPI
	if conf.LDAP.Enabled {
		ldapHandler = api.NewLDAP(conf, db, userChangeNotifier)
	}
	auditHandler := api.AuditAPI{DB: db}
	systemHandler := api.SystemAPI{
		DB: db,
		Dialect: conf.Database.Dialect,
		DataDir: operations.DataDirectory(conf.Database.Dialect, conf.Database.Connection),
		DatabaseFile: operations.DatabaseFile(conf.Database.Dialect, conf.Database.Connection),
		VersionInfo: vInfo,
		NotifyDeleted: streamHandler.NotifyDeletedClient,
	}
	groupHandler := api.UserGroupAPI{DB: db}
	updateHandler := api.NewUpdateAPIFromEnv()
	automationHandler := api.AutomationAPI{
		DB: db,
		Engine: automationEngine,
		WebhookLimiter: security.NewDynamicLimiter(),
		WebhookReplay: security.NewReplayCache(),
	}
	collaborationHandler := api.CollaborationAPI{
		DB: db,
		Dispatcher: automationEngine,
		AttachmentDir: attachmentDir,
	}
	connectorHandler := api.ConnectorAPI{DB: db, Runtime: connectorManager}
	serviceHandler := api.ServiceAccountAPI{DB: db, Publisher: automationEngine}
	loginLimiter := security.NewFixedWindowLimiter(10, 5*time.Minute)

	pluginManager, err := plugin.NewManager(db, conf.PluginsDir, g.Group("/plugin/:id/custom/"), streamHandler)
	if err != nil {
		panic(err)
	}
	pluginManager.SetDispatcher(automationEngine)
	pluginHandler := api.PluginAPI{
		Manager:  pluginManager,
		Notifier: streamHandler,
		DB:       db,
	}

	userChangeNotifier.OnUserDeleted(streamHandler.NotifyDeletedUser)
	userChangeNotifier.OnUserDeleted(pluginManager.RemoveUser)
	userChangeNotifier.OnUserAdded(pluginManager.InitializeForUserID)

	ui.Register(
		g, *vInfo, conf.Registration, conf.LocalAuthEnabled,
		conf.OIDC.Enabled, conf.OIDC.IDPName, conf.OIDC.AutoRedirect,
		conf.LDAP.Enabled, conf.LDAP.IDPName,
	)

	loginRateLimit := security.RateLimitMiddleware(loginLimiter, func(ctx *gin.Context) string { return ctx.ClientIP() })
	g.POST("/auth/passkey/login/options", loginRateLimit, passkeyHandler.LoginOptions)
	g.POST("/auth/passkey/login/verify", loginRateLimit, passkeyHandler.LoginVerify)
	if conf.LDAP.Enabled {
		g.POST("/auth/ldap/login", loginRateLimit, ldapHandler.Login)
	}

	if conf.OIDC.Enabled {
		oidcHandler := api.NewOIDC(conf, db, userChangeNotifier)
		oidcGroup := g.Group("/auth/oidc")
		oidcGroup.GET("/login", oidcHandler.LoginHandler())
		oidcGroup.GET("/callback", oidcHandler.CallbackHandler())
		oidcGroup.POST("/external/authorize", oidcHandler.ExternalAuthorizeHandler)
		oidcGroup.POST("/external/token", oidcHandler.ExternalTokenHandler)
		oidcGroup.GET("/elevate", oidcHandler.ElevateHandler)
	}

	g.Match([]string{"GET", "HEAD"}, "/health", healthHandler.Health)
	g.POST("/integrations/webhook/:secret", automationHandler.ReceiveWebhook)
	g.GET("/swagger", docs.Serve)
	g.StaticFS("/image", &onlyImageFS{inner: gin.Dir(conf.UploadedImagesDir, false)})

	g.GET("/docs", docs.UI)

	g.GET("/service/channels", auth.RequireServiceScope(db, "channel:read"), serviceHandler.GetChannels)
	g.POST("/service/application/:id/message", auth.RequireServiceScope(db, "message:write"), serviceHandler.PublishMessage)
	g.GET("/service/application/:id/message", auth.RequireServiceScope(db, "message:read"), serviceHandler.GetMessages)

	g.Use(func(ctx *gin.Context) {
		ctx.Header("Content-Type", "application/json")
		for header, value := range conf.Server.ResponseHeaders {
			ctx.Header(header, value)
		}
	})
	g.Use(cors.New(auth.CorsConfig(conf)))

	{
		g.GET("/plugin", authentication.RequireClient, pluginHandler.GetPlugins)
		g.GET("/plugin/catalog", authentication.RequireAdmin, pluginHandler.GetCatalog)
		g.POST("/plugin/catalog/install", authentication.RequireAdmin, pluginHandler.InstallCatalogPlugin)
		g.POST("/plugin/install", authentication.RequireAdmin, pluginHandler.InstallPlugin)
		g.POST("/plugin/:id/update", authentication.RequireAdmin, pluginHandler.StagePluginUpdate)
		g.DELETE("/plugin/:id/uninstall", authentication.RequireAdmin, pluginHandler.UninstallPlugin)
		pluginRoute := g.Group("/plugin/", authentication.RequireClient)
		{
			pluginRoute.GET("/:id/config", pluginHandler.GetConfig)
			pluginRoute.POST("/:id/config", pluginHandler.UpdateConfig)
			pluginRoute.GET("/:id/display", pluginHandler.GetDisplay)
			pluginRoute.POST("/:id/enable", pluginHandler.EnablePlugin)
			pluginRoute.POST("/:id/disable", pluginHandler.DisablePlugin)
		}
	}

	g.Group("/user").Use(authentication.OptionalAdmin).POST("", userHandler.CreateUser)

	g.POST("/auth/local/login", loginRateLimit, sessionHandler.Login)

	g.OPTIONS("/*any")

	// swagger:operation GET /version info getVersion
	//
	// Get version information.
	//
	// ---
	// produces: [application/json]
	// responses:
	//   200:
	//     description: Ok
	//     schema:
	//         $ref: "#/definitions/VersionInfo"
	g.GET("version", func(ctx *gin.Context) {
		ctx.JSON(200, vInfo)
	})

	// swagger:operation GET /gotifyinfo info getInfo
	//
	// Get gotify information.
	//
	// ---
	// produces: [application/json]
	// responses:
	//   200:
	//     description: Ok
	//     schema:
	//         $ref: "#/definitions/GotifyInfo"
	g.GET("gotifyinfo", func(ctx *gin.Context) {
		ctx.JSON(200, &model.GotifyInfo{
			Version:          vInfo.Version,
			Oidc:             conf.OIDC.Enabled,
			Register:         conf.Registration,
			LocalAuth:        conf.LocalAuthEnabled,
			OIDCIDPName:      conf.OIDC.IDPName,
			OIDCAutoRedirect: conf.OIDC.AutoRedirect,
			LDAP:             conf.LDAP.Enabled,
			LDAPIDPName:      conf.LDAP.IDPName,
		})
	})

	g.Group("/").Use(authentication.RequireApplicationOrClient).POST("/message", messageHandler.CreateMessage)

	clientAuth := g.Group("")
	{
		clientAuth.Use(authentication.RequireClient)
		app := clientAuth.Group("/application")
		{
			app.GET("", applicationHandler.GetApplications)
			app.POST("", applicationHandler.CreateApplication)
			app.POST("/:id/image", applicationHandler.UploadApplicationImage)
			app.DELETE("/:id/image", applicationHandler.RemoveApplicationImage)
			app.PUT("/:id", applicationHandler.UpdateApplication)
			app.PUT("/:id/notifications", applicationMembershipHandler.SetCurrentUserNotifications)

			tokenMessage := app.Group("/:id/message")
			{
				tokenMessage.GET("", messageHandler.GetMessagesWithApplication)
				tokenMessage.DELETE("", messageHandler.DeleteMessageWithApplication)
				tokenMessage.POST("/archive", messageHandler.ArchiveMessageWithApplication)
				tokenMessage.DELETE("/archive", messageHandler.UnarchiveMessageWithApplication)
			}
		}

		client := clientAuth.Group("/client")
		{
			client.GET("", clientHandler.GetClients)
			client.POST("", clientHandler.CreateClient)
			client.PUT("/:id", clientHandler.UpdateClient)
		}

		message := clientAuth.Group("/message")
		{
			message.GET("", messageHandler.GetMessages)
			message.DELETE("", messageHandler.DeleteMessages)
			message.POST("/archive", messageHandler.ArchiveMessages)
			message.DELETE("/archive", messageHandler.UnarchiveMessages)
			message.DELETE("/:id", messageHandler.DeleteMessage)
			message.POST("/:id/archive", messageHandler.ArchiveMessage)
			message.DELETE("/:id/archive", messageHandler.UnarchiveMessage)
		message.GET("/:id/acknowledgement", automationHandler.GetAcknowledgement)
		message.POST("/:id/acknowledgement", automationHandler.AcknowledgeMessage)
		message.DELETE("/:id/acknowledgement", automationHandler.UnacknowledgeMessage)
		message.GET("/:id/thread", collaborationHandler.Thread)
		message.POST("/:id/reply", collaborationHandler.Reply)
		message.POST("/:id/reaction", collaborationHandler.AddReaction)
		message.DELETE("/:id/reaction", collaborationHandler.DeleteReaction)
		message.PUT("/:id/assignment", collaborationHandler.Assign)
		message.PUT("/:id/status", collaborationHandler.SetStatus)
		message.POST("/:id/read", collaborationHandler.MarkRead)
		message.DELETE("/:id/read", collaborationHandler.MarkUnread)
		message.POST("/:id/attachment", collaborationHandler.UploadAttachment)
		message.GET("/:id/attachment/:attachmentId", collaborationHandler.DownloadAttachment)
		message.DELETE("/:id/attachment/:attachmentId", collaborationHandler.DeleteAttachment)
		}

		clientAuth.GET("/stream", streamHandler.Handle)
		clientAuth.GET("current/user", userHandler.GetCurrentUser)
		clientAuth.GET("/current/user/mfa/status", mfaHandler.Status)
		clientAuth.GET("/current/user/passkeys", passkeyHandler.List)
		clientAuth.POST("/current/user/passkeys/elevate/options", passkeyHandler.ElevationOptions)
		clientAuth.POST("/current/user/passkeys/elevate/verify", passkeyHandler.ElevationVerify)
		if conf.LDAP.Enabled {
			clientAuth.POST("/auth/ldap/elevate", ldapHandler.Elevate)
		}
		clientAuth.POST("/auth/logout", sessionHandler.Logout)
		clientAuth.GET("/automation/quiet-hours", automationHandler.GetQuietHours)
		clientAuth.PUT("/automation/quiet-hours", automationHandler.SaveQuietHours)
		clientAuth.GET("/automation/digest", automationHandler.GetDigest)
		clientAuth.PUT("/automation/digest", automationHandler.SaveDigest)
		clientAuth.GET("/message/search", collaborationHandler.Search)
		clientAuth.GET("/message-template", collaborationHandler.GetTemplates)
		clientAuth.POST("/message-template", collaborationHandler.SaveTemplate)
		clientAuth.PUT("/message-template/:id", collaborationHandler.UpdateTemplate)
		clientAuth.DELETE("/message-template/:id", collaborationHandler.DeleteTemplate)
		clientAuth.GET("/saved-search", collaborationHandler.GetSavedSearches)
		clientAuth.POST("/saved-search", collaborationHandler.SaveSearch)
		clientAuth.PUT("/saved-search/:id", collaborationHandler.UpdateSearch)
		clientAuth.DELETE("/saved-search/:id", collaborationHandler.DeleteSearch)
	}

	clientElevated := g.Group("")
	{
		clientElevated.Use(authentication.RequireElevatedClient)
		clientElevated.POST("/client/:id/elevate", clientHandler.ElevateClient)
		clientElevated.DELETE("/application/:id", applicationHandler.DeleteApplication)
		clientElevated.PUT("/application/:id/security", applicationHandler.UpdateApplicationSecurity)
		clientElevated.GET("/application/:id/members", applicationMembershipHandler.GetMembers)
		clientElevated.GET("/application/:id/assignable-users", applicationMembershipHandler.GetAssignableUsers)
		clientElevated.GET("/application/:id/assignable-groups", applicationMembershipHandler.GetAssignableGroups)
		clientElevated.GET("/application/:id/groups", applicationMembershipHandler.GetGroupAssignments)
		clientElevated.POST("/application/:id/groups", applicationMembershipHandler.UpsertGroupAssignment)
		clientElevated.DELETE("/application/:id/groups/:groupId", applicationMembershipHandler.DeleteGroupAssignment)
		clientElevated.POST("/application/:id/members", applicationMembershipHandler.UpsertMember)
		clientElevated.DELETE("/application/:id/members/:userId", applicationMembershipHandler.DeleteMember)
		clientElevated.PUT("/application/:id/auto-assign", applicationMembershipHandler.SetAutoAssign)
		clientElevated.PUT("/application/:id/owner", applicationMembershipHandler.TransferOwnership)
		clientElevated.PUT("/application/:id/member-posting", applicationMembershipHandler.SetMemberPosting)
		clientElevated.DELETE("/application/:id/message/all", messageHandler.DeleteMessagesForEveryone)
		clientElevated.DELETE("/client/:id", clientHandler.DeleteClient)
		clientElevated.POST("/current/user/password", userHandler.ChangePassword)
		clientElevated.POST("/current/user/mfa/setup", mfaHandler.Setup)
		clientElevated.POST("/current/user/mfa/enable", mfaHandler.Enable)
		clientElevated.POST("/current/user/mfa/disable", mfaHandler.Disable)
		clientElevated.POST("/current/user/mfa/recovery-codes", mfaHandler.RegenerateRecoveryCodes)
		clientElevated.POST("/current/user/passkeys/options", passkeyHandler.RegistrationOptions)
		clientElevated.POST("/current/user/passkeys", passkeyHandler.RegistrationVerify)
		clientElevated.DELETE("/current/user/passkeys/:id", passkeyHandler.Delete)
	}

	authAdmin := g.Group("/user")
	{
		authAdmin.Use(authentication.RequireAdmin)
		authAdmin.GET("", userHandler.GetUsers)
		authAdmin.DELETE("/:id", userHandler.DeleteUserByID)
		authAdmin.GET("/:id", userHandler.GetUserByID)
		authAdmin.POST("/:id", userHandler.UpdateUserByID)
	}

	adminPlatform := g.Group("")
	{
		adminPlatform.Use(authentication.RequireAdmin)
		adminPlatform.GET("/audit", auditHandler.GetAuditEvents)
		adminPlatform.GET("/audit/export", systemHandler.ExportAudit)
		adminPlatform.POST("/audit/retention/apply", systemHandler.ApplyAuditRetention)
		adminPlatform.GET("/admin/security-policy", systemHandler.GetSecurityPolicy)
		adminPlatform.PUT("/admin/security-policy", systemHandler.SaveSecurityPolicy)
		adminPlatform.GET("/admin/operations", systemHandler.GetOperations)
		adminPlatform.GET("/admin/backup", systemHandler.DownloadBackup)
		adminPlatform.POST("/admin/restore", systemHandler.StageRestore)
		adminPlatform.DELETE("/admin/restore", systemHandler.CancelRestore)
		adminPlatform.GET("/admin/diagnostics", systemHandler.DownloadDiagnostics)
		adminPlatform.GET("/admin/sessions", systemHandler.GetSessions)
		adminPlatform.DELETE("/admin/sessions/:id", systemHandler.RevokeSession)
		adminPlatform.GET("/admin/service-accounts", serviceHandler.GetAccounts)
		adminPlatform.POST("/admin/service-accounts", serviceHandler.CreateAccount)
		adminPlatform.PUT("/admin/service-accounts/:id", serviceHandler.UpdateAccount)
		adminPlatform.DELETE("/admin/service-accounts/:id", serviceHandler.DeleteAccount)
		adminPlatform.GET("/admin/service-accounts/:id/tokens", serviceHandler.GetTokens)
		adminPlatform.POST("/admin/service-accounts/:id/tokens", serviceHandler.CreateToken)
		adminPlatform.DELETE("/admin/service-accounts/:id/tokens/:tokenId", serviceHandler.DeleteToken)
		adminPlatform.GET("/group", groupHandler.GetGroups)
		adminPlatform.POST("/group", groupHandler.CreateGroup)
		adminPlatform.PUT("/group/:id", groupHandler.UpdateGroup)
		adminPlatform.DELETE("/group/:id", groupHandler.DeleteGroup)
		adminPlatform.GET("/group/:id/members", groupHandler.GetMembers)
		adminPlatform.POST("/group/:id/members", groupHandler.AddMember)
		adminPlatform.DELETE("/group/:id/members/:userId", groupHandler.RemoveMember)
		adminPlatform.GET("/update/status", updateHandler.Status)
		adminPlatform.POST("/update/install", updateHandler.Install)

		adminPlatform.GET("/integration/webhook", automationHandler.GetWebhookRoutes)
		adminPlatform.POST("/integration/webhook", automationHandler.CreateWebhookRoute)
		adminPlatform.PUT("/integration/webhook/:id", automationHandler.UpdateWebhookRoute)
		adminPlatform.POST("/integration/webhook/:id/regenerate", automationHandler.RegenerateWebhookSecret)
		adminPlatform.POST("/integration/webhook/:id/test", automationHandler.TestWebhookRoute)
		adminPlatform.DELETE("/integration/webhook/:id", automationHandler.DeleteWebhookRoute)

		adminPlatform.GET("/integration/mqtt", automationHandler.GetMQTT)
		adminPlatform.POST("/integration/mqtt", automationHandler.CreateMQTT)
		adminPlatform.PUT("/integration/mqtt/:id", automationHandler.UpdateMQTT)
		adminPlatform.POST("/integration/mqtt/:id/test", automationHandler.TestMQTT)
		adminPlatform.DELETE("/integration/mqtt/:id", automationHandler.DeleteMQTT)

		adminPlatform.GET("/integration/home-assistant", automationHandler.GetHomeAssistant)
		adminPlatform.POST("/integration/home-assistant", automationHandler.CreateHomeAssistant)
		adminPlatform.PUT("/integration/home-assistant/:id", automationHandler.UpdateHomeAssistant)
		adminPlatform.POST("/integration/home-assistant/:id/event", automationHandler.SendHomeAssistantEvent)
		adminPlatform.DELETE("/integration/home-assistant/:id", automationHandler.DeleteHomeAssistant)

		adminPlatform.GET("/automation/schedule", automationHandler.GetSchedules)
		adminPlatform.POST("/automation/schedule", automationHandler.CreateSchedule)
		adminPlatform.PUT("/automation/schedule/:id", automationHandler.UpdateSchedule)
		adminPlatform.GET("/automation/schedule/:id/runs", automationHandler.GetScheduleRuns)
		adminPlatform.DELETE("/automation/schedule/:id", automationHandler.DeleteSchedule)

		adminPlatform.GET("/automation/escalation", automationHandler.GetEscalations)
		adminPlatform.POST("/automation/escalation", automationHandler.CreateEscalation)
		adminPlatform.PUT("/automation/escalation/:id", automationHandler.UpdateEscalation)
		adminPlatform.DELETE("/automation/escalation/:id", automationHandler.DeleteEscalation)

		adminPlatform.GET("/connector/email", connectorHandler.GetEmailGateways)
		adminPlatform.POST("/connector/email", connectorHandler.CreateEmailGateway)
		adminPlatform.PUT("/connector/email/:id", connectorHandler.UpdateEmailGateway)
		adminPlatform.POST("/connector/email/:id/test", connectorHandler.TestEmailGateway)
		adminPlatform.DELETE("/connector/email/:id", connectorHandler.DeleteEmailGateway)

		adminPlatform.GET("/connector/smtp", connectorHandler.GetSMTPRoutes)
		adminPlatform.POST("/connector/smtp", connectorHandler.CreateSMTPRoute)
		adminPlatform.PUT("/connector/smtp/:id", connectorHandler.UpdateSMTPRoute)
		adminPlatform.DELETE("/connector/smtp/:id", connectorHandler.DeleteSMTPRoute)

		adminPlatform.GET("/connector/rss", connectorHandler.GetRSS)
		adminPlatform.POST("/connector/rss", connectorHandler.CreateRSS)
		adminPlatform.PUT("/connector/rss/:id", connectorHandler.UpdateRSS)
		adminPlatform.DELETE("/connector/rss/:id", connectorHandler.DeleteRSS)

		adminPlatform.GET("/connector/syslog", connectorHandler.GetSyslog)
		adminPlatform.POST("/connector/syslog", connectorHandler.CreateSyslog)
		adminPlatform.PUT("/connector/syslog/:id", connectorHandler.UpdateSyslog)
		adminPlatform.DELETE("/connector/syslog/:id", connectorHandler.DeleteSyslog)

		adminPlatform.GET("/connector/calendar", connectorHandler.GetCalendars)
		adminPlatform.POST("/connector/calendar", connectorHandler.CreateCalendar)
		adminPlatform.PUT("/connector/calendar/:id", connectorHandler.UpdateCalendar)
		adminPlatform.DELETE("/connector/calendar/:id", connectorHandler.DeleteCalendar)
	}
	return g, func() {
		close(maintenanceStop)
		connectorManager.Close()
		automationEngine.Close()
		streamHandler.Close()
	}
}

func auditMutations(db *database.GormDatabase) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Next()

		method := ctx.Request.Method
		if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
			return
		}
		if ctx.Writer.Status() >= 400 {
			return
		}

		path := ctx.FullPath()
		if path == "" {
			path = ctx.Request.URL.Path
		}
		if !shouldAuditMutation(path) {
			return
		}

		event := &model.AuditEvent{
			Action:    strings.ToLower(method),
			Target:    path,
			IPAddress: ctx.ClientIP(),
		}
		if id := ctx.Param("id"); id != "" {
			event.TargetID = id
		} else if userID := ctx.Param("userId"); userID != "" {
			event.TargetID = userID
		}
		if userID := auth.TryGetUserID(ctx); userID != nil {
			event.UserID = *userID
			if user, err := db.GetUserByID(*userID); err == nil && user != nil {
				event.Username = user.Name
			}
		}
		if err := db.CreateAuditEvent(event); err != nil {
			log.Error().Err(err).Str("path", path).Msg("Could not persist audit event")
		}
	}
}

func shouldAuditMutation(path string) bool {
	switch {
	case path == "/auth/logout":
		return true
	case path == "/current/user/password":
		return true
	case strings.HasPrefix(path, "/user"):
		return true
	case strings.HasPrefix(path, "/client"):
		return true
	case strings.HasPrefix(path, "/group"):
		return true
	case path == "/plugin/install":
		return true
	case strings.HasPrefix(path, "/plugin/") && !strings.Contains(path, "/custom/"):
		return true
	case strings.HasPrefix(path, "/application") && !strings.Contains(path, "/message"):
		return true
	case strings.HasPrefix(path, "/update"):
		return true
	case strings.HasPrefix(path, "/integration"):
		return true
	case strings.HasPrefix(path, "/connector"):
		return true
	case strings.HasPrefix(path, "/admin"):
		return true
	case strings.HasPrefix(path, "/automation"):
		return true
	case strings.Contains(path, "/acknowledgement"):
		return true
	case strings.HasPrefix(path, "/message-template"):
		return true
	case strings.HasPrefix(path, "/saved-search"):
		return true
	case strings.HasPrefix(path, "/message/") &&
		(strings.Contains(path, "/reply") || strings.Contains(path, "/reaction") ||
			strings.Contains(path, "/assignment") || strings.Contains(path, "/status") ||
			strings.Contains(path, "/read") || strings.Contains(path, "/attachment")):
		return true
	default:
		return false
	}
}

func sanitizeRequestPath(path, rawQuery string) string {
	if strings.HasPrefix(path, "/integrations/webhook/") {
		path = "/integrations/webhook/[masked]"
	}
	if rawQuery == "" {
		return path
	}
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return path + "?[masked]"
	}
	for key := range values {
		switch strings.ToLower(key) {
		case "token", "code", "state", "access_token", "id_token", "secret", "key", "password", "client_secret", "refresh_token":
			values.Set(key, "[masked]")
		}
	}
	return path + "?" + values.Encode()
}

func accessLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		rawQuery := c.Request.URL.RawQuery
		path := c.Request.URL.Path

		c.Next()

		clientIP := c.ClientIP()
		if (clientIP == "127.0.0.1" || clientIP == "::1") && path == "/health" {
			return
		}

		path = sanitizeRequestPath(path, rawQuery)

		latency := time.Since(start)
		if latency > time.Minute {
			latency = latency - latency%time.Second
		}

		status := c.Writer.Status()
		evt := log.Info()
		switch {
		case status >= 500:
			evt = log.Error()
		case status >= 400:
			evt = log.Warn()
		}

		evt.
			Int("status", status).
			Str("duration", latency.String()).
			Str("ip", clientIP).
			Str("method", c.Request.Method).
			Str("path", path)

		if errs := c.Errors.ByType(gin.ErrorTypePrivate).String(); errs != "" {
			evt.Str("errors", strings.TrimSpace(errs))
		}

		evt.Msg("HTTP")
	}
}

type onlyImageFS struct {
	inner http.FileSystem
}

func (fs *onlyImageFS) Open(name string) (http.File, error) {
	ext := filepath.Ext(name)
	if !api.ValidApplicationImageExt(ext) {
		return nil, fmt.Errorf("invalid file")
	}
	return fs.inner.Open(name)
}

func cleanupOrphanAttachments(db *database.GormDatabase, directory string) error {
	names, err := db.GetAttachmentStorageNames()
	if err != nil { return err }
	keep := make(map[string]struct{}, len(names))
	for _, name := range names { keep[filepath.Base(name)] = struct{}{} }
	entries, err := os.ReadDir(directory)
	if errors.Is(err, os.ErrNotExist) { return nil }
	if err != nil { return err }
	for _, entry := range entries {
		if entry.IsDir() { continue }
		if _, ok := keep[entry.Name()]; ok { continue }
		if err := os.Remove(filepath.Join(directory, entry.Name())); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}
