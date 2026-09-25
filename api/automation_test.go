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
