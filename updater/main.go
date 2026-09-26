package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	defaultListen     = ":8099"
	defaultRepository = "gigabytegrove/gotify-mu"
	defaultTarget     = "gotify-mu"
)

var versionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)

type installRequest struct {
	Version string `json:"version"`
}

type activityEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
}

type updateStatus struct {
	Ready      bool            `json:"ready"`
	State      string          `json:"state"`
	Version    string          `json:"version,omitempty"`
	Message    string          `json:"message,omitempty"`
	Step       string          `json:"step,omitempty"`
	Progress   int             `json:"progress"`
	Activity   []activityEntry `json:"activity,omitempty"`
	StartedAt  *time.Time      `json:"startedAt,omitempty"`
	FinishedAt *time.Time      `json:"finishedAt,omitempty"`
}

type portBinding struct {
	HostIP   string `json:"HostIp"`
	HostPort string `json:"HostPort"`
}

type restartPolicy struct {
	Name              string `json:"Name"`
	MaximumRetryCount int    `json:"MaximumRetryCount"`
}

type mountInfo struct {
	Type        string `json:"Type"`
	Name        string `json:"Name"`
	Source      string `json:"Source"`
	Destination string `json:"Destination"`
	RW          bool   `json:"RW"`
}

type endpointInfo struct {
	Aliases []string `json:"Aliases"`
}

type inspectedContainer struct {
	Config struct {
		Env        []string          `json:"Env"`
		User       string            `json:"User"`
		WorkingDir string            `json:"WorkingDir"`
		Labels     map[string]string `json:"Labels"`
	} `json:"Config"`
	HostConfig struct {
		Binds         []string                 `json:"Binds"`
		PortBindings  map[string][]portBinding `json:"PortBindings"`
		RestartPolicy restartPolicy            `json:"RestartPolicy"`
		NetworkMode   string                   `json:"NetworkMode"`
		ExtraHosts    []string                 `json:"ExtraHosts"`
		DNS           []string                 `json:"Dns"`
		DNSSearch     []string                 `json:"DnsSearch"`
	} `json:"HostConfig"`
	Mounts []mountInfo `json:"Mounts"`
	NetworkSettings struct {
		Networks map[string]endpointInfo `json:"Networks"`
	} `json:"NetworkSettings"`
}

type manager struct {
	mu         sync.RWMutex
	status     updateStatus
	token      string
	repository string
	target     string
}

func newManager() *manager {
	token := strings.TrimSpace(os.Getenv("GOTIFY_MU_UPDATER_TOKEN"))
	repository := strings.TrimSpace(os.Getenv("GOTIFY_MU_REPOSITORY"))
	if repository == "" {
		repository = defaultRepository
	}
	target := strings.TrimSpace(os.Getenv("GOTIFY_MU_TARGET_CONTAINER"))
	if target == "" {
		target = defaultTarget
	}
	return &manager{
		status:     updateStatus{Ready: token != "", State: "idle"},
		token:      token,
		repository: repository,
		target:     target,
	}
}

func (m *manager) beginUpdate(version string, started time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status = updateStatus{
		Ready:     m.token != "",
		State:     "preparing",
		Version:   version,
		Message:   "Preparing update",
		Step:      "Preparing update",
		Progress:  2,
		StartedAt: &started,
		Activity: []activityEntry{{
			Timestamp: time.Now().UTC(),
			Message:   "Update started",
		}},
	}
}

func (m *manager) updateProgress(state, step, message string, progress int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if progress < m.status.Progress {
		progress = m.status.Progress
	}
	if progress > 100 {
		progress = 100
	}

	changed := step != "" && step != m.status.Step
	m.status.Ready = m.token != ""
	m.status.State = state
	m.status.Step = step
	m.status.Message = message
	m.status.Progress = progress

	if changed {
		m.status.Activity = append(m.status.Activity, activityEntry{
			Timestamp: time.Now().UTC(),
			Message:   step,
		})
		if len(m.status.Activity) > 40 {
			m.status.Activity = append([]activityEntry(nil), m.status.Activity[len(m.status.Activity)-40:]...)
		}
	}
}

func (m *manager) finishUpdate(state, step, message string, progress int, finished time.Time) {
	m.updateProgress(state, step, message, progress)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status.FinishedAt = &finished
}

func (m *manager) snapshot() updateStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	status := m.status
	status.Activity = append([]activityEntry(nil), m.status.Activity...)
	return status
}

