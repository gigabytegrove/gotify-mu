package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultRepository = "gigabytegrove/gotify-mu"
	defaultTarget     = "gotify-mu"
	defaultStatusFile  = "/app/data/.gotify-mu-update-status.json"
	defaultRuntimeImage = "gotify-mu:managed"
)

var versionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)

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

type deviceMapping struct {
	PathOnHost        string `json:"PathOnHost"`
	PathInContainer   string `json:"PathInContainer"`
	CgroupPermissions string `json:"CgroupPermissions"`
}

type logConfig struct {
	Type   string            `json:"Type"`
	Config map[string]string `json:"Config"`
}

type ulimit struct {
	Name string `json:"Name"`
	Hard int64  `json:"Hard"`
	Soft int64  `json:"Soft"`
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
		Hostname   string            `json:"Hostname"`
		Cmd        []string          `json:"Cmd"`
		Entrypoint []string          `json:"Entrypoint"`
	} `json:"Config"`
	HostConfig struct {
		Binds         []string                 `json:"Binds"`
		PortBindings  map[string][]portBinding `json:"PortBindings"`
		RestartPolicy restartPolicy            `json:"RestartPolicy"`
		NetworkMode   string                   `json:"NetworkMode"`
		ExtraHosts    []string                 `json:"ExtraHosts"`
		DNS           []string                 `json:"Dns"`
		DNSSearch     []string                 `json:"DnsSearch"`
		Memory        int64                    `json:"Memory"`
		NanoCPUs      int64                    `json:"NanoCpus"`
		PidsLimit     int64                    `json:"PidsLimit"`
		CapAdd        []string                 `json:"CapAdd"`
		CapDrop       []string                 `json:"CapDrop"`
		ReadonlyRootfs bool                    `json:"ReadonlyRootfs"`
		Privileged    bool                     `json:"Privileged"`
		SecurityOpt   []string                 `json:"SecurityOpt"`
		ShmSize       int64                    `json:"ShmSize"`
		Tmpfs         map[string]string        `json:"Tmpfs"`
		Devices       []deviceMapping          `json:"Devices"`
		LogConfig     logConfig                `json:"LogConfig"`
		Ulimits       []ulimit                 `json:"Ulimits"`
	} `json:"HostConfig"`
	Mounts []mountInfo `json:"Mounts"`
	NetworkSettings struct {
		Networks map[string]endpointInfo `json:"Networks"`
	} `json:"NetworkSettings"`
}

type manager struct {
	mu         sync.RWMutex
	status     updateStatus
	repository string
	target     string
	statusFile   string
	runtimeImage string
}

func newManager() *manager {
	repository := strings.TrimSpace(os.Getenv("GOTIFY_MU_REPOSITORY"))
	if repository == "" {
		repository = defaultRepository
	}
	target := strings.TrimSpace(os.Getenv("GOTIFY_MU_TARGET_CONTAINER"))
	if target == "" {
		target = defaultTarget
	}
	statusFile := strings.TrimSpace(os.Getenv("GOTIFY_MU_UPDATE_STATUS_FILE"))
	if statusFile == "" {
		statusFile = defaultStatusFile
	}
	runtimeImage := strings.TrimSpace(os.Getenv("GOTIFY_MU_RUNTIME_IMAGE"))
	if runtimeImage == "" {
		runtimeImage = defaultRuntimeImage
	}
	return &manager{
		status:       updateStatus{Ready: true, State: "idle"},
		repository:   repository,
		target:       target,
		statusFile:   statusFile,
		runtimeImage: runtimeImage,
	}
}

func (m *manager) beginUpdate(version string, started time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status = updateStatus{
		Ready:     true,
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
	m.persistStatusLocked()
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
	m.status.Ready = true
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
	m.persistStatusLocked()
}

func (m *manager) finishUpdate(state, step, message string, progress int, finished time.Time) {
	m.updateProgress(state, step, message, progress)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status.FinishedAt = &finished
	m.persistStatusLocked()
}

func (m *manager) persistStatusLocked() {
	if m.statusFile == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(m.statusFile), 0o755); err != nil {
		log.Printf("warning: could not create updater status directory: %v", err)
		return
	}
	payload, err := json.MarshalIndent(m.status, "", "  ")
	if err != nil {
		log.Printf("warning: could not encode updater status: %v", err)
		return
	}
	payload = append(payload, '\n')
	temp := m.statusFile + ".tmp"
	if err := os.WriteFile(temp, payload, 0o600); err != nil {
		log.Printf("warning: could not write updater status: %v", err)
		return
	}
	if err := os.Rename(temp, m.statusFile); err != nil {
		_ = os.Remove(temp)
		log.Printf("warning: could not publish updater status: %v", err)
	}
}

