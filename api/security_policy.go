package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gotify/server/v3/model"
)

type SecurityPolicyDatabase interface {
	GetSecurityPolicy() (*model.SecurityPolicy, error)
	SaveSecurityPolicy(item *model.SecurityPolicy) error
}

type SecurityPolicyAPI struct {
	DB SecurityPolicyDatabase
}

func (a *SecurityPolicyAPI) Get(ctx *gin.Context) {
	item, err := a.DB.GetSecurityPolicy()
	if !successOrAbort(ctx, 500, err) { return }
	ctx.JSON(200, item)
}

func (a *SecurityPolicyAPI) Save(ctx *gin.Context) {
	item := model.DefaultSecurityPolicy()
	if err := ctx.ShouldBindJSON(item); err != nil { return }
	item.ID = 1

	switch {
	case item.MinPasswordLength < 8 || item.MinPasswordLength > 72:
		ctx.AbortWithError(http.StatusBadRequest, errors.New("minimum password length must be between 8 and 72 characters"))
		return
	case item.SessionLifetimeHours < 1 || item.SessionLifetimeHours > 24*365:
		ctx.AbortWithError(http.StatusBadRequest, errors.New("session lifetime must be between 1 hour and 365 days"))
		return
	case item.ElevationMinutes < 1 || item.ElevationMinutes > 24*60:
		ctx.AbortWithError(http.StatusBadRequest, errors.New("elevation lifetime must be between 1 minute and 24 hours"))
		return
	case item.AuditRetentionDays < 1 || item.AuditRetentionDays > 3650:
		ctx.AbortWithError(http.StatusBadRequest, errors.New("audit retention must be between 1 and 3650 days"))
		return
	case item.AutomationRetentionDays < 1 || item.AutomationRetentionDays > 3650:
		ctx.AbortWithError(http.StatusBadRequest, errors.New("automation history retention must be between 1 and 3650 days"))
		return
	}

	if !successOrAbort(ctx, 500, a.DB.SaveSecurityPolicy(item)) { return }
	ctx.JSON(200, item)
}
