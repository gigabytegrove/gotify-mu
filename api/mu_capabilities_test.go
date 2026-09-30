package api

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMUCapabilitiesContract(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	handler := MUCapabilitiesAPI{Version: "1.0.0"}
	handler.Get(ctx)

	if recorder.Code != 200 {
		t.Fatalf("expected HTTP 200, got %d", recorder.Code)
	}

	var payload MUCapabilities
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Product != "gotify-mu" || payload.Version != "1.0.0" || payload.APIVersion != 1 {
		t.Fatalf("unexpected capability identity: %#v", payload)
	}
	if !payload.Features.SharedChannels || !payload.Features.ChannelTypes ||
		!payload.Features.ChannelImages || !payload.Features.ChatImages ||
		!payload.Features.NotificationImages || !payload.Features.MessageControls ||
		!payload.Features.ChatChannels ||
		!payload.Features.TypingPresence ||
		!payload.Features.ChatNotifications || !payload.Features.Mentions {
		t.Fatalf("required MU feature flags are not advertised: %#v", payload.Features)
	}
}
