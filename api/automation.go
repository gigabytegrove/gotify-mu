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
	"github.com/gotify/server/v3/security"
	"github.com/robfig/cron"
)

type AutomationEngine interface {
	Publish(applicationID uint, title, message string, priority int) (*model.Message, error)
	ReloadIntegrations()
	SendHomeAssistantEvent(id uint, eventType string, data map[string]any) error
	TestMQTTConnection(id uint) error
}

type AutomationDatabase interface {
	GetApplicationByID(id uint) (*model.Application, error)
	GetUserByID(id uint) (*model.User, error)
	GetUserGroupByID(id uint) (*model.UserGroup, error)
	GetApplicationMembership(applicationID, userID uint) (*model.ApplicationMembership, error)
	GetMessageByID(id uint) (*model.Message, error)

	GetWebhookRoutes() ([]*model.WebhookRoute, error)
	GetWebhookRouteByID(id uint) (*model.WebhookRoute, error)
	GetWebhookRouteBySecret(secret string) (*model.WebhookRoute, error)
	SaveWebhookRoute(item *model.WebhookRoute) error
	DeleteWebhookRoute(id uint) error
	CreateWebhookDelivery(item *model.WebhookDelivery) error
	GetWebhookDeliveries(routeID uint, limit int) ([]*model.WebhookDelivery, error)

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
	GetScheduledNotificationRuns(scheduleID uint, limit int) ([]*model.ScheduledNotificationRun, error)

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
	DB             AutomationDatabase
	Engine         AutomationEngine
	WebhookLimiter *security.DynamicLimiter
	WebhookReplay  *security.ReplayCache
}

type webhookParams struct {
	Name            string `json:"name" binding:"required"`
	ApplicationID   uint   `json:"applicationId" binding:"required"`
	Enabled         bool   `json:"enabled"`
	TitleField      string `json:"titleField"`
	MessageField    string `json:"messageField"`
	PriorityField   string `json:"priorityField"`
	MatchField      string `json:"matchField"`
	MatchValue      string `json:"matchValue"`
	TitleTemplate   string `json:"titleTemplate"`
	MessageTemplate string `json:"messageTemplate"`
	DefaultTitle    string `json:"defaultTitle"`
	DefaultPriority    int    `json:"defaultPriority"`
	RequireSignature   bool   `json:"requireSignature"`
	AllowedCIDRs       string `json:"allowedCidrs"`
	RateLimitPerMinute int    `json:"rateLimitPerMinute"`
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
	secret, err := generateIntegrationSecret()
	if !successOrAbort(ctx, 500, err) { return }
	item := &model.WebhookRoute{
		Name: params.Name, ApplicationID: params.ApplicationID, Secret: secret, Enabled: params.Enabled,
		TitleField: valueOr(params.TitleField, "title"), MessageField: valueOr(params.MessageField, "message"),
		PriorityField: valueOr(params.PriorityField, "priority"),
		MatchField: strings.TrimSpace(params.MatchField), MatchValue: params.MatchValue,
		TitleTemplate: params.TitleTemplate, MessageTemplate: params.MessageTemplate,
		DefaultTitle: params.DefaultTitle, DefaultPriority: params.DefaultPriority,
		RequireSignature: params.RequireSignature,
		AllowedCIDRs: strings.TrimSpace(params.AllowedCIDRs),
		RateLimitPerMinute: normalizedWebhookRateLimit(params.RateLimitPerMinute),
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
		item.PriorityField = valueOr(params.PriorityField, "priority")
		item.MatchField, item.MatchValue = strings.TrimSpace(params.MatchField), params.MatchValue
		item.TitleTemplate, item.MessageTemplate = params.TitleTemplate, params.MessageTemplate
		item.DefaultTitle, item.DefaultPriority = params.DefaultTitle, params.DefaultPriority
		item.RequireSignature = params.RequireSignature
		item.AllowedCIDRs = strings.TrimSpace(params.AllowedCIDRs)
		item.RateLimitPerMinute = normalizedWebhookRateLimit(params.RateLimitPerMinute)
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

func (a *AutomationAPI) TestWebhookRoute(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		item, err := a.DB.GetWebhookRouteByID(id)
		if !successOrAbort(ctx, 500, err) { return }
		if item == nil { ctx.AbortWithError(404, errors.New("webhook not found")); return }
		msg, err := a.Engine.Publish(item.ApplicationID, "Webhook test", "Gotify MU webhook test completed successfully.", item.DefaultPriority)
		if !successOrAbort(ctx, 500, err) { return }
		ctx.JSON(200, gin.H{"sent":true, "messageId":msg.ID})
	})
}

