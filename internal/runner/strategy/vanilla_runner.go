package strategy

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type VanillaRunner struct {
	JarName string
}

func (r *VanillaRunner) BuildCommand(javaPath string, absServerDir string, ram int, customArgs string) (*exec.Cmd, error) {
	jarPath := r.JarName
	if jarPath == "" {
		jarPath = "server.jar"
	}

	jarFull := filepath.Join(absServerDir, jarPath)
	if _, err := os.Stat(jarFull); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("server jar not found at %s", jarFull)
		}
		return nil, fmt.Errorf("error accessing %s: %w", jarFull, err)
	}

	args := []string{
		fmt.Sprintf("-Xmx%dM", ram),
		"-Xms512M",
	}

	if customArgs != "" {
		args = append(args, FilterJVMArgs(customArgs)...)
	}

	args = append(args, "-jar", jarPath, "nogui")

	cmd := exec.Command(javaPath, args...)
	cmd.Dir = absServerDir
	return cmd, nil
}
