package api

import (
	"bytes"
	"encoding/csv"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gotify/server/v3/model"
)

type AuditDatabase interface {
	GetAuditEvents(limit int, action, target string) ([]*model.AuditEvent, error)
	GetAuditEventsForExport(limit int) ([]*model.AuditEvent, error)
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


func (a *AuditAPI) ExportAuditEvents(ctx *gin.Context) {
	events, err := a.DB.GetAuditEventsForExport(10000)
	if !successOrAbort(ctx, 500, err) { return }

	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	_ = writer.Write([]string{"timestamp","user_id","username","action","target","target_id","ip_address","details"})
	for _, event := range events {
		_ = writer.Write([]string{
			event.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
			strconv.FormatUint(uint64(event.UserID), 10),
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
		ctx.AbortWithError(500, err)
		return
	}
	ctx.Header("Content-Disposition", "attachment; filename=gotify-mu-audit.csv")
	ctx.Data(200, "text/csv; charset=utf-8", buffer.Bytes())
}