func (a *AutomationAPI) DeleteWebhookRoute(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) { successOrAbort(ctx, 500, a.DB.DeleteWebhookRoute(id)) })
}

func (a *AutomationAPI) GetWebhookDeliveries(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		item, err := a.DB.GetWebhookRouteByID(id)
		if !successOrAbort(ctx, 500, err) { return }
		if item == nil { ctx.AbortWithError(404, errors.New("webhook not found")); return }
		limit := 100
		if raw := ctx.Query("limit"); raw != "" {
			if parsed, parseErr := strconv.Atoi(raw); parseErr == nil { limit = parsed }
		}
		items, err := a.DB.GetWebhookDeliveries(id, limit)
		if !successOrAbort(ctx, 500, err) { return }
		ctx.JSON(200, items)
	})
}


func (a *AutomationAPI) ReceiveWebhook(ctx *gin.Context) {
	secret := strings.TrimSpace(ctx.Param("secret"))
	item, err := a.DB.GetWebhookRouteBySecret(secret)
	if !successOrAbort(ctx, 500, err) { return }
	if item == nil { ctx.AbortWithStatus(http.StatusNotFound); return }

	record := func(status, detail string, messageID uint) {
		if len(detail) > 500 { detail = detail[:500] }
		// Delivery history is diagnostic and must not break webhook delivery.
		_ = a.DB.CreateWebhookDelivery(&model.WebhookDelivery{
			WebhookRouteID:item.ID, IPAddress:ctx.ClientIP(), Status:status, Detail:detail, MessageID:messageID,
		})
	}

	if !webhookIPAllowed(ctx.ClientIP(), item.AllowedCIDRs) {
		record("rejected", "source IP is not allowed", 0)
		ctx.AbortWithStatus(http.StatusForbidden)
		return
	}

	limit := normalizedWebhookRateLimit(item.RateLimitPerMinute)
	if a.WebhookLimiter != nil {
		key := fmt.Sprintf("%d|%s", item.ID, ctx.ClientIP())
		if allowed, retry := a.WebhookLimiter.Allow(key, limit, time.Minute); !allowed {
			seconds := int(retry.Round(time.Second).Seconds())
			if seconds < 1 { seconds = 1 }
			record("rate_limited", "rate limit exceeded", 0)
			ctx.Header("Retry-After", strconv.Itoa(seconds))
			ctx.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error":"webhook rate limit exceeded"})
			return
		}
	}

	const maxWebhookBytes = 1 << 20
	body, err := io.ReadAll(io.LimitReader(ctx.Request.Body, maxWebhookBytes+1))
	if err != nil {
		record("failed", "request body could not be read", 0)
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}
	if len(body) > maxWebhookBytes {
		record("rejected", "payload exceeds 1 MiB", 0)
		ctx.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{"error":"webhook payload exceeds 1 MiB"})
		return
	}

	if item.RequireSignature {
		timestamp := strings.TrimSpace(ctx.GetHeader("X-Gotify-MU-Timestamp"))
		signature := strings.TrimSpace(ctx.GetHeader("X-Gotify-MU-Signature"))
		if !validWebhookSignature(secret, timestamp, signature, body, time.Now()) {
			record("rejected", "signature verification failed", 0)
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		if a.WebhookReplay != nil {
			replayKey := security.HashSecret(secret + "|" + timestamp + "|" + signature)
			if !a.WebhookReplay.Remember(replayKey, 10*time.Minute) {
				record("rejected", "duplicate signed request", 0)
				ctx.AbortWithStatusJSON(http.StatusConflict, gin.H{"error":"duplicate webhook request"})
				return
			}
		}
	}

	title := item.DefaultTitle
	message := strings.TrimSpace(string(body))
	priority := item.DefaultPriority
	var payload any
	jsonPayload := json.Unmarshal(body, &payload) == nil

	if item.MatchField != "" {
		if !jsonPayload {
			record("ignored", "conditional rule requires a JSON payload", 0)
			ctx.JSON(http.StatusAccepted, gin.H{"accepted":false,"matched":false})
			return
		}
		value, ok := lookupPayload(payload, item.MatchField)
		if !ok || (item.MatchValue != "" && payloadString(value) != item.MatchValue) {
			record("ignored", "conditional routing rule did not match", 0)
			ctx.JSON(http.StatusAccepted, gin.H{"accepted":false,"matched":false})
			return
		}
	}

	if jsonPayload {
		if item.TitleTemplate != "" {
			title = renderPayloadTemplate(item.TitleTemplate, payload, string(body))
		} else if value, ok := lookupPayload(payload, item.TitleField); ok {
			if text := payloadString(value); text != "" { title = text }
		}

		if item.MessageTemplate != "" {
			message = renderPayloadTemplate(item.MessageTemplate, payload, string(body))
		} else if value, ok := lookupPayload(payload, item.MessageField); ok {
			if text := payloadString(value); text != "" { message = text }
		} else if encoded, marshalErr := json.MarshalIndent(payload, "", "  "); marshalErr == nil {
			message = string(encoded)
		}

		if value, ok := lookupPayload(payload, item.PriorityField); ok {
			if number, numberOK := payloadInt(value); numberOK { priority = number }
		}
	}

	if strings.TrimSpace(message) == "" {
		record("failed", "payload did not contain a message", 0)
		ctx.AbortWithError(http.StatusBadRequest, errors.New("webhook payload did not contain a message"))
		return
	}
	msg, err := a.Engine.Publish(item.ApplicationID, title, message, priority)
	if err != nil {
		record("failed", "message delivery failed", 0)
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	record("delivered", "", msg.ID)
	ctx.JSON(http.StatusAccepted, gin.H{"accepted": true, "matched":true, "messageId": msg.ID})
}

type mqttParams struct {
	Name              string `json:"name" binding:"required"`
	ApplicationID     uint   `json:"applicationId" binding:"required"`
	BrokerURL         string `json:"brokerUrl" binding:"required"`
	ClientID          string `json:"clientId"`
	Username          string `json:"username"`
	Password          string `json:"password"`
	ProtocolVersion   int    `json:"protocolVersion"`
	QoS               int    `json:"qos"`
	CACertificate     string `json:"caCertificate"`
	ClientCertificate string `json:"clientCertificate"`
	ClientKey         string `json:"clientKey"`
	Topic             string `json:"topic" binding:"required"`
	Enabled           bool   `json:"enabled"`
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
	if params.ProtocolVersion == 0 { params.ProtocolVersion = 5 }
	if params.ProtocolVersion != 4 && params.ProtocolVersion != 5 {
		ctx.AbortWithError(400, errors.New("MQTT protocol version must be 4 (3.1.1) or 5")); return
	}
	if params.QoS < 0 || params.QoS > 2 {
		ctx.AbortWithError(400, errors.New("MQTT QoS must be 0, 1, or 2")); return
	}
	if (strings.TrimSpace(params.ClientCertificate) == "") != (strings.TrimSpace(params.ClientKey) == "") {
		ctx.AbortWithError(400, errors.New("MQTT client certificate and private key must be provided together")); return
	}
	item := &model.MQTTIntegration{
		Name: params.Name, ApplicationID: params.ApplicationID, BrokerURL: params.BrokerURL,
		ClientID: params.ClientID, Username: params.Username, Password: params.Password,
		ProtocolVersion: params.ProtocolVersion, QoS: params.QoS,
		CACertificate: params.CACertificate, ClientCertificate: params.ClientCertificate, ClientKey: params.ClientKey,
		Topic: params.Topic, Enabled: params.Enabled,
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
		if params.ProtocolVersion == 0 { params.ProtocolVersion = item.ProtocolVersion }
		if params.ProtocolVersion == 0 { params.ProtocolVersion = 5 }
		if params.ProtocolVersion != 4 && params.ProtocolVersion != 5 {
			ctx.AbortWithError(400, errors.New("MQTT protocol version must be 4 (3.1.1) or 5")); return
		}
		if params.QoS < 0 || params.QoS > 2 {
			ctx.AbortWithError(400, errors.New("MQTT QoS must be 0, 1, or 2")); return
		}
		if strings.TrimSpace(params.ClientCertificate) != "" && strings.TrimSpace(params.ClientKey) == "" && item.ClientKey == "" {
			ctx.AbortWithError(400, errors.New("MQTT client private key is required with a client certificate")); return
		}
		item.Name, item.ApplicationID, item.BrokerURL, item.ClientID = params.Name, params.ApplicationID, params.BrokerURL, params.ClientID
		item.Username, item.Topic, item.Enabled = params.Username, params.Topic, params.Enabled
		item.ProtocolVersion, item.QoS = params.ProtocolVersion, params.QoS
		item.CACertificate = params.CACertificate
		item.ClientCertificate = params.ClientCertificate
		if params.Password != "" { item.Password = params.Password }
		if params.ClientKey != "" { item.ClientKey = params.ClientKey }
		if !successOrAbort(ctx, 500, a.DB.SaveMQTTIntegration(item)) { return }
		a.Engine.ReloadIntegrations()
		ctx.JSON(200, mqttView(item))
	})
}

