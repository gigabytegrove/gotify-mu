package api

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gotify/server/v3/model"
)

type SystemDatabase interface {
	GetSecurityPolicy() (model.SecurityPolicy, error)
	SaveSecurityPolicy(model.SecurityPolicy) error
	GetOperationsSummary(dialect string) (model.OperationsSummary, error)
	GetAllClients() ([]*model.Client, error)
	GetClientByID(id uint) (*model.Client, error)
	GetUserByID(id uint) (*model.User, error)
	DeleteClientByID(id uint) error
	GetAuditEventsForExport(limit int) ([]*model.AuditEvent, error)
	DeleteAuditEventsBefore(before time.Time) error
	CreateBackupSnapshot(destination string) error
}

type SystemAPI struct {
	DB            SystemDatabase
	Dialect       string
	DataDir       string
	DatabaseFile  string
	VersionInfo   *model.VersionInfo
	NotifyDeleted     func(uint, string)
	ConnectedClients func() int
}

type sessionView struct {
	ID            uint       `json:"id"`
	UserID        uint       `json:"userId"`
	Username      string     `json:"username"`
	Name          string     `json:"name"`
	CreatedAt     time.Time  `json:"createdAt"`
	LastUsed      *time.Time `json:"lastUsed,omitempty"`
	ElevatedUntil *time.Time `json:"elevatedUntil,omitempty"`
	ExpiresAt     *time.Time `json:"expiresAt,omitempty"`
}

func (a *SystemAPI) GetSecurityPolicy(ctx *gin.Context) {
	policy, err := a.DB.GetSecurityPolicy()
	if !successOrAbort(ctx, http.StatusInternalServerError, err) { return }
	ctx.JSON(http.StatusOK, policy)
}

func validateSecurityPolicy(policy model.SecurityPolicy) error {
	if policy.MinimumPasswordLength < 8 || policy.MinimumPasswordLength > 72 {
		return errors.New("minimum password length must be between 8 and 72")
	}
	if policy.SessionInactivityMinutes < 5 || policy.SessionInactivityMinutes > 525600 {
		return errors.New("session inactivity must be between 5 minutes and 1 year")
	}
	if policy.ElevationMinutes < 1 || policy.ElevationMinutes > 1440 {
		return errors.New("elevation duration must be between 1 minute and 24 hours")
	}
	if policy.AuditRetentionDays < 1 || policy.AuditRetentionDays > 3650 {
		return errors.New("audit retention must be between 1 and 3650 days")
	}
	return nil
}

func (a *SystemAPI) SaveSecurityPolicy(ctx *gin.Context) {
	var policy model.SecurityPolicy
	if err := ctx.ShouldBindJSON(&policy); err != nil { return }
	if err := validateSecurityPolicy(policy); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}
	if !successOrAbort(ctx, http.StatusInternalServerError, a.DB.SaveSecurityPolicy(policy)) { return }
	ctx.JSON(http.StatusOK, policy)
}

func (a *SystemAPI) GetOperations(ctx *gin.Context) {
	summary, err := a.DB.GetOperationsSummary(a.Dialect)
	if !successOrAbort(ctx, http.StatusInternalServerError, err) { return }
	if a.ConnectedClients != nil {
		summary.ConnectedClients = a.ConnectedClients()
	}
	if a.DataDir != "" {
		attachmentDir := filepath.Join(a.DataDir, "attachments")
		_ = filepath.Walk(a.DataDir, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil || info == nil {
				return nil
			}
			if info.Mode().IsRegular() {
				summary.StorageFiles++
				summary.StorageBytes += info.Size()
				if strings.HasPrefix(filepath.Clean(path), filepath.Clean(attachmentDir)+string(os.PathSeparator)) {
					summary.AttachmentFiles++
				}
			}
			return nil
		})
	}
	ctx.JSON(http.StatusOK, summary)
}

func (a *SystemAPI) GetSessions(ctx *gin.Context) {
	items, err := a.DB.GetAllClients()
	if !successOrAbort(ctx, http.StatusInternalServerError, err) { return }
	out := make([]sessionView, 0, len(items))
	for _, item := range items {
		username := ""
		if user, userErr := a.DB.GetUserByID(item.UserID); userErr == nil && user != nil {
			username = user.Name
		}
		out = append(out, sessionView{
			ID:item.ID, UserID:item.UserID, Username:username, Name:item.Name,
			CreatedAt:item.CreatedAt, LastUsed:item.LastUsed, ElevatedUntil:item.ElevatedUntil,
			ExpiresAt:item.ExpiresAt,
		})
	}
	ctx.JSON(http.StatusOK, out)
}

func (a *SystemAPI) RevokeSession(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		client, err := a.DB.GetClientByID(id)
		if !successOrAbort(ctx, http.StatusInternalServerError, err) { return }
		if client == nil {
			ctx.AbortWithError(http.StatusNotFound, errors.New("session not found"))
			return
		}
		if !successOrAbort(ctx, http.StatusInternalServerError, a.DB.DeleteClientByID(id)) { return }
		if a.NotifyDeleted != nil { a.NotifyDeleted(client.UserID, client.Token) }
		ctx.Status(http.StatusNoContent)
	})
}

func (a *SystemAPI) ExportAudit(ctx *gin.Context) {
	limit := 10000
	if raw := ctx.Query("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 100000 {
			limit = parsed
		}
	}
	items, err := a.DB.GetAuditEventsForExport(limit)
	if !successOrAbort(ctx, http.StatusInternalServerError, err) { return }

	if ctx.Query("format") == "json" {
		body, err := json.MarshalIndent(items, "", "  ")
		if !successOrAbort(ctx, http.StatusInternalServerError, err) { return }
		ctx.Header("Content-Disposition", "attachment; filename=gotify-mu-audit.json")
		ctx.Data(http.StatusOK, "application/json", body)
		return
	}

	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	_ = writer.Write([]string{"id","createdAt","userId","username","action","target","targetId","ipAddress","details"})
	for _, item := range items {
		_ = writer.Write([]string{
			strconv.FormatUint(uint64(item.ID),10),
			item.CreatedAt.UTC().Format(time.RFC3339Nano),
			strconv.FormatUint(uint64(item.UserID),10),
			item.Username,item.Action,item.Target,item.TargetID,item.IPAddress,item.Details,
		})
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	ctx.Header("Content-Disposition", "attachment; filename=gotify-mu-audit.csv")
	ctx.Data(http.StatusOK, "text/csv; charset=utf-8", buffer.Bytes())
}

func (a *SystemAPI) ApplyAuditRetention(ctx *gin.Context) {
	policy, err := a.DB.GetSecurityPolicy()
	if !successOrAbort(ctx, http.StatusInternalServerError, err) { return }
	before := time.Now().AddDate(0,0,-policy.AuditRetentionDays)
	if !successOrAbort(ctx, http.StatusInternalServerError, a.DB.DeleteAuditEventsBefore(before)) { return }
	ctx.JSON(http.StatusOK, gin.H{"deletedBefore":before})
}
