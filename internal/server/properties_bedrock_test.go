package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateServerPropertiesForBedrockConfiguresManagedPorts(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "server.properties")
	if err := os.WriteFile(path, []byte("server-name=Test\nserver-port=19132\nenable-lan-visibility=true\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := UpdateServerPropertiesForLoader(dir, 25570, "bedrock"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	for _, expected := range []string{"server-port=25570", "server-portv6=25570", "enable-lan-visibility=false"} {
		if !strings.Contains(content, expected) {
			t.Fatalf("properties missing %q:\n%s", expected, content)
		}
	}
}
