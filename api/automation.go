package api

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gotify/server/v3/auth"
	"github.com/gotify/server/v3/automation"
	"github.com/gotify/server/v3/model"
)

type AutomationEngine interface {
	Publish(applicationID uint, title, message string, priority int) (*model.Message, error)
	ReloadIntegrations()
	SendHomeAssistantEvent(id uint, eventType string, data map[string]any) error
	TestMQTTConnection(id uint) error
}

type AutomationDatabase interface {
	GetApplicationByID(id uint) (*model.Application, error)
	GetApplicationMembership(applicationID, userID uint) (*model.ApplicationMembership, error)
	GetMessageByID(id uint) (*model.Message, error)

	GetWebhookRoutes() ([]*model.WebhookRoute, error)
	GetWebhookRouteByID(id uint) (*model.WebhookRoute, error)
	GetWebhookRouteBySecret(secret string) (*model.WebhookRoute, error)
	SaveWebhookRoute(item *model.WebhookRoute) error
	DeleteWebhookRoute(id uint) error

	GetMQTTIntegrations() ([]*model.MQTTIntegration, error)
	GetMQTTIntegrationByID(id uint) (*model.MQTTIntegration, error)
	SaveMQTTIntegration(item *model.MQTTIntegration) error
	DeleteMQTTIntegration(id uint) error

	GetHomeAssistantIntegrations() ([]*model.HomeAssistantIntegration, error)
	GetHomeAssistantIntegrationByID(id uint) (*model.HomeAssistantIntegration, error)
	SaveHomeAssistantIntegration(item *model.HomeAssistantIntegration) error
	DeleteHomeAssistantIntegration(id uint) error

	GetScheduledNotifications() ([]*model.ScheduledNotification, error)
	GetScheduledNotificationByID(id uint) (*model.ScheduledNotification, error)
	SaveScheduledNotification(item *model.ScheduledNotification) error
	DeleteScheduledNotification(id uint) error

	GetQuietHoursPolicy(userID uint) (*model.QuietHoursPolicy, error)
	SaveQuietHoursPolicy(item *model.QuietHoursPolicy) error
	GetDigestPolicy(userID uint) (*model.DigestPolicy, error)
	SaveDigestPolicy(item *model.DigestPolicy) error
	DeleteDigestItems(userID uint) error

	GetEscalationRules() ([]*model.EscalationRule, error)
	GetEscalationRuleByID(id uint) (*model.EscalationRule, error)
	SaveEscalationRule(item *model.EscalationRule) error
	DeleteEscalationRule(id uint) error

	SetMessageAcknowledgement(userID, messageID uint, acknowledged bool, now time.Time) error
	IsMessageAcknowledgedByUser(userID, messageID uint) (bool, error)
	GetMessageAcknowledgements(messageID uint) ([]model.MessageAcknowledgementView, error)

	GetIntegrationStatus(kind string, integrationID uint) (*model.IntegrationStatus, error)
	GetIntegrationEvents(kind string, integrationID uint, limit int) ([]*model.IntegrationEvent, error)
	RecordIntegrationEvent(item *model.IntegrationEvent) error
	SaveIntegrationStatus(item *model.IntegrationStatus) error
}

type AutomationAPI struct {
	DB     AutomationDatabase
	Engine AutomationEngine
}

type webhookParams struct {
	Name            string `json:"name" binding:"required"`
	ApplicationID   uint   `json:"applicationId" binding:"required"`
	Enabled         bool   `json:"enabled"`
	TitleField      string `json:"titleField"`
	MessageField    string `json:"messageField"`
	PriorityField   string `json:"priorityField"`
	DefaultTitle    string `json:"defaultTitle"`
	DefaultPriority     int      `json:"defaultPriority"`
	AllowedCIDRs        []string `json:"allowedCidrs"`
	RequireSignature    bool     `json:"requireSignature"`
	SigningSecret       string   `json:"signingSecret"`
	ReplayWindowSeconds int      `json:"replayWindowSeconds"`
}

func (a *AutomationAPI) GetWebhookRoutes(ctx *gin.Context) {
	items, err := a.DB.GetWebhookRoutes()
	if !successOrAbort(ctx, 500, err) { return }
	out := make([]model.WebhookRouteView, 0, len(items))
	for _, item := range items {
		view := webhookView(item)
		view.Status, _ = a.DB.GetIntegrationStatus("webhook", item.ID)
		out = append(out, view)
	}
	ctx.JSON(200, out)
}

