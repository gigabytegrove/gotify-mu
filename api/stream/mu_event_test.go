package stream

import (
	"testing"
	"time"
)

func TestNotifyMUEventUsesSeparateRealtimeQueue(t *testing.T) {
	api := New(time.Second, time.Second, nil)
	client := &muEventClient{
		write:  make(chan any, 1),
		closed: make(chan struct{}),
		userID: 7,
	}
	api.muClients[7] = []*muEventClient{client}

	event := map[string]any{"type": "typing", "applicationId": uint(12)}
	api.NotifyMUEvent(7, event)

	select {
	case received := <-client.write:
		payload, ok := received.(map[string]any)
		if !ok || payload["type"] != "typing" || payload["applicationId"] != uint(12) {
			t.Fatalf("unexpected MU realtime payload: %#v", received)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for MU realtime event")
	}

	if got := api.ConnectedClientCount(); got != 1 {
		t.Fatalf("expected one connected MU client, got %d", got)
	}
}
