package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateAPIStatusUnavailableWithoutDockerAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	updateAPI := UpdateAPI{
		ReadyCheck: func() error {
			return errors.New("docker unavailable")
		},
	}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/update/status", nil)

	updateAPI.Status(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)
	var payload UpdateStatus
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	assert.False(t, payload.Ready)
	assert.Equal(t, "unavailable", payload.State)
	assert.Equal(t, "docker unavailable", payload.Message)
}

func TestUpdateAPIStatusIdleWithSingleContainerUpdaterReady(t *testing.T) {
	gin.SetMode(gin.TestMode)
	updateAPI := UpdateAPI{
		StatusFile: filepath.Join(t.TempDir(), "status.json"),
		ReadyCheck: func() error { return nil },
	}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/update/status", nil)

	updateAPI.Status(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)
	var payload UpdateStatus
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	assert.True(t, payload.Ready)
	assert.Equal(t, "idle", payload.State)
	assert.Equal(t, 0, payload.Progress)
}

func TestUpdateAPIInstallLaunchesShortLivedWorkerFromCurrentImage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	statusFile := filepath.Join(t.TempDir(), "status.json")
	fixedNow := time.Date(2026, 9, 27, 15, 30, 0, 0, time.UTC)
	var commands [][]string

	updateAPI := UpdateAPI{
		TargetContainer: "gotify-mu",
		Repository:      "gigabytegrove/gotify-mu",
		WorkerContainer: "gotify-mu-update-worker",
		StatusFile:      statusFile,
		ReadyCheck:      func() error { return nil },
		Now:             func() time.Time { return fixedNow },
		RunDocker: func(args ...string) (string, error) {
			copied := append([]string(nil), args...)
			commands = append(commands, copied)

			if slices.Equal(args, []string{"inspect", "--format", "{{.State.Running}}", "gotify-mu-update-worker"}) {
				return "Error: No such object: gotify-mu-update-worker", errors.New("not found")
			}
			if slices.Equal(args, []string{"rm", "-f", "gotify-mu-update-worker"}) {
				return "", errors.New("not found")
			}
			if slices.Equal(args, []string{"inspect", "--format", "{{.Config.Image}}", "gotify-mu"}) {
				return "gotify-mu:master\n", nil
			}
			if len(args) > 0 && args[0] == "run" {
				return "worker-id\n", nil
			}
			return "", errors.New("unexpected docker command")
		},
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(
		http.MethodPost,
		"/update/install",
		strings.NewReader(`{"version":"1.0.2"}`),
	)
	ctx.Request.Header.Set("Content-Type", "application/json")

	updateAPI.Install(ctx)

	assert.Equal(t, http.StatusAccepted, recorder.Code, recorder.Body.String())

	var payload UpdateStatus
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	assert.True(t, payload.Ready)
	assert.Equal(t, "preparing", payload.State)
	assert.Equal(t, "1.0.2", payload.Version)

	var workerCommand []string
	for _, command := range commands {
		if len(command) > 0 && command[0] == "run" {
			workerCommand = command
			break
		}
	}
	require.NotEmpty(t, workerCommand)
	assert.True(t, containsUpdateArgs(workerCommand, "--rm"))
	assert.True(t, containsUpdateArgs(workerCommand, "--name", "gotify-mu-update-worker"))
	assert.True(t, containsUpdateArgs(workerCommand, "--volumes-from", "gotify-mu:rw"))
	assert.True(t, containsUpdateArgs(workerCommand, "--entrypoint", "/usr/local/bin/gotify-mu-updater"))
	assert.True(t, containsUpdateArgs(workerCommand, "gotify-mu:master", "--oneshot", "1.0.2"))

	for _, argument := range workerCommand {
		assert.NotContains(t, argument, "GOTIFY_MU_UPDATER_TOKEN")
		assert.NotContains(t, argument, "GOTIFY_MU_UPDATER_URL")
	}

	data, err := os.ReadFile(statusFile)
	require.NoError(t, err)
	var stored UpdateStatus
	require.NoError(t, json.Unmarshal(data, &stored))
	assert.Equal(t, "preparing", stored.State)
	assert.Equal(t, "1.0.2", stored.Version)
}

func TestUpdateAPIRejectsNonSemanticVersion(t *testing.T) {
	gin.SetMode(gin.TestMode)
	updateAPI := UpdateAPI{
		StatusFile: filepath.Join(t.TempDir(), "status.json"),
		ReadyCheck: func() error { return nil },
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(
		http.MethodPost,
		"/update/install",
		strings.NewReader(`{"version":"master"}`),
	)
	ctx.Request.Header.Set("Content-Type", "application/json")

	updateAPI.Install(ctx)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

func containsUpdateArgs(values []string, expected ...string) bool {
	for index := 0; index+len(expected) <= len(values); index++ {
		if slices.Equal(values[index:index+len(expected)], expected) {
			return true
		}
	}
	return false
}