func (m *manager) snapshot() updateStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	status := m.status
	status.Activity = append([]activityEntry(nil), m.status.Activity...)
	return status
}

func (m *manager) performInstall(version string, started time.Time) {
	m.updateProgress("preparing", "Checking release", "Checking release", 5)
	commit, err := m.resolveCommit(version)
	if err != nil {
		m.fail(version, started, "The release could not be verified.", err)
		return
	}

	image := fmt.Sprintf("ghcr.io/%s:%s", strings.ToLower(m.repository), version)
	m.updateProgress("downloading", "Downloading update", "Downloading update", 15)
	if err := m.pullReleaseImage(image); err != nil {
		m.fail(version, started, "The update image could not be downloaded.", err)
		return
	}

	m.updateProgress("preparing", "Verifying update", "Verifying update", 72)
	if err := verifyReleaseImage(image, version, commit, m.repository); err != nil {
		m.fail(version, started, "The downloaded update failed verification.", err)
		return
	}

	m.updateProgress("preparing", "Preparing restart", "Preparing restart", 78)
	if _, err := runDocker("tag", image, m.runtimeImage); err != nil {
		m.fail(version, started, "The verified update image could not be prepared.", err)
		return
	}

	m.updateProgress("replacing", "Applying update", "Applying update", 82)
	if err := m.replaceContainer(m.runtimeImage, version, started); err != nil {
		return
	}

	finished := time.Now().UTC()
	m.finishUpdate("completed", "Update complete", "Update installed successfully", 100, finished)
}

func (m *manager) pullReleaseImage(image string) error {
	command := exec.Command("docker", "pull", image)
	writer := &pullProgressWriter{manager: m}
	command.Stdout = writer
	command.Stderr = writer
	if err := command.Run(); err != nil {
		return fmt.Errorf("docker pull %s: %w", image, err)
	}
	m.updateProgress("downloading", "Download complete", "Download complete", 70)
	return nil
}

type pullProgressWriter struct {
	mu      sync.Mutex
	manager *manager
	buffer  string
}

func (w *pullProgressWriter) Write(p []byte) (int, error) {
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
		w.manager.handlePullProgress(line)
	}
	return len(p), nil
}

func (m *manager) handlePullProgress(line string) {
	switch {
	case strings.Contains(line, "Pulling from"):
		m.updateProgress("downloading", "Downloading update", "Downloading update", 20)
	case strings.Contains(line, "Downloading"):
		m.updateProgress("downloading", "Downloading update", "Downloading update", 35)
	case strings.Contains(line, "Extracting"):
		m.updateProgress("downloading", "Preparing update", "Preparing update", 55)
	case strings.Contains(line, "Pull complete"):
		m.updateProgress("downloading", "Preparing update", "Preparing update", 65)
	case strings.HasPrefix(line, "Digest:"):
		m.updateProgress("downloading", "Download complete", "Download complete", 70)
	}
}

