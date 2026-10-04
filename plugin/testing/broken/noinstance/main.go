package main

import (
	"github.com/gotify/plugin-api"
)

// GetGotifyPluginInfo returns the legacy plugin ABI information
func GetGotifyPluginInfo() plugin.Info {
	return plugin.Info{
		ModulePath: "github.com/gigabytegrove/monita/plugin/testing/broken/noinstance",
	}
}

func main() {
	panic("this is a broken plugin for testing purposes")
}
