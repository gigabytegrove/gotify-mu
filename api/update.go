package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	defaultUpdateStatusFile = "/app/data/.monita-update-status.json"
	defaultRuntimeDir       = "/app/data/.monita-runtime"
	defaultUpdateRepository = "gigabytegrove/monita"
)

var updateVersionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)

type UpdateAPI struct {
	StatusFile string
	RuntimeDir string
	Repository string
	HTTPClient *http.Client
	Exec       func(string, []string, []string) error
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

func firstUpdateEnv(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}

func NewUpdateAPIFromEnv() UpdateAPI {
	statusFile := firstUpdateEnv("MONITA_UPDATE_STATUS_FILE", "GOTIFY_MU_UPDATE_STATUS_FILE")
	if statusFile == "" {
		statusFile = defaultUpdateStatusFile
	}

	runtimeDir := firstUpdateEnv("MONITA_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = defaultRuntimeDir
	}

	repository := firstUpdateEnv("MONITA_REPOSITORY", "GOTIFY_MU_REPOSITORY")
	if repository == "" {
		repository = defaultUpdateRepository
	}

	api := UpdateAPI{
		StatusFile: statusFile,
		RuntimeDir: runtimeDir,
		Repository: repository,
		HTTPClient: &http.Client{Timeout: 10 * time.Minute},
		Exec:       syscall.Exec,
	}
	api.finalizeRestart()
	return api
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

	ctx.JSON(http.StatusAccepted, initial)
	go a.performInstall(request.Version, started)
}

func (a *UpdateAPI) performInstall(version string, started time.Time) {
	assetName, err := runtimeAssetName(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		a.fail(version, started, "This system architecture is not supported by automatic updates.", err)
		return
	}

	a.updateProgress("preparing", "Checking release", "Checking release", 5)
	assetURL, checksumURL, err := a.resolveReleaseAssets(version, assetName)
	if err != nil {
		a.fail(version, started, "The release could not be verified.", err)
		return
	}

	if err := os.MkdirAll(a.RuntimeDir, 0o755); err != nil {
		a.fail(version, started, "The update directory could not be prepared.", err)
		return
	}

	tempPath := filepath.Join(a.RuntimeDir, ".monita-app-"+version+".tmp")
	_ = os.Remove(tempPath)

	a.updateProgress("downloading", "Downloading update", "Downloading update", 12)
	if err := a.downloadFile(assetURL, tempPath, 12, 68); err != nil {
		_ = os.Remove(tempPath)
		a.fail(version, started, "The update could not be downloaded.", err)
		return
	}

	a.updateProgress("verifying", "Verifying update", "Verifying update", 72)
	checksums, err := a.downloadText(checksumURL)
	if err != nil {
		_ = os.Remove(tempPath)
		a.fail(version, started, "The release checksum could not be downloaded.", err)
		return
	}
	if err := verifyReleaseChecksum(tempPath, checksums, assetName); err != nil {
		_ = os.Remove(tempPath)
		a.fail(version, started, "The downloaded update failed integrity verification.", err)
		return
	}
	if err := os.Chmod(tempPath, 0o755); err != nil {
		_ = os.Remove(tempPath)
		a.fail(version, started, "The update could not be prepared.", err)
		return
	}
	if err := verifyRuntimeBinary(tempPath, version); err != nil {
		_ = os.Remove(tempPath)
		a.fail(version, started, "The downloaded update could not be verified.", err)
		return
	}

	a.updateProgress("installing", "Installing update", "Installing update", 84)

	runtimePath := filepath.Join(a.RuntimeDir, "monita-app")
	previousPath := filepath.Join(a.RuntimeDir, "monita-app.previous")
	_ = os.Remove(previousPath)

	hadPrevious := false
	if _, err := os.Stat(runtimePath); err == nil {
		if err := os.Rename(runtimePath, previousPath); err != nil {
			_ = os.Remove(tempPath)
			a.fail(version, started, "The current runtime could not be preserved.", err)
			return
		}
		hadPrevious = true
	} else if !errors.Is(err, os.ErrNotExist) {
		_ = os.Remove(tempPath)
		a.fail(version, started, "The current runtime could not be inspected.", err)
		return
	}

	if err := os.Rename(tempPath, runtimePath); err != nil {
		if hadPrevious {
			_ = os.Rename(previousPath, runtimePath)
		}
		a.fail(version, started, "The update could not be installed.", err)
		return
	}

	a.updateProgress("restarting", "Restarting Monita", "Restarting Monita", 98)

	execFn := a.Exec
	if execFn == nil {
		execFn = syscall.Exec
	}
	args := append([]string{runtimePath}, os.Args[1:]...)
	if err := execFn(runtimePath, args, os.Environ()); err != nil {
		_ = os.Remove(runtimePath)
		if hadPrevious {
			_ = os.Rename(previousPath, runtimePath)
		}
		a.fail(version, started, "Monita could not restart after the update.", err)
	}
}

func (a *UpdateAPI) resolveReleaseAssets(version, assetName string) (string, string, error) {
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/releases/tags/v%s", a.Repository, version)
	request, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return "", "", err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "monita")

	response, err := a.client().Do(request)
	if err != nil {
		return "", "", err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("GitHub release lookup returned HTTP %d", response.StatusCode)
	}

	var payload struct {
		Assets []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return "", "", err
	}

	var assetURL, checksumURL string
	for _, asset := range payload.Assets {
		switch asset.Name {
		case assetName:
			assetURL = asset.BrowserDownloadURL
		case "SHA256SUMS":
			checksumURL = asset.BrowserDownloadURL
		}
	}
	if assetURL == "" || checksumURL == "" {
		return "", "", errors.New("release is missing the runtime binary or SHA256SUMS")
	}
	return assetURL, checksumURL, nil
}

