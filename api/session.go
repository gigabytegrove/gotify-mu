package api

import (
	"errors"
	"net/http"
	"strconv"
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
}

// SessionAPI provides handlers for cookie-based session authentication.
type SessionAPI struct {
	DB               SessionDatabase
	NotifyDeleted    func(uint, string)
	SecureCookie     bool
	LocalAuthEnabled bool
	LoginLimiter     *security.FailureLimiter
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

	limitKey := ctx.ClientIP() + "|" + strings.ToLower(strings.TrimSpace(name))
	if a.LoginLimiter != nil {
		if allowed, retry := a.LoginLimiter.Allow(limitKey); !allowed {
			seconds := int(retry.Seconds())
			if seconds < 1 {
				seconds = 1
			}
			ctx.Header("Retry-After", strconv.Itoa(seconds))
			ctx.AbortWithError(http.StatusTooManyRequests, errors.New("too many failed authentication attempts"))
			return
		}
	}

	user, err := a.DB.GetUserByName(name)
	if err != nil {
		ctx.AbortWithError(500, err)
		return
	}
	if user == nil || !password.ComparePassword(user.Pass, []byte(pass)) {
		if a.LoginLimiter != nil {
			a.LoginLimiter.Failure(limitKey)
		}
		ctx.AbortWithError(401, errors.New("invalid credentials"))
		return
	}
	if a.LoginLimiter != nil {
		a.LoginLimiter.Success(limitKey)
	}

	clientParams := ClientParams{}
	if err := ctx.Bind(&clientParams); err != nil {
		return
	}

	elevatedUntil := time.Now().Add(model.DefaultElevationDuration)
	tokenPublic, tokenPrivate := generateClientToken()
	client := model.Client{
		Name:                          clientParams.Name,
		Token:                         tokenPublic,
		UserID:                        user.ID,
		ElevatedUntil:                 &elevatedUntil,
		ExpiresAfterInactivitySeconds: auth.CookieMaxAge,
	}
	if success := successOrAbort(ctx, 500, a.DB.CreateClient(&client)); !success {
		return
	}

	auth.SetCookie(ctx.Writer, tokenPrivate, auth.CookieMaxAge, a.SecureCookie)

	ctx.JSON(200, &model.CurrentUserExternal{
		ID:            user.ID,
		Name:          user.Name,
		DisplayName:   user.DisplayName,
		Admin:         user.Admin,
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
