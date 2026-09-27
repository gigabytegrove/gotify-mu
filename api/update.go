package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	defaultUpdateDockerSocket = "/var/run/docker.sock"
	defaultUpdateTarget       = "gotify-mu"
	defaultUpdateWorker       = "gotify-mu-update-worker"
	defaultUpdateStatusFile   = "/app/data/.gotify-mu-update-status.json"
	defaultUpdateRepository   = "gigabytegrove/gotify-mu"
)

var updateVersionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)

type UpdateAPI struct {
	DockerSocket    string
	TargetContainer string
	WorkerName      string
	StatusFile      string
	Repository      string
	RunDocker       func(args ...string) (string, error)
}

type UpdateInstallRequest struct {
	Version string `json:"version" binding:"required"`
}

type updateActivity struct {
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
}

type managedUpdateStatus struct {
	Ready      bool             `json:"ready"`
	State      string           `json:"state"`
	Version    string           `json:"version,omitempty"`
	Message    string           `json:"message,omitempty"`
	Step       string           `json:"step,omitempty"`
	Progress   int              `json:"progress"`
	Activity   []updateActivity `json:"activity,omitempty"`
	StartedAt  *time.Time       `json:"startedAt,omitempty"`
	FinishedAt *time.Time       `json:"finishedAt,omitempty"`
}

func NewUpdateAPIFromEnv() UpdateAPI {
	socket := strings.TrimSpace(os.Getenv("GOTIFY_MU_UPDATE_DOCKER_SOCKET"))
	if socket == "" {
		socket = defaultUpdateDockerSocket
	}
	target := strings.TrimSpace(os.Getenv("GOTIFY_MU_UPDATE_TARGET_CONTAINER"))
	if target == "" {
		target = defaultUpdateTarget
	}
	worker := strings.TrimSpace(os.Getenv("GOTIFY_MU_UPDATE_WORKER_NAME"))
	if worker == "" {
		worker = defaultUpdateWorker
	}
	statusFile := strings.TrimSpace(os.Getenv("GOTIFY_MU_UPDATE_STATUS_FILE"))
	if statusFile == "" {
		statusFile = defaultUpdateStatusFile
	}
	repository := strings.TrimSpace(os.Getenv("GOTIFY_MU_REPOSITORY"))
	if repository == "" {
		repository = defaultUpdateRepository
	}

	return UpdateAPI{
		DockerSocket:    socket,
		TargetContainer: target,
		WorkerName:      worker,
		StatusFile:      statusFile,
		Repository:      repository,
		RunDocker:       runUpdateDocker,
	}
}

func (a *UpdateAPI) Status(ctx *gin.Context) {
	if !a.available() {
		ctx.JSON(http.StatusOK, managedUpdateStatus{
			Ready:   false,
			State:   "unavailable",
			Message: "Automatic updates are unavailable on this installation.",
		})
		return
	}

	status, err := a.readStatus()
	if err != nil {
		ctx.JSON(http.StatusOK, managedUpdateStatus{
			Ready:   false,
			State:   "unavailable",
			Message: "Automatic update status could not be read.",
		})
		return
	}

	status.Ready = true
	ctx.JSON(http.StatusOK, status)
}

