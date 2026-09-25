package main

import (
	"archive/zip"
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

type updateStatus struct {
	Ready      bool       `json:"ready"`
	State      string     `json:"state"`
	Version    string     `json:"version,omitempty"`
	Message    string     `json:"message,omitempty"`
	StartedAt  *time.Time `json:"startedAt,omitempty"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
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

func (m *manager) setStatus(state, version, message string, startedAt, finishedAt *time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status = updateStatus{
		Ready:      m.token != "",
		State:      state,
		Version:    version,
		Message:    message,
		StartedAt:  startedAt,
		FinishedAt: finishedAt,
	}
}

func (m *manager) snapshot() updateStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status
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
	case "downloading", "building", "replacing", "verifying":
		writeJSON(w, http.StatusConflict, current)
		return
	}

	started := time.Now().UTC()
	m.setStatus("downloading", request.Version, "Downloading release source", &started, nil)
	go m.performInstall(request.Version, started)

	writeJSON(w, http.StatusAccepted, m.snapshot())
}

func (m *manager) performInstall(version string, started time.Time) {
	tempDir, err := os.MkdirTemp("", "gotify-mu-update-*")
	if err != nil {
		m.fail(version, started, "Could not create update workspace: "+err.Error())
		return
	}
	defer os.RemoveAll(tempDir)

	commit, err := m.resolveCommit(version)
	if err != nil {
		m.fail(version, started, err.Error())
		return
	}

	archivePath := filepath.Join(tempDir, "source.zip")
	sourceURL := fmt.Sprintf("https://github.com/%s/archive/refs/tags/v%s.zip", m.repository, version)
	if err := downloadFile(sourceURL, archivePath); err != nil {
		m.fail(version, started, "Could not download release source: "+err.Error())
		return
	}

	sourceDir := filepath.Join(tempDir, "source")
	if err := unzip(archivePath, sourceDir); err != nil {
		m.fail(version, started, "Could not unpack release source: "+err.Error())
		return
	}
	root, err := singleDirectory(sourceDir)
	if err != nil {
		m.fail(version, started, "Could not locate release source root: "+err.Error())
		return
	}

	m.setStatus("building", version, "Building release container", &started, nil)
	image := "gotify-mu:release-" + version
	buildDate := time.Now().UTC().Format(time.RFC3339)
	if _, err := runDocker(
		"build",
		"--pull",
		"--build-arg", "BUILD_JS=1",
		"--build-arg", "GO_VERSION=1.26.0",
		"--build-arg", "GOTIFY_MU_VERSION="+version,
		"--build-arg", "GOTIFY_MU_COMMIT="+commit,
		"--build-arg", "GOTIFY_MU_BUILD_DATE="+buildDate,
		"-f", filepath.Join(root, "docker", "Dockerfile"),
		"-t", image,
		root,
	); err != nil {
		m.fail(version, started, "Container build failed: "+err.Error())
		return
	}

	m.setStatus("replacing", version, "Replacing Gotify MU container", &started, nil)
	if err := m.replaceContainer(image, version, started); err != nil {
		return
	}

	finished := time.Now().UTC()
	m.setStatus("completed", version, "Update installed successfully", &started, &finished)
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
	inspection, err := inspectContainer(m.target)
	if err != nil {
		m.fail(version, started, "Could not inspect current container: "+err.Error())
		return err
	}

	rollback := fmt.Sprintf("%s-rollback-%d", m.target, time.Now().Unix())

	if _, err := runDocker("stop", "-t", "20", m.target); err != nil {
		m.fail(version, started, "Could not stop current container: "+err.Error())
		return err
	}
	if _, err := runDocker("rename", m.target, rollback); err != nil {
		_, _ = runDocker("start", m.target)
		m.fail(version, started, "Could not preserve current container: "+err.Error())
		return err
	}

	restore := func(cause error) error {
		_, _ = runDocker("rm", "-f", m.target)
		_, renameErr := runDocker("rename", rollback, m.target)
		_, startErr := runDocker("start", m.target)
		finished := time.Now().UTC()
		message := "Update failed and previous container was restored: " + cause.Error()
		if renameErr != nil || startErr != nil {
			message = "Update failed and automatic rollback also failed; manual recovery is required"
		}
		m.setStatus("rolled_back", version, message, &started, &finished)
		return cause
	}

	args := createArgs(m.target, image, inspection)
	if _, err := runDocker(args...); err != nil {
		return restore(fmt.Errorf("could not create replacement container: %w", err))
	}

	for network := range inspection.NetworkSettings.Networks {
		if network == inspection.HostConfig.NetworkMode || network == "bridge" || network == "default" {
			continue
		}
		_, _ = runDocker("network", "connect", network, m.target)
	}

	if _, err := runDocker("start", m.target); err != nil {
		return restore(fmt.Errorf("could not start replacement container: %w", err))
	}

	m.setStatus("verifying", version, "Waiting for replacement container health check", &started, nil)
	if err := waitForHealthy(m.target, 120*time.Second); err != nil {
		return restore(err)
	}

	if _, err := runDocker("rm", "-f", rollback); err != nil {
		log.Printf("warning: could not remove rollback container %s: %v", rollback, err)
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

func waitForHealthy(name string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		output, err := runDocker("inspect", "--format", "{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}", name)
		if err == nil {
			status := strings.TrimSpace(output)
			if status == "healthy" || status == "running" {
				return nil
			}
			if status == "unhealthy" || status == "exited" || status == "dead" {
				return fmt.Errorf("replacement container entered state %s", status)
			}
		}
		time.Sleep(2 * time.Second)
	}
	return errors.New("replacement container did not become healthy before timeout")
}

func runDocker(args ...string) (string, error) {
	command := exec.Command("docker", args...)
	output, err := command.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("docker %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return string(output), nil
}

func downloadFile(url, path string) error {
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
	_, err = io.Copy(file, response.Body)
	return err
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

func (m *manager) fail(version string, started time.Time, message string) {
	finished := time.Now().UTC()
	m.setStatus("failed", version, message, &started, &finished)
	log.Print(message)
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
