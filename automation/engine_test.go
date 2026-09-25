package automation

import (
	"bufio"
	"bytes"
	"testing"
	"time"

	"github.com/gotify/server/v3/model"
)

func TestQuietHoursCrossMidnight(t *testing.T) {
	policy := &model.QuietHoursPolicy{
		Enabled: true,
		StartMinute: 22 * 60,
		EndMinute: 7 * 60,
		Timezone: "UTC",
	}

	if !quietNow(policy, time.Date(2026, 9, 25, 23, 0, 0, 0, time.UTC)) {
		t.Fatal("23:00 should be inside overnight quiet hours")
	}
	if !quietNow(policy, time.Date(2026, 9, 26, 6, 59, 0, 0, time.UTC)) {
		t.Fatal("06:59 should be inside overnight quiet hours")
	}
	if quietNow(policy, time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)) {
		t.Fatal("12:00 should be outside overnight quiet hours")
	}
}

func TestNextScheduleRunDailyTimezone(t *testing.T) {
	item := &model.ScheduledNotification{
		ScheduleType: "daily",
		Hour: 9,
		Minute: 30,
		Timezone: "America/New_York",
	}
	now := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	next := NextScheduleRun(item, now)
	if next == nil {
		t.Fatal("expected next run")
	}
	expected := time.Date(2026, 9, 25, 13, 30, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Fatalf("expected %s, got %s", expected, next)
	}
}

func TestNextScheduleRunWeeklyRollsForward(t *testing.T) {
	item := &model.ScheduledNotification{
		ScheduleType: "weekly",
		Weekday: int(time.Friday),
		Hour: 8,
		Minute: 0,
		Timezone: "UTC",
	}
	now := time.Date(2026, 9, 25, 9, 0, 0, 0, time.UTC)
	next := NextScheduleRun(item, now)
	expected := time.Date(2026, 10, 2, 8, 0, 0, 0, time.UTC)
	if next == nil || !next.Equal(expected) {
		t.Fatalf("expected %s, got %v", expected, next)
	}
}

func TestDecodeMQTTPublishQoS0(t *testing.T) {
	topic := "home/alerts"
	payload := []byte("door open")
	body := appendMQTTString(nil, topic)
	body = append(body, payload...)

	packet := []byte{0x30}
	packet = append(packet, encodeRemainingLength(len(body))...)
	packet = append(packet, body...)

	reader := bufio.NewReader(bytes.NewReader(packet))
	header, encodedBody, err := readMQTTPacket(reader)
	if err != nil {
		t.Fatal(err)
	}
	gotTopic, gotPayload, packetID, qos, err := decodePublish(header, encodedBody, 4)
	if err != nil {
		t.Fatal(err)
	}
	if gotTopic != topic || string(gotPayload) != string(payload) || packetID != 0 || qos != 0 {
		t.Fatalf("unexpected publish decode: topic=%q payload=%q id=%d qos=%d", gotTopic, gotPayload, packetID, qos)
	}
}

func TestEncodeRemainingLength(t *testing.T) {
	got := encodeRemainingLength(321)
	expected := []byte{0xC1, 0x02}
	if !bytes.Equal(got, expected) {
		t.Fatalf("expected %v, got %v", expected, got)
	}
}

func TestAppendMQTTString(t *testing.T) {
	got := appendMQTTString(nil, "MQTT")
	expected := []byte{0x00, 0x04, 'M', 'Q', 'T', 'T'}
	if !bytes.Equal(got, expected) {
		t.Fatalf("expected %v, got %v", expected, got)
	}
}


func TestDecodeMQTTPublishV5QoS1(t *testing.T) {
	topic := "alerts/critical"
	payload := []byte("server down")
	body := appendMQTTString(nil, topic)
	body = append(body, 0x12, 0x34) // packet id
	body = append(body, 0x00)       // MQTT 5 properties length
	body = append(body, payload...)

	header := byte(0x32) // PUBLISH QoS 1
	gotTopic, gotPayload, packetID, qos, err := decodePublish(header, body, 5)
	if err != nil {
		t.Fatal(err)
	}
	if gotTopic != topic || string(gotPayload) != string(payload) || packetID != 0x1234 || qos != 1 {
		t.Fatalf("unexpected v5 publish decode: topic=%q payload=%q id=%d qos=%d", gotTopic, gotPayload, packetID, qos)
	}
}

func TestDecodeMQTTVarInt(t *testing.T) {
	value, consumed, err := decodeMQTTVarInt([]byte{0xC1, 0x02})
	if err != nil {
		t.Fatal(err)
	}
	if value != 321 || consumed != 2 {
		t.Fatalf("expected value=321 consumed=2, got value=%d consumed=%d", value, consumed)
	}
}

func TestReadMQTTPacketRejectsOversize(t *testing.T) {
	encoded := append([]byte{0x30}, encodeRemainingLength(maxMQTTPacketBytes+1)...)
	reader := bufio.NewReader(bytes.NewReader(encoded))
	if _, _, err := readMQTTPacket(reader); err == nil {
		t.Fatal("expected oversized MQTT packet to be rejected")
	}
}
