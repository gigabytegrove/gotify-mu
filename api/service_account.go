package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gotify/server/v3/auth"
	"github.com/gotify/server/v3/model"
)

var serviceAccountScopes = map[string]struct{}{
	"channels:read": {},
	"message:read":  {},
	"message:write": {},
}

type ServiceAccountDatabase interface {
	CreateServiceAccount(item *model.ServiceAccount) error
	GetServiceAccounts() ([]*model.ServiceAccount, error)
	GetServiceAccountByID(id uint) (*model.ServiceAccount, error)
	GetServiceAccountByTokenHash(hash string) (*model.ServiceAccount, error)
	TouchServiceAccount(id uint, now time.Time) error
	DeleteServiceAccount(id uint) error
	GetApplicationByID(id uint) (*model.Application, error)
	GetMessagesByApplicationSince(appID uint, limit int, since uint) ([]*model.Message, error)
}

type ServiceAccountAPI struct {
	DB         ServiceAccountDatabase
	Dispatcher MessageDispatcher
}

type serviceAccountParams struct {
	Name       string     `json:"name" binding:"required"`
	Scopes     []string   `json:"scopes" binding:"required"`
	ChannelIDs []uint     `json:"channelIds" binding:"required"`
	ExpiresAt  *time.Time `json:"expiresAt"`
}

type serviceAccountCreated struct {
	*model.ServiceAccount
	Token string `json:"token"`
}

func (a *ServiceAccountAPI) List(ctx *gin.Context) {
	items, err := a.DB.GetServiceAccounts()
	if !successOrAbort(ctx, 500, err) { return }
	ctx.JSON(200, items)
}

func (a *ServiceAccountAPI) Create(ctx *gin.Context) {
	var params serviceAccountParams
	if err := ctx.ShouldBindJSON(&params); err != nil { return }
	name := strings.TrimSpace(params.Name)
	if name == "" {
		ctx.AbortWithError(400, errors.New("service account name is required"))
		return
	}
	scopes, err := normalizeServiceScopes(params.Scopes)
	if err != nil {
		ctx.AbortWithError(400, err)
		return
	}
	channelIDs, err := normalizeServiceChannels(params.ChannelIDs)
	if err != nil {
		ctx.AbortWithError(400, err)
		return
	}
	for _, id := range params.ChannelIDs {
		app, loadErr := a.DB.GetApplicationByID(id)
		if !successOrAbort(ctx, 500, loadErr) { return }
		if app == nil {
			ctx.AbortWithError(400, errors.New("one or more selected Channels do not exist"))
			return
		}
	}
	if params.ExpiresAt != nil && !params.ExpiresAt.After(time.Now()) {
		ctx.AbortWithError(400, errors.New("expiration must be in the future"))
		return
	}

	token, hash, err := newServiceAccountToken()
	if !successOrAbort(ctx, 500, err) { return }
	item := &model.ServiceAccount{
		Name: name,
		UserID: auth.GetUserID(ctx),
		TokenHash: hash,
		Scopes: scopes,
		ChannelIDs: channelIDs,
		ExpiresAt: params.ExpiresAt,
	}
	if !successOrAbort(ctx, 500, a.DB.CreateServiceAccount(item)) { return }
	ctx.JSON(201, serviceAccountCreated{ServiceAccount:item, Token:token})
}

func (a *ServiceAccountAPI) Delete(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		item, err := a.DB.GetServiceAccountByID(id)
		if !successOrAbort(ctx, 500, err) { return }
		if item == nil { ctx.AbortWithStatus(404); return }
		if !successOrAbort(ctx, 500, a.DB.DeleteServiceAccount(id)) { return }
		ctx.Status(204)
	})
}

func (a *ServiceAccountAPI) Channels(ctx *gin.Context) {
	account, ok := a.authenticate(ctx, "channels:read")
	if !ok { return }
	ids := parseServiceChannelIDs(account.ChannelIDs)
	result := make([]*model.Application, 0, len(ids))
	for _, id := range ids {
		app, err := a.DB.GetApplicationByID(id)
		if err != nil { ctx.AbortWithError(500, err); return }
		if app == nil { continue }
		copy := *app
		copy.Token = ""
		result = append(result, &copy)
	}
	ctx.JSON(200, result)
}

func (a *ServiceAccountAPI) Messages(ctx *gin.Context) {
	account, ok := a.authenticate(ctx, "message:read")
	if !ok { return }
	channelID, err := strconv.ParseUint(ctx.Query("channelId"), 10, 64)
	if err != nil || channelID == 0 {
		ctx.AbortWithError(400, errors.New("channelId is required"))
		return
	}
	if !serviceChannelAllowed(account.ChannelIDs, uint(channelID)) {
		ctx.AbortWithStatus(403)
		return
	}
	limit := 100
	if raw := ctx.Query("limit"); raw != "" {
		if parsed, parseErr := strconv.Atoi(raw); parseErr == nil && parsed > 0 && parsed <= 500 {
			limit = parsed
		}
	}
	var since uint64
	if raw := ctx.Query("since"); raw != "" { since, _ = strconv.ParseUint(raw, 10, 64) }
	items, err := a.DB.GetMessagesByApplicationSince(uint(channelID), limit, uint(since))
	if !successOrAbort(ctx, 500, err) { return }
	result := make([]*model.MessageExternal, 0, len(items))
	for _, item := range items { result = append(result, toExternalMessage(item)) }
	ctx.JSON(200, result)
}