func (a *AutomationAPI) TestMQTT(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		if !successOrAbort(ctx, 502, a.Engine.TestMQTTConnection(id)) { return }
		ctx.JSON(200, gin.H{"connected":true})
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
	EntityIDs     string `json:"entityIds"`
	DataField     string `json:"dataField"`
	DataValue     string `json:"dataValue"`
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
		Token: params.Token, EventType: strings.TrimSpace(params.EventType),
		EntityIDs: strings.TrimSpace(params.EntityIDs), DataField: strings.TrimSpace(params.DataField), DataValue: params.DataValue,
		Enabled: params.Enabled,
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
		item.EventType = strings.TrimSpace(params.EventType)
		item.EntityIDs = strings.TrimSpace(params.EntityIDs)
		item.DataField = strings.TrimSpace(params.DataField)
		item.DataValue = params.DataValue
		item.Enabled = params.Enabled
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
	Name           string     `json:"name" binding:"required"`
	ApplicationID  uint       `json:"applicationId" binding:"required"`
	Title          string     `json:"title"`
	Message        string     `json:"message" binding:"required"`
	Priority       int        `json:"priority"`
	ScheduleType   string     `json:"scheduleType" binding:"required"`
	RunAt          *time.Time `json:"runAt"`
	Hour           int        `json:"hour"`
	Minute         int        `json:"minute"`
	Weekday        int        `json:"weekday"`
	CronExpression string     `json:"cronExpression"`
	ExcludedDates  string     `json:"excludedDates"`
	Timezone       string     `json:"timezone"`
	EndAt          *time.Time `json:"endAt"`
	MaxRuns        int        `json:"maxRuns"`
	MisfirePolicy  string     `json:"misfirePolicy"`
	Enabled        bool       `json:"enabled"`
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

func (a *AutomationAPI) GetScheduleRuns(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		item, err := a.DB.GetScheduledNotificationByID(id)
		if !successOrAbort(ctx, 500, err) { return }
		if item == nil { ctx.AbortWithError(404, errors.New("schedule not found")); return }
		limit := 50
		if raw := ctx.Query("limit"); raw != "" {
			if parsed, parseErr := strconv.Atoi(raw); parseErr == nil { limit = parsed }
		}
		runs, err := a.DB.GetScheduledNotificationRuns(id, limit)
		if !successOrAbort(ctx, 500, err) { return }
		ctx.JSON(200, runs)
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
	Mode          string `json:"mode"`
}

func (a *AutomationAPI) GetQuietHours(ctx *gin.Context) {
	userID := auth.GetUserID(ctx)
	item, err := a.DB.GetQuietHoursPolicy(userID)
	if !successOrAbort(ctx, 500, err) { return }
	if item == nil {
		item = &model.QuietHoursPolicy{UserID:userID, StartMinute:1320, EndMinute:420, Timezone:"UTC", AllowPriority:8, Mode:"suppress"}
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
	mode := strings.ToLower(strings.TrimSpace(params.Mode))
	if mode == "" { mode = "suppress" }
	if mode != "suppress" && mode != "defer" {
		ctx.AbortWithError(400, errors.New("quiet-hours mode must be suppress or defer")); return
	}
	item := &model.QuietHoursPolicy{UserID:auth.GetUserID(ctx), Enabled:params.Enabled, StartMinute:params.StartMinute, EndMinute:params.EndMinute, Timezone:valueOr(params.Timezone,"UTC"), AllowPriority:params.AllowPriority, Mode:mode}
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
	TargetApplicationID uint   `json:"targetApplicationId"`
	TargetType          string `json:"targetType"`
	TargetID            uint   `json:"targetId"`
	MinPriority         int    `json:"minPriority"`
	DelayMinutes        int    `json:"delayMinutes" binding:"min=1,max=10080"`
	RepeatMinutes       int    `json:"repeatMinutes"`
	MaxRepeats          int    `json:"maxRepeats"`
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
	if !a.channelExists(ctx, params.SourceApplicationID) { return }
	targetType, targetID, ok := a.validateEscalationTarget(ctx, params)
	if !ok { return }
	if params.RepeatMinutes < 0 || params.RepeatMinutes > 10080 || params.MaxRepeats < 0 || params.MaxRepeats > 1000 {
		ctx.AbortWithError(400, errors.New("invalid escalation repeat settings")); return
	}
	item := &model.EscalationRule{
		Name:params.Name,SourceApplicationID:params.SourceApplicationID,
		TargetType:targetType,TargetID:targetID,MinPriority:params.MinPriority,
		DelayMinutes:params.DelayMinutes,RepeatMinutes:params.RepeatMinutes,MaxRepeats:params.MaxRepeats,Enabled:params.Enabled,
	}
	if targetType == "channel" { item.TargetApplicationID = targetID }
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
		if !a.channelExists(ctx, params.SourceApplicationID) { return }
		targetType, targetID, ok := a.validateEscalationTarget(ctx, params)
		if !ok { return }
		if params.RepeatMinutes < 0 || params.RepeatMinutes > 10080 || params.MaxRepeats < 0 || params.MaxRepeats > 1000 {
			ctx.AbortWithError(400, errors.New("invalid escalation repeat settings")); return
		}
		item.Name = params.Name
		item.SourceApplicationID = params.SourceApplicationID
		item.TargetType = targetType
		item.TargetID = targetID
		item.TargetApplicationID = 0
		if targetType == "channel" { item.TargetApplicationID = targetID }
		item.MinPriority = params.MinPriority
		item.DelayMinutes = params.DelayMinutes
		item.RepeatMinutes = params.RepeatMinutes
		item.MaxRepeats = params.MaxRepeats
		item.Enabled = params.Enabled
		if !successOrAbort(ctx, 500, a.DB.SaveEscalationRule(item)) { return }
		ctx.JSON(200, item)
	})
}

