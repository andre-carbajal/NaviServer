package strategy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBedrockRunnerBuildsLinuxCommand(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "bedrock_server")
	if err := os.WriteFile(binary, []byte("binary"), 0755); err != nil {
		t.Fatal(err)
	}

	runner := &BedrockRunner{GOOS: "linux"}
	cmd, err := runner.BuildCommand("ignored-java", dir, 4096, "ServerTest=true")
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Path != binary {
		t.Fatalf("command path = %s, want %s", cmd.Path, binary)
	}
	if len(cmd.Args) != 2 || cmd.Args[1] != "ServerTest=true" {
		t.Fatalf("unexpected command args: %v", cmd.Args)
	}
	if !containsEnvironment(cmd.Env, "LD_LIBRARY_PATH=.") {
		t.Fatalf("LD_LIBRARY_PATH was not configured: %v", cmd.Env)
	}
}

func TestBedrockRunnerBuildsWindowsCommand(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "bedrock_server.exe")
	if err := os.WriteFile(binary, []byte("binary"), 0644); err != nil {
		t.Fatal(err)
	}

	runner := &BedrockRunner{GOOS: "windows"}
	cmd, err := runner.BuildCommand("", dir, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Path != binary {
		t.Fatalf("command path = %s, want %s", cmd.Path, binary)
	}
	if containsEnvironment(cmd.Env, "LD_LIBRARY_PATH=.") {
		t.Fatal("Windows command unexpectedly contains LD_LIBRARY_PATH")
	}
}

func TestBedrockRunnerRejectsUnsupportedPlatform(t *testing.T) {
	runner := &BedrockRunner{GOOS: "darwin"}
	_, err := runner.BuildCommand("", t.TempDir(), 0, "")
	if err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("expected unsupported platform error, got %v", err)
	}
}

func containsEnvironment(environment []string, expected string) bool {
	for _, entry := range environment {
		if entry == expected {
			return true
		}
	}
	return false
}
