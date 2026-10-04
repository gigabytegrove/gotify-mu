package main

import (
	"github.com/gotify/plugin-api"
)

// GetGotifyPluginInfo returns the legacy plugin ABI information
func GetGotifyPluginInfo() plugin.Info {
	return plugin.Info{
		Name:       "minimal plugin",
		ModulePath: "github.com/gigabytegrove/monita/example/minimal",
	}
}

// Plugin is plugin instance
type Plugin struct{}

// Enable implements plugin.Plugin
func (c *Plugin) Enable() error {
	return nil
}

// Disable implements plugin.Plugin
func (c *Plugin) Disable() error {
	return nil
}

// NewGotifyPluginInstance is the legacy ABI entrypoint for creating a plugin instance.
func NewGotifyPluginInstance(ctx plugin.UserContext) plugin.Plugin {
	return &Plugin{}
}

func main() {
	panic("this should be built as go plugin")
}
