package server

import (
	"fmt"
	"path/filepath"
)

func UpdateServerProperties(serverDir string, port int) error {
	return UpdateServerPropertiesForLoader(serverDir, port, "")
}

func UpdateServerPropertiesForLoader(serverDir string, port int, loaderType string) error {
	path := filepath.Join(serverDir, "server.properties")
	props, err := parsePropertiesFile(path)
	if err != nil {
		return err
	}
	values := map[string]string{
		"server-port": fmt.Sprintf("%d", port),
	}
	if loaderType == "bedrock" {
		values["server-portv6"] = fmt.Sprintf("%d", port)
		values["enable-lan-visibility"] = "false"
	}
	props.SetMany(values)
	return props.Write(path)
}
