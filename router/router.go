package router

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
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
	"github.com/gotify/server/v3/database"
	"github.com/gotify/server/v3/docs"
	gerror "github.com/gotify/server/v3/error"
	"github.com/gotify/server/v3/model"
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
	streamHandler := stream.New(
		time.Duration(conf.Server.Stream.PingPeriodSeconds)*time.Second, 15*time.Second, conf.Server.Stream.AllowedOrigins,
	)
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		for range ticker.C {
			connectedTokens := streamHandler.CollectConnectedClientTokens()
			now := time.Now()
			if err := db.UpdateClientTokensLastUsedAndExpiresAt(connectedTokens, &now); err != nil {
				log.Error().Err(err).Msg("Error updating last used")
			}
			if expired, err := db.CleanupExpiredClients(now); err == nil {
				for _, c := range expired {
					streamHandler.NotifyDeletedClient(c.UserID, c.Token)
				}
			} else {
				log.Error().Err(err).Msg("Error cleaning up expired clients")
			}
		}
	}()
	loginLimiter := security.NewLimiter(12, 5)
	webhookLimiter := security.NewLimiter(240, 30)
	policyProvider := func() *model.SecurityPolicy {
		policy, err := db.GetSecurityPolicy()
		if err != nil {
			log.Error().Err(err).Msg("Could not load security policy")
			return model.DefaultSecurityPolicy()
		}
		return policy
	}
	authentication := auth.Auth{
		DB:               db,
		SecureCookie:     conf.Server.SecureCookie,
		LocalAuthEnabled: conf.LocalAuthEnabled,
		CrossOrigin:      http.NewCrossOriginProtection(),
		LoginLimiter:     loginLimiter,
	}
	automationEngine := automation.New(db, streamHandler)
	messageHandler := api.MessageAPI{Notifier: streamHandler, DB: db, Dispatcher: automationEngine}
	healthHandler := api.HealthAPI{DB: db}
	clientHandler := api.ClientAPI{
		DB:            db,
		ImageDir:      conf.UploadedImagesDir,
		NotifyDeleted: streamHandler.NotifyDeletedClient,
		Policy:        policyProvider,
	}
	applicationHandler := api.ApplicationAPI{
		DB:       db,
		ImageDir: conf.UploadedImagesDir,
		OnDelete: func(uint) { automationEngine.ReloadIntegrations() },
	}
	applicationMembershipHandler := api.ApplicationMembershipAPI{
		DB: db,
	}
	sessionHandler := api.SessionAPI{DB: db, NotifyDeleted: streamHandler.NotifyDeletedClient, SecureCookie: conf.Server.SecureCookie, LocalAuthEnabled: conf.LocalAuthEnabled, Policy: policyProvider, LoginLimiter: loginLimiter}
	userChangeNotifier := new(api.UserChangeNotifier)
	userHandler := api.UserAPI{DB: db, PasswordStrength: conf.PassStrength, UserChangeNotifier: userChangeNotifier, Registration: conf.Registration, Policy: policyProvider}
	auditHandler := api.AuditAPI{DB: db}
	groupHandler := api.UserGroupAPI{DB: db}
	updateHandler := api.NewUpdateAPIFromEnv()
	automationHandler := api.AutomationAPI{DB: db, Engine: automationEngine}
	securityPolicyHandler := api.SecurityPolicyAPI{DB: db}
	mfaHandler := api.MFAAPI{DB: db}
	operationsHandler := api.OperationsAPI{
		DB: db,
		NotifyDeleted: streamHandler.NotifyDeletedClient,
		DatabaseDialect: conf.Database.Dialect,
		DatabaseConnection: conf.Database.Connection,
		DataPaths: []string{conf.UploadedImagesDir, conf.PluginsDir, filepath.Dir(conf.Database.Connection)},
		UploadedImagesDir: conf.UploadedImagesDir,
		PluginsDir: conf.PluginsDir,
		Version: vInfo,
	}

	pluginManager, err := plugin.NewManager(db, conf.PluginsDir, g.Group("/plugin/:id/custom/"), streamHandler)
	if err != nil {
		panic(err)
	}
	pluginManager.SetMessageDispatcher(automationEngine)
	pluginHandler := api.PluginAPI{
		Manager:  pluginManager,
		Notifier: streamHandler,
		DB:       db,
		Policy:   policyProvider,
	}

	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				now := time.Now()
				loginLimiter.Cleanup(24*time.Hour)
				webhookLimiter.Cleanup(24*time.Hour)
				policy := policyProvider()
				if policy.AuditRetentionDays > 0 {
					if err := db.DeleteAuditEventsBefore(now.AddDate(0, 0, -policy.AuditRetentionDays)); err != nil {
						log.Error().Err(err).Msg("Could not apply audit retention policy")
					}
				}
				if policy.AutomationRetentionDays > 0 {
					if err := db.DeleteAutomationRunsBefore(now.AddDate(0, 0, -policy.AutomationRetentionDays)); err != nil {
						log.Error().Err(err).Msg("Could not apply automation history retention policy")
					}
				}
			}
		}
	}()

	userChangeNotifier.OnUserDeleted(streamHandler.NotifyDeletedUser)
	userChangeNotifier.OnUserDeleted(pluginManager.RemoveUser)
	userChangeNotifier.OnUserAdded(pluginManager.InitializeForUserID)

	ui.Register(g, *vInfo, conf.Registration, conf.LocalAuthEnabled, conf.OIDC.Enabled, conf.OIDC.IDPName, conf.OIDC.AutoRedirect)

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
	g.POST("/integrations/webhook/:secret", func(ctx *gin.Context) {
		if !webhookLimiter.Allow(ctx.ClientIP()) {
			ctx.Header("Retry-After", "60")
			ctx.AbortWithStatus(http.StatusTooManyRequests)
			return
		}
		automationHandler.ReceiveWebhook(ctx)
	})
	g.GET("/swagger", docs.Serve)
	g.StaticFS("/image", &onlyImageFS{inner: gin.Dir(conf.UploadedImagesDir, false)})

	g.GET("/docs", docs.UI)

	g.Use(func(ctx *gin.Context) {
		ctx.Header("Content-Type", "application/json")
		for header, value := range conf.Server.ResponseHeaders {
			ctx.Header(header, value)
		}
	})
	g.Use(cors.New(auth.CorsConfig(conf)))

	{
		g.GET("/plugin", authentication.RequireClient, pluginHandler.GetPlugins)
		g.POST("/plugin/install", authentication.RequireAdmin, pluginHandler.InstallPlugin)
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

	g.POST("/auth/local/login", sessionHandler.Login)

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
		}

		clientAuth.GET("/stream", streamHandler.Handle)
		clientAuth.GET("current/user", userHandler.GetCurrentUser)
		clientAuth.POST("/auth/logout", sessionHandler.Logout)
		clientAuth.GET("/automation/quiet-hours", automationHandler.GetQuietHours)
		clientAuth.PUT("/automation/quiet-hours", automationHandler.SaveQuietHours)
		clientAuth.GET("/automation/digest", automationHandler.GetDigest)
		clientAuth.PUT("/automation/digest", automationHandler.SaveDigest)
		clientAuth.GET("/security/mfa", mfaHandler.Status)
	}

	clientElevated := g.Group("")
	{
		clientElevated.Use(authentication.RequireElevatedClient)
		clientElevated.POST("/client/:id/elevate", clientHandler.ElevateClient)
		clientElevated.DELETE("/application/:id", applicationHandler.DeleteApplication)
		clientElevated.PUT("/application/:id/security", applicationHandler.UpdateApplicationSecurity)
		clientElevated.GET("/application/:id/members", applicationMembershipHandler.GetMembers)
		clientElevated.GET("/application/:id/assignable-users", applicationMembershipHandler.GetAssignableUsers)
		clientElevated.POST("/application/:id/members", applicationMembershipHandler.UpsertMember)
		clientElevated.DELETE("/application/:id/members/:userId", applicationMembershipHandler.DeleteMember)
		clientElevated.GET("/application/:id/groups", applicationMembershipHandler.GetGroups)
		clientElevated.POST("/application/:id/groups", applicationMembershipHandler.UpsertGroup)
		clientElevated.DELETE("/application/:id/groups/:groupId", applicationMembershipHandler.DeleteGroup)
		clientElevated.PUT("/application/:id/auto-assign", applicationMembershipHandler.SetAutoAssign)
		clientElevated.PUT("/application/:id/owner", applicationMembershipHandler.TransferOwnership)
		clientElevated.PUT("/application/:id/member-posting", applicationMembershipHandler.SetMemberPosting)
		clientElevated.DELETE("/application/:id/message/all", messageHandler.DeleteMessagesForEveryone)
		clientElevated.DELETE("/client/:id", clientHandler.DeleteClient)
		clientElevated.POST("/current/user/password", userHandler.ChangePassword)
		clientElevated.POST("/security/mfa/start", mfaHandler.Start)
		clientElevated.POST("/security/mfa/enable", mfaHandler.Enable)
		clientElevated.POST("/security/mfa/disable", mfaHandler.Disable)
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
		adminPlatform.GET("/operations/sessions", operationsHandler.Sessions)
		adminPlatform.DELETE("/operations/sessions/:id", operationsHandler.RevokeSession)
		adminPlatform.GET("/operations/stats", operationsHandler.Stats)
		adminPlatform.GET("/operations/diagnostics", operationsHandler.Diagnostics)
		adminPlatform.GET("/operations/backup", operationsHandler.Backup)
		adminPlatform.POST("/operations/restore", operationsHandler.StageRestore)

		adminPlatform.GET("/security/policy", securityPolicyHandler.Get)
		adminPlatform.PUT("/security/policy", securityPolicyHandler.Save)
		adminPlatform.GET("/audit", auditHandler.GetAuditEvents)
		adminPlatform.GET("/group", groupHandler.GetGroups)
		adminPlatform.POST("/group", groupHandler.CreateGroup)
		adminPlatform.PUT("/group/:id", groupHandler.UpdateGroup)
		adminPlatform.DELETE("/group/:id", groupHandler.DeleteGroup)
		adminPlatform.GET("/group/:id/members", groupHandler.GetMembers)
		adminPlatform.POST("/group/:id/members", groupHandler.AddMember)
		adminPlatform.DELETE("/group/:id/members/:userId", groupHandler.RemoveMember)
		adminPlatform.GET("/update/status", updateHandler.Status)
		adminPlatform.POST("/update/install", updateHandler.Install)

		adminPlatform.GET("/integration/status", automationHandler.GetIntegrationStatuses)
		adminPlatform.GET("/automation/run", automationHandler.GetAutomationRuns)
		adminPlatform.DELETE("/automation/run", automationHandler.CleanupAutomationRuns)

		adminPlatform.GET("/integration/webhook", automationHandler.GetWebhookRoutes)
		adminPlatform.POST("/integration/webhook", automationHandler.CreateWebhookRoute)
		adminPlatform.PUT("/integration/webhook/:id", automationHandler.UpdateWebhookRoute)
		adminPlatform.POST("/integration/webhook/:id/regenerate", automationHandler.RegenerateWebhookSecret)
		adminPlatform.DELETE("/integration/webhook/:id", automationHandler.DeleteWebhookRoute)

		adminPlatform.GET("/integration/mqtt", automationHandler.GetMQTT)
		adminPlatform.POST("/integration/mqtt", automationHandler.CreateMQTT)
		adminPlatform.PUT("/integration/mqtt/:id", automationHandler.UpdateMQTT)
		adminPlatform.POST("/integration/mqtt/:id/test", automationHandler.TestMQTT)
		adminPlatform.DELETE("/integration/mqtt/:id", automationHandler.DeleteMQTT)

		adminPlatform.GET("/integration/home-assistant", automationHandler.GetHomeAssistant)
		adminPlatform.POST("/integration/home-assistant", automationHandler.CreateHomeAssistant)
		adminPlatform.PUT("/integration/home-assistant/:id", automationHandler.UpdateHomeAssistant)
		adminPlatform.POST("/integration/home-assistant/:id/test", automationHandler.TestHomeAssistant)
		adminPlatform.POST("/integration/home-assistant/:id/event", automationHandler.SendHomeAssistantEvent)
		adminPlatform.DELETE("/integration/home-assistant/:id", automationHandler.DeleteHomeAssistant)

		adminPlatform.GET("/integration/rss", automationHandler.GetRSS)
		adminPlatform.POST("/integration/rss", automationHandler.CreateRSS)
		adminPlatform.PUT("/integration/rss/:id", automationHandler.UpdateRSS)
		adminPlatform.DELETE("/integration/rss/:id", automationHandler.DeleteRSS)

		adminPlatform.GET("/integration/calendar", automationHandler.GetCalendars)
		adminPlatform.POST("/integration/calendar", automationHandler.CreateCalendar)
		adminPlatform.PUT("/integration/calendar/:id", automationHandler.UpdateCalendar)
		adminPlatform.DELETE("/integration/calendar/:id", automationHandler.DeleteCalendar)

		adminPlatform.GET("/integration/email-gateway", automationHandler.GetEmailGateways)
		adminPlatform.POST("/integration/email-gateway", automationHandler.CreateEmailGateway)
		adminPlatform.PUT("/integration/email-gateway/:id", automationHandler.UpdateEmailGateway)
		adminPlatform.DELETE("/integration/email-gateway/:id", automationHandler.DeleteEmailGateway)

		adminPlatform.GET("/integration/smtp-receiver", automationHandler.GetSMTPReceiver)
		adminPlatform.PUT("/integration/smtp-receiver", automationHandler.SaveSMTPReceiver)
		adminPlatform.GET("/integration/smtp-route", automationHandler.GetSMTPRoutes)
		adminPlatform.POST("/integration/smtp-route", automationHandler.CreateSMTPRoute)
		adminPlatform.PUT("/integration/smtp-route/:id", automationHandler.UpdateSMTPRoute)
		adminPlatform.DELETE("/integration/smtp-route/:id", automationHandler.DeleteSMTPRoute)

		adminPlatform.GET("/integration/syslog", automationHandler.GetSyslog)
		adminPlatform.POST("/integration/syslog", automationHandler.CreateSyslog)
		adminPlatform.PUT("/integration/syslog/:id", automationHandler.UpdateSyslog)
		adminPlatform.DELETE("/integration/syslog/:id", automationHandler.DeleteSyslog)

		adminPlatform.GET("/automation/schedule", automationHandler.GetSchedules)
		adminPlatform.POST("/automation/schedule", automationHandler.CreateSchedule)
		adminPlatform.PUT("/automation/schedule/:id", automationHandler.UpdateSchedule)
		adminPlatform.DELETE("/automation/schedule/:id", automationHandler.DeleteSchedule)

		adminPlatform.GET("/automation/escalation", automationHandler.GetEscalations)
		adminPlatform.POST("/automation/escalation", automationHandler.CreateEscalation)
		adminPlatform.PUT("/automation/escalation/:id", automationHandler.UpdateEscalation)
		adminPlatform.DELETE("/automation/escalation/:id", automationHandler.DeleteEscalation)
	}
	return g, func() {
		automationEngine.Close()
		streamHandler.Close()
	}
}