func (a *AutomationAPI) CreateWebhookRoute(ctx *gin.Context) {
	var params webhookParams
	if err := ctx.ShouldBindJSON(&params); err != nil { return }
	if !a.channelExists(ctx, params.ApplicationID) { return }
	secret, err := generateIntegrationSecret()
	if !successOrAbort(ctx, 500, err) { return }
	if err := validateWebhookSecurity(params.AllowedCIDRs, params.RequireSignature, params.SigningSecret, params.ReplayWindowSeconds); err != nil {
		ctx.AbortWithError(400, err)
		return
	}
	replayWindow := params.ReplayWindowSeconds
	if replayWindow == 0 { replayWindow = 300 }
	item := &model.WebhookRoute{
		Name: params.Name, ApplicationID: params.ApplicationID, Secret: secret, Enabled: params.Enabled,
		TitleField: valueOr(params.TitleField, "title"), MessageField: valueOr(params.MessageField, "message"),
		PriorityField: valueOr(params.PriorityField, "priority"), DefaultTitle: params.DefaultTitle,
		DefaultPriority: params.DefaultPriority, AllowedCIDRs: strings.Join(params.AllowedCIDRs, ","),
		RequireSignature: params.RequireSignature, SigningSecret: params.SigningSecret,
		ReplayWindowSeconds: replayWindow,
	}
	if !successOrAbort(ctx, 500, a.DB.SaveWebhookRoute(item)) { return }
	ctx.JSON(201, webhookView(item))
}

func (a *AutomationAPI) UpdateWebhookRoute(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		item, err := a.DB.GetWebhookRouteByID(id)
		if !successOrAbort(ctx, 500, err) { return }
		if item == nil { ctx.AbortWithError(404, errors.New("webhook not found")); return }
		var params webhookParams
		if err := ctx.ShouldBindJSON(&params); err != nil { return }
		if !a.channelExists(ctx, params.ApplicationID) { return }
		item.Name, item.ApplicationID, item.Enabled = params.Name, params.ApplicationID, params.Enabled
		item.TitleField, item.MessageField = valueOr(params.TitleField, "title"), valueOr(params.MessageField, "message")
		if err := validateWebhookSecurity(params.AllowedCIDRs, params.RequireSignature, valueOr(params.SigningSecret, item.SigningSecret), params.ReplayWindowSeconds); err != nil {
			ctx.AbortWithError(400, err)
			return
		}
		item.PriorityField, item.DefaultTitle, item.DefaultPriority = valueOr(params.PriorityField, "priority"), params.DefaultTitle, params.DefaultPriority
		item.AllowedCIDRs = strings.Join(params.AllowedCIDRs, ",")
		item.RequireSignature = params.RequireSignature
		if params.SigningSecret != "" { item.SigningSecret = params.SigningSecret }
		item.ReplayWindowSeconds = params.ReplayWindowSeconds
		if item.ReplayWindowSeconds == 0 { item.ReplayWindowSeconds = 300 }
		if !successOrAbort(ctx, 500, a.DB.SaveWebhookRoute(item)) { return }
		ctx.JSON(200, webhookView(item))
	})
}

func (a *AutomationAPI) RegenerateWebhookSecret(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		item, err := a.DB.GetWebhookRouteByID(id)
		if !successOrAbort(ctx, 500, err) { return }
		if item == nil { ctx.AbortWithError(404, errors.New("webhook not found")); return }
		secret, err := generateIntegrationSecret()
		if !successOrAbort(ctx, 500, err) { return }
		item.Secret = secret
		if !successOrAbort(ctx, 500, a.DB.SaveWebhookRoute(item)) { return }
		ctx.JSON(200, webhookView(item))
	})
}

func (a *AutomationAPI) DeleteWebhookRoute(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) { successOrAbort(ctx, 500, a.DB.DeleteWebhookRoute(id)) })
}

