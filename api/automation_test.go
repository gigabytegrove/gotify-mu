package api

import (
	"testing"
	"time"

	"github.com/gotify/server/v3/model"
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


func TestWebhookSourceAllowed(t *testing.T) {
	if !webhookSourceAllowed("", "203.0.113.20") {
		t.Fatal("blank source policy should allow requests")
	}
	if !webhookSourceAllowed("192.168.0.0/16,10.0.0.0/8", "192.168.10.20") {
		t.Fatal("address inside configured network should be allowed")
	}
	if webhookSourceAllowed("192.168.0.0/16", "203.0.113.20") {
		t.Fatal("address outside configured network should be rejected")
	}
	if webhookSourceAllowed("not-a-network", "192.168.1.1") {
		t.Fatal("invalid network must not match")
	}
}

func TestValidateWebhookSecurity(t *testing.T) {
	if err := validateWebhookSecurity(&model.WebhookRoute{
		RequireSignature: true,
	}); err == nil {
		t.Fatal("signed Webhook without signing secret should fail")
	}
	if err := validateWebhookSecurity(&model.WebhookRoute{
		RequireSignature: true,
		SigningSecret:    "secret",
		AllowedCIDRs:     "192.168.0.0/16,10.0.0.0/8",
	}); err != nil {
		t.Fatalf("valid Webhook security rejected: %v", err)
	}
	if err := validateWebhookSecurity(&model.WebhookRoute{
		AllowedCIDRs: "invalid",
	}); err == nil {
		t.Fatal("invalid source network should fail")
	}
}