func auditMutations(db *database.GormDatabase) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		method := ctx.Request.Method
		mutation := method != http.MethodGet && method != http.MethodHead && method != http.MethodOptions
		var capturedBody []byte
		contentType := strings.ToLower(ctx.GetHeader("Content-Type"))
		if mutation && strings.Contains(contentType, "application/json") && ctx.Request.Body != nil {
			capturedBody, _ = io.ReadAll(io.LimitReader(ctx.Request.Body, 64*1024))
			ctx.Request.Body = io.NopCloser(bytes.NewReader(capturedBody))
		}

		ctx.Next()

		path := ctx.FullPath()
		if path == "" {
			path = ctx.Request.URL.Path
		}
		status := ctx.Writer.Status()
		authFailure := (status == http.StatusUnauthorized || status == http.StatusForbidden) &&
			(strings.HasPrefix(path, "/auth/") || strings.TrimSpace(ctx.GetHeader("Authorization")) != "")

		if !shouldAuditMutation(path) && !authFailure {
			return
		}
		if status >= 400 && !authFailure {
			return
		}

		action := strings.ToLower(method)
		if authFailure {
			action = "authentication_failed"
		}

		details := map[string]any{
			"status":    status,
			"userAgent": ctx.GetHeader("User-Agent"),
		}
		if len(capturedBody) > 0 && !authFailure {
			var payload any
			if json.Unmarshal(capturedBody, &payload) == nil {
				details["request"] = redactAuditValue(payload)
			}
		}
		detailJSON, _ := json.Marshal(details)

		event := &model.AuditEvent{
			Action:    action,
			Target:    path,
			IPAddress: ctx.ClientIP(),
			Details:   string(detailJSON),
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

func redactAuditValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			lower := strings.ToLower(key)
			if strings.Contains(lower, "password") ||
				strings.Contains(lower, "token") ||
				strings.Contains(lower, "secret") ||
				strings.Contains(lower, "credential") ||
				strings.Contains(lower, "authorization") ||
				strings.HasSuffix(lower, "key") {
				out[key] = "[redacted]"
				continue
			}
			out[key] = redactAuditValue(child)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i := range typed {
			out[i] = redactAuditValue(typed[i])
		}
		return out
	default:
		return value
	}
}

