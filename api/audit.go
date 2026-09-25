package api

import (
	"encoding/csv"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gotify/server/v3/model"
)

type AuditDatabase interface {
	GetAuditEvents(limit int, action, target string) ([]*model.AuditEvent, error)
	GetAuditSettings() (*model.AuditSettings, error)
	SaveAuditSettings(item *model.AuditSettings) error
	DeleteAuditEventsBefore(before time.Time) error
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


type auditSettingsInput struct {
	RetentionDays int `json:"retentionDays" binding:"min=7,max=3650"`
}

func (a *AuditAPI) GetSettings(ctx *gin.Context) {
	item, err := a.DB.GetAuditSettings()
	if !successOrAbort(ctx, 500, err) { return }
	ctx.JSON(200, item)
}

func (a *AuditAPI) SaveSettings(ctx *gin.Context) {
	var input auditSettingsInput
	if err := ctx.ShouldBindJSON(&input); err != nil { return }
	item := &model.AuditSettings{ID:1, RetentionDays:input.RetentionDays}
	if !successOrAbort(ctx, 500, a.DB.SaveAuditSettings(item)) { return }
	before := time.Now().AddDate(0, 0, -item.RetentionDays)
	if !successOrAbort(ctx, 500, a.DB.DeleteAuditEventsBefore(before)) { return }
	ctx.JSON(200, item)
}

func (a *AuditAPI) ExportCSV(ctx *gin.Context) {
	events, err := a.DB.GetAuditEvents(500, ctx.Query("action"), ctx.Query("target"))
	if !successOrAbort(ctx, 500, err) { return }

	ctx.Header("Content-Type", "text/csv; charset=utf-8")
	ctx.Header("Content-Disposition", "attachment; filename=gotify-mu-audit.csv")
	writer := csv.NewWriter(ctx.Writer)
	_ = writer.Write([]string{"time","user","action","target","target_id","ip_address","details"})
	for _, event := range events {
		_ = writer.Write([]string{
			event.CreatedAt.UTC().Format(time.RFC3339),
			event.Username,
			event.Action,
			event.Target,
			event.TargetID,
			event.IPAddress,
			event.Details,
		})
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		ctx.Error(err)
	}
}
