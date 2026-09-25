package router

import (
	"fmt"
	"net/http"
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
	"github.com/gotify/server/v3/config"
	"github.com/gotify/server/v3/database"
	"github.com/gotify/server/v3/docs"
	gerror "github.com/gotify/server/v3/error"
	"github.com/gotify/server/v3/model"
	"github.com/gotify/server/v3/plugin"
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
	authentication := auth.Auth{
		DB:               db,
		SecureCookie:     conf.Server.SecureCookie,
		LocalAuthEnabled: conf.LocalAuthEnabled,
		CrossOrigin:      http.NewCrossOriginProtection(),
	}
	messageHandler := api.MessageAPI{Notifier: streamHandler, DB: db}
	healthHandler := api.HealthAPI{DB: db}
	clientHandler := api.ClientAPI{
		DB:            db,
		ImageDir:      conf.UploadedImagesDir,
		NotifyDeleted: streamHandler.NotifyDeletedClient,
	}
	applicationHandler := api.ApplicationAPI{
		DB:       db,
		ImageDir: conf.UploadedImagesDir,
	}
	applicationMembershipHandler := api.ApplicationMembershipAPI{
		DB: db,
	}
	sessionHandler := api.SessionAPI{DB: db, NotifyDeleted: streamHandler.NotifyDeletedClient, SecureCookie: conf.Server.SecureCookie, LocalAuthEnabled: conf.LocalAuthEnabled}
	userChangeNotifier := new(api.UserChangeNotifier)
	userHandler := api.UserAPI{DB: db, PasswordStrength: conf.PassStrength, UserChangeNotifier: userChangeNotifier, Registration: conf.Registration}
	auditHandler := api.AuditAPI{DB: db}
	groupHandler := api.UserGroupAPI{DB: db}

	pluginManager, err := plugin.NewManager(db, conf.PluginsDir, g.Group("/plugin/:id/custom/"), streamHandler)
	if err != nil {
		panic(err)
	}
	pluginHandler := api.PluginAPI{
		Manager:  pluginManager,
		Notifier: streamHandler,
		DB:       db,
	}

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
		}

		clientAuth.GET("/stream", streamHandler.Handle)
		clientAuth.GET("current/user", userHandler.GetCurrentUser)
		clientAuth.POST("/auth/logout", sessionHandler.Logout)
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
		clientElevated.PUT("/application/:id/auto-assign", applicationMembershipHandler.SetAutoAssign)
		clientElevated.PUT("/application/:id/owner", applicationMembershipHandler.TransferOwnership)
		clientElevated.PUT("/application/:id/member-posting", applicationMembershipHandler.SetMemberPosting)
		clientElevated.DELETE("/application/:id/message/all", messageHandler.DeleteMessagesForEveryone)
		clientElevated.DELETE("/client/:id", clientHandler.DeleteClient)
		clientElevated.POST("/current/user/password", userHandler.ChangePassword)
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
		adminPlatform.GET("/group", groupHandler.GetGroups)
		adminPlatform.POST("/group", groupHandler.CreateGroup)
		adminPlatform.PUT("/group/:id", groupHandler.UpdateGroup)
		adminPlatform.DELETE("/group/:id", groupHandler.DeleteGroup)
		adminPlatform.GET("/group/:id/members", groupHandler.GetMembers)
		adminPlatform.POST("/group/:id/members", groupHandler.AddMember)
		adminPlatform.DELETE("/group/:id/members/:userId", groupHandler.RemoveMember)
	}
	return g, streamHandler.Close
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
	default:
		return false
	}
}

var tokenRegexp = regexp.MustCompile("token=[^&]+")

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

		if rawQuery != "" {
			path = path + "?" + rawQuery
		}
		path = tokenRegexp.ReplaceAllString(path, "token=[masked]")

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
