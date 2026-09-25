package main

import (
	"slices"
	"testing"
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

	args := createArgs("gotify-mu", "gotify-mu:release-0.2.1", inspected)

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
	if !slices.Equal(args[len(args)-1:], []string{"gotify-mu:release-0.2.1"}) {
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

func containsAdjacent(values []string, first string, rest ...string) bool {
	needle := append([]string{first}, rest...)
	for i := 0; i+len(needle) <= len(values); i++ {
		if slices.Equal(values[i:i+len(needle)], needle) {
			return true
		}
	}
	return false
}