func (m *manager) authenticate(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if m.token == "" {
			writeJSON(w, http.StatusServiceUnavailable, updateStatus{
				Ready:   false,
				State:   "unavailable",
				Message: "updater token is not configured",
			})
			return
		}
		if r.Header.Get("Authorization") != "Bearer "+m.token {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		next(w, r)
	}
}

func (m *manager) statusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, m.snapshot())
}

func (m *manager) installHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var request installRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	request.Version = strings.TrimSpace(strings.TrimPrefix(request.Version, "v"))
	if !versionPattern.MatchString(request.Version) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "version must be semantic x.y.z"})
		return
	}

	current := m.snapshot()
	switch current.State {
	case "preparing", "downloading", "building", "replacing", "verifying":
		writeJSON(w, http.StatusConflict, current)
		return
	}

	started := time.Now().UTC()
	m.beginUpdate(request.Version, started)
	go m.performInstall(request.Version, started)

	writeJSON(w, http.StatusAccepted, m.snapshot())
}

func (m *manager) performInstall(version string, started time.Time) {
	tempDir, err := os.MkdirTemp("", "gotify-mu-update-*")
	if err != nil {
		m.fail(version, started, "The update could not be prepared.", err)
		return
	}
	defer os.RemoveAll(tempDir)

	m.updateProgress("preparing", "Checking release", "Checking release", 5)
	release, err := m.resolveRelease(version)
	if err != nil {
		m.fail(version, started, "The release could not be verified.", err)
		return
	}

	archivePath := filepath.Join(tempDir, release.SourceName)
	checksumPath := filepath.Join(tempDir, "SHA256SUMS")
	m.updateProgress("downloading", "Downloading update", "Downloading update", 10)
	if err := m.downloadFile(release.SourceURL, archivePath, 10, 23); err != nil {
		m.fail(version, started, "The update could not be downloaded.", err)
		return
	}
	if err := m.downloadFile(release.ChecksumURL, checksumPath, 23, 25); err != nil {
		m.fail(version, started, "The release checksum could not be downloaded.", err)
		return
	}
	m.updateProgress("preparing", "Verifying update", "Verifying update", 26)
	if err := verifyReleaseChecksum(archivePath, checksumPath, release.SourceName); err != nil {
		m.fail(version, started, "The update failed integrity verification.", err)
		return
	}

	m.updateProgress("preparing", "Preparing update files", "Preparing update files", 27)
	sourceDir := filepath.Join(tempDir, "source")
	if err := unzip(archivePath, sourceDir); err != nil {
		m.fail(version, started, "The update files could not be prepared.", err)
		return
	}
	root, err := singleDirectory(sourceDir)
	if err != nil {
		m.fail(version, started, "The update files could not be prepared.", err)
		return
	}

	m.updateProgress("building", "Installing update", "Installing update", 30)
	image := "gotify-mu:release-" + version
	buildDate := time.Now().UTC().Format(time.RFC3339)
	if err := m.buildRelease(root, image, version, release.Commit, buildDate); err != nil {
		m.fail(version, started, "The update could not be installed.", err)
		return
	}

	m.updateProgress("replacing", "Applying update", "Applying update", 82)
	if err := m.replaceContainer(image, version, started); err != nil {
		return
	}

	finished := time.Now().UTC()
	m.finishUpdate("completed", "Update complete", "Update installed successfully", 100, finished)
}

type releaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type releaseMetadata struct {
	TagName  string         `json:"tag_name"`
	Assets   []releaseAsset `json:"assets"`
	Commit   string         `json:"-"`
	SourceURL string        `json:"-"`
	ChecksumURL string      `json:"-"`
	SourceName string       `json:"-"`
}

func (m *manager) resolveRelease(version string) (*releaseMetadata, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/tags/v%s", m.repository, version)
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "gotify-mu-updater")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("could not load release metadata: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("could not load release metadata: GitHub returned HTTP %d", response.StatusCode)
	}

	var release releaseMetadata
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&release); err != nil {
		return nil, fmt.Errorf("could not decode release metadata: %w", err)
	}
	if release.TagName != "v"+version {
		return nil, errors.New("release tag did not match requested version")
	}
	release.SourceName = "gotify-mu-v" + version + "-source.zip"
	for _, asset := range release.Assets {
		switch asset.Name {
		case release.SourceName:
			release.SourceURL = asset.BrowserDownloadURL
		case "SHA256SUMS":
			release.ChecksumURL = asset.BrowserDownloadURL
		}
	}
	if release.SourceURL == "" || release.ChecksumURL == "" {
		return nil, errors.New("release is missing the required source or checksum asset")
	}
	commit, err := m.resolveCommit(version)
	if err != nil {
		return nil, err
	}
	release.Commit = commit
	return &release, nil
}

