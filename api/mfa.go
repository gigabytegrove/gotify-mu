package api

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gotify/server/v3/auth"
	"github.com/gotify/server/v3/auth/mfa"
	"github.com/gotify/server/v3/model"
)

type MFADatabase interface {
	GetUserMFA(userID uint) (*model.UserMFA, error)
	SaveUserMFA(item *model.UserMFA) error
	DisableUserMFA(userID uint) error
}

type MFAAPI struct {
	DB MFADatabase
}

type mfaEnableInput struct {
	Secret string `json:"secret" binding:"required"`
	Code   string `json:"code" binding:"required"`
}

type mfaVerifyInput struct {
	Code string `json:"code" binding:"required"`
}

func (a *MFAAPI) Status(ctx *gin.Context) {
	item, err := a.DB.GetUserMFA(auth.GetUserID(ctx))
	if !successOrAbort(ctx, 500, err) { return }
	status := model.MFAStatus{}
	if item != nil && item.Enabled {
		status.Enabled = true
		status.RecoveryRemaining = len(mfa.ParseRecoveryHashes(item.RecoveryHashes))
	}
	ctx.JSON(http.StatusOK, status)
}

func (a *MFAAPI) Setup(ctx *gin.Context) {
	userID := auth.GetUserID(ctx)
	secret, err := mfa.GenerateSecret()
	if !successOrAbort(ctx, 500, err) { return }
	userName := "user"
	if value := auth.GetUser(ctx); value != nil { userName = value.Name }
	ctx.JSON(http.StatusOK, gin.H{
		"secret": secret,
		"provisioningUri": mfa.ProvisioningURI("Gotify MU", userName, secret),
	})
}

func (a *MFAAPI) Enable(ctx *gin.Context) {
	var input mfaEnableInput
	if err := ctx.ShouldBindJSON(&input); err != nil { return }
	if !mfa.VerifyTOTP(input.Secret, input.Code, time.Now()) {
		ctx.AbortWithError(http.StatusBadRequest, errors.New("verification code is invalid"))
		return
	}
	codes, hashes, err := mfa.GenerateRecoveryCodes(10)
	if !successOrAbort(ctx, 500, err) { return }
	item := &model.UserMFA{
		UserID: auth.GetUserID(ctx),
		Enabled: true,
		Secret: strings.TrimSpace(input.Secret),
		RecoveryHashes: mfa.SerializeRecoveryHashes(hashes),
	}
	if !successOrAbort(ctx, 500, a.DB.SaveUserMFA(item)) { return }
	ctx.JSON(http.StatusOK, gin.H{
		"enabled": true,
		"recoveryCodes": codes,
	})
}

func (a *MFAAPI) Disable(ctx *gin.Context) {
	if !successOrAbort(ctx, 500, a.DB.DisableUserMFA(auth.GetUserID(ctx))) { return }
	ctx.Status(http.StatusNoContent)
}

func (a *MFAAPI) RegenerateRecoveryCodes(ctx *gin.Context) {
	item, err := a.DB.GetUserMFA(auth.GetUserID(ctx))
	if !successOrAbort(ctx, 500, err) { return }
	if item == nil || !item.Enabled {
		ctx.AbortWithError(http.StatusBadRequest, errors.New("multi-factor authentication is not enabled"))
		return
	}
	var input mfaVerifyInput
	if err := ctx.ShouldBindJSON(&input); err != nil { return }
	if !mfa.VerifyTOTP(item.Secret, input.Code, time.Now()) {
		ctx.AbortWithError(http.StatusBadRequest, errors.New("verification code is invalid"))
		return
	}
	codes, hashes, err := mfa.GenerateRecoveryCodes(10)
	if !successOrAbort(ctx, 500, err) { return }
	item.RecoveryHashes = mfa.SerializeRecoveryHashes(hashes)
	if !successOrAbort(ctx, 500, a.DB.SaveUserMFA(item)) { return }
	ctx.JSON(http.StatusOK, gin.H{"recoveryCodes": codes})
}

func sanitizeProvisioningURI(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil { return "" }
	if parsed.Scheme != "otpauth" { return "" }
	return parsed.String()
}
