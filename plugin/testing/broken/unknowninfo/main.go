package main

// GetGotifyPluginInfo returns the legacy plugin ABI information
func GetGotifyPluginInfo() string {
	return "github.com/gigabytegrove/monita/plugin/testing/broken/unknowninfo"
}

func main() {
	panic("this is a broken plugin for testing purposes")
}
