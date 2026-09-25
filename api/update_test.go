package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestUpdateAPIStatusWhenUnconfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)
	api := UpdateAPI{}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/update/status", nil)

	api.Status(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)
	var payload map[string]any
	assert.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	assert.Equal(t, false, payload["ready"])
	assert.Equal(t, "unavailable", payload["state"])
}

func TestUpdateAPIProxiesStatus(t *testing.T) {
	helper := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ready":true,"state":"idle"}`))
	}))
	defer helper.Close()

	gin.SetMode(gin.TestMode)
	api := UpdateAPI{
		BaseURL: helper.URL,
		Token:   "test-token",
		Client:  helper.Client(),
	}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/update/status", nil)

	api.Status(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `{"ready":true,"state":"idle"}`, recorder.Body.String())
}
