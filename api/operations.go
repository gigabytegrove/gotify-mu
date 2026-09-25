package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gotify/server/v3/model"
)

type OperationsDatabase interface {
	GetAdminSessions() ([]*model.AdminSession, error)
	GetClientByID(id uint) (*model.Client, error)
	DeleteClientByID(id uint) error
	GetSystemStats() (*model.SystemStats, error)
	GetIntegrationStatuses() ([]*model.IntegrationStatus, error)
	GetAutomationRuns(kind string, objectID uint, limit int) ([]*model.AutomationRun, error)
}

type OperationsAPI struct {
	DB OperationsDatabase
	NotifyDeleted func(uint, string)
	DatabaseDialect string
	DatabaseConnection string
	DataPaths []string
	Version *model.VersionInfo
}

func (a *OperationsAPI) Sessions(ctx *gin.Context) {
	items, err := a.DB.GetAdminSessions()
	if !successOrAbort(ctx, 500, err) { return }
	ctx.JSON(200, items)
}

func (a *OperationsAPI) RevokeSession(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		client, err := a.DB.GetClientByID(id)
		if !successOrAbort(ctx, 500, err) { return }
		if client == nil { ctx.AbortWithStatus(404); return }
		if a.NotifyDeleted != nil { a.NotifyDeleted(client.UserID, client.Token) }
		if !successOrAbort(ctx, 500, a.DB.DeleteClientByID(id)) { return }
		ctx.Status(204)
	})
}

func (a *OperationsAPI) Stats(ctx *gin.Context) {
	stats, err := a.DB.GetSystemStats()
	if !successOrAbort(ctx, 500, err) { return }
	if strings.EqualFold(a.DatabaseDialect, "sqlite3") {
		if info, statErr := os.Stat(a.DatabaseConnection); statErr == nil { stats.DatabaseBytes = info.Size() }
	}
	var total int64
	seen := make(map[string]struct{})
	for _, root := range a.DataPaths {
		clean := filepath.Clean(root)
		if _, ok := seen[clean]; ok { continue }
		seen[clean] = struct{}{}
		_ = filepath.Walk(clean, func(_ string, info os.FileInfo, err error) error {
			if err == nil && info != nil && !info.IsDir() { total += info.Size() }
			return nil
		})
	}
	stats.DataBytes = total
	ctx.JSON(200, stats)
}

func (a *OperationsAPI) Diagnostics(ctx *gin.Context) {
	stats, err := a.DB.GetSystemStats()
	if !successOrAbort(ctx, 500, err) { return }
	integrations, err := a.DB.GetIntegrationStatuses()
	if !successOrAbort(ctx, 500, err) { return }
	runs, err := a.DB.GetAutomationRuns("", 0, 100)
	if !successOrAbort(ctx, 500, err) { return }

	payload := map[string]any{
		"generatedAt": time.Now().UTC(),
		"version": a.Version,
		"databaseDialect": a.DatabaseDialect,
		"stats": stats,
		"integrationStatus": integrations,
		"recentAutomationRuns": runs,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if !successOrAbort(ctx, 500, err) { return }
	filename := "gotify-mu-diagnostics-" + time.Now().UTC().Format("20060102-150405") + ".json"
	ctx.Header("Content-Disposition", "attachment; filename="+filename)
	ctx.Data(http.StatusOK, "application/json", data)
}
