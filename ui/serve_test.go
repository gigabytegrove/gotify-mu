package ui

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestFreshUIHeadersPreventStaleFrontend(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

	setFreshUIHeaders(ctx)

	assert.Equal(t, "no-store, no-cache, must-revalidate, max-age=0", ctx.Writer.Header().Get("Cache-Control"))
	assert.Equal(t, "no-cache", ctx.Writer.Header().Get("Pragma"))
	assert.Equal(t, "0", ctx.Writer.Header().Get("Expires"))
}
