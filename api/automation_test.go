package api

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gotify/server/v3/model"
	"github.com/gotify/server/v3/security"
)

func TestLookupPayloadNestedField(t *testing.T) {
	payload := map[string]any{
		"alert": map[string]any{
			"title": "Disk warning",
			"priority": float64(8),
		},
	}
	value, ok := lookupPayload(payload, "alert.title")
	if !ok || value != "Disk warning" {
		t.Fatalf("unexpected nested value: %#v %v", value, ok)
	}
	priority, ok := lookupPayload(payload, "alert.priority")
	if !ok {
		t.Fatal("priority not found")
	}
	number, ok := payloadInt(priority)
	if !ok || number != 8 {
		t.Fatalf("unexpected priority: %v %v", number, ok)
	}
}

func TestValidateScheduleRejectsInvalidValues(t *testing.T) {
	cases := []*model.ScheduledNotification{
		{ScheduleType: "once", Timezone: "UTC"},
		{ScheduleType: "hourly", Minute: 60, Timezone: "UTC"},
		{ScheduleType: "daily", Hour: 24, Minute: 0, Timezone: "UTC"},
		{ScheduleType: "weekly", Weekday: 7, Hour: 8, Timezone: "UTC"},
		{ScheduleType: "daily", Hour: 8, Timezone: "Not/A_Timezone"},
	}
	for _, item := range cases {
		if err := validateSchedule(item); err == nil {
			t.Fatalf("expected validation failure for %#v", item)
		}
	}
}

func TestValidateOneTimeSchedule(t *testing.T) {
	runAt := time.Now().Add(time.Hour)
	item := &model.ScheduledNotification{
		ScheduleType: "once",
		RunAt: &runAt,
		Timezone: "UTC",
	}
	if err := validateSchedule(item); err != nil {
		t.Fatal(err)
	}
}

func TestURLValidation(t *testing.T) {
	if !validMQTTURL("mqtts://broker.example:8883") {
		t.Fatal("mqtts URL should be valid")
	}
	if validMQTTURL("https://broker.example") {
		t.Fatal("https URL should not be accepted for MQTT")
	}
	if !validHTTPURL("https://home.example") {
		t.Fatal("https URL should be valid for Home Assistant")
	}
}


func TestLookupPayloadSupportsArrays(t *testing.T) {
	payload := map[string]any{
		"items": []any{
			map[string]any{"name":"first"},
			map[string]any{"name":"second"},
		},
	}
	value, ok := lookupPayload(payload, "items.1.name")
	if !ok || value != "second" {
		t.Fatalf("unexpected array lookup: %#v ok=%v", value, ok)
	}
}

func TestRenderPayloadTemplate(t *testing.T) {
	payload := map[string]any{
		"alert": map[string]any{"title":"Disk full"},
		"items": []any{map[string]any{"name":"server-1"}},
	}
	got := renderPayloadTemplate("{{alert.title}} on {{items.0.name}}", payload, "raw")
	if got != "Disk full on server-1" {
		t.Fatalf("unexpected template output %q", got)
	}
	if raw := renderPayloadTemplate("body={{raw}}", payload, "original"); raw != "body=original" {
		t.Fatalf("unexpected raw template output %q", raw)
	}
}


func TestNormalizeHomeAssistantMode(t *testing.T) {
	cases := map[string]string{
		"":            "token",
		"token":       "token",
		"LLT":         "token",
		"integration": "integration",
		"Native":      "integration",
		"invalid":     "",
	}
	for input, expected := range cases {
		if got := normalizeHomeAssistantMode(input); got != expected {
			t.Fatalf("normalizeHomeAssistantMode(%q) = %q, want %q", input, got, expected)
		}
	}
}

func TestPrepareHomeAssistantPairingCreatesOneTimeState(t *testing.T) {
	item := &model.HomeAssistantIntegration{Status: "not_paired"}
	before := time.Now()

	code, err := prepareHomeAssistantPairing(item)
	if err != nil {
		t.Fatal(err)
	}
	if code == "" {
		t.Fatal("pairing code is empty")
	}
	if item.PairingCodeHash != security.HashSecret(code) {
		t.Fatal("pairing code hash does not match generated code")
	}
	if item.PairingExpiresAt == nil {
		t.Fatal("pairing expiry was not set")
	}
	if item.PairingExpiresAt.Before(before.Add(14*time.Minute)) ||
		item.PairingExpiresAt.After(before.Add(16*time.Minute)) {
		t.Fatalf("unexpected pairing expiry: %s", item.PairingExpiresAt)
	}
	if item.NativeWebhookURL != "" || item.NativeSecret != "" {
		t.Fatal("fresh pairing unexpectedly created bridge credentials")
	}
	if item.Status != "pairing" {
		t.Fatalf("unexpected pairing status %q", item.Status)
	}
}


