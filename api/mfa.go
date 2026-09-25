package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gotify/server/v3/auth"
	"github.com/gotify/server/v3/model"
	"github.com/gotify/server/v3/security"
)

type MFADatabase interface {
	GetUserByID(id uint) (*model.User, error)
	GetUserMFA(userID uint) (*model.UserMFA, error)
	SaveUserMFA(item *model.UserMFA) error
	DeleteUserMFA(userID uint) error
	ConsumeRecoveryCode(userID uint, codeHash string) (bool, error)
	GetSecurityPolicy() (model.SecurityPolicy, error)
}

type MFAAPI struct {
	DB MFADatabase
}

type mfaCodeParams struct {
	Code string `json:"code" binding:"required"`
}

func recoveryHashes(codes []string) (string, error) {
	hashes := make([]string, 0, len(codes))
	for _, code := range codes { hashes = append(hashes, security.HashRecoveryCode(code)) }
	raw, err := json.Marshal(hashes)
	return string(raw), err
}

func (a *MFAAPI) Status(ctx *gin.Context) {
	userID := auth.GetUserID(ctx)
	item, err := a.DB.GetUserMFA(userID)
	if !successOrAbort(ctx, http.StatusInternalServerError, err) { return }
	status := model.MFAStatus{}
	if item != nil {
		status.Enabled = item.Enabled
		status.EnrolledAt = item.EnrolledAt
		var hashes []string
		_ = json.Unmarshal([]byte(item.RecoveryHashes), &hashes)
		status.RecoveryCodes = len(hashes)
	}
	ctx.JSON(http.StatusOK, status)
}

func (a *MFAAPI) Setup(ctx *gin.Context) {
	userID := auth.GetUserID(ctx)
	user, err := a.DB.GetUserByID(userID)
	if !successOrAbort(ctx, http.StatusInternalServerError, err) { return }
	if user == nil {
		ctx.AbortWithError(http.StatusNotFound, errors.New("user not found"))
		return
	}

	secret, err := security.GenerateTOTPSecret()
	if !successOrAbort(ctx, http.StatusInternalServerError, err) { return }
	codes, err := security.GenerateRecoveryCodes(10)
	if !successOrAbort(ctx, http.StatusInternalServerError, err) { return }
	hashes, err := recoveryHashes(codes)
	if !successOrAbort(ctx, http.StatusInternalServerError, err) { return }

	item := &model.UserMFA{UserID:userID, Secret:secret, RecoveryHashes:hashes, Enabled:false}
	if !successOrAbort(ctx, http.StatusInternalServerError, a.DB.SaveUserMFA(item)) { return }

	label := url.QueryEscape("Gotify MU:" + user.Name)
	issuer := url.QueryEscape("Gotify MU")
	uri := "otpauth://totp/" + label + "?secret=" + url.QueryEscape(secret) + "&issuer=" + issuer + "&digits=6&period=30"
	ctx.JSON(http.StatusOK, model.MFASetupResult{Secret:secret, ProvisioningURI:uri, RecoveryCodes:codes})
}

func verifyMFAItem(item *model.UserMFA, code string, now time.Time) bool {
	if item == nil { return false }
	return security.VerifyTOTP(item.Secret, strings.TrimSpace(code), now)
}

func (a *MFAAPI) Enable(ctx *gin.Context) {
	var params mfaCodeParams
	if err := ctx.ShouldBindJSON(&params); err != nil { return }
	userID := auth.GetUserID(ctx)
	item, err := a.DB.GetUserMFA(userID)
	if !successOrAbort(ctx, http.StatusInternalServerError, err) { return }
	if item == nil {
		ctx.AbortWithError(http.StatusBadRequest, errors.New("MFA setup has not been started"))
		return
	}
	if !verifyMFAItem(item, params.Code, time.Now()) {
		ctx.AbortWithError(http.StatusBadRequest, errors.New("invalid verification code"))
		return
	}
	now := time.Now()
	item.Enabled = true
	item.EnrolledAt = &now
	if !successOrAbort(ctx, http.StatusInternalServerError, a.DB.SaveUserMFA(item)) { return }
	ctx.JSON(http.StatusOK, model.MFAStatus{Enabled:true,EnrolledAt:&now,RecoveryCodes:10})
}

func (a *MFAAPI) Disable(ctx *gin.Context) {
	var params mfaCodeParams
	if err := ctx.ShouldBindJSON(&params); err != nil { return }
	userID := auth.GetUserID(ctx)
	item, err := a.DB.GetUserMFA(userID)
	if !successOrAbort(ctx, http.StatusInternalServerError, err) { return }
	if item == nil || !item.Enabled {
		ctx.AbortWithError(http.StatusBadRequest, errors.New("MFA is not enabled"))
		return
	}
	valid := verifyMFAItem(item, params.Code, time.Now())
	if !valid {
		valid, err = a.DB.ConsumeRecoveryCode(userID, security.HashRecoveryCode(params.Code))
		if !successOrAbort(ctx, http.StatusInternalServerError, err) { return }
	}
	if !valid {
		ctx.AbortWithError(http.StatusBadRequest, errors.New("invalid verification code"))
		return
	}
	if !successOrAbort(ctx, http.StatusInternalServerError, a.DB.DeleteUserMFA(userID)) { return }
	ctx.JSON(http.StatusOK, model.MFAStatus{Enabled:false})
}

func (a *MFAAPI) RegenerateRecoveryCodes(ctx *gin.Context) {
	var params mfaCodeParams
	if err := ctx.ShouldBindJSON(&params); err != nil { return }
	userID := auth.GetUserID(ctx)
	item, err := a.DB.GetUserMFA(userID)
	if !successOrAbort(ctx, http.StatusInternalServerError, err) { return }
	if item == nil || !item.Enabled || !verifyMFAItem(item, params.Code, time.Now()) {
		ctx.AbortWithError(http.StatusBadRequest, errors.New("valid authenticator code required"))
		return
	}
	codes, err := security.GenerateRecoveryCodes(10)
	if !successOrAbort(ctx, http.StatusInternalServerError, err) { return }
	hashes, err := recoveryHashes(codes)
	if !successOrAbort(ctx, http.StatusInternalServerError, err) { return }
	item.RecoveryHashes = hashes
	if !successOrAbort(ctx, http.StatusInternalServerError, a.DB.SaveUserMFA(item)) { return }
	ctx.JSON(http.StatusOK, gin.H{"recoveryCodes":codes})
}