func (a *AutomationAPI) ReceiveWebhook(ctx *gin.Context) {
	secret := strings.TrimSpace(ctx.Param("secret"))
	item, err := a.DB.GetWebhookRouteBySecret(secret)
	if !successOrAbort(ctx, 500, err) { return }
	if item == nil { ctx.AbortWithStatus(404); return }

	if !webhookSourceAllowed(item.AllowedCIDRs, ctx.ClientIP()) {
		ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error":"Webhook source is not allowed"})
		return
	}

	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, 1024*1024)
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			ctx.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{"error":"Webhook payload exceeds 1 MiB"})
			return
		}
		ctx.AbortWithError(400, err)
		return
	}
	if err := verifyWebhookSignature(item, body, ctx.Request.Header.Get("X-Gotify-Timestamp"), ctx.Request.Header.Get("X-Gotify-Signature"), time.Now()); err != nil {
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error":err.Error()})
		return
	}

	title := item.DefaultTitle
	message := strings.TrimSpace(string(body))
	priority := item.DefaultPriority
	var payload any
	if json.Unmarshal(body, &payload) == nil {
		if value, ok := lookupPayload(payload, item.TitleField); ok {
			if text := payloadString(value); text != "" { title = text }
		}
		if value, ok := lookupPayload(payload, item.MessageField); ok {
			if text := payloadString(value); text != "" { message = text }
		} else if encoded, marshalErr := json.MarshalIndent(payload, "", "  "); marshalErr == nil {
			message = string(encoded)
		}
		if value, ok := lookupPayload(payload, item.PriorityField); ok {
			if number, numberOK := payloadInt(value); numberOK { priority = number }
		}
	}
	if strings.TrimSpace(message) == "" {
		ctx.AbortWithError(400, errors.New("webhook payload did not contain a message"))
		return
	}
	msg, err := a.Engine.Publish(item.ApplicationID, title, message, priority)
	if err != nil {
		now := time.Now().UTC()
		_ = a.DB.SaveIntegrationStatus(&model.IntegrationStatus{Kind:"webhook", IntegrationID:item.ID, State:"error", Message:err.Error(), LastErrorAt:&now, UpdatedAt:now})
		_ = a.DB.RecordIntegrationEvent(&model.IntegrationEvent{Kind:"webhook",IntegrationID:item.ID,Level:"error",Event:"delivery_error",Message:err.Error(),CreatedAt:now})
		ctx.AbortWithError(500, err)
		return
	}
	now := time.Now().UTC()
	_ = a.DB.SaveIntegrationStatus(&model.IntegrationStatus{Kind:"webhook", IntegrationID:item.ID, State:"ready", Message:"Request accepted", LastEventAt:&now, UpdatedAt:now})
	_ = a.DB.RecordIntegrationEvent(&model.IntegrationEvent{Kind:"webhook",IntegrationID:item.ID,Level:"info",Event:"request_accepted",Message:"Webhook request accepted",CreatedAt:now})
	ctx.JSON(202, gin.H{"accepted": true, "messageId": msg.ID})
}

