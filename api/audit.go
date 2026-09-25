package api

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gotify/server/v3/model"
)

type AuditDatabase interface {
	GetAuditEvents(limit int, action, target string) ([]*model.AuditEvent, error)
}

// AuditAPI exposes security/admin audit history.
type AuditAPI struct {
	DB AuditDatabase
}

func (a *AuditAPI) GetAuditEvents(ctx *gin.Context) {
	limit := 200
	if raw := ctx.Query("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}

	events, err := a.DB.GetAuditEvents(limit, ctx.Query("action"), ctx.Query("target"))
	if success := successOrAbort(ctx, 500, err); !success {
		return
	}
	ctx.JSON(200, events)
}