func runtimeAssetName(goos, goarch string) (string, error) {
	if goos != "linux" {
		return "", fmt.Errorf("unsupported operating system %s", goos)
	}
	switch goarch {
	case "amd64":
		return "monita-linux-amd64", nil
	case "arm64":
		return "monita-linux-arm64", nil
	default:
		return "", fmt.Errorf("unsupported architecture %s", goarch)
	}
}

func (a *UpdateAPI) downloadFile(url, path string, startProgress, endProgress int) error {
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	request.Header.Set("User-Agent", "monita")

	response, err := a.client().Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned HTTP %d", response.StatusCode)
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()

	if response.ContentLength <= 0 {
		if _, err := io.Copy(file, response.Body); err != nil {
			return err
		}
		a.updateProgress("downloading", "Downloading update", "Downloading update", endProgress)
		return nil
	}

	buffer := make([]byte, 128*1024)
	var written int64
	for {
		n, readErr := response.Body.Read(buffer)
		if n > 0 {
			if _, err := file.Write(buffer[:n]); err != nil {
				return err
			}
			written += int64(n)
			fraction := float64(written) / float64(response.ContentLength)
			progress := startProgress + int(fraction*float64(endProgress-startProgress))
			a.updateProgress("downloading", "Downloading update", "Downloading update", progress)
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	return file.Sync()
}

func (a *UpdateAPI) downloadText(url string) (string, error) {
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("User-Agent", "monita")

	response, err := a.client().Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download returned HTTP %d", response.StatusCode)
	}

	content, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func verifyReleaseChecksum(path, checksumContent, expectedName string) error {
	var expected string
	for _, line := range strings.Split(checksumContent, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if filepath.Base(strings.TrimPrefix(fields[len(fields)-1], "*")) == expectedName {
			expected = strings.ToLower(fields[0])
			break
		}
	}
	if len(expected) != 64 {
		return errors.New("SHA256SUMS does not contain the runtime binary")
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}
	actual := hex.EncodeToString(hash.Sum(nil))
	if actual != expected {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expected, actual)
	}
	return nil
}

func verifyRuntimeBinary(path, version string) error {
	output, err := exec.Command(path, "version").CombinedOutput()
	if err != nil {
		return fmt.Errorf("runtime verification failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	if !strings.Contains(string(output), "Version: "+version) {
		return fmt.Errorf("runtime reported an unexpected version: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

func (a *UpdateAPI) client() *http.Client {
	if a.HTTPClient != nil {
		return a.HTTPClient
	}
	return &http.Client{Timeout: 10 * time.Minute}
}

func (a *UpdateAPI) available() bool {
	return strings.TrimSpace(a.StatusFile) != "" &&
		strings.TrimSpace(a.RuntimeDir) != "" &&
		strings.TrimSpace(a.Repository) != ""
}

func (a *UpdateAPI) finalizeRestart() {
	status, err := a.readStatus()
	if err != nil || status.State != "restarting" {
		return
	}
	finished := time.Now().UTC()
	status.Ready = true
	status.State = "completed"
	status.Step = "Update complete"
	status.Message = "Update installed successfully"
	status.Progress = 100
	status.FinishedAt = &finished
	status.Activity = append(status.Activity, updateActivity{
		Timestamp: finished,
		Message:   "Update complete",
	})
	_ = a.writeStatus(status)
}

func (a *UpdateAPI) updateProgress(state, step, message string, progress int) {
	status, err := a.readStatus()
	if err != nil {
		return
	}
	if progress < status.Progress {
		progress = status.Progress
	}
	if progress > 100 {
		progress = 100
	}
	if step != "" && step != status.Step {
		status.Activity = append(status.Activity, updateActivity{
			Timestamp: time.Now().UTC(),
			Message:   step,
		})
		if len(status.Activity) > 40 {
			status.Activity = append([]updateActivity(nil), status.Activity[len(status.Activity)-40:]...)
		}
	}
	status.Ready = true
	status.State = state
	status.Step = step
	status.Message = message
	status.Progress = progress
	_ = a.writeStatus(status)
}

func (a *UpdateAPI) fail(version string, started time.Time, message string, err error) {
	status, readErr := a.readStatus()
	if readErr != nil {
		status = managedUpdateStatus{
			Ready:     true,
			Version:   version,
			StartedAt: &started,
		}
	}
	finished := time.Now().UTC()
	status.Ready = true
	status.State = "failed"
	status.Step = "Update stopped"
	status.Message = message
	status.FinishedAt = &finished
	status.Activity = append(status.Activity, updateActivity{
		Timestamp: finished,
		Message:   "Update stopped",
	})
	_ = a.writeStatus(status)
	fmt.Fprintf(os.Stderr, "Monita update v%s failed: %v\n", version, err)
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
	case "preparing", "downloading", "verifying", "installing", "restarting":
		return true
	default:
		return false
	}
}
