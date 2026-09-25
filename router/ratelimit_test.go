package router

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRequestLimiterBlocksAfterLimitAndResets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 9, 25, 20, 0, 0, 0, time.UTC)
	limiter := newRequestLimiter(2, time.Minute)
	limiter.now = func() time.Time { return now }

	engine := gin.New()
	engine.GET("/test", limiter.Middleware(), func(ctx *gin.Context) {
		ctx.Status(http.StatusNoContent)
	})

	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.0.2.20:1234"
		engine.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("request %d expected 204, got %d", i+1, rec.Code)
		}
	}

	blocked := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.0.2.20:1234"
	engine.ServeHTTP(blocked, req)
	if blocked.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", blocked.Code)
	}
	if blocked.Header().Get("Retry-After") == "" {
		t.Fatal("expected Retry-After header")
	}

	now = now.Add(61 * time.Second)
	reset := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.0.2.20:1234"
	engine.ServeHTTP(reset, req)
	if reset.Code != http.StatusNoContent {
		t.Fatalf("expected reset limiter to allow request, got %d", reset.Code)
	}
}