func (a *AutomationAPI) validateEscalationTarget(ctx *gin.Context, params escalationParams) (string, uint, bool) {
	targetType := strings.ToLower(strings.TrimSpace(params.TargetType))
	if targetType == "" { targetType = "channel" }
	targetID := params.TargetID
	if targetID == 0 { targetID = params.TargetApplicationID }
	if targetID == 0 {
		ctx.AbortWithError(400, errors.New("escalation target is required"))
		return "", 0, false
	}
	switch targetType {
	case "channel":
		if params.SourceApplicationID == targetID {
			ctx.AbortWithError(400, errors.New("source and target Channels must be different"))
			return "", 0, false
		}
		if !a.channelExists(ctx, targetID) { return "", 0, false }
	case "user":
		user, err := a.DB.GetUserByID(targetID)
		if !successOrAbort(ctx, 500, err) { return "", 0, false }
		if user == nil {
			ctx.AbortWithError(400, errors.New("escalation user not found"))
			return "", 0, false
		}
	case "group":
		group, err := a.DB.GetUserGroupByID(targetID)
		if !successOrAbort(ctx, 500, err) { return "", 0, false }
		if group == nil {
			ctx.AbortWithError(400, errors.New("escalation Group not found"))
			return "", 0, false
		}
	default:
		ctx.AbortWithError(400, errors.New("escalation target type must be channel, user, or group"))
		return "", 0, false
	}
	return targetType, targetID, true
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
			"acknowledged":value,
			"acknowledgedByAnyone":len(history) > 0,
			"acknowledgementCount":len(history),
			"acknowledgements":history,
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
	misfire := strings.ToLower(strings.TrimSpace(params.MisfirePolicy))
	if misfire == "" { misfire = "send" }
	return &model.ScheduledNotification{
		Name:params.Name,ApplicationID:params.ApplicationID,Title:params.Title,Message:params.Message,Priority:params.Priority,
		ScheduleType:strings.ToLower(strings.TrimSpace(params.ScheduleType)),RunAt:params.RunAt,Hour:params.Hour,Minute:params.Minute,Weekday:params.Weekday,
		CronExpression:strings.TrimSpace(params.CronExpression),ExcludedDates:strings.TrimSpace(params.ExcludedDates),
		Timezone:valueOr(params.Timezone,"UTC"),EndAt:params.EndAt,MaxRuns:params.MaxRuns,MisfirePolicy:misfire,Enabled:params.Enabled,
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
	case "cron":
		if strings.TrimSpace(item.CronExpression) == "" { return errors.New("cron schedules require an expression") }
		if _, err := cron.Parse(item.CronExpression); err != nil { return errors.New("invalid cron expression") }
	default:
		return errors.New("schedule type must be once, hourly, daily, weekly, or cron")
	}
	if _, err := time.LoadLocation(valueOr(item.Timezone,"UTC")); err != nil { return errors.New("invalid timezone") }
	if item.MaxRuns < 0 { return errors.New("maximum runs cannot be negative") }
	if item.MisfirePolicy != "send" && item.MisfirePolicy != "skip" {
		return errors.New("misfire policy must be send or skip")
	}
	if item.EndAt != nil && item.RunAt != nil && item.ScheduleType == "once" && item.RunAt.After(*item.EndAt) {
		return errors.New("one-time schedule is after the configured end date")
	}
	for _, raw := range strings.FieldsFunc(item.ExcludedDates, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == ' ' || r == '\t'
	}) {
		if _, err := time.Parse("2006-01-02", strings.TrimSpace(raw)); err != nil {
			return fmt.Errorf("invalid excluded date %q; use YYYY-MM-DD", raw)
		}
	}
	return nil
}

