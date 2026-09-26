package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gotify/server/v3/auth"
	"github.com/gotify/server/v3/auth/password"
	"github.com/gotify/server/v3/model"
	"github.com/gotify/server/v3/security"
)

// SessionDatabase is the interface for session-related database access.
type SessionDatabase interface {
	GetUserByName(name string) (*model.User, error)
	CreateClient(client *model.Client) error
	GetClientByToken(token string) (*model.Client, error)
	DeleteClientByID(id uint) error
	GetUserMFA(userID uint) (*model.UserMFA, error)
	ConsumeMFARecoveryCode(userID uint, hash string) (bool, error)
}

// SessionAPI provides handlers for cookie-based session authentication.
type SessionAPI struct {
	DB               SessionDatabase
	NotifyDeleted    func(uint, string)
	SecureCookie     bool
	LocalAuthEnabled bool
	Directory        auth.DirectoryAuthenticator
	Policy           func() *model.SecurityPolicy
	LoginLimiter     *security.Limiter
}

// swagger:operation POST /auth/local/login auth localLogin
//
// Authenticate via basic auth and create a session.
//
//	---
//	consumes: [application/x-www-form-urlencoded]
//	produces: [application/json]
//	security:
//	- basicAuth: []
//	parameters:
//	- name: name
//	  in: formData
//	  description: the client name to create
//	  required: true
//	  type: string
//	responses:
//	  200:
//	    description: Ok
//	    schema:
//	        $ref: "#/definitions/CurrentUser"
//	    headers:
//	      Set-Cookie:
//	        type: string
//	        description: session cookie
//	  401:
//	    description: Unauthorized
//	    schema:
//	        $ref: "#/definitions/Error"
//	  403:
//	    description: Forbidden
//	    schema:
//	        $ref: "#/definitions/Error"
func (a *SessionAPI) Login(ctx *gin.Context) {
	directoryEnabled := a.Directory != nil && a.Directory.Enabled()
	if !a.LocalAuthEnabled && !directoryEnabled {
		ctx.AbortWithError(403, errors.New("password and directory sign-in are disabled"))
		return
	}

	name, pass, ok := ctx.Request.BasicAuth()
	if !ok {
		ctx.AbortWithError(401, errors.New("credentials required"))
		return
	}

	limiterKey := ctx.ClientIP() + "|" + strings.ToLower(strings.TrimSpace(name))
	if a.LoginLimiter != nil && !a.LoginLimiter.Allow(limiterKey) {
		ctx.Header("Retry-After", "60")
		ctx.AbortWithError(http.StatusTooManyRequests, errors.New("too many authentication attempts"))
		return
	}

	var user *model.User
	if a.LocalAuthEnabled {
		local, err := a.DB.GetUserByName(name)
		if err != nil {
			ctx.AbortWithError(500, err)
			return
		}
		if local != nil && password.ComparePassword(local.Pass, []byte(pass)) {
			user = local
		}
	}
	if user == nil && directoryEnabled {
		directoryUser, err := a.Directory.Authenticate(name, pass)
		if err != nil {
			ctx.AbortWithError(503, errors.New("directory authentication is temporarily unavailable"))
			return
		}
		user = directoryUser
	}
	if user == nil {
		ctx.AbortWithError(401, errors.New("invalid credentials"))
		return
	}
	if a.LoginLimiter != nil { a.LoginLimiter.Reset(limiterKey) }

	policy := model.DefaultSecurityPolicy()
	if a.Policy != nil {
		if configured := a.Policy(); configured != nil { policy = configured }
	}
	mfa, err := a.DB.GetUserMFA(user.ID)
	if err != nil {
		ctx.AbortWithError(500, err)
		return
	}
	mfaRequired := (mfa != nil && mfa.Enabled) || policy.RequireMFAAll || (policy.RequireMFAAdmins && user.Admin)
	if mfaRequired {
		if mfa == nil || !mfa.Enabled {
			ctx.AbortWithError(http.StatusForbidden, errors.New("MFA enrollment is required for this account"))
			return
		}
		code := strings.TrimSpace(ctx.GetHeader("X-Gotify-MU-MFA"))
		if code == "" {
			ctx.JSON(http.StatusPreconditionRequired, gin.H{"error":"mfa_required","errorDescription":"Enter an authenticator or recovery code."})
			return
		}
		valid, verifyErr := verifyMFAWithRecovery(a.DB, user.ID, code, time.Now())
		if verifyErr != nil {
			ctx.AbortWithError(500, verifyErr)
			return
		}
		if !valid {
			ctx.AbortWithError(401, errors.New("invalid MFA code"))
			return
		}
	}

	clientParams := ClientParams{}
	if err := ctx.Bind(&clientParams); err != nil {
		return
	}

	sessionSeconds := policy.SessionLifetimeHours * 60 * 60
	if sessionSeconds <= 0 { sessionSeconds = auth.CookieMaxAge }
	elevation := time.Duration(policy.ElevationMinutes) * time.Minute
	if elevation <= 0 { elevation = model.DefaultElevationDuration }
	elevatedUntil := time.Now().Add(elevation)
	tokenPublic, tokenPrivate := generateClientToken()
	client := model.Client{
		Name:                          clientParams.Name,
		Token:                         tokenPublic,
		UserID:                        user.ID,
		ElevatedUntil:                 &elevatedUntil,
		ExpiresAfterInactivitySeconds: uint(sessionSeconds),
	}
	if success := successOrAbort(ctx, 500, a.DB.CreateClient(&client)); !success {
		return
	}

	auth.SetCookie(ctx.Writer, tokenPrivate, sessionSeconds, a.SecureCookie)

	ctx.JSON(200, &model.CurrentUserExternal{
		ID:            user.ID,
		Name:          user.Name,
		DisplayName:   user.DisplayName,
		Admin:         user.Admin,
		DirectoryManaged: user.DirectoryManaged,
		CreatedAt:     user.CreatedAt,
		ClientID:      client.ID,
		ElevatedUntil: client.ElevatedUntil,
	})
}

// swagger:operation POST /auth/logout auth logout
//
// End the current session.
//
// Clears the session cookie and deletes the associated client.
//
//	---
//	produces: [application/json]
//	security:
//	- clientTokenHeader: []
//	- clientTokenQuery: []
//	- basicAuth: []
//	responses:
//	  200:
//	    description: Ok
//	    headers:
//	      Set-Cookie:
//	        type: string
//	        description: cleared session cookie
//	  400:
//	    description: Bad Request
//	    schema:
//	        $ref: "#/definitions/Error"
func (a *SessionAPI) Logout(ctx *gin.Context) {
	auth.SetCookie(ctx.Writer, "", -1, a.SecureCookie)

	client := auth.GetClient(ctx)
	if client == nil {
		ctx.AbortWithError(403, errors.New("no client auth provided"))
		return
	}

	a.NotifyDeleted(client.UserID, client.Token)
	if success := successOrAbort(ctx, 500, a.DB.DeleteClientByID(client.ID)); !success {
		return
	}

	ctx.Status(200)
}
