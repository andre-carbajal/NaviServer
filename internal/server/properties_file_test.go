package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPropertiesFileSetManyPreservesCommentsAndUpdatesValues(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "server.properties")

	initial := strings.Join([]string{
		"# Minecraft Server Properties",
		"motd=A Minecraft Server",
		"pvp=true",
		"",
		"max-players=20",
	}, "\n")

	if err := os.WriteFile(path, []byte(initial), 0o644); err != nil {
		t.Fatalf("failed to seed properties file: %v", err)
	}

	props, err := parsePropertiesFile(path)
	if err != nil {
		t.Fatalf("parsePropertiesFile failed: %v", err)
	}

	props.SetMany(map[string]string{
		"motd":                "Hosted by NaviServer",
		"pvp":                 "false",
		"simulation-distance": "12",
	})

	if err := props.Write(path); err != nil {
		t.Fatalf("failed to write properties file: %v", err)
	}

	updated, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read properties file: %v", err)
	}

	content := string(updated)
	if !strings.Contains(content, "# Minecraft Server Properties") {
		t.Fatalf("expected comment to be preserved, got: %s", content)
	}
	if !strings.Contains(content, "motd=Hosted by NaviServer") {
		t.Fatalf("expected motd update, got: %s", content)
	}
	if !strings.Contains(content, "pvp=false") {
		t.Fatalf("expected pvp update, got: %s", content)
	}
	if !strings.Contains(content, "simulation-distance=12") {
		t.Fatalf("expected new key append, got: %s", content)
	}
}
