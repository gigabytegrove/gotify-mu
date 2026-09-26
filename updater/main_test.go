package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func TestCreateArgsPreservesRuntimeConfiguration(t *testing.T) {
	inspected := &inspectedContainer{}
	inspected.Config.Env = []string{"GOTIFY_DEFAULTUSER_NAME=admin", "GOTIFY_MU_UPDATER_TOKEN=secret"}
	inspected.Config.User = "1000:1000"
	inspected.Config.WorkingDir = "/app"
	inspected.HostConfig.RestartPolicy = restartPolicy{Name: "unless-stopped"}
	inspected.HostConfig.NetworkMode = "gotify-mu-system"
	inspected.HostConfig.Binds = []string{"/opt/gotify-mu-data:/app/data"}
	inspected.HostConfig.PortBindings = map[string][]portBinding{
		"80/tcp": {{HostIP: "0.0.0.0", HostPort: "8799"}},
	}
	inspected.HostConfig.ExtraHosts = []string{"example.local:192.0.2.10"}
	inspected.HostConfig.DNS = []string{"1.1.1.1"}
	inspected.HostConfig.DNSSearch = []string{"example.local"}
	inspected.HostConfig.Memory = 536870912
	inspected.HostConfig.NanoCpus = 1500000000
	inspected.HostConfig.CPUShares = 512
	inspected.HostConfig.CPUSetCPUs = "0-1"
	inspected.HostConfig.CapAdd = []string{"NET_ADMIN"}
	inspected.HostConfig.CapDrop = []string{"MKNOD"}
	inspected.HostConfig.SecurityOpt = []string{"no-new-privileges:true"}
	inspected.HostConfig.ReadonlyRootfs = true
	inspected.HostConfig.Tmpfs = map[string]string{"/tmp": "rw,noexec"}
	inspected.HostConfig.LogConfig = logConfig{Type: "local", Config: map[string]string{"max-size": "10m"}}
	inspected.HostConfig.Devices = []deviceMapping{{PathOnHost:"/dev/null",PathInContainer:"/dev/test",CgroupPermissions:"r"}}
	inspected.HostConfig.PidsLimit = func() *int64 { value := int64(256); return &value }()
	inspected.HostConfig.ShmSize = 67108864

	args := createArgs("gotify-mu", "gotify-mu:release-0.2.2", inspected)

	expected := [][]string{
		{"create", "--name", "gotify-mu"},
		{"--restart", "unless-stopped"},
		{"--env", "GOTIFY_DEFAULTUSER_NAME=admin"},
		{"--env", "GOTIFY_MU_UPDATER_TOKEN=secret"},
		{"--volume", "/opt/gotify-mu-data:/app/data"},
		{"--publish", "8799:80/tcp"},
		{"--network", "gotify-mu-system"},
		{"--add-host", "example.local:192.0.2.10"},
		{"--dns", "1.1.1.1"},
		{"--dns-search", "example.local"},
		{"--user", "1000:1000"},
		{"--workdir", "/app"},
		{"--memory", "536870912"},
		{"--cpus", "1.500"},
		{"--cpu-shares", "512"},
		{"--cpuset-cpus", "0-1"},
		{"--cap-add", "NET_ADMIN"},
		{"--cap-drop", "MKNOD"},
		{"--security-opt", "no-new-privileges:true"},
		{"--tmpfs", "/tmp:rw,noexec"},
		{"--log-driver", "local"},
		{"--log-opt", "max-size=10m"},
		{"--device", "/dev/null:/dev/test:r"},
		{"--pids-limit", "256"},
		{"--shm-size", "67108864"},
	}

	for _, pair := range expected {
		if !containsAdjacent(args, pair[0], pair[1:]...) {
			t.Fatalf("expected %v in args: %v", pair, args)
		}
	}
	if !slices.Equal(args[len(args)-1:], []string{"gotify-mu:release-0.2.2"}) {
		t.Fatalf("expected image at end of args: %v", args)
	}
}

func TestCreateArgsFallsBackToMountInspection(t *testing.T) {
	inspected := &inspectedContainer{}
	inspected.Mounts = []mountInfo{
		{Type: "bind", Source: "/srv/data", Destination: "/app/data", RW: true},
		{Type: "volume", Name: "plugins", Destination: "/app/plugins", RW: false},
	}

	args := createArgs("gotify-mu", "image", inspected)
	if !containsAdjacent(args, "--volume", "/srv/data:/app/data") {
		t.Fatalf("missing bind mount: %v", args)
	}
	if !containsAdjacent(args, "--volume", "plugins:/app/plugins:ro") {
		t.Fatalf("missing volume mount: %v", args)
	}
}