func (a *UpdateAPI) Install(ctx *gin.Context) {
	if !a.available() {
		ctx.AbortWithError(http.StatusServiceUnavailable, errors.New("automatic updates are unavailable"))
		return
	}

	var request UpdateInstallRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}
	request.Version = strings.TrimSpace(strings.TrimPrefix(request.Version, "v"))
	if !updateVersionPattern.MatchString(request.Version) {
		ctx.AbortWithError(http.StatusBadRequest, errors.New("version must be semantic x.y.z"))
		return
	}

	current, err := a.readStatus()
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if updateStateActive(current.State) {
		ctx.JSON(http.StatusConflict, current)
		return
	}

	runner := a.RunDocker
	if runner == nil {
		runner = runUpdateDocker
	}

	if running, _ := runner("inspect", "--format", "{{.State.Running}}", a.WorkerName); strings.TrimSpace(running) == "true" {
		ctx.JSON(http.StatusConflict, managedUpdateStatus{
			Ready:    true,
			State:    "preparing",
			Version:  request.Version,
			Message:  "An update is already running.",
			Step:     "Preparing update",
			Progress: 2,
		})
		return
	}
	_, _ = runner("rm", "-f", a.WorkerName)

	image, err := runner("inspect", "--format", "{{.Config.Image}}", a.TargetContainer)
	if err != nil {
		ctx.AbortWithError(http.StatusBadGateway, fmt.Errorf("could not inspect current Gotify MU container: %w", err))
		return
	}
	image = strings.TrimSpace(image)
	if image == "" {
		ctx.AbortWithError(http.StatusBadGateway, errors.New("current Gotify MU image could not be determined"))
		return
	}

	started := time.Now().UTC()
	initial := managedUpdateStatus{
		Ready:    true,
		State:    "preparing",
		Version:  request.Version,
		Message:  "Preparing update",
		Step:     "Preparing update",
		Progress: 2,
		StartedAt: &started,
		Activity: []updateActivity{{
			Timestamp: started,
			Message:   "Update started",
		}},
	}
	if err := a.writeStatus(initial); err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	_, err = runner(
		"run", "-d", "--rm",
		"--name", a.WorkerName,
		"--volume", a.DockerSocket+":"+defaultUpdateDockerSocket,
		"--volumes-from", a.TargetContainer,
		"--env", "GOTIFY_MU_TARGET_CONTAINER="+a.TargetContainer,
		"--env", "GOTIFY_MU_UPDATE_STATUS_FILE="+a.StatusFile,
		"--env", "GOTIFY_MU_REPOSITORY="+a.Repository,
		"--entrypoint", "/usr/local/bin/gotify-mu-updater",
		image,
		"install", request.Version,
	)
	if err != nil {
		finished := time.Now().UTC()
		initial.State = "failed"
		initial.Step = "Update stopped"
		initial.Message = "The update worker could not be started."
		initial.FinishedAt = &finished
		_ = a.writeStatus(initial)
		ctx.AbortWithError(http.StatusBadGateway, fmt.Errorf("could not start update worker: %w", err))
		return
	}

	ctx.JSON(http.StatusAccepted, initial)
}

func (a *UpdateAPI) available() bool {
	if strings.TrimSpace(a.DockerSocket) == "" || strings.TrimSpace(a.TargetContainer) == "" {
		return false
	}
	if _, err := os.Stat(a.DockerSocket); err != nil {
		return false
	}
	return true
}

func (a *UpdateAPI) readStatus() (managedUpdateStatus, error) {
	status := managedUpdateStatus{Ready: true, State: "idle"}
	content, err := os.ReadFile(a.StatusFile)
	if errors.Is(err, os.ErrNotExist) {
		return status, nil
	}
	if err != nil {
		return managedUpdateStatus{}, err
	}
	if len(strings.TrimSpace(string(content))) == 0 {
		return status, nil
	}
	if err := json.Unmarshal(content, &status); err != nil {
		return managedUpdateStatus{}, err
	}
	return status, nil
}

func (a *UpdateAPI) writeStatus(status managedUpdateStatus) error {
	if err := os.MkdirAll(filepath.Dir(a.StatusFile), 0o700); err != nil {
		return err
	}
	content, err := json.Marshal(status)
	if err != nil {
		return err
	}
	temp := a.StatusFile + ".tmp"
	if err := os.WriteFile(temp, content, 0o600); err != nil {
		return err
	}
	return os.Rename(temp, a.StatusFile)
}

func updateStateActive(state string) bool {
	switch state {
	case "preparing", "downloading", "building", "replacing", "verifying":
		return true
	default:
		return false
	}
}

func runUpdateDocker(args ...string) (string, error) {
	command := exec.Command("docker", args...)
	output, err := command.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("docker %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return string(output), nil
}
