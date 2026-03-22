package strategy

import (
	"os/exec"
	"strings"
)

type ServerRunner interface {
	BuildCommand(javaPath string, serverDir string, ram int, customArgs string) (*exec.Cmd, error)
}

func FilterJVMArgs(customArgs string) []string {
	if customArgs == "" {
		return nil
	}

	fields := strings.Fields(customArgs)
	blocklist := []string{
		"-Xbootclasspath",
		"-javaagent",
		"-agentpath",
		"-agentlib",
		"-Djava.library.path",
		"-XX:OnOutOfMemoryError",
		"-XX:OnError",
	}

	var filtered []string
	for _, arg := range fields {
		lowerArg := strings.ToLower(arg)
		blocked := false
		for _, b := range blocklist {
			if strings.HasPrefix(lowerArg, strings.ToLower(b)) {
				blocked = true
				break
			}
		}
		if !blocked {
			filtered = append(filtered, arg)
		}
	}
	return filtered
}
