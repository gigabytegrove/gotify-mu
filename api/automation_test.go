package api

import (
	"testing"
	"time"

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
	item := &model.HomeAssistantIntegration{
		NativeWebhookURL: "https://old.example/api/webhook/old",
		NativeSecret:     "old-secret",
		Status:           "connected",
	}
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
		t.Fatal("old native bridge credentials were not cleared")
	}
	if item.Status != "pairing" {
		t.Fatalf("unexpected pairing status %q", item.Status)
	}
}
\n\nfunc TestPrepareHomeAssistantPairingPreservesActiveNativeBridge(t *testing.T) {
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