func webhookView(item *model.WebhookRoute) model.WebhookRouteView {
	return model.WebhookRouteView{
		ID:item.ID,Name:item.Name,ApplicationID:item.ApplicationID,Enabled:item.Enabled,
		RequireSignature:item.RequireSignature,AllowedCIDRs:item.AllowedCIDRs,RateLimitPerMinute:normalizedWebhookRateLimit(item.RateLimitPerMinute),
		Path:"/integrations/webhook/"+item.Secret,TitleField:item.TitleField,MessageField:item.MessageField,
		PriorityField:item.PriorityField,MatchField:item.MatchField,MatchValue:item.MatchValue,
		TitleTemplate:item.TitleTemplate,MessageTemplate:item.MessageTemplate,
		DefaultTitle:item.DefaultTitle,DefaultPriority:item.DefaultPriority,
		CreatedAt:item.CreatedAt,UpdatedAt:item.UpdatedAt,
	}
}

func mqttView(item *model.MQTTIntegration) model.MQTTIntegrationView {
	return model.MQTTIntegrationView{
		ID:item.ID,Name:item.Name,ApplicationID:item.ApplicationID,BrokerURL:item.BrokerURL,ClientID:item.ClientID,
		Username:item.Username,PasswordConfigured:item.Password!="",
		ProtocolVersion:item.ProtocolVersion,QoS:item.QoS,
		CACertificate:item.CACertificate,ClientCertificate:item.ClientCertificate,ClientKeyConfigured:item.ClientKey!="",
		Topic:item.Topic,Enabled:item.Enabled,
		Status:item.Status,LastConnectedAt:item.LastConnectedAt,LastMessageAt:item.LastMessageAt,
		LastError:item.LastError,LastErrorAt:item.LastErrorAt,ReconnectCount:item.ReconnectCount,
		CreatedAt:item.CreatedAt,UpdatedAt:item.UpdatedAt,
	}
}