type mqttParams struct {
	Name          string `json:"name" binding:"required"`
	ApplicationID uint   `json:"applicationId" binding:"required"`
	BrokerURL     string `json:"brokerUrl" binding:"required"`
	ClientID      string `json:"clientId"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	Topic         string `json:"topic" binding:"required"`
	Enabled       bool   `json:"enabled"`
}

func (a *AutomationAPI) GetMQTT(ctx *gin.Context) {
	items, err := a.DB.GetMQTTIntegrations()
	if !successOrAbort(ctx, 500, err) { return }
	out := make([]model.MQTTIntegrationView, 0, len(items))
	for _, item := range items {
		view := mqttView(item)
		view.Status, _ = a.DB.GetIntegrationStatus("mqtt", item.ID)
		out = append(out, view)
	}
	ctx.JSON(200, out)
}

func (a *AutomationAPI) CreateMQTT(ctx *gin.Context) {
	var params mqttParams
	if err := ctx.ShouldBindJSON(&params); err != nil { return }
	if !a.channelExists(ctx, params.ApplicationID) { return }
	if !validMQTTURL(params.BrokerURL) { ctx.AbortWithError(400, errors.New("broker URL must use mqtt, mqtts, tcp, or tls")); return }
	item := &model.MQTTIntegration{
		Name: params.Name, ApplicationID: params.ApplicationID, BrokerURL: params.BrokerURL,
		ClientID: params.ClientID, Username: params.Username, Password: params.Password, Topic: params.Topic, Enabled: params.Enabled,
	}
	if !successOrAbort(ctx, 500, a.DB.SaveMQTTIntegration(item)) { return }
	a.Engine.ReloadIntegrations()
	ctx.JSON(201, mqttView(item))
}

func (a *AutomationAPI) UpdateMQTT(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		item, err := a.DB.GetMQTTIntegrationByID(id)
		if !successOrAbort(ctx, 500, err) { return }
		if item == nil { ctx.AbortWithError(404, errors.New("MQTT connection not found")); return }
		var params mqttParams
		if err := ctx.ShouldBindJSON(&params); err != nil { return }
		if !a.channelExists(ctx, params.ApplicationID) { return }
		if !validMQTTURL(params.BrokerURL) { ctx.AbortWithError(400, errors.New("broker URL must use mqtt, mqtts, tcp, or tls")); return }
		item.Name, item.ApplicationID, item.BrokerURL, item.ClientID = params.Name, params.ApplicationID, params.BrokerURL, params.ClientID
		item.Username, item.Topic, item.Enabled = params.Username, params.Topic, params.Enabled
		if params.Password != "" { item.Password = params.Password }
		if !successOrAbort(ctx, 500, a.DB.SaveMQTTIntegration(item)) { return }
		a.Engine.ReloadIntegrations()
		ctx.JSON(200, mqttView(item))
	})
}

func (a *AutomationAPI) TestMQTTConnection(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		if !successOrAbort(ctx, 502, a.Engine.TestMQTTConnection(id)) { return }
		ctx.JSON(200, gin.H{"connected": true})
	})
}

func (a *AutomationAPI) DeleteMQTT(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		if successOrAbort(ctx, 500, a.DB.DeleteMQTTIntegration(id)) { a.Engine.ReloadIntegrations() }
	})
}

type homeAssistantParams struct {
	Name          string `json:"name" binding:"required"`
	ApplicationID uint   `json:"applicationId" binding:"required"`
	BaseURL       string `json:"baseUrl" binding:"required"`
	Token         string `json:"token"`
	EventType     string `json:"eventType"`
	Enabled       bool   `json:"enabled"`
}

func (a *AutomationAPI) GetHomeAssistant(ctx *gin.Context) {
	items, err := a.DB.GetHomeAssistantIntegrations()
	if !successOrAbort(ctx, 500, err) { return }
	out := make([]model.HomeAssistantIntegrationView, 0, len(items))
	for _, item := range items {
		view := homeAssistantView(item)
		view.Status, _ = a.DB.GetIntegrationStatus("home-assistant", item.ID)
		out = append(out, view)
	}
	ctx.JSON(200, out)
}

func (a *AutomationAPI) CreateHomeAssistant(ctx *gin.Context) {
	var params homeAssistantParams
	if err := ctx.ShouldBindJSON(&params); err != nil { return }
	if !a.channelExists(ctx, params.ApplicationID) { return }
	if !validHTTPURL(params.BaseURL) { ctx.AbortWithError(400, errors.New("Home Assistant URL must use http or https")); return }
	if strings.TrimSpace(params.Token) == "" { ctx.AbortWithError(400, errors.New("access token is required")); return }
	item := &model.HomeAssistantIntegration{
		Name: params.Name, ApplicationID: params.ApplicationID, BaseURL: strings.TrimRight(params.BaseURL, "/"),
		Token: params.Token, EventType: params.EventType, Enabled: params.Enabled,
	}
	if !successOrAbort(ctx, 500, a.DB.SaveHomeAssistantIntegration(item)) { return }
	a.Engine.ReloadIntegrations()
	ctx.JSON(201, homeAssistantView(item))
}

func (a *AutomationAPI) UpdateHomeAssistant(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		item, err := a.DB.GetHomeAssistantIntegrationByID(id)
		if !successOrAbort(ctx, 500, err) { return }
		if item == nil { ctx.AbortWithError(404, errors.New("Home Assistant connection not found")); return }
		var params homeAssistantParams
		if err := ctx.ShouldBindJSON(&params); err != nil { return }
		if !a.channelExists(ctx, params.ApplicationID) { return }
		if !validHTTPURL(params.BaseURL) { ctx.AbortWithError(400, errors.New("Home Assistant URL must use http or https")); return }
		item.Name, item.ApplicationID, item.BaseURL = params.Name, params.ApplicationID, strings.TrimRight(params.BaseURL, "/")
		item.EventType, item.Enabled = params.EventType, params.Enabled
		if params.Token != "" { item.Token = params.Token }
		if !successOrAbort(ctx, 500, a.DB.SaveHomeAssistantIntegration(item)) { return }
		a.Engine.ReloadIntegrations()
		ctx.JSON(200, homeAssistantView(item))
	})
}

type homeAssistantEventParams struct {
	EventType string         `json:"eventType" binding:"required"`
	Data      map[string]any `json:"data"`
}

func (a *AutomationAPI) SendHomeAssistantEvent(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		var params homeAssistantEventParams
		if err := ctx.ShouldBindJSON(&params); err != nil { return }
		if params.Data == nil { params.Data = map[string]any{} }
		if !successOrAbort(ctx, 502, a.Engine.SendHomeAssistantEvent(id, params.EventType, params.Data)) { return }
		ctx.JSON(200, gin.H{"sent": true})
	})
}

func (a *AutomationAPI) DeleteHomeAssistant(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		if successOrAbort(ctx, 500, a.DB.DeleteHomeAssistantIntegration(id)) { a.Engine.ReloadIntegrations() }
	})
}

type scheduleParams struct {
	Name          string     `json:"name" binding:"required"`
	ApplicationID uint       `json:"applicationId" binding:"required"`
	Title         string     `json:"title"`
	Message       string     `json:"message" binding:"required"`
	Priority      int        `json:"priority"`
	ScheduleType  string     `json:"scheduleType" binding:"required"`
	RunAt         *time.Time `json:"runAt"`
	Hour          int        `json:"hour"`
	Minute        int        `json:"minute"`
	Weekday       int        `json:"weekday"`
	Timezone      string     `json:"timezone"`
	Enabled       bool       `json:"enabled"`
}

func (a *AutomationAPI) GetSchedules(ctx *gin.Context) {
	items, err := a.DB.GetScheduledNotifications()
	if !successOrAbort(ctx, 500, err) { return }
	ctx.JSON(200, items)
}

func (a *AutomationAPI) CreateSchedule(ctx *gin.Context) {
	var params scheduleParams
	if err := ctx.ShouldBindJSON(&params); err != nil { return }
	if !a.channelExists(ctx, params.ApplicationID) { return }
	item := scheduleFromParams(params)
	if err := validateSchedule(item); err != nil { ctx.AbortWithError(400, err); return }
	item.NextRunAt = automation.NextScheduleRun(item, time.Now())
	if item.Enabled && item.NextRunAt == nil { ctx.AbortWithError(400, errors.New("schedule does not have a future run time")); return }
	if !successOrAbort(ctx, 500, a.DB.SaveScheduledNotification(item)) { return }
	ctx.JSON(201, item)
}

func (a *AutomationAPI) UpdateSchedule(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		item, err := a.DB.GetScheduledNotificationByID(id)
		if !successOrAbort(ctx, 500, err) { return }
		if item == nil { ctx.AbortWithError(404, errors.New("schedule not found")); return }
		var params scheduleParams
		if err := ctx.ShouldBindJSON(&params); err != nil { return }
		if !a.channelExists(ctx, params.ApplicationID) { return }
		updated := scheduleFromParams(params)
		updated.ID, updated.CreatedAt = item.ID, item.CreatedAt
		if err := validateSchedule(updated); err != nil { ctx.AbortWithError(400, err); return }
		updated.NextRunAt = automation.NextScheduleRun(updated, time.Now())
		if updated.Enabled && updated.NextRunAt == nil { ctx.AbortWithError(400, errors.New("schedule does not have a future run time")); return }
		if !successOrAbort(ctx, 500, a.DB.SaveScheduledNotification(updated)) { return }
		ctx.JSON(200, updated)
	})
}

func (a *AutomationAPI) DeleteSchedule(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) { successOrAbort(ctx, 500, a.DB.DeleteScheduledNotification(id)) })
}

type quietHoursParams struct {
	Enabled       bool   `json:"enabled"`
	StartMinute   int    `json:"startMinute"`
	EndMinute     int    `json:"endMinute"`
	Timezone      string `json:"timezone"`
	AllowPriority int    `json:"allowPriority"`
}

func (a *AutomationAPI) GetQuietHours(ctx *gin.Context) {
	userID := auth.GetUserID(ctx)
	item, err := a.DB.GetQuietHoursPolicy(userID)
	if !successOrAbort(ctx, 500, err) { return }
	if item == nil {
		item = &model.QuietHoursPolicy{UserID:userID, StartMinute:1320, EndMinute:420, Timezone:"UTC", AllowPriority:8}
	}
	ctx.JSON(200, item)
}

func (a *AutomationAPI) SaveQuietHours(ctx *gin.Context) {
	var params quietHoursParams
	if err := ctx.ShouldBindJSON(&params); err != nil { return }
	if params.StartMinute < 0 || params.StartMinute > 1439 || params.EndMinute < 0 || params.EndMinute > 1439 {
		ctx.AbortWithError(400, errors.New("quiet hours must be valid times of day")); return
	}
	if _, err := time.LoadLocation(valueOr(params.Timezone, "UTC")); err != nil {
		ctx.AbortWithError(400, errors.New("invalid timezone")); return
	}
	item := &model.QuietHoursPolicy{UserID:auth.GetUserID(ctx), Enabled:params.Enabled, StartMinute:params.StartMinute, EndMinute:params.EndMinute, Timezone:valueOr(params.Timezone,"UTC"), AllowPriority:params.AllowPriority}
	if !successOrAbort(ctx, 500, a.DB.SaveQuietHoursPolicy(item)) { return }
	ctx.JSON(200, item)
}

type digestParams struct {
	Enabled           bool `json:"enabled"`
	IntervalMinutes   int  `json:"intervalMinutes"`
	ImmediatePriority int  `json:"immediatePriority"`
}

func (a *AutomationAPI) GetDigest(ctx *gin.Context) {
	userID := auth.GetUserID(ctx)
	item, err := a.DB.GetDigestPolicy(userID)
	if !successOrAbort(ctx, 500, err) { return }
	if item == nil { item = &model.DigestPolicy{UserID:userID, IntervalMinutes:60, ImmediatePriority:8} }
	ctx.JSON(200, item)
}

func (a *AutomationAPI) SaveDigest(ctx *gin.Context) {
	var params digestParams
	if err := ctx.ShouldBindJSON(&params); err != nil { return }
	if params.IntervalMinutes < 15 || params.IntervalMinutes > 10080 {
		ctx.AbortWithError(400, errors.New("digest interval must be between 15 minutes and 7 days")); return
	}
	userID := auth.GetUserID(ctx)
	existing, err := a.DB.GetDigestPolicy(userID)
	if !successOrAbort(ctx, 500, err) { return }
	item := &model.DigestPolicy{UserID:userID, Enabled:params.Enabled, IntervalMinutes:params.IntervalMinutes, ImmediatePriority:params.ImmediatePriority}
	if existing != nil { item.ID, item.CreatedAt, item.LastSentAt = existing.ID, existing.CreatedAt, existing.LastSentAt }
	if params.Enabled {
		next := time.Now().Add(time.Duration(params.IntervalMinutes)*time.Minute)
		item.NextRunAt = &next
	}
	if !successOrAbort(ctx, 500, a.DB.SaveDigestPolicy(item)) { return }
	if !params.Enabled {
		if !successOrAbort(ctx, 500, a.DB.DeleteDigestItems(userID)) { return }
	}
	ctx.JSON(200, item)
}

type escalationParams struct {
	Name                string `json:"name" binding:"required"`
	SourceApplicationID uint   `json:"sourceApplicationId" binding:"required"`
	TargetApplicationID uint   `json:"targetApplicationId" binding:"required"`
	MinPriority         int    `json:"minPriority"`
	DelayMinutes        int    `json:"delayMinutes" binding:"min=1,max=10080"`
	Enabled             bool   `json:"enabled"`
}

func (a *AutomationAPI) GetEscalations(ctx *gin.Context) {
	items, err := a.DB.GetEscalationRules()
	if !successOrAbort(ctx, 500, err) { return }
	ctx.JSON(200, items)
}

func (a *AutomationAPI) CreateEscalation(ctx *gin.Context) {
	var params escalationParams
	if err := ctx.ShouldBindJSON(&params); err != nil { return }
	if params.SourceApplicationID == params.TargetApplicationID { ctx.AbortWithError(400, errors.New("source and target Channels must be different")); return }
	if !a.channelExists(ctx, params.SourceApplicationID) || !a.channelExists(ctx, params.TargetApplicationID) { return }
	item := &model.EscalationRule{Name:params.Name,SourceApplicationID:params.SourceApplicationID,TargetApplicationID:params.TargetApplicationID,MinPriority:params.MinPriority,DelayMinutes:params.DelayMinutes,Enabled:params.Enabled}
	if !successOrAbort(ctx, 500, a.DB.SaveEscalationRule(item)) { return }
	ctx.JSON(201, item)
}

func (a *AutomationAPI) UpdateEscalation(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		item, err := a.DB.GetEscalationRuleByID(id)
		if !successOrAbort(ctx, 500, err) { return }
		if item == nil { ctx.AbortWithError(404, errors.New("escalation not found")); return }
		var params escalationParams
		if err := ctx.ShouldBindJSON(&params); err != nil { return }
		if params.SourceApplicationID == params.TargetApplicationID { ctx.AbortWithError(400, errors.New("source and target Channels must be different")); return }
		if !a.channelExists(ctx, params.SourceApplicationID) || !a.channelExists(ctx, params.TargetApplicationID) { return }
		item.Name, item.SourceApplicationID, item.TargetApplicationID = params.Name, params.SourceApplicationID, params.TargetApplicationID
		item.MinPriority, item.DelayMinutes, item.Enabled = params.MinPriority, params.DelayMinutes, params.Enabled
		if !successOrAbort(ctx, 500, a.DB.SaveEscalationRule(item)) { return }
		ctx.JSON(200, item)
	})
}

func (a *AutomationAPI) DeleteEscalation(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) { successOrAbort(ctx, 500, a.DB.DeleteEscalationRule(id)) })
}

func (a *AutomationAPI) GetAcknowledgement(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		if !a.canAccessMessage(ctx, id) { return }
		value, err := a.DB.IsMessageAcknowledgedByUser(auth.GetUserID(ctx), id)
		if !successOrAbort(ctx, 500, err) { return }
		history, err := a.DB.GetMessageAcknowledgements(id)
		if !successOrAbort(ctx, 500, err) { return }
		ctx.JSON(200, gin.H{
			"acknowledged": value,
			"acknowledgedAny": len(history) > 0,
			"count": len(history),
			"acknowledgedBy": history,
		})
	})
}

func (a *AutomationAPI) AcknowledgeMessage(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		if !a.canAccessMessage(ctx, id) { return }
		if !successOrAbort(ctx, 500, a.DB.SetMessageAcknowledgement(auth.GetUserID(ctx), id, true, time.Now())) { return }
		ctx.JSON(200, gin.H{"acknowledged":true})
	})
}

func (a *AutomationAPI) UnacknowledgeMessage(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		if !a.canAccessMessage(ctx, id) { return }
		if !successOrAbort(ctx, 500, a.DB.SetMessageAcknowledgement(auth.GetUserID(ctx), id, false, time.Now())) { return }
		ctx.JSON(200, gin.H{"acknowledged":false})
	})
}

func (a *AutomationAPI) GetIntegrationEvents(ctx *gin.Context) {
	kind := strings.TrimSpace(ctx.Param("kind"))
	if kind != "webhook" && kind != "mqtt" && kind != "home-assistant" {
		ctx.AbortWithError(400, errors.New("unknown integration type"))
		return
	}
	withID(ctx, "id", func(id uint) {
		limit := 50
		if raw := ctx.Query("limit"); raw != "" {
			if parsed, err := strconv.Atoi(raw); err == nil { limit = parsed }
		}
		items, err := a.DB.GetIntegrationEvents(kind, id, limit)
		if !successOrAbort(ctx, 500, err) { return }
		ctx.JSON(200, items)
	})
}

func (a *AutomationAPI) channelExists(ctx *gin.Context, id uint) bool {
	item, err := a.DB.GetApplicationByID(id)
	if !successOrAbort(ctx, 500, err) { return false }
	if item == nil { ctx.AbortWithError(400, errors.New("Channel not found")); return false }
	return true
}

func (a *AutomationAPI) canAccessMessage(ctx *gin.Context, messageID uint) bool {
	msg, err := a.DB.GetMessageByID(messageID)
	if !successOrAbort(ctx, 500, err) { return false }
	if msg == nil { ctx.AbortWithError(404, errors.New("message not found")); return false }
	membership, err := a.DB.GetApplicationMembership(msg.ApplicationID, auth.GetUserID(ctx))
	if !successOrAbort(ctx, 500, err) { return false }
	if membership == nil { ctx.AbortWithError(404, errors.New("message not found")); return false }
	return true
}

func scheduleFromParams(params scheduleParams) *model.ScheduledNotification {
	return &model.ScheduledNotification{
		Name:params.Name,ApplicationID:params.ApplicationID,Title:params.Title,Message:params.Message,Priority:params.Priority,
		ScheduleType:params.ScheduleType,RunAt:params.RunAt,Hour:params.Hour,Minute:params.Minute,Weekday:params.Weekday,
		Timezone:valueOr(params.Timezone,"UTC"),Enabled:params.Enabled,
	}
}

func validateSchedule(item *model.ScheduledNotification) error {
	switch item.ScheduleType {
	case "once":
		if item.RunAt == nil { return errors.New("one-time schedules require a date and time") }
	case "hourly":
		if item.Minute < 0 || item.Minute > 59 { return errors.New("minute must be between 0 and 59") }
	case "daily":
		if item.Hour < 0 || item.Hour > 23 || item.Minute < 0 || item.Minute > 59 { return errors.New("invalid daily time") }
	case "weekly":
		if item.Weekday < 0 || item.Weekday > 6 || item.Hour < 0 || item.Hour > 23 || item.Minute < 0 || item.Minute > 59 { return errors.New("invalid weekly schedule") }
	default:
		return errors.New("schedule type must be once, hourly, daily, or weekly")
	}
	if _, err := time.LoadLocation(valueOr(item.Timezone,"UTC")); err != nil { return errors.New("invalid timezone") }
	return nil
}

func webhookView(item *model.WebhookRoute) model.WebhookRouteView {
	return model.WebhookRouteView{
		ID:item.ID,Name:item.Name,ApplicationID:item.ApplicationID,Enabled:item.Enabled,
		Path:"/integrations/webhook/"+item.Secret,TitleField:item.TitleField,MessageField:item.MessageField,
		PriorityField:item.PriorityField,DefaultTitle:item.DefaultTitle,DefaultPriority:item.DefaultPriority,
		AllowedCIDRs:splitCSV(item.AllowedCIDRs),RequireSignature:item.RequireSignature,
		SignatureConfigured:item.SigningSecret!="",ReplayWindowSeconds:item.ReplayWindowSeconds,
		CreatedAt:item.CreatedAt,UpdatedAt:item.UpdatedAt,
	}
}

func mqttView(item *model.MQTTIntegration) model.MQTTIntegrationView {
	return model.MQTTIntegrationView{
		ID:item.ID,Name:item.Name,ApplicationID:item.ApplicationID,BrokerURL:item.BrokerURL,ClientID:item.ClientID,
		Username:item.Username,PasswordConfigured:item.Password!="",Topic:item.Topic,Enabled:item.Enabled,
		CreatedAt:item.CreatedAt,UpdatedAt:item.UpdatedAt,
	}
}

func homeAssistantView(item *model.HomeAssistantIntegration) model.HomeAssistantIntegrationView {
	return model.HomeAssistantIntegrationView{
		ID:item.ID,Name:item.Name,ApplicationID:item.ApplicationID,BaseURL:item.BaseURL,
		TokenConfigured:item.Token!="",EventType:item.EventType,Enabled:item.Enabled,
		CreatedAt:item.CreatedAt,UpdatedAt:item.UpdatedAt,
	}
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" { return nil }
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" { result = append(result, trimmed) }
	}
	return result
}

func validateWebhookSecurity(cidrs []string, requireSignature bool, signingSecret string, replayWindow int) error {
	for _, raw := range cidrs {
		if _, _, err := net.ParseCIDR(strings.TrimSpace(raw)); err != nil {
			return fmt.Errorf("invalid allowed CIDR %q", raw)
		}
	}
	if requireSignature && len(signingSecret) < 16 {
		return errors.New("signed webhooks require a signing secret of at least 16 characters")
	}
	if replayWindow < 0 || replayWindow > 3600 {
		return errors.New("webhook replay window must be between 0 and 3600 seconds")
	}
	return nil
}

func webhookSourceAllowed(raw, clientIP string) bool {
	cidrs := splitCSV(raw)
	if len(cidrs) == 0 { return true }
	ip := net.ParseIP(clientIP)
	if ip == nil { return false }
	for _, value := range cidrs {
		_, network, err := net.ParseCIDR(value)
		if err == nil && network.Contains(ip) { return true }
	}
	return false
}

func verifyWebhookSignature(item *model.WebhookRoute, body []byte, timestampHeader, signatureHeader string, now time.Time) error {
	if !item.RequireSignature { return nil }
	if item.SigningSecret == "" { return errors.New("Webhook signing is not configured") }
	timestamp, err := strconv.ParseInt(strings.TrimSpace(timestampHeader), 10, 64)
	if err != nil { return errors.New("Webhook timestamp is required") }
	window := item.ReplayWindowSeconds
	if window <= 0 { window = 300 }
	sentAt := time.Unix(timestamp, 0)
	delta := now.Sub(sentAt)
	if delta < 0 { delta = -delta }
	if delta > time.Duration(window)*time.Second {
		return errors.New("Webhook timestamp is outside the allowed replay window")
	}
	provided := strings.TrimSpace(strings.TrimPrefix(signatureHeader, "sha256="))
	providedBytes, err := hex.DecodeString(provided)
	if err != nil || len(providedBytes) != sha256.Size {
		return errors.New("Webhook signature is invalid")
	}
	mac := hmac.New(sha256.New, []byte(item.SigningSecret))
	_, _ = mac.Write([]byte(timestampHeader))
	_, _ = mac.Write([]byte("."))
	_, _ = mac.Write(body)
	if !hmac.Equal(providedBytes, mac.Sum(nil)) {
		return errors.New("Webhook signature is invalid")
	}
	return nil
}

func generateIntegrationSecret() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil { return "", err }
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func lookupPayload(payload any, path string) (any, bool) {
	if strings.TrimSpace(path) == "" { return nil, false }
	current := payload
	for _, part := range strings.Split(path, ".") {
		object, ok := current.(map[string]any)
		if !ok { return nil, false }
		current, ok = object[part]
		if !ok { return nil, false }
	}
	return current, true
}

func payloadString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		encoded, err := json.Marshal(typed)
		if err != nil { return "" }
		return string(encoded)
	}
}

func payloadInt(value any) (int, bool) {
	switch typed := value.(type) {
	case float64: return int(typed), true
	case int: return typed, true
	case string:
		parsed, err := strconv.Atoi(typed)
		return parsed, err == nil
	default: return 0, false
	}
}

func valueOr(value, fallback string) string {
	if strings.TrimSpace(value) == "" { return fallback }
	return strings.TrimSpace(value)
}

func validMQTTURL(raw string) bool {
	lower := strings.ToLower(strings.TrimSpace(raw))
	return strings.HasPrefix(lower,"mqtt://") || strings.HasPrefix(lower,"mqtts://") || strings.HasPrefix(lower,"tcp://") || strings.HasPrefix(lower,"tls://")
}

func validHTTPURL(raw string) bool {
	lower := strings.ToLower(strings.TrimSpace(raw))
	return strings.HasPrefix(lower,"http://") || strings.HasPrefix(lower,"https://")
}

func integrationID(raw string) (uint, error) {
	value, err := strconv.ParseUint(raw, 10, 64)
	return uint(value), err
}

func integrationError(name string, id uint) error {
	return fmt.Errorf("%s %d not found", name, id)
}