func verifyReleaseChecksum(archivePath, checksumPath, sourceName string) error {
	data, err := os.ReadFile(checksumPath)
	if err != nil {
		return fmt.Errorf("read release checksum: %w", err)
	}
	var expected string
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) < 2 {
			continue
		}
		name := strings.TrimPrefix(fields[len(fields)-1], "*")
		if filepath.Base(name) == sourceName {
			expected = strings.ToLower(fields[0])
			break
		}
	}
	if expected == "" {
		return errors.New("release checksum file does not contain the source archive")
	}
	if _, err := hex.DecodeString(expected); err != nil || len(expected) != sha256.Size*2 {
		return errors.New("release checksum is not a valid SHA-256 value")
	}
	file, err := os.Open(archivePath)
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
		return fmt.Errorf("release checksum mismatch: expected %s, got %s", expected, actual)
	}
	return nil
}

func (m *manager) resolveCommit(version string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/commits/v%s", m.repository, version)
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "gotify-mu-updater")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("could not resolve release commit: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("could not resolve release commit: GitHub returned HTTP %d", response.StatusCode)
	}

	var payload struct {
		SHA string `json:"sha"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("could not decode release commit: %w", err)
	}
	if payload.SHA == "" {
		return "", errors.New("GitHub did not return a release commit")
	}
	return payload.SHA, nil
}

func (m *manager) replaceContainer(image, version string, started time.Time) error {
	m.updateProgress("replacing", "Saving current installation", "Saving current installation", 84)
	inspection, err := inspectContainer(m.target)
	if err != nil {
		m.fail(version, started, "The current installation could not be prepared for updating.", err)
		return err
	}

	rollback := fmt.Sprintf("%s-rollback-%d", m.target, time.Now().Unix())

	m.updateProgress("replacing", "Preparing restart", "Preparing restart", 87)
	if _, err := runDocker("stop", "-t", "20", m.target); err != nil {
		m.fail(version, started, "The service could not be stopped safely.", err)
		return err
	}
	if _, err := runDocker("rename", m.target, rollback); err != nil {
		_, _ = runDocker("start", m.target)
		m.fail(version, started, "The current installation could not be preserved.", err)
		return err
	}

	restore := func(cause error) error {
		_, _ = runDocker("rm", "-f", m.target)
		_, renameErr := runDocker("rename", rollback, m.target)
		_, startErr := runDocker("start", m.target)
		finished := time.Now().UTC()
		if renameErr != nil || startErr != nil {
			m.finishUpdate(
				"failed",
				"Recovery required",
				"The update could not be completed and automatic recovery was unsuccessful. The server administrator should review the service.",
				m.snapshot().Progress,
				finished,
			)
			log.Printf("update failed and rollback failed: update=%v rename=%v start=%v", cause, renameErr, startErr)
			return cause
		}
		m.finishUpdate(
			"rolled_back",
			"Previous version restored",
			"The update could not be completed. The previous version was restored automatically.",
			m.snapshot().Progress,
			finished,
		)
		log.Printf("update failed and previous version was restored: %v", cause)
		return cause
	}

	m.updateProgress("replacing", "Applying new version", "Applying new version", 90)
	args := createArgs(m.target, image, inspection)
	if _, err := runDocker(args...); err != nil {
		return restore(fmt.Errorf("could not create replacement service: %w", err))
	}

	for network := range inspection.NetworkSettings.Networks {
		if network == inspection.HostConfig.NetworkMode || network == "bridge" || network == "default" {
			continue
		}
		_, _ = runDocker("network", "connect", network, m.target)
	}

	m.updateProgress("replacing", "Starting updated version", "Starting updated version", 94)
	if _, err := runDocker("start", m.target); err != nil {
		return restore(fmt.Errorf("could not start replacement service: %w", err))
	}

	m.updateProgress("verifying", "Checking updated version", "Checking updated version", 96)
	if err := m.waitForHealthy(m.target, 120*time.Second); err != nil {
		return restore(err)
	}

	if _, err := runDocker("rm", "-f", rollback); err != nil {
		log.Printf("warning: could not remove rollback service %s: %v", rollback, err)
	}
	return nil
}

func createArgs(name, image string, inspected *inspectedContainer) []string {
	args := []string{"create", "--name", name}

	if policy := inspected.HostConfig.RestartPolicy; policy.Name != "" && policy.Name != "no" {
		value := policy.Name
		if policy.Name == "on-failure" && policy.MaximumRetryCount > 0 {
			value = fmt.Sprintf("%s:%d", policy.Name, policy.MaximumRetryCount)
		}
		args = append(args, "--restart", value)
	}

	for _, env := range inspected.Config.Env {
		args = append(args, "--env", env)
	}

	binds := inspected.HostConfig.Binds
	if len(binds) == 0 {
		for _, mount := range inspected.Mounts {
			switch mount.Type {
			case "bind":
				value := mount.Source + ":" + mount.Destination
				if !mount.RW {
					value += ":ro"
				}
				binds = append(binds, value)
			case "volume":
				source := mount.Name
				if source == "" {
					source = mount.Source
				}
				value := source + ":" + mount.Destination
				if !mount.RW {
					value += ":ro"
				}
				binds = append(binds, value)
			}
		}
	}
	for _, bind := range binds {
		args = append(args, "--volume", bind)
	}

	for containerPort, bindings := range inspected.HostConfig.PortBindings {
		for _, binding := range bindings {
			published := binding.HostPort + ":" + containerPort
			if binding.HostIP != "" && binding.HostIP != "0.0.0.0" {
				published = binding.HostIP + ":" + published
			}
			args = append(args, "--publish", published)
		}
	}

	networkMode := inspected.HostConfig.NetworkMode
	if networkMode != "" && networkMode != "default" && networkMode != "bridge" && networkMode != "host" && !strings.HasPrefix(networkMode, "container:") {
		args = append(args, "--network", networkMode)
	} else if networkMode == "host" {
		args = append(args, "--network", "host")
	}

	for _, host := range inspected.HostConfig.ExtraHosts {
		args = append(args, "--add-host", host)
	}
	for _, dns := range inspected.HostConfig.DNS {
		args = append(args, "--dns", dns)
	}
	for _, search := range inspected.HostConfig.DNSSearch {
		args = append(args, "--dns-search", search)
	}
	if inspected.Config.User != "" {
		args = append(args, "--user", inspected.Config.User)
	}
	if inspected.Config.WorkingDir != "" {
		args = append(args, "--workdir", inspected.Config.WorkingDir)
	}

	args = append(args, image)
	return args
}

func inspectContainer(name string) (*inspectedContainer, error) {
	output, err := runDocker("inspect", name)
	if err != nil {
		return nil, err
	}
	var payload []inspectedContainer
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		return nil, err
	}
	if len(payload) != 1 {
		return nil, fmt.Errorf("unexpected inspect response")
	}
	return &payload[0], nil
}

func (m *manager) waitForHealthy(name string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		output, err := runDocker("inspect", "--format", "{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}", name)
		if err == nil {
			status := strings.TrimSpace(output)
			if status == "healthy" || status == "running" {
				m.updateProgress("verifying", "Final checks", "Final checks", 99)
				return nil
			}
			if status == "unhealthy" || status == "exited" || status == "dead" {
				return fmt.Errorf("replacement service entered state %s", status)
			}
		}
		elapsed := timeout - time.Until(deadline)
		progress := 96 + int(elapsed.Seconds()/30)
		if progress > 99 {
			progress = 99
		}
		m.updateProgress("verifying", "Checking updated version", "Checking updated version", progress)
		time.Sleep(2 * time.Second)
	}
	return errors.New("replacement service did not become healthy before timeout")
}

func formatDockerArgs(args []string) string {
	safe := append([]string(nil), args...)
	for i := 0; i < len(safe); i++ {
		if (safe[i] == "--env" || safe[i] == "-e") && i+1 < len(safe) {
			if key, _, ok := strings.Cut(safe[i+1], "="); ok {
				safe[i+1] = key + "=[masked]"
			} else {
				safe[i+1] = "[masked]"
			}
			i++
			continue
		}
		if strings.HasPrefix(safe[i], "--env=") {
			raw := strings.TrimPrefix(safe[i], "--env=")
			if key, _, ok := strings.Cut(raw, "="); ok {
				safe[i] = "--env=" + key + "=[masked]"
			} else {
				safe[i] = "--env=[masked]"
			}
		}
	}
	return strings.Join(safe, " ")
}

func runDocker(args ...string) (string, error) {
	command := exec.Command("docker", args...)
	output, err := command.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("docker %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return string(output), nil
}

func (m *manager) downloadFile(url, path string, startProgress, endProgress int) error {
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	request.Header.Set("User-Agent", "gotify-mu-updater")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", response.StatusCode)
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	if response.ContentLength <= 0 {
		_, err = io.Copy(file, response.Body)
		if err == nil {
			m.updateProgress("downloading", "Downloading update", "Downloading update", endProgress)
		}
		return err
	}

	buffer := make([]byte, 64*1024)
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
			m.updateProgress("downloading", "Downloading update", "Downloading update", progress)
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	return nil
}

func unzip(archivePath, destination string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer reader.Close()
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return err
	}
	cleanRoot := filepath.Clean(destination) + string(os.PathSeparator)

	for _, file := range reader.File {
		target := filepath.Join(destination, file.Name)
		if !strings.HasPrefix(filepath.Clean(target)+string(os.PathSeparator), cleanRoot) {
			return fmt.Errorf("archive contains invalid path %q", file.Name)
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		source, err := file.Open()
		if err != nil {
			return err
		}
		destinationFile, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, file.Mode())
		if err != nil {
			source.Close()
			return err
		}
		_, copyErr := io.Copy(destinationFile, source)
		closeErr := destinationFile.Close()
		source.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

func singleDirectory(root string) (string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			return filepath.Join(root, entry.Name()), nil
		}
	}
	return "", errors.New("archive did not contain a source directory")
}

func (m *manager) fail(version string, started time.Time, publicMessage string, err error) {
	finished := time.Now().UTC()
	m.finishUpdate("failed", "Update stopped", publicMessage, m.snapshot().Progress, finished)
	log.Printf("update v%s failed: %v", version, err)
}

type buildProgressWriter struct {
	mu      sync.Mutex
	manager *manager
	buffer  string
}

func (w *buildProgressWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.buffer += string(p)
	for {
		index := strings.IndexByte(w.buffer, '\n')
		if index < 0 {
			break
		}
		line := strings.TrimSpace(w.buffer[:index])
		w.buffer = w.buffer[index+1:]
		w.manager.handleBuildProgress(line)
	}
	return len(p), nil
}

func (m *manager) buildRelease(root, image, version, commit, buildDate string) error {
	command := exec.Command(
		"docker",
		"build",
		"--progress=plain",
		"--pull",
		"--build-arg", "BUILD_JS=1",
		"--build-arg", "RUN_TESTS=1",
		"--build-arg", "GO_VERSION=1.26.0",
		"--build-arg", "GOTIFY_MU_VERSION="+version,
		"--build-arg", "GOTIFY_MU_COMMIT="+commit,
		"--build-arg", "GOTIFY_MU_BUILD_DATE="+buildDate,
		"-f", filepath.Join(root, "docker", "Dockerfile"),
		"-t", image,
		root,
	)
	writer := &buildProgressWriter{manager: m}
	command.Stdout = writer
	command.Stderr = writer
	if err := command.Run(); err != nil {
		return err
	}
	m.updateProgress("building", "Update files ready", "Update files ready", 80)
	return nil
}

func (m *manager) handleBuildProgress(line string) {
	if line == "" {
		return
	}
	switch {
	case strings.Contains(line, "load build definition"):
		m.updateProgress("building", "Reading update package", "Reading update package", 33)
	case strings.Contains(line, "load metadata"):
		m.updateProgress("building", "Checking required components", "Checking required components", 37)
	case strings.Contains(line, "js-builder"):
		m.updateProgress("building", "Preparing interface", "Preparing interface", 47)
	case strings.Contains(line, "[builder "):
		m.updateProgress("building", "Preparing application", "Preparing application", 62)
	case strings.Contains(line, "[stage-2 "):
		m.updateProgress("building", "Assembling update", "Assembling update", 73)
	case strings.Contains(line, "exporting to image"):
		m.updateProgress("building", "Finalizing update files", "Finalizing update files", 78)
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func main() {
	manager := newManager()
	listen := strings.TrimSpace(os.Getenv("GOTIFY_MU_UPDATER_LISTEN"))
	if listen == "" {
		listen = defaultListen
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/status", manager.authenticate(manager.statusHandler))
	mux.HandleFunc("/install", manager.authenticate(manager.installHandler))

	server := &http.Server{
		Addr:              listen,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("Gotify MU updater listening on %s for container %s", listen, manager.target)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