func verifyReleaseImage(image, version, commit, repository string) error {
	output, err := runDocker(
		"image", "inspect",
		"--format",
		`{{index .Config.Labels "org.opencontainers.image.version"}}|{{index .Config.Labels "org.opencontainers.image.revision"}}|{{index .Config.Labels "org.opencontainers.image.source"}}`,
		image,
	)
	if err != nil {
		return err
	}
	parts := strings.Split(strings.TrimSpace(output), "|")
	if len(parts) != 3 {
		return errors.New("release image metadata is incomplete")
	}
	if parts[0] != version {
		return fmt.Errorf("release image version mismatch: expected %s, got %s", version, parts[0])
	}
	if parts[1] != commit {
		return fmt.Errorf("release image revision mismatch: expected %s, got %s", commit, parts[1])
	}
	expectedSource := "https://github.com/" + repository
	if parts[2] != expectedSource {
		return fmt.Errorf("release image source mismatch: expected %s, got %s", expectedSource, parts[2])
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
		key, _, _ := strings.Cut(env, "=")
		switch key {
		case "GOTIFY_MU_UPDATER_URL", "GOTIFY_MU_UPDATER_TOKEN":
			continue
		}
		args = append(args, "--env", env)
	}
	for key, value := range inspected.Config.Labels {
		args = append(args, "--label", key+"="+value)
	}
	if inspected.Config.Hostname != "" {
		args = append(args, "--hostname", inspected.Config.Hostname)
	}
	if len(inspected.Config.Entrypoint) > 0 && inspected.Config.Entrypoint[0] != "" {
		args = append(args, "--entrypoint", inspected.Config.Entrypoint[0])
	}

	if inspected.HostConfig.Memory > 0 {
		args = append(args, "--memory", strconv.FormatInt(inspected.HostConfig.Memory, 10))
	}
	if inspected.HostConfig.NanoCPUs > 0 {
		args = append(args, "--cpus", strconv.FormatFloat(float64(inspected.HostConfig.NanoCPUs)/1e9, 'f', -1, 64))
	}
	if inspected.HostConfig.PidsLimit > 0 {
		args = append(args, "--pids-limit", strconv.FormatInt(inspected.HostConfig.PidsLimit, 10))
	}
	for _, capability := range inspected.HostConfig.CapAdd {
		args = append(args, "--cap-add", capability)
	}
	for _, capability := range inspected.HostConfig.CapDrop {
		args = append(args, "--cap-drop", capability)
	}
	if inspected.HostConfig.ReadonlyRootfs {
		args = append(args, "--read-only")
	}
	if inspected.HostConfig.Privileged {
		args = append(args, "--privileged")
	}
	for _, option := range inspected.HostConfig.SecurityOpt {
		args = append(args, "--security-opt", option)
	}
	if inspected.HostConfig.ShmSize > 0 {
		args = append(args, "--shm-size", strconv.FormatInt(inspected.HostConfig.ShmSize, 10))
	}
	for target, options := range inspected.HostConfig.Tmpfs {
		value := target
		if options != "" { value += ":" + options }
		args = append(args, "--tmpfs", value)
	}
	for _, device := range inspected.HostConfig.Devices {
		value := device.PathOnHost
		if device.PathInContainer != "" { value += ":" + device.PathInContainer }
		if device.CgroupPermissions != "" { value += ":" + device.CgroupPermissions }
		args = append(args, "--device", value)
	}
	if inspected.HostConfig.LogConfig.Type != "" && inspected.HostConfig.LogConfig.Type != "json-file" {
		args = append(args, "--log-driver", inspected.HostConfig.LogConfig.Type)
	}
	for key, value := range inspected.HostConfig.LogConfig.Config {
		args = append(args, "--log-opt", key+"="+value)
	}
	for _, limit := range inspected.HostConfig.Ulimits {
		if limit.Name == "" { continue }
		args = append(args, "--ulimit", fmt.Sprintf("%s=%d:%d", limit.Name, limit.Soft, limit.Hard))
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
	if len(inspected.Config.Cmd) > 0 {
		args = append(args, inspected.Config.Cmd...)
	}
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
		output, err := runDocker("inspect", "--format", "{{if .State.Health}}{{.State.Health.Status}}{{else}}missing-healthcheck{{end}}", name)
		if err == nil {
			status := strings.TrimSpace(output)
			if status == "healthy" {
				m.updateProgress("verifying", "Final checks", "Final checks", 99)
				return nil
			}
			if status == "missing-healthcheck" {
				return errors.New("replacement service does not define a Docker health check")
			}
			if status == "unhealthy" {
				return errors.New("replacement service became unhealthy")
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

func runDocker(args ...string) (string, error) {
	command := exec.Command("docker", args...)
	output, err := command.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("docker %s: %w: %s", strings.Join(redactDockerArgs(args), " "), err, strings.TrimSpace(string(output)))
	}
	return string(output), nil
}

func redactDockerArgs(args []string) []string {
	out := append([]string(nil), args...)
	for i := 0; i < len(out); i++ {
		if out[i] == "--env" && i+1 < len(out) {
			if key, _, found := strings.Cut(out[i+1], "="); found {
				out[i+1] = key + "=[masked]"
			} else {
				out[i+1] = "[masked]"
			}
			i++
		}
	}
	return out
}

func (m *manager) fail(version string, started time.Time, publicMessage string, err error) {
	finished := time.Now().UTC()
	m.finishUpdate("failed", "Update stopped", publicMessage, m.snapshot().Progress, finished)
	log.Printf("update v%s failed: %v", version, err)
}

func main() {
	if len(os.Args) != 3 || os.Args[1] != "--oneshot" {
		log.Fatal("usage: gotify-mu-updater --oneshot <x.y.z>")
	}

	version := strings.TrimSpace(strings.TrimPrefix(os.Args[2], "v"))
	if !versionPattern.MatchString(version) {
		log.Fatal("version must be semantic x.y.z")
	}

	manager := newManager()
	started := time.Now().UTC()
	manager.beginUpdate(version, started)
	manager.performInstall(version, started)

	status := manager.snapshot()
	if status.State != "completed" {
		os.Exit(1)
	}
}