func TestPrepareHomeAssistantPairingPreservesActiveNativeBridge(t *testing.T) {
	item := &model.HomeAssistantIntegration{
		NativeWebhookURL: "https://ha.example/api/webhook/existing",
		NativeSecret:     "existing-secret",
		Status:           "connected",
	}
	if _, err := prepareHomeAssistantPairing(item); err != nil {
		t.Fatal(err)
	}
	if item.NativeWebhookURL != "https://ha.example/api/webhook/existing" {
		t.Fatal("repair pairing must preserve the active webhook until replacement succeeds")
	}
	if item.NativeSecret != "existing-secret" {
		t.Fatal("repair pairing must preserve the active secret until replacement succeeds")
	}
	if item.Status != "connected" {
		t.Fatalf("expected connected status to be preserved, got %q", item.Status)
	}
	if item.PairingCodeHash == "" || item.PairingExpiresAt == nil {
		t.Fatal("expected a fresh pairing code and expiry")
	}
}

func TestHomeAssistantNativeAuthorization(t *testing.T) {
	if !validHomeAssistantNativeAuthorization("Bearer shared-secret", "shared-secret") {
		t.Fatal("valid Bearer secret should be accepted")
	}
	for _, header := range []string{
		"",
		"shared-secret",
		"Basic shared-secret",
		"Bearer wrong-secret",
		"Bearer ",
	} {
		if validHomeAssistantNativeAuthorization(header, "shared-secret") {
			t.Fatalf("unexpected authorization success for %q", header)
		}
	}
}

func TestClearHomeAssistantNativePairing(t *testing.T) {
	expires := time.Now().Add(time.Minute)
	item := &model.HomeAssistantIntegration{
		NativeWebhookURL:  "https://ha.example/api/webhook/existing",
		NativeSecret:      "shared-secret",
		PairingCodeHash:   "hash",
		PairingExpiresAt:  &expires,
		Status:            "connected",
		LastError:         "old error",
		LastErrorAt:       &expires,
	}
	clearHomeAssistantNativePairing(item)
	if item.NativeWebhookURL != "" || item.NativeSecret != "" || item.PairingCodeHash != "" {
		t.Fatal("native bridge credentials were not cleared")
	}
	if item.PairingExpiresAt != nil {
		t.Fatal("pairing expiry should be cleared")
	}
	if item.Status != "not_paired" {
		t.Fatalf("expected not_paired status, got %q", item.Status)
	}
	if item.LastError != "" || item.LastErrorAt != nil {
		t.Fatal("stale native bridge errors should be cleared")
	}
}


type homeAssistantEndpointTestDB struct {
	AutomationDatabase
	item *model.HomeAssistantIntegration
}

func (d *homeAssistantEndpointTestDB) GetHomeAssistantIntegrationByID(id uint) (*model.HomeAssistantIntegration, error) {
	if d.item == nil || d.item.ID != id {
		return nil, nil
	}
	return d.item, nil
}

func (d *homeAssistantEndpointTestDB) SaveHomeAssistantIntegration(item *model.HomeAssistantIntegration) error {
	d.item = item
	return nil
}

type homeAssistantEndpointTestEngine struct {
	AutomationEngine
	received bool
	reloaded int
	routed   bool
}

func (e *homeAssistantEndpointTestEngine) ReceiveHomeAssistantEvent(id uint, eventType string, data map[string]any) (bool, error) {
	e.received = true
	return e.routed, nil
}

func (e *homeAssistantEndpointTestEngine) ReloadIntegrations() {
	e.reloaded++
}

func nativePairingContext(t *testing.T, body any) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("POST", "/", bytes.NewReader(raw))
	ctx.Request.Header.Set("Content-Type", "application/json")
	return ctx, recorder
}

func TestPairNativeHomeAssistantEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	expires := time.Now().Add(10 * time.Minute)
	db := &homeAssistantEndpointTestDB{
		item: &model.HomeAssistantIntegration{
			ID:               7,
			ConnectionMode:   "integration",
			PairingCodeHash:  security.HashSecret("pair-secret"),
			PairingExpiresAt: &expires,
			Enabled:          true,
		},
	}
	api := &AutomationAPI{DB: db, Engine: &homeAssistantEndpointTestEngine{}}
	ctx, recorder := nativePairingContext(t, map[string]any{
		"pairingCode": "7.pair-secret",
		"webhookUrl":  "https://ha.example/api/webhook/bridge",
	})

	api.PairNativeHomeAssistant(ctx)

	if recorder.Code != 200 {
		t.Fatalf("unexpected status %d: %s", recorder.Code, recorder.Body.String())
	}
	var response homeAssistantNativePairResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.IntegrationID != 7 || response.Secret == "" || response.EventPath != "/integrations/home-assistant/native/7/event" {
		t.Fatalf("unexpected pairing response: %#v", response)
	}
	if db.item.NativeWebhookURL != "https://ha.example/api/webhook/bridge" || db.item.NativeSecret == "" {
		t.Fatal("native bridge credentials were not stored")
	}
	if db.item.PairingCodeHash != "" || db.item.PairingExpiresAt != nil {
		t.Fatal("pairing code was not invalidated after successful pairing")
	}
}

