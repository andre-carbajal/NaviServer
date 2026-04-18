package main

import (
	"naviserver/internal/cli/cmd"
	"naviserver/internal/config"
)

func main() {
	port := config.GetPort()
	cmd.Execute(port)
}
