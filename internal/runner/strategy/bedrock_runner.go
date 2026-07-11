package strategy

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type BedrockRunner struct {
	GOOS string
}

func (r *BedrockRunner) BuildCommand(_ string, absServerDir string, _ int, customArgs string) (*exec.Cmd, error) {
	var executable string
	switch r.GOOS {
	case "windows":
		executable = filepath.Join(absServerDir, "bedrock_server.exe")
	case "linux":
		executable = filepath.Join(absServerDir, "bedrock_server")
	default:
		return nil, fmt.Errorf("Bedrock runner is not supported on %s", r.GOOS)
	}

	if _, err := os.Stat(executable); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("Bedrock server binary not found at %s", executable)
		}
		return nil, fmt.Errorf("error accessing %s: %w", executable, err)
	}

	args := strings.Fields(customArgs)
	cmd := exec.Command(executable, args...)
	cmd.Dir = absServerDir
	if r.GOOS == "linux" {
		cmd.Env = appendEnvironment(os.Environ(), "LD_LIBRARY_PATH", ".")
	}
	return cmd, nil
}

func appendEnvironment(environment []string, key, value string) []string {
	prefix := key + "="
	result := make([]string, 0, len(environment)+1)
	for _, entry := range environment {
		if !strings.HasPrefix(entry, prefix) {
			result = append(result, entry)
		}
	}
	return append(result, prefix+value)
}
