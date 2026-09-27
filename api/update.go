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
	defaultUpdateTarget = "gotify-mu"
	defaultUpdateRepo   = "gigabytegrove/gotify-mu"
	defaultUpdateWorker = "gotify-mu-update-worker"
	defaultUpdateStatus = "/app/data/.gotify-mu-update-status.json"
	defaultDockerSocket = "/var/run/docker.sock"
)

var (
	updateVersionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)
	activeUpdateStates    = map[string]struct{}{
		"preparing":   {},
		"downloading": {},
		"building":    {},
		"replacing":   {},
		"verifying":   {},
	}
)

type UpdateAPI struct {
	TargetContainer string
	Repository      string
	WorkerContainer string
	StatusFile      string
	DockerBin       string
	RunDocker       func(args ...string) (string, error)
	ReadyCheck      func() error
	Now             func() time.Time
}

type UpdateInstallRequest struct {
	Version string `json:"version" binding:"required"`
}

type UpdateActivity struct {
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
}

type UpdateStatus struct {
	Ready      bool             `json:"ready"`
	State      string           `json:"state"`
	Version    string           `json:"version,omitempty"`
	Message    string           `json:"message,omitempty"`
	Step       string           `json:"step,omitempty"`
	Progress   int              `json:"progress"`
	Activity   []UpdateActivity `json:"activity,omitempty"`
	StartedAt  *time.Time       `json:"startedAt,omitempty"`
	FinishedAt *time.Time       `json:"finishedAt,omitempty"`
}

func NewUpdateAPIFromEnv() UpdateAPI {
	target := strings.TrimSpace(os.Getenv("GOTIFY_MU_TARGET_CONTAINER"))
	if target == "" {
		target = defaultUpdateTarget
	}
	repository := strings.TrimSpace(os.Getenv("GOTIFY_MU_REPOSITORY"))
	if repository == "" {
		repository = defaultUpdateRepo
	}

	return UpdateAPI{
		TargetContainer: target,
		Repository:      repository,
		WorkerContainer: defaultUpdateWorker,
		StatusFile:      defaultUpdateStatus,
		DockerBin:       "docker",
		Now:             time.Now,
	}
}

func (a *UpdateAPI) Status(ctx *gin.Context) {
	if err := a.ready(); err != nil {
		ctx.JSON(http.StatusOK, UpdateStatus{
			Ready:   false,
			State:   "unavailable",
			Message: err.Error(),
		})
		return
	}

	status, err := a.readStatus()
	if err != nil {
		ctx.JSON(http.StatusOK, UpdateStatus{
			Ready:   true,
			State:   "failed",
			Message: "Update status could not be read. You can retry the update.",
		})
		return
	}

	if _, active := activeUpdateStates[status.State]; active {
		running, runErr := a.workerRunning()
		if runErr != nil {
			status.Ready = false
			status.State = "unavailable"
			status.Message = "The update worker could not be checked."
			ctx.JSON(http.StatusOK, status)
			return
		}
		if !running {
			finished := a.now().UTC()
			status.Ready = true
			status.State = "failed"
			status.Step = "Update stopped"
			status.Message = "The update worker stopped before the update completed."
			status.FinishedAt = &finished
			_ = a.writeStatus(status)
		}
	}

	status.Ready = true
	ctx.JSON(http.StatusOK, status)
}