func TestUpdateProgressIsMonotonicAndTracksActivity(t *testing.T) {
	manager := &manager{token: "token"}
	started := time.Now().UTC()
	manager.beginUpdate("0.2.2", started)
	manager.updateProgress("building", "Installing update", "Installing update", 60)
	manager.updateProgress("building", "Installing update", "Installing update", 40)
	manager.updateProgress("verifying", "Checking updated version", "Checking updated version", 96)

	status := manager.snapshot()
	if status.Progress != 96 {
		t.Fatalf("expected progress 96, got %d", status.Progress)
	}
	if status.Step != "Checking updated version" {
		t.Fatalf("unexpected step %q", status.Step)
	}
	if len(status.Activity) < 3 {
		t.Fatalf("expected activity history, got %#v", status.Activity)
	}

	status.Activity[0].Message = "changed outside manager"
	fresh := manager.snapshot()
	if fresh.Activity[0].Message == "changed outside manager" {
		t.Fatal("snapshot activity must not share mutable backing storage")
	}
}

func TestBuildProgressUsesUserFacingStages(t *testing.T) {
	manager := &manager{token: "token"}
	started := time.Now().UTC()
	manager.beginUpdate("0.2.2", started)

	manager.handleBuildProgress("#7 [js-builder 4/4] RUN make build-js")
	status := manager.snapshot()
	if status.Step != "Preparing interface" || status.Progress < 47 {
		t.Fatalf("unexpected web build status: %#v", status)
	}

	manager.handleBuildProgress("#11 [builder 6/6] RUN make")
	status = manager.snapshot()
	if status.Step != "Preparing application" || status.Progress < 62 {
		t.Fatalf("unexpected server build status: %#v", status)
	}

	manager.handleBuildProgress("#18 exporting to image")
	status = manager.snapshot()
	if status.Step != "Finalizing update files" || status.Progress < 78 {
		t.Fatalf("unexpected final build status: %#v", status)
	}
}

func containsAdjacent(values []string, first string, rest ...string) bool {
	needle := append([]string{first}, rest...)
	for i := 0; i+len(needle) <= len(values); i++ {
		if slices.Equal(values[i:i+len(needle)], needle) {
			return true
		}
	}
	return false
}


func TestRedactDockerArgsMasksEnvironmentValues(t *testing.T) {
	args := []string{"create", "--env", "TOKEN=secret-value", "-e", "PASSWORD=hunter2", "--name", "gotify-mu"}
	got := redactDockerArgs(args)
	if got[2] != "TOKEN=[masked]" || got[4] != "PASSWORD=[masked]" {
		t.Fatalf("environment values were not redacted: %v", got)
	}
	if args[2] != "TOKEN=secret-value" {
		t.Fatal("redaction mutated original arguments")
	}
}

func TestVerifyChecksum(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "source.zip")
	if err := os.WriteFile(archive, []byte("release bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte("release bytes"))
	checksums := filepath.Join(dir, "SHA256SUMS")
	content := hex.EncodeToString(sum[:]) + "  source.zip\n"
	if err := os.WriteFile(checksums, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := verifyChecksum(archive, checksums); err != nil {
		t.Fatalf("valid checksum rejected: %v", err)
	}
	if err := os.WriteFile(archive, []byte("tampered"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := verifyChecksum(archive, checksums); err == nil {
		t.Fatal("tampered archive should fail checksum verification")
	}
}

func TestVerifyChecksumSupportsHistoricalSingleEntry(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "source.zip")
	payload := []byte("release bytes")
	if err := os.WriteFile(archive, payload, 0o600); err != nil { t.Fatal(err) }
	sum := sha256.Sum256(payload)
	checksums := filepath.Join(dir, "SHA256SUMS")
	content := hex.EncodeToString(sum[:]) + "  /tmp/build/gotify-mu-v0.5.0-source.zip\n"
	if err := os.WriteFile(checksums, []byte(content), 0o600); err != nil { t.Fatal(err) }
	if err := verifyChecksum(archive, checksums); err != nil {
		t.Fatalf("historical single-entry checksum rejected: %v", err)
	}
}
