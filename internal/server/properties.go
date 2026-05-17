package server

import (
	"fmt"
	"path/filepath"
)

func UpdateServerProperties(serverDir string, port int) error {
	path := filepath.Join(serverDir, "server.properties")
	props, err := parsePropertiesFile(path)
	if err != nil {
		return err
	}
	props.SetMany(map[string]string{
		"server-port": fmt.Sprintf("%d", port),
	})
	return props.Write(path)
}
