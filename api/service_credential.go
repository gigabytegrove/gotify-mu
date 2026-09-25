package api

import (
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gotify/server/v3/auth"
	"github.com/gotify/server/v3/model"
	"github.com/gotify/server/v3/secretstore"
)

var allowedServiceScopes = []string{
	"message:write",
	"message:read",
	"channel:read",
}

type ServiceCredentialDatabase interface {
	CreateServiceCredential(item *model.ServiceCredential) error
	GetServiceCredentialsByUser(userID uint) ([]*model.ServiceCredential, error)
	DeleteServiceCredential(id, userID uint) error
	GetApplicationByID(id uint) (*model.Application, error)
	GetApplicationMembership(applicationID, userID uint) (*model.ApplicationMembership, error)
}

type ServiceCredentialAPI struct {
	DB ServiceCredentialDatabase
}

type serviceCredentialInput struct {
	Name          string   `json:"name" binding:"required"`
	ApplicationID *uint    `json:"applicationId"`
	Scopes        []string `json:"scopes" binding:"required,min=1"`
	ExpiresAt     *time.Time `json:"expiresAt"`
}

func (a *ServiceCredentialAPI) List(ctx *gin.Context) {
	userID := auth.GetUserID(ctx)
	items, err := a.DB.GetServiceCredentialsByUser(userID)
	if !successOrAbort(ctx, 500, err) { return }
	out := make([]model.ServiceCredentialExternal, 0, len(items))
	for _, item := range items {
		out = append(out, serviceCredentialExternal(item, ""))
	}
	ctx.JSON(http.StatusOK, out)
}

func (a *ServiceCredentialAPI) Create(ctx *gin.Context) {
	var input serviceCredentialInput
	if err := ctx.ShouldBindJSON(&input); err != nil { return }
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		ctx.AbortWithError(400, errors.New("name is required"))
		return
	}
	scopes := normalizeServiceScopes(input.Scopes)
	if len(scopes) == 0 {
		ctx.AbortWithError(400, errors.New("at least one supported scope is required"))
		return
	}
	for _, scope := range scopes {
		if !slices.Contains(allowedServiceScopes, scope) {
			ctx.AbortWithError(400, errors.New("unsupported service credential scope"))
			return
		}
	}
	userID := auth.GetUserID(ctx)
	if input.ApplicationID != nil {
		app, err := a.DB.GetApplicationByID(*input.ApplicationID)
		if !successOrAbort(ctx, 500, err) { return }
		if app == nil {
			ctx.AbortWithError(404, errors.New("Channel does not exist"))
			return
		}
		if app.UserID != userID {
			membership, err := a.DB.GetApplicationMembership(app.ID, userID)
			if !successOrAbort(ctx, 500, err) { return }
			if membership == nil || (membership.Role != model.ApplicationRoleManager && membership.Role != model.ApplicationRolePublisher) {
				ctx.AbortWithError(403, errors.New("you do not have permission to create a credential for this Channel"))
				return
			}
		}
	}
	if input.ExpiresAt != nil && !input.ExpiresAt.After(time.Now()) {
		ctx.AbortWithError(400, errors.New("expiration must be in the future"))
		return
	}

	token := auth.GenerateServiceToken()
	item := &model.ServiceCredential{
		UserID: userID,
		ApplicationID: input.ApplicationID,
		Name: input.Name,
		TokenHash: secretstore.Hash(token),
		Scopes: strings.Join(scopes, ","),
		ExpiresAt: input.ExpiresAt,
	}
	if !successOrAbort(ctx, 500, a.DB.CreateServiceCredential(item)) { return }
	ctx.JSON(http.StatusCreated, serviceCredentialExternal(item, token))
}

func (a *ServiceCredentialAPI) Delete(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		if !successOrAbort(ctx, 500, a.DB.DeleteServiceCredential(id, auth.GetUserID(ctx))) { return }
		ctx.Status(http.StatusNoContent)
	})
}

func serviceCredentialExternal(item *model.ServiceCredential, token string) model.ServiceCredentialExternal {
	return model.ServiceCredentialExternal{
		ID:item.ID, UserID:item.UserID, ApplicationID:item.ApplicationID, Name:item.Name,
		Scopes:normalizeServiceScopes(strings.Split(item.Scopes, ",")),
		CreatedAt:item.CreatedAt, LastUsed:item.LastUsed, ExpiresAt:item.ExpiresAt, Token:token,
	}
}

func normalizeServiceScopes(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		scope := strings.ToLower(strings.TrimSpace(value))
		if scope == "" { continue }
		if _, ok := seen[scope]; ok { continue }
		seen[scope] = struct{}{}
		out = append(out, scope)
	}
	return out
}
