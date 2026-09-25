package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gotify/server/v3/operations"
)

const maxRestoreUploadBytes int64 = 4 << 30

type diagnosticsReport struct {
	GeneratedAt    time.Time         `json:"generatedAt"`
	Version        any               `json:"version"`
	Operations     any               `json:"operations"`
	SecurityPolicy any               `json:"securityPolicy"`
	Storage        diagnosticsStorage `json:"storage"`
	PendingRestore bool               `json:"pendingRestore"`
}

type diagnosticsStorage struct {
	DataDirectory string `json:"dataDirectory"`
	Files         int64  `json:"files"`
	Bytes         int64  `json:"bytes"`
}

func (a *SystemAPI) DownloadBackup(ctx *gin.Context) {
	if a.Dialect != "sqlite3" && a.Dialect != "sqlite" {
		ctx.AbortWithError(http.StatusNotImplemented, errors.New("online backup bundles currently require SQLite"))
		return
	}
	if a.DataDir == "" || a.DatabaseFile == "" {
		ctx.AbortWithError(http.StatusServiceUnavailable, errors.New("backup paths are not configured"))
		return
	}

	tempDir, err := os.MkdirTemp("", "gotify-mu-backup-*")
	if !successOrAbort(ctx, http.StatusInternalServerError, err) { return }
	defer os.RemoveAll(tempDir)

	snapshot := filepath.Join(tempDir, "gotify.db")
	if !successOrAbort(ctx, http.StatusInternalServerError, a.DB.CreateBackupSnapshot(snapshot)) { return }

	version, commit := "unknown", "unknown"
	if a.VersionInfo != nil {
		version = a.VersionInfo.Version
		commit = a.VersionInfo.Commit
	}
	filename := "gotify-mu-backup-" + time.Now().UTC().Format("20060102-150405") + ".zip"
	bundle := filepath.Join(tempDir, filename)
	if !successOrAbort(
		ctx,
		http.StatusInternalServerError,
		operations.CreateBackupBundle(a.DataDir, a.DatabaseFile, snapshot, bundle, version, commit),
	) {
		return
	}

	ctx.Header("Cache-Control", "no-store")
	ctx.FileAttachment(bundle, filename)
}

func (a *SystemAPI) StageRestore(ctx *gin.Context) {
	if a.Dialect != "sqlite3" && a.Dialect != "sqlite" {
		ctx.AbortWithError(http.StatusNotImplemented, errors.New("restore staging currently requires SQLite"))
		return
	}
	if a.DataDir == "" {
		ctx.AbortWithError(http.StatusServiceUnavailable, errors.New("restore path is not configured"))
		return
	}

	header, err := ctx.FormFile("backup")
	if err != nil {
		ctx.AbortWithError(http.StatusBadRequest, errors.New("backup file is required"))
		return
	}
	if header.Size <= 0 || header.Size > maxRestoreUploadBytes {
		ctx.AbortWithError(http.StatusRequestEntityTooLarge, errors.New("backup file is too large"))
		return
	}
	file, err := header.Open()
	if !successOrAbort(ctx, http.StatusInternalServerError, err) { return }
	defer file.Close()

	manifest, err := operations.ValidateAndStageRestore(file, a.DataDir, maxRestoreUploadBytes)
	if !successOrAbort(ctx, http.StatusBadRequest, err) { return }
	ctx.JSON(http.StatusAccepted, gin.H{
		"staged": true,
		"restartRequired": true,
		"manifest": manifest,
		"message": "Backup validated and staged. Restart Gotify MU to apply the restore before the database opens.",
	})
}

func (a *SystemAPI) CancelRestore(ctx *gin.Context) {
	if a.DataDir == "" {
		ctx.AbortWithError(http.StatusServiceUnavailable, errors.New("restore path is not configured"))
		return
	}
	path := filepath.Join(a.DataDir, operations.PendingRestoreName)
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

func (a *SystemAPI) DownloadDiagnostics(ctx *gin.Context) {
	operationsSummary, err := a.DB.GetOperationsSummary(a.Dialect)
	if !successOrAbort(ctx, http.StatusInternalServerError, err) { return }
	policy, err := a.DB.GetSecurityPolicy()
	if !successOrAbort(ctx, http.StatusInternalServerError, err) { return }

	storage := diagnosticsStorage{DataDirectory: a.DataDir}
	if a.DataDir != "" {
		_ = filepath.Walk(a.DataDir, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil { return nil }
			if info.Mode().IsRegular() {
				storage.Files++
				storage.Bytes += info.Size()
			}
			return nil
		})
	}
	pending := false
	if a.DataDir != "" {
		_, statErr := os.Stat(filepath.Join(a.DataDir, operations.PendingRestoreName))
		pending = statErr == nil
	}

	report := diagnosticsReport{
		GeneratedAt: time.Now().UTC(),
		Version: a.VersionInfo,
		Operations: operationsSummary,
		SecurityPolicy: policy,
		Storage: storage,
		PendingRestore: pending,
	}
	body, err := json.MarshalIndent(report, "", "  ")
	if !successOrAbort(ctx, http.StatusInternalServerError, err) { return }
	ctx.Header("Cache-Control", "no-store")
	ctx.Header("Content-Disposition", "attachment; filename=gotify-mu-diagnostics.json")
	ctx.Data(http.StatusOK, "application/json", body)
}
