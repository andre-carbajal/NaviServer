package loader

import (
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
)

func installerCommand(javaPath, destDir string) (*exec.Cmd, error) {
	javaPath = strings.TrimSpace(javaPath)
	if javaPath == "" || !filepath.IsAbs(javaPath) {
		return nil, fmt.Errorf("an absolute Java path is required to run the installer")
	}

	cmd := exec.Command(javaPath, "-jar", "installer.jar", "--installServer")
	cmd.Dir = destDir
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd, nil
}
