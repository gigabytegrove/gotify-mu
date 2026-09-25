package api

import (
	"encoding/json"
	"errors"
	"fmt"
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
	ConsumeMFARecoveryCode(userID uint, hash string) (bool, error)
}

type MFAAPI struct {
	DB MFADatabase
}

type mfaCodeParams struct {
	Code string `json:"code" binding:"required"`
}

func (a *MFAAPI) Status(ctx *gin.Context) {
	userID := auth.GetUserID(ctx)
	item, err := a.DB.GetUserMFA(userID)
	if !successOrAbort(ctx, 500, err) { return }
	status := model.MFAStatus{}
	if item != nil {
		status.Enabled = item.Enabled
		var hashes []string
		_ = json.Unmarshal([]byte(item.RecoveryCodes), &hashes)
		status.RecoveryCodesRemaining = len(hashes)
	}
	ctx.JSON(200, status)
}

func (a *MFAAPI) Start(ctx *gin.Context) {
	userID := auth.GetUserID(ctx)
	user, err := a.DB.GetUserByID(userID)
	if !successOrAbort(ctx, 500, err) { return }
	if user == nil { ctx.AbortWithError(404, errors.New("user not found")); return }

	secret, err := security.GenerateTOTPSecret()
	if !successOrAbort(ctx, 500, err) { return }
	protected, err := security.Protect(secret)
	if !successOrAbort(ctx, 500, err) { return }
	item := &model.UserMFA{UserID:userID, Secret:protected, Enabled:false}
	if !successOrAbort(ctx, 500, a.DB.SaveUserMFA(item)) { return }

	label := url.QueryEscape("Gotify MU:" + user.Name)
	issuer := url.QueryEscape("Gotify MU")
	uri := fmt.Sprintf("otpauth://totp/%s?secret=%s&issuer=%s&period=30&digits=6", label, secret, issuer)
	ctx.JSON(200, gin.H{"secret":secret,"otpauthUri":uri})
}

func (a *MFAAPI) Enable(ctx *gin.Context) {
	var params mfaCodeParams
	if err := ctx.ShouldBindJSON(&params); err != nil { return }
	userID := auth.GetUserID(ctx)
	item, err := a.DB.GetUserMFA(userID)
	if !successOrAbort(ctx, 500, err) { return }
	if item == nil || item.Secret == "" {
		ctx.AbortWithError(400, errors.New("MFA setup has not been started"))
		return
	}
	secret, err := security.Reveal(item.Secret)
	if !successOrAbort(ctx, 500, err) { return }
	if !security.VerifyTOTP(secret, params.Code, time.Now()) {
		ctx.AbortWithError(400, errors.New("verification code is invalid"))
		return
	}
	plain, hashes, err := security.GenerateRecoveryCodes(10)
	if !successOrAbort(ctx, 500, err) { return }
	encoded, _ := json.Marshal(hashes)
	item.Enabled = true
	item.RecoveryCodes = string(encoded)
	if !successOrAbort(ctx, 500, a.DB.SaveUserMFA(item)) { return }
	ctx.JSON(200, gin.H{"enabled":true,"recoveryCodes":plain})
}

func (a *MFAAPI) Disable(ctx *gin.Context) {
	var params mfaCodeParams
	if err := ctx.ShouldBindJSON(&params); err != nil { return }
	userID := auth.GetUserID(ctx)
	ok, err := verifyMFAWithRecovery(a.DB, userID, params.Code, time.Now())
	if !successOrAbort(ctx, 500, err) { return }
	if !ok {
		ctx.AbortWithError(http.StatusBadRequest, errors.New("verification or recovery code is invalid"))
		return
	}
	if !successOrAbort(ctx, 500, a.DB.DeleteUserMFA(userID)) { return }
	ctx.JSON(200, gin.H{"enabled":false})
}

func verifyMFAWithRecovery(db interface {
	GetUserMFA(userID uint) (*model.UserMFA, error)
	ConsumeMFARecoveryCode(userID uint, hash string) (bool, error)
}, userID uint, code string, now time.Time) (bool, error) {
	item, err := db.GetUserMFA(userID)
	if err != nil || item == nil || !item.Enabled { return false, err }
	secret, err := security.Reveal(item.Secret)
	if err != nil { return false, err }
	code = strings.TrimSpace(code)
	if security.VerifyTOTP(secret, code, now) { return true, nil }
	return db.ConsumeMFARecoveryCode(userID, security.RecoveryCodeHash(code))
}
