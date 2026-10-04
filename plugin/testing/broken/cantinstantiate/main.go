package main

import (
	"errors"

	"github.com/gotify/plugin-api"
)

// GetGotifyPluginInfo returns the legacy plugin ABI information
func GetGotifyPluginInfo() plugin.Info {
	return plugin.Info{
		ModulePath: "github.com/gigabytegrove/monita/plugin/testing/broken/noinstance",
	}
}

// Plugin is plugin instance
type Plugin struct{}

// Enable implements plugin.Plugin
func (c *Plugin) Enable() error {
	return errors.New("cannot instantiate")
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
	panic("this is a broken plugin for testing purposes")
}
