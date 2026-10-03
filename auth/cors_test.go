package auth

import (
	"testing"
	"time"

	"github.com/gigabytegrove/monita/config"
	"github.com/gigabytegrove/monita/mode"
	"github.com/gin-contrib/cors"
	"github.com/stretchr/testify/assert"
)

func TestCorsConfig(t *testing.T) {
	mode.Set(mode.Prod)
	serverConf := config.Configuration{}
	serverConf.Server.Cors.AllowOrigins = []string{"http://monita\\.test|http://push\\.monita\\.test", "http://other\\.monita\\.test"}
	serverConf.Server.Cors.AllowHeaders = []string{"content-type"}
	serverConf.Server.Cors.AllowMethods = []string{"GET"}

	actual := CorsConfig(&serverConf)
	allowF := actual.AllowOriginFunc
	actual.AllowOriginFunc = nil // func cannot be checked with equal

	assert.Equal(t, cors.Config{
		AllowAllOrigins:        false,
		AllowHeaders:           []string{"content-type"},
		AllowMethods:           []string{"GET"},
		MaxAge:                 12 * time.Hour,
		AllowBrowserExtensions: true,
	}, actual)
	assert.NotNil(t, allowF)
	assert.True(t, allowF("http://monita.test"))
	assert.True(t, allowF("http://push.monita.test"))
	assert.True(t, allowF("http://other.monita.test"))
	assert.False(t, allowF("http://monita.test.evil.net"))
	assert.False(t, allowF("http://evil-monita.test"))
}

func TestEmptyCorsConfigWithResponseHeaders(t *testing.T) {
	mode.Set(mode.Prod)
	serverConf := config.Configuration{}
	serverConf.Server.ResponseHeaders = map[string]string{"Access-control-allow-origin": "https://example.com"}

	actual := CorsConfig(&serverConf)
	assert.NotNil(t, actual.AllowOriginFunc)
	actual.AllowOriginFunc = nil // func cannot be checked with equal

	assert.Equal(t, cors.Config{
		AllowAllOrigins:        false,
		AllowOrigins:           []string{"https://example.com"},
		MaxAge:                 12 * time.Hour,
		AllowBrowserExtensions: true,
	}, actual)
}
