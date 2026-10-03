package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
)

var osStat = os.Stat

func loadFiles() []FutureLog {
	if configFile := firstConfigEnv("MONITA_CONFIG_FILE", "GOTIFY_CONFIG_FILE"); configFile != "" {
		log, _ := loadFile(configFile)
		return []FutureLog{log}
	}

	var logs []FutureLog
	for _, file := range getFiles() {
		log, found := loadFile(file)
		logs = append(logs, log)
		if found {
			break
		}
	}
	return logs
}

func loadFile(file string) (log FutureLog, found bool) {
	if _, err := osStat(file); err != nil {
		if os.IsNotExist(err) {
			return FutureLog{Level: zerolog.DebugLevel, Msg: fmt.Sprintf("config file %s does not exist, skipping", file)}, false
		}
		return futureFatal(fmt.Sprintf("cannot read file %s: %s", file, err)), true
	}
	if err := godotenv.Load(file); err != nil {
		return futureFatal(fmt.Sprintf("cannot load file %s: %s", file, err)), true
	}
	return FutureLog{Level: zerolog.InfoLevel, Msg: fmt.Sprintf("Loading file %s", file)}, true
}

func getFiles() []string {
	result := []string{"monita-server.env", "gotify-server.env"}
	if configHome := getConfigHome(); configHome != "" {
		result = append(result, filepath.Join(configHome, "monita/monita-server.env"), filepath.Join(configHome, "gotify/gotify-server.env"))
	}
	return append(result, "/etc/monita/server.env", "/etc/gotify/server.env")
}

func getConfigHome() string {
	if configHome := os.Getenv("XDG_CONFIG_HOME"); configHome != "" {
		return configHome
	}
	if homeDir, err := os.UserHomeDir(); err == nil {
		return filepath.Join(homeDir, ".config")
	}
	return ""
}

func firstConfigEnv(names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			return value
		}
	}
	return ""
}
