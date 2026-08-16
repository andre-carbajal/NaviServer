package server

import (
	"errors"
	"naviserver/internal/domain"
	"naviserver/internal/loader"
	"path/filepath"
	"testing"
)

type recordingJavaEnsurer struct {
	path    string
	version int
}

func (r *recordingJavaEnsurer) EnsureJava(version int) (string, error) {
	r.version = version
	return r.path, nil
}

type supportedVersionsLoader struct {
	versions []string
}

func (l supportedVersionsLoader) Load(loader.LoaderOptions, string, chan<- domain.ProgressEvent) (string, error) {
	return "", errors.New("not used")
}

func (l supportedVersionsLoader) GetSupportedVersions(loader.LoaderOptions) ([]string, error) {
	return l.versions, nil
}

func (l supportedVersionsLoader) GetMetadata(loader.LoaderOptions) (*loader.LoaderMetadata, error) {
	return nil, errors.New("not used")
}

func TestPrepareLoaderOptionsUsesManagedAbsoluteJavaForForge(t *testing.T) {
	managedRoot := t.TempDir()
	javaPath, err := filepath.Abs(filepath.Join(managedRoot, "java-17", "bin", "java"))
	if err != nil {
		t.Fatalf("failed to resolve managed Java path: %v", err)
	}
	java := &recordingJavaEnsurer{path: javaPath}
	manager := &Manager{Java: java}

	options, err := manager.prepareLoaderOptions(
		"forge",
		supportedVersionsLoader{versions: []string{"1.20.4"}},
		loader.LoaderOptions{JavaPath: "/untrusted/client/path"},
		"",
	)
	if err != nil {
		t.Fatalf("prepareLoaderOptions failed: %v", err)
	}
	if java.version != 17 {
		t.Fatalf("expected Java 17, got %d", java.version)
	}
	if options.MCVersion != "1.20.4" {
		t.Fatalf("expected selected Minecraft version 1.20.4, got %q", options.MCVersion)
	}
	if options.JavaPath != java.path {
		t.Fatalf("expected managed Java path %q, got %q", java.path, options.JavaPath)
	}
}

func TestPrepareLoaderOptionsRejectsForgeWithoutManagedJava(t *testing.T) {
	manager := &Manager{}
	_, err := manager.prepareLoaderOptions(
		"neoforge",
		supportedVersionsLoader{versions: []string{"1.21.1"}},
		loader.LoaderOptions{},
		"1.21.1",
	)
	if err == nil {
		t.Fatal("expected Forge-family installation to require managed Java")
	}
}