func homeAssistantView(item *model.HomeAssistantIntegration) model.HomeAssistantIntegrationView {
	return model.HomeAssistantIntegrationView{
		ID:item.ID,Name:item.Name,ApplicationID:item.ApplicationID,BaseURL:item.BaseURL,
		TokenConfigured:item.Token!="",EventType:item.EventType,EntityIDs:item.EntityIDs,DataField:item.DataField,DataValue:item.DataValue,Enabled:item.Enabled,
		Status:item.Status,LastConnectedAt:item.LastConnectedAt,LastEventAt:item.LastEventAt,
		LastError:item.LastError,LastErrorAt:item.LastErrorAt,ReconnectCount:item.ReconnectCount,
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
		switch typed := current.(type) {
		case map[string]any:
			next, ok := typed[part]
			if !ok { return nil, false }
			current = next
		case []any:
			index, err := strconv.Atoi(part)
			if err != nil || index < 0 || index >= len(typed) { return nil, false }
			current = typed[index]
		default:
			return nil, false
		}
	}
	return current, true
}

func renderPayloadTemplate(template string, payload any, raw string) string {
	var result strings.Builder
	for {
		start := strings.Index(template, "{{")
		if start < 0 {
			result.WriteString(template)
			break
		}
		result.WriteString(template[:start])
		template = template[start+2:]
		end := strings.Index(template, "}}")
		if end < 0 {
			result.WriteString("{{")
			result.WriteString(template)
			break
		}
		key := strings.TrimSpace(template[:end])
		template = template[end+2:]
		switch key {
		case "raw":
			result.WriteString(raw)
		default:
			if value, ok := lookupPayload(payload, key); ok {
				result.WriteString(payloadString(value))
			}
		}
	}
	return result.String()
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

func normalizedWebhookRateLimit(value int) int {
	if value <= 0 { return 120 }
	if value > 10000 { return 10000 }
	return value
}

func webhookIPAllowed(clientIP, allowed string) bool {
	allowed = strings.TrimSpace(allowed)
	if allowed == "" { return true }
	ip := net.ParseIP(strings.TrimSpace(clientIP))
	if ip == nil { return false }
	for _, raw := range strings.FieldsFunc(allowed, func(r rune) bool { return r == ',' || r == ';' || r == '\n' || r == ' ' || r == '\t' }) {
		raw = strings.TrimSpace(raw)
		if raw == "" { continue }
		if candidate := net.ParseIP(raw); candidate != nil && candidate.Equal(ip) { return true }
		if _, network, err := net.ParseCIDR(raw); err == nil && network.Contains(ip) { return true }
	}
	return false
}

func validWebhookSignature(secret, timestamp, signature string, body []byte, now time.Time) bool {
	if secret == "" || timestamp == "" || signature == "" { return false }
	seconds, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil { return false }
	signedAt := time.Unix(seconds, 0)
	delta := now.Sub(signedAt)
	if delta < 0 { delta = -delta }
	if delta > 5*time.Minute { return false }
	signature = strings.TrimPrefix(strings.ToLower(signature), "sha256=")
	provided, err := hex.DecodeString(signature)
	if err != nil { return false }
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(timestamp))
	_, _ = mac.Write([]byte("."))
	_, _ = mac.Write(body)
	return hmac.Equal(provided, mac.Sum(nil))
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
