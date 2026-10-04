package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompileAllowedOrigins(t *testing.T) {
	assert.Equal(t, 0, len(CompileAllowedOrigins([]string{})))
	assert.Equal(t, 3, len(CompileAllowedOrigins([]string{"^.*$", "", "abc"})))
}

func TestMatchesFully(t *testing.T) {
	compiledOrigins := CompileAllowedOrigins([]string{"monita\\.net|push\\.monita\\.net", "other\\.monita\\.net"})

	assert.True(t, MatchesFully(compiledOrigins, "monita.net"))
	assert.True(t, MatchesFully(compiledOrigins, "push.monita.net"))
	assert.True(t, MatchesFully(compiledOrigins, "other.monita.net"))
	assert.False(t, MatchesFully(compiledOrigins, "monita.net.evil.net"))
	assert.False(t, MatchesFully(compiledOrigins, "evil-monita.net"))
}