func (a *ServiceAccountAPI) Publish(ctx *gin.Context) {
	account, ok := a.authenticate(ctx, "message:write")
	if !ok { return }
	var params model.CreateMessage
	if err := ctx.ShouldBindJSON(&params); err != nil { return }
	if params.ApplicationID == 0 || !serviceChannelAllowed(account.ChannelIDs, params.ApplicationID) {
		ctx.AbortWithStatus(403)
		return
	}
	app, err := a.DB.GetApplicationByID(params.ApplicationID)
	if !successOrAbort(ctx, 500, err) { return }
	if app == nil { ctx.AbortWithStatus(404); return }

	priority := app.DefaultPriority
	if params.Priority != nil { priority = *params.Priority }
	msg := &model.Message{
		ApplicationID: params.ApplicationID,
		Title: params.Title,
		Message: params.Message,
		Priority: priority,
		Date: time.Now(),
		SenderName: "Service: " + account.Name,
	}
	if params.Extras != nil {
		if encoded, encodeErr := json.Marshal(params.Extras); encodeErr == nil { msg.Extras = encoded }
	}
	external, err := a.Dispatcher.StoreAndDeliver(msg)
	if !successOrAbort(ctx, 500, err) { return }
	ctx.JSON(200, external)
}

func (a *ServiceAccountAPI) authenticate(ctx *gin.Context, scope string) (*model.ServiceAccount, bool) {
	const prefix = "Bearer "
	header := strings.TrimSpace(ctx.GetHeader("Authorization"))
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		ctx.AbortWithError(http.StatusUnauthorized, errors.New("service account bearer token required"))
		return nil, false
	}
	token := strings.TrimSpace(header[len(prefix):])
	if !strings.HasPrefix(token, "gmu_sa_") {
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return nil, false
	}
	sum := sha256.Sum256([]byte(token))
	item, err := a.DB.GetServiceAccountByTokenHash(hex.EncodeToString(sum[:]))
	if err != nil { ctx.AbortWithError(500, err); return nil, false }
	if item == nil || (item.ExpiresAt != nil && !item.ExpiresAt.After(time.Now())) {
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return nil, false
	}
	if !serviceScopeAllowed(item.Scopes, scope) {
		ctx.AbortWithStatus(http.StatusForbidden)
		return nil, false
	}
	now := time.Now()
	if item.LastUsed == nil || item.LastUsed.Add(5*time.Minute).Before(now) {
		_ = a.DB.TouchServiceAccount(item.ID, now)
	}
	return item, true
}

func newServiceAccountToken() (string, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil { return "", "", err }
	token := "gmu_sa_" + base64.RawURLEncoding.EncodeToString(raw)
	sum := sha256.Sum256([]byte(token))
	return token, hex.EncodeToString(sum[:]), nil
}

func normalizeServiceScopes(values []string) (string, error) {
	set := make(map[string]struct{})
	for _, raw := range values {
		value := strings.TrimSpace(raw)
		if _, ok := serviceAccountScopes[value]; !ok {
			return "", errors.New("unsupported service account scope: " + value)
		}
		set[value] = struct{}{}
	}
	if len(set) == 0 { return "", errors.New("at least one service account scope is required") }
	ordered := []string{"channels:read","message:read","message:write"}
	result := make([]string,0,len(set))
	for _, value := range ordered { if _, ok := set[value]; ok { result=append(result,value) } }
	return strings.Join(result, ","), nil
}

func normalizeServiceChannels(values []uint) (string, error) {
	if len(values) == 0 { return "", errors.New("at least one Channel is required") }
	set := make(map[uint]struct{})
	result := make([]string,0,len(values))
	for _, id := range values {
		if id == 0 { return "", errors.New("Channel ids must be non-zero") }
		if _, ok := set[id]; ok { continue }
		set[id]=struct{}{}
		result=append(result,strconv.FormatUint(uint64(id),10))
	}
	return strings.Join(result,","),nil
}

func parseServiceChannelIDs(raw string) []uint {
	var result []uint
	for _, part := range strings.Split(raw,",") {
		value, err := strconv.ParseUint(strings.TrimSpace(part),10,64)
		if err == nil && value > 0 { result=append(result,uint(value)) }
	}
	return result
}

func serviceChannelAllowed(raw string, id uint) bool {
	for _, value := range parseServiceChannelIDs(raw) { if value==id { return true } }
	return false
}

func serviceScopeAllowed(raw, scope string) bool {
	for _, value := range strings.Split(raw,",") {
		if strings.TrimSpace(value)==scope { return true }
	}
	return false
}
