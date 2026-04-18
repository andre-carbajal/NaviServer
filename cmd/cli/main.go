package main

import (
	"naviserver/internal/cli/cmd"
	"naviserver/internal/config"
	"os"
	"path/filepath"
)

func main() {
	port := config.GetPort()

	if userConfigDir, err := os.UserConfigDir(); err == nil {
		appName := "naviserver"
		if config.IsDev() {
			appName = "naviserver-dev"
		}

		configDir := filepath.Join(userConfigDir, appName)
		if cfg, err := config.LoadConfig(configDir); err == nil && cfg.API.Port > 0 {
			port = cfg.API.Port
		}
	}

	cmd.Execute(port)
}
