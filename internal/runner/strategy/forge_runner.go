package strategy

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

type ForgeRunner struct{}

func (r *ForgeRunner) BuildCommand(javaPath string, absServerDir string, ram int, customArgs string) (*exec.Cmd, error) {
	librariesDir := filepath.Join(absServerDir, "libraries")
	var argsFile string
	targetFile := "unix_args.txt"
	if runtime.GOOS == "windows" {
		targetFile = "win_args.txt"
	}

	if _, err := os.Stat(librariesDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("libraries directory not found in %s (required for Forge/NeoForge)", librariesDir)
	}

	err := filepath.WalkDir(librariesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && d.Name() == targetFile {
			argsFile = path
			return io.EOF
		}
		return nil
	})

	if argsFile == "" {
		if err != io.EOF {
			return nil, fmt.Errorf("args file %s not found in libraries", targetFile)
		}
	}

	args := []string{
		fmt.Sprintf("-Xmx%dM", ram),
		"-Xms512M",
	}

	userJvmArgs := filepath.Join(absServerDir, "user_jvm_args.txt")
	if _, err := os.Stat(userJvmArgs); err == nil {
		args = append(args, fmt.Sprintf("@%s", userJvmArgs))
	}

	if customArgs != "" {
		args = append(args, FilterJVMArgs(customArgs)...)
	}

	args = append(args, fmt.Sprintf("@%s", argsFile))
	args = append(args, "nogui")

	cmd := exec.Command(javaPath, args...)
	cmd.Dir = absServerDir
	return cmd, nil
}
