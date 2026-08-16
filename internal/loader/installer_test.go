package loader

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallerCommandRequiresAbsoluteJavaPath(t *testing.T) {
	if _, err := installerCommand("java", t.TempDir()); err == nil {
		t.Fatal("expected relative Java path to be rejected")
	}
}

func TestInstallerCommandUsesProvidedAbsoluteJavaPath(t *testing.T) {
	javaPath := filepath.Join(t.TempDir(), "java")
	cmd, err := installerCommand(javaPath, t.TempDir())
	if err != nil {
		t.Fatalf("installerCommand failed: %v", err)
	}

	if cmd.Path != javaPath {
		t.Fatalf("expected command path %q, got %q", javaPath, cmd.Path)
	}
	if got := strings.Join(cmd.Args, " "); got != javaPath+" -jar installer.jar --installServer" {
		t.Fatalf("unexpected installer arguments: %s", got)
	}
}
