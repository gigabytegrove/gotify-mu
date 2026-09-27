package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateAPIStatusWhenDockerSocketMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	api := UpdateAPI{
		DockerSocket:    filepath.Join(t.TempDir(), "missing.sock"),
		TargetContainer: "monita",
		StatusFile:      filepath.Join(t.TempDir(), "status.json"),
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/update/status", nil)

	api.Status(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	assert.Equal(t, false, payload["ready"])
	assert.Equal(t, "unavailable", payload["state"])
}

func TestUpdateAPIStatusDefaultsToIdle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	socket := filepath.Join(dir, "docker.sock")
	require.NoError(t, os.WriteFile(socket, nil, 0o600))

	api := UpdateAPI{
		DockerSocket:    socket,
		TargetContainer: "monita",
		StatusFile:      filepath.Join(dir, "status.json"),
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/update/status", nil)

	api.Status(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `{"ready":true,"state":"idle","progress":0}`, recorder.Body.String())
}

func TestUpdateAPIStartsTransientWorker(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	socket := filepath.Join(dir, "docker.sock")
	require.NoError(t, os.WriteFile(socket, nil, 0o600))

	var calls [][]string
	runner := func(args ...string) (string, error) {
		copied := append([]string(nil), args...)
		calls = append(calls, copied)
		if len(args) >= 3 && args[0] == "inspect" && args[1] == "--format" && strings.Contains(args[2], ".State.Running") {
			return "false\n", nil
		}
		if len(args) >= 3 && args[0] == "inspect" && args[1] == "--format" && strings.Contains(args[2], ".Config.Image") {
			return "monita:1.1.0\n", nil
		}
		if len(args) > 0 && args[0] == "run" {
			return "worker-id\n", nil
		}
		return "", nil
	}

	api := UpdateAPI{
		DockerSocket:    socket,
		TargetContainer: "monita",
		WorkerName:      "monita-update-worker",
		StatusFile:      filepath.Join(dir, "status.json"),
		Repository:      "gigabytegrove/gotify-mu",
		RunDocker:       runner,
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(
		http.MethodPost,
		"/update/install",
		strings.NewReader(`{"version":"1.1.0"}`),
	)
	ctx.Request.Header.Set("Content-Type", "application/json")

	api.Install(ctx)

	assert.Equal(t, http.StatusAccepted, recorder.Code)
	require.NotEmpty(t, calls)

	var runArgs []string
	for _, call := range calls {
		if len(call) > 0 && call[0] == "run" {
			runArgs = call
			break
		}
	}
	require.NotEmpty(t, runArgs)
	assert.Contains(t, runArgs, "--rm")
	assert.Contains(t, runArgs, "--volumes-from")
	assert.Contains(t, runArgs, "monita")
	assert.Contains(t, runArgs, "/usr/local/bin/monita-updater")
	assert.Equal(t, []string{"install", "1.1.0"}, runArgs[len(runArgs)-2:])

	content, err := os.ReadFile(api.StatusFile)
	require.NoError(t, err)
	var status managedUpdateStatus
	require.NoError(t, json.Unmarshal(content, &status))
	assert.Equal(t, "preparing", status.State)
	assert.Equal(t, "1.1.0", status.Version)
	assert.True(t, status.Ready)
}

func TestUpdateStateActive(t *testing.T) {
	for _, state := range []string{"preparing", "downloading", "building", "replacing", "verifying"} {
		assert.True(t, updateStateActive(state), state)
	}
	for _, state := range []string{"", "idle", "completed", "failed", "rolled_back"} {
		assert.False(t, updateStateActive(state), state)
	}
}