func shouldAuditMutation(path string) bool {
	switch {
	case path == "/auth/logout" || path == "/auth/local/login":
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
	case strings.HasPrefix(path, "/automation"):
		return true
	case strings.Contains(path, "/acknowledgement"):
		return true
	default:
		return false
	}
}

var tokenRegexp = regexp.MustCompile("(?i)(token|code|state|key|secret|password|access_token|refresh_token|id_token)=[^&]+")

func sanitizedLogPath(requestURL *url.URL) string {
	path := requestURL.Path
	if strings.HasPrefix(path, "/integrations/webhook/") {
		path = "/integrations/webhook/[masked]"
	}
	if requestURL.RawQuery == "" {
		return path
	}

	values, err := url.ParseQuery(requestURL.RawQuery)
	if err != nil {
		return tokenRegexp.ReplaceAllString(path+"?"+requestURL.RawQuery, "$1=[masked]")
	}
	for key := range values {
		lower := strings.ToLower(key)
		if strings.Contains(lower, "token") ||
			strings.Contains(lower, "secret") ||
			strings.Contains(lower, "password") ||
			lower == "code" ||
			lower == "state" ||
			lower == "key" {
			values.Set(key, "[masked]")
		}
	}
	return path + "?" + values.Encode()
}

func accessLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		clientIP := c.ClientIP()
		if (clientIP == "127.0.0.1" || clientIP == "::1") && path == "/health" {
			return
		}

		path = sanitizedLogPath(c.Request.URL)

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
