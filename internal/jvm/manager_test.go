package jvm

import (
	"os"
	"runtime"
	"testing"
)

func TestRestrictJavaBinaryUsesOwnerOnlyPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix file modes are not available on Windows")
	}

	path := t.TempDir() + string(os.PathSeparator) + "java"
	if err := os.WriteFile(path, []byte("java"), 0755); err != nil {
		t.Fatalf("failed to create test binary: %v", err)
	}

	if err := restrictJavaBinary(path); err != nil {
		t.Fatalf("restrictJavaBinary failed: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("failed to stat test binary: %v", err)
	}
	if got := info.Mode().Perm(); got != 0700 {
		t.Fatalf("expected permissions 0700, got %04o", got)
	}
}