func (a *UpdateAPI) Install(ctx *gin.Context) {
	if err := a.ready(); err != nil {
		ctx.AbortWithError(http.StatusServiceUnavailable, err)
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

	status, err := a.readStatus()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if _, active := activeUpdateStates[status.State]; active {
		ctx.JSON(http.StatusConflict, status)
		return
	}

	running, err := a.workerRunning()
	if err != nil {
		ctx.AbortWithError(http.StatusBadGateway, err)
		return
	}
	if running {
		ctx.AbortWithError(http.StatusConflict, errors.New("an update is already running"))
		return
	}
	_, _ = a.runDocker("rm", "-f", a.workerContainer())

	image, err := a.currentImage()
	if err != nil {
		ctx.AbortWithError(http.StatusBadGateway, err)
		return
	}

	started := a.now().UTC()
	status = UpdateStatus{
		Ready:     true,
		State:     "preparing",
		Version:   request.Version,
		Message:   "Preparing update",
		Step:      "Preparing update",
		Progress:  2,
		StartedAt: &started,
		Activity: []UpdateActivity{{
			Timestamp: started,
			Message:   "Update started",
		}},
	}
	if err := a.writeStatus(status); err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	args := []string{
		"run",
		"--rm",
		"-d",
		"--name", a.workerContainer(),
		"--volumes-from", a.targetContainer() + ":rw",
		"--env", "GOTIFY_MU_TARGET_CONTAINER=" + a.targetContainer(),
		"--env", "GOTIFY_MU_REPOSITORY=" + a.repository(),
		"--env", "GOTIFY_MU_UPDATE_STATUS_FILE=" + a.statusFile(),
		"--entrypoint", "/usr/local/bin/gotify-mu-updater",
		image,
		"--oneshot", request.Version,
	}
	if _, err := a.runDocker(args...); err != nil {
		finished := a.now().UTC()
		status.State = "failed"
		status.Step = "Update stopped"
		status.Message = "The update worker could not be started."
		status.FinishedAt = &finished
		_ = a.writeStatus(status)
		ctx.AbortWithError(http.StatusBadGateway, err)
		return
	}

	ctx.JSON(http.StatusAccepted, status)
}

func (a *UpdateAPI) currentImage() (string, error) {
	output, err := a.runDocker(
		"inspect",
		"--format", "{{.Config.Image}}",
		a.targetContainer(),
	)
	if err != nil {
		return "", fmt.Errorf("could not inspect the current Gotify MU container: %w", err)
	}
	image := strings.TrimSpace(output)
	if image == "" {
		return "", errors.New("the current Gotify MU image could not be determined")
	}
	return image, nil
}

func (a *UpdateAPI) workerRunning() (bool, error) {
	output, err := a.runDocker(
		"inspect",
		"--format", "{{.State.Running}}",
		a.workerContainer(),
	)
	if err != nil {
		lower := strings.ToLower(output)
		if strings.Contains(lower, "no such object") || strings.Contains(lower, "no such container") {
			return false, nil
		}
		return false, err
	}
	return strings.EqualFold(strings.TrimSpace(output), "true"), nil
}

func (a *UpdateAPI) ready() error {
	if a.ReadyCheck != nil {
		return a.ReadyCheck()
	}

	dockerBin := a.dockerBin()
	if _, err := exec.LookPath(dockerBin); err != nil {
		return errors.New("Managed updates are unavailable because the Docker client is not installed")
	}

	info, err := os.Stat(defaultDockerSocket)
	if err != nil {
		return errors.New("Managed updates are unavailable because Docker socket access is not available")
	}
	if info.Mode()&os.ModeSocket == 0 {
		return errors.New("Managed updates are unavailable because the Docker socket mount is invalid")
	}
	return nil
}

func (a *UpdateAPI) readStatus() (UpdateStatus, error) {
	file, err := os.Open(a.statusFile())
	if errors.Is(err, os.ErrNotExist) {
		return UpdateStatus{Ready: true, State: "idle", Progress: 0}, nil
	}
	if err != nil {
		return UpdateStatus{}, err
	}
	defer file.Close()

	var status UpdateStatus
	if err := json.NewDecoder(file).Decode(&status); err != nil {
		return UpdateStatus{}, err
	}
	if status.State == "" {
		status.State = "idle"
	}
	status.Ready = true
	return status, nil
}

func (a *UpdateAPI) writeStatus(status UpdateStatus) error {
	path := a.statusFile()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return err
	}
	payload = append(payload, '\n')

	temp := path + ".tmp"
	if err := os.WriteFile(temp, payload, 0o600); err != nil {
		return err
	}
	if err := os.Rename(temp, path); err != nil {
		_ = os.Remove(temp)
		return err
	}
	return nil
}

func (a *UpdateAPI) runDocker(args ...string) (string, error) {
	if a.RunDocker != nil {
		return a.RunDocker(args...)
	}

	command := exec.Command(a.dockerBin(), args...)
	output, err := command.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf(
			"docker %s: %w: %s",
			strings.Join(redactUpdateDockerArgs(args), " "),
			err,
			strings.TrimSpace(string(output)),
		)
	}
	return string(output), nil
}

func (a *UpdateAPI) targetContainer() string {
	if value := strings.TrimSpace(a.TargetContainer); value != "" {
		return value
	}
	return defaultUpdateTarget
}

func (a *UpdateAPI) repository() string {
	if value := strings.TrimSpace(a.Repository); value != "" {
		return value
	}
	return defaultUpdateRepo
}

func (a *UpdateAPI) workerContainer() string {
	if value := strings.TrimSpace(a.WorkerContainer); value != "" {
		return value
	}
	return defaultUpdateWorker
}

func (a *UpdateAPI) statusFile() string {
	if value := strings.TrimSpace(a.StatusFile); value != "" {
		return value
	}
	return defaultUpdateStatus
}

func (a *UpdateAPI) dockerBin() string {
	if value := strings.TrimSpace(a.DockerBin); value != "" {
		return value
	}
	return "docker"
}

func (a *UpdateAPI) now() time.Time {
	if a.Now != nil {
		return a.Now()
	}
	return time.Now()
}

func redactUpdateDockerArgs(args []string) []string {
	out := append([]string(nil), args...)
	for index := 0; index < len(out); index++ {
		if out[index] != "--env" || index+1 >= len(out) {
			continue
		}
		key, _, found := strings.Cut(out[index+1], "=")
		if found {
			out[index+1] = key + "=[masked]"
		} else {
			out[index+1] = "[masked]"
		}
		index++
	}
	return out
}