func TestPairNativeHomeAssistantRejectsBadAndExpiredCodes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	future := time.Now().Add(10 * time.Minute)
	past := time.Now().Add(-time.Minute)

	for _, tc := range []struct {
		name       string
		secret     string
		expires    *time.Time
		wantStatus int
	}{
		{name: "bad secret", secret: "wrong-secret", expires: &future, wantStatus: 401},
		{name: "expired", secret: "pair-secret", expires: &past, wantStatus: 410},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := &homeAssistantEndpointTestDB{
				item: &model.HomeAssistantIntegration{
					ID:               7,
					ConnectionMode:   "integration",
					PairingCodeHash:  security.HashSecret("pair-secret"),
					PairingExpiresAt: tc.expires,
				},
			}
			api := &AutomationAPI{DB: db, Engine: &homeAssistantEndpointTestEngine{}}
			ctx, recorder := nativePairingContext(t, map[string]any{
				"pairingCode": "7." + tc.secret,
				"webhookUrl":  "https://ha.example/api/webhook/bridge",
			})

			api.PairNativeHomeAssistant(ctx)

			if recorder.Code != tc.wantStatus {
				t.Fatalf("unexpected status %d, want %d", recorder.Code, tc.wantStatus)
			}
		})
	}
}

func TestReceiveNativeHomeAssistantEventAuthorizationAndRouting(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := &homeAssistantEndpointTestDB{
		item: &model.HomeAssistantIntegration{
			ID:             7,
			ConnectionMode: "integration",
			NativeSecret:   "shared-secret",
			Enabled:        true,
		},
	}
	engine := &homeAssistantEndpointTestEngine{routed: true}
	api := &AutomationAPI{DB: db, Engine: engine}

	for _, tc := range []struct {
		name         string
		authHeader   string
		wantStatus   int
		wantReceived bool
	}{
		{name: "bad bearer", authHeader: "Bearer wrong-secret", wantStatus: 401, wantReceived: false},
		{name: "valid bearer", authHeader: "Bearer shared-secret", wantStatus: 202, wantReceived: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			engine.received = false
			ctx, recorder := nativePairingContext(t, map[string]any{
				"eventType": "state_changed",
				"data":      map[string]any{"entity_id": "light.kitchen"},
			})
			ctx.Params = gin.Params{{Key: "id", Value: "7"}}
			ctx.Request.Header.Set("Authorization", tc.authHeader)

			api.ReceiveNativeHomeAssistantEvent(ctx)

			if recorder.Code != tc.wantStatus {
				t.Fatalf("unexpected status %d, want %d: %s", recorder.Code, tc.wantStatus, recorder.Body.String())
			}
			if engine.received != tc.wantReceived {
				t.Fatalf("received=%v, want %v", engine.received, tc.wantReceived)
			}
		})
	}
}

func TestRevokeNativeHomeAssistantEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := &homeAssistantEndpointTestDB{
		item: &model.HomeAssistantIntegration{
			ID:               7,
			ConnectionMode:   "integration",
			NativeWebhookURL: "https://ha.example/api/webhook/bridge",
			NativeSecret:     "shared-secret",
			Status:           "connected",
		},
	}
	engine := &homeAssistantEndpointTestEngine{}
	api := &AutomationAPI{DB: db, Engine: engine}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("DELETE", "/", nil)
	ctx.Request.Header.Set("Authorization", "Bearer shared-secret")
	ctx.Params = gin.Params{{Key: "id", Value: "7"}}

	api.RevokeNativeHomeAssistant(ctx)

	if ctx.Writer.Status() != 204 {
		t.Fatalf("unexpected status %d: %s", ctx.Writer.Status(), recorder.Body.String())
	}
	if db.item.NativeWebhookURL != "" || db.item.NativeSecret != "" || db.item.Status != "not_paired" {
		t.Fatal("native bridge was not revoked")
	}
	if engine.reloaded != 1 {
		t.Fatalf("expected integration reload, got %d", engine.reloaded)
	}
}


func TestWebhookSummaryViewHidesCredentialPath(t *testing.T) {
	item := &model.WebhookRoute{
		ID: 7,
		Name: "Doorbell",
		Secret: "super-secret-route-token",
	}
	full := webhookView(item)
	summary := webhookSummaryView(item)
	if full.Path == "" || !strings.Contains(full.Path, item.Secret) {
		t.Fatalf("expected elevated webhook view to contain generated path, got %q", full.Path)
	}
	if summary.Path != "" {
		t.Fatalf("read-only webhook summary exposed credential path: %q", summary.Path)
	}
}
