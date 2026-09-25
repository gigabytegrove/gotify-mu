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
	ConsumeWebhookReplay(routeID uint, signature string, now time.Time, ttl time.Duration) (bool, error)

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
	GetMessageAcknowledgements(messageID uint) ([]*model.MessageAcknowledgementView, error)
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
	DefaultPriority  int    `json:"defaultPriority"`
	RequireSignature bool   `json:"requireSignature"`
	SigningSecret    string `json:"signingSecret"`
	AllowedCIDRs     string `json:"allowedCidrs"`
}

func (a *AutomationAPI) GetWebhookRoutes(ctx *gin.Context) {
	items, err := a.DB.GetWebhookRoutes()
	if !successOrAbort(ctx, 500, err) { return }
	out := make([]model.WebhookRouteView, 0, len(items))
	for _, item := range items { out = append(out, webhookView(item)) }
	ctx.JSON(200, out)
}

func (a *AutomationAPI) CreateWebhookRoute(ctx *gin.Context) {
	var params webhookParams
	if err := ctx.ShouldBindJSON(&params); err != nil { return }
	if !a.channelExists(ctx, params.ApplicationID) { return }
	if params.RequireSignature && strings.TrimSpace(params.SigningSecret) == "" {
		ctx.AbortWithError(400, errors.New("signing secret is required when request signing is enabled"))
		return
	}
	if err := validateCIDRList(params.AllowedCIDRs); err != nil {
		ctx.AbortWithError(400, err)
		return
	}
	secret, err := generateIntegrationSecret()
	if !successOrAbort(ctx, 500, err) { return }
	item := &model.WebhookRoute{
		Name: params.Name, ApplicationID: params.ApplicationID, Secret: secret, Enabled: params.Enabled,
		TitleField: valueOr(params.TitleField, "title"), MessageField: valueOr(params.MessageField, "message"),
		PriorityField: valueOr(params.PriorityField, "priority"), DefaultTitle: params.DefaultTitle,
		DefaultPriority: params.DefaultPriority, RequireSignature: params.RequireSignature,
		SigningSecret: params.SigningSecret, AllowedCIDRs: strings.TrimSpace(params.AllowedCIDRs),
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
		item.PriorityField, item.DefaultTitle, item.DefaultPriority = valueOr(params.PriorityField, "priority"), params.DefaultTitle, params.DefaultPriority
		item.RequireSignature, item.AllowedCIDRs = params.RequireSignature, strings.TrimSpace(params.AllowedCIDRs)
		if params.SigningSecret != "" { item.SigningSecret = params.SigningSecret }
		if item.RequireSignature && item.SigningSecret == "" {
			ctx.AbortWithError(400, errors.New("signing secret is required when request signing is enabled"))
			return
		}
		if err := validateCIDRList(item.AllowedCIDRs); err != nil {
			ctx.AbortWithError(400, err)
			return
		}
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
	if !ipAllowed(ctx.ClientIP(), item.AllowedCIDRs) {
		ctx.AbortWithStatus(403)
		return
	}

	const maxWebhookBody = int64(1024 * 1024)
	body, err := io.ReadAll(io.LimitReader(ctx.Request.Body, maxWebhookBody+1))
	if !successOrAbort(ctx, 400, err) { return }
	if int64(len(body)) > maxWebhookBody {
		ctx.AbortWithError(413, errors.New("webhook payload exceeds the 1 MiB limit"))
		return
	}
	if item.RequireSignature {
		if err := verifyWebhookSignature(item.SigningSecret, ctx.GetHeader("X-Gotify-MU-Timestamp"), ctx.GetHeader("X-Gotify-MU-Signature"), body, time.Now()); err != nil {
			ctx.AbortWithError(401, err)
			return
		}
		signature := strings.TrimSpace(ctx.GetHeader("X-Gotify-MU-Signature"))
		accepted, err := a.DB.ConsumeWebhookReplay(item.ID, signature, time.Now(), 10*time.Minute)
		if !successOrAbort(ctx, 500, err) { return }
		if !accepted {
			ctx.AbortWithError(409, errors.New("webhook request was already received"))
			return
		}
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
	if !successOrAbort(ctx, 500, err) { return }
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
	for _, item := range items { out = append(out, mqttView(item)) }
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
	for _, item := range items { out = append(out, homeAssistantView(item)) }
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
	} else {
		item.NextRunAt = nil
		if !successOrAbort(ctx, 500, a.DB.DeleteDigestItems(userID)) { return }
	}
	if !successOrAbort(ctx, 500, a.DB.SaveDigestPolicy(item)) { return }
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
		ctx.JSON(200, gin.H{"acknowledged":value})
	})
}

func (a *AutomationAPI) GetAcknowledgements(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		if !a.canAccessMessage(ctx, id) { return }
		items, err := a.DB.GetMessageAcknowledgements(id)
		if !successOrAbort(ctx, 500, err) { return }
		ctx.JSON(200, items)
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
		RequireSignature:item.RequireSignature,SignatureConfigured:item.SigningSecret!="",AllowedCIDRs:item.AllowedCIDRs,
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


func validateCIDRList(raw string) error {
	for _, entry := range strings.Split(raw, ",") {
		value := strings.TrimSpace(entry)
		if value == "" { continue }
		if _, _, err := net.ParseCIDR(value); err != nil {
			return fmt.Errorf("invalid allowed network %q", value)
		}
	}
	return nil
}

func ipAllowed(ipText, rawCIDRs string) bool {
	if strings.TrimSpace(rawCIDRs) == "" {
		return true
	}
	ip := net.ParseIP(strings.TrimSpace(ipText))
	if ip == nil {
		return false
	}
	for _, entry := range strings.Split(rawCIDRs, ",") {
		value := strings.TrimSpace(entry)
		if value == "" { continue }
		_, network, err := net.ParseCIDR(value)
		if err == nil && network.Contains(ip) {
			return true
		}
	}
	return false
}

func verifyWebhookSignature(secret, timestamp, signature string, body []byte, now time.Time) error {
	if secret == "" {
		return errors.New("webhook signing is enabled but no signing secret is configured")
	}
	unixTime, err := strconv.ParseInt(strings.TrimSpace(timestamp), 10, 64)
	if err != nil {
		return errors.New("invalid webhook timestamp")
	}
	requestTime := time.Unix(unixTime, 0)
	delta := now.Sub(requestTime)
	if delta < -5*time.Minute || delta > 5*time.Minute {
		return errors.New("webhook timestamp is outside the allowed window")
	}
	provided := strings.TrimSpace(signature)
	provided = strings.TrimPrefix(provided, "sha256=")
	decoded, err := hex.DecodeString(provided)
	if err != nil {
		return errors.New("invalid webhook signature")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(timestamp))
	_, _ = mac.Write([]byte("."))
	_, _ = mac.Write(body)
	if !hmac.Equal(decoded, mac.Sum(nil)) {
		return errors.New("invalid webhook signature")
	}
	return nil
}
