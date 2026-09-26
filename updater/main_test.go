package main

import (
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


func TestParseExpectedSHA256(t *testing.T) {
	content := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa  gotify-mu-v0.5.0-source.zip\n"
	got, err := parseExpectedSHA256(content, "gotify-mu-v0.5.0-source.zip")
	if err != nil { t.Fatal(err) }
	if got != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("unexpected checksum %q", got)
	}
}

func TestVerifyFileSHA256(t *testing.T) {
	path := filepath.Join(t.TempDir(), "source.zip")
	if err := os.WriteFile(path, []byte("hello"), 0o600); err != nil { t.Fatal(err) }
	const expected = "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if err := verifyFileSHA256(path, expected); err != nil { t.Fatal(err) }
	if err := verifyFileSHA256(path, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"); err == nil {
		t.Fatal("expected checksum mismatch")
	}
}

func TestRedactDockerArgs(t *testing.T) {
	args := []string{"create", "--env", "PASSWORD=secret", "--env", "SAFE=value", "--name", "gotify-mu"}
	got := redactDockerArgs(args)
	if slices.Contains(got, "PASSWORD=secret") || slices.Contains(got, "SAFE=value") {
		t.Fatalf("environment values leaked: %v", got)
	}
	if !slices.Contains(got, "PASSWORD=[masked]") || !slices.Contains(got, "SAFE=[masked]") {
		t.Fatalf("expected masked environment values: %v", got)
	}
}
