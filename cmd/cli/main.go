package main

import (
	"naviserver/internal/cli/cmd"
	"naviserver/internal/config"
)

func main() {
	port := config.GetPort()
	cliToken := ""
	if configDir, err := config.ResolveConfigDir(); err == nil {
		if token, err := config.LoadOrGenerateCLIToken(configDir); err == nil {
			cliToken = token
		}
	}
	cmd.Execute(port, cliToken)
}
