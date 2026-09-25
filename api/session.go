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
	ConsumeRecoveryCode(userID uint, codeHash string) (bool, error)
	GetSecurityPolicy() (model.SecurityPolicy, error)
	CreateAuditEvent(event *model.AuditEvent) error
}

// SessionAPI provides handlers for cookie-based session authentication.
type SessionAPI struct {
	DB               SessionDatabase
	NotifyDeleted    func(uint, string)
	SecureCookie     bool
	LocalAuthEnabled bool
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
	if !a.LocalAuthEnabled {
		ctx.AbortWithError(403, errors.New("local authentication is disabled"))
		return
	}

	name, pass, ok := ctx.Request.BasicAuth()
	if !ok {
		ctx.AbortWithError(401, errors.New("basic auth required"))
		return
	}

	user, err := a.DB.GetUserByName(name)
	if err != nil {
		ctx.AbortWithError(500, err)
		return
	}
	if user == nil || !password.ComparePassword(user.Pass, []byte(pass)) {
		_ = a.DB.CreateAuditEvent(&model.AuditEvent{
			Username:name, Action:"login_failed", Target:"local_auth", IPAddress:ctx.ClientIP(),
		})
		ctx.AbortWithError(401, errors.New("invalid credentials"))
		return
	}

	policy, err := a.DB.GetSecurityPolicy()
	if err != nil {
		ctx.AbortWithError(500, err)
		return
	}
	mfa, err := a.DB.GetUserMFA(user.ID)
	if err != nil {
		ctx.AbortWithError(500, err)
		return
	}
	mfaRequired := (policy.RequireMFAForAdmins && user.Admin) || policy.RequireMFAForAllLocalUsers
	mfaAuthenticated := false
	if mfa != nil && mfa.Enabled {
		code := strings.TrimSpace(ctx.GetHeader("X-Gotify-MFA-Code"))
		if code == "" {
			ctx.AbortWithStatusJSON(http.StatusPreconditionRequired, gin.H{
				"error":"mfa_required", "mfaRequired":true,
			})
			return
		}
		mfaAuthenticated = security.VerifyTOTP(mfa.Secret, code, time.Now())
		if !mfaAuthenticated {
			mfaAuthenticated, err = a.DB.ConsumeRecoveryCode(user.ID, security.HashRecoveryCode(code))
			if err != nil {
				ctx.AbortWithError(500, err)
				return
			}
		}
		if !mfaAuthenticated {
			_ = a.DB.CreateAuditEvent(&model.AuditEvent{
				UserID:user.ID, Username:user.Name, Action:"mfa_failed", Target:"local_auth", IPAddress:ctx.ClientIP(),
			})
			ctx.AbortWithError(http.StatusUnauthorized, errors.New("invalid MFA code"))
			return
		}
	}

	clientParams := ClientParams{}
	if err := ctx.Bind(&clientParams); err != nil {
		return
	}

	elevationMinutes := policy.ElevationMinutes
	if elevationMinutes <= 0 { elevationMinutes = 60 }
	sessionMinutes := policy.SessionInactivityMinutes
	if sessionMinutes <= 0 { sessionMinutes = auth.CookieMaxAge / 60 }
	elevatedUntil := time.Now().Add(time.Duration(elevationMinutes) * time.Minute)
	tokenPublic, tokenPrivate := generateClientToken()
	client := model.Client{
		Name:                          clientParams.Name,
		Token:                         tokenPublic,
		UserID:                        user.ID,
		ElevatedUntil:                 &elevatedUntil,
		ExpiresAfterInactivitySeconds: uint(sessionMinutes * 60),
		MFAAuthenticated:              mfaAuthenticated,
	}
	if success := successOrAbort(ctx, 500, a.DB.CreateClient(&client)); !success {
		return
	}

	auth.SetCookie(ctx.Writer, tokenPrivate, sessionMinutes*60, a.SecureCookie)
	_ = a.DB.CreateAuditEvent(&model.AuditEvent{
		UserID:user.ID, Username:user.Name, Action:"login_success", Target:"local_auth", IPAddress:ctx.ClientIP(),
	})

	ctx.JSON(200, &model.CurrentUserExternal{
		ID:            user.ID,
		Name:          user.Name,
		DisplayName:   user.DisplayName,
		Admin:         user.Admin,
		CreatedAt:     user.CreatedAt,
		ClientID:      client.ID,
		ElevatedUntil: client.ElevatedUntil,
		MFAEnabled:    mfa != nil && mfa.Enabled,
		MFARequired:   mfaRequired && (mfa == nil || !mfa.Enabled),
		AuthProvider:  "local",
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
