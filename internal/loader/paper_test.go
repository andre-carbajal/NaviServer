package loader

import "testing"

func TestStablePaperVersionsIncludesNewCalendarVersionsFirst(t *testing.T) {
	versions := stablePaperVersions(map[string][]string{
		"26.1": {"26.1.2", "26.1.1"},
		"1.21": {"1.21.11", "1.21.11-rc1", "1.21.10"},
	})

	expected := []string{"26.1.2", "26.1.1", "1.21.11", "1.21.10"}
	if len(versions) != len(expected) {
		t.Fatalf("expected %d versions, got %d: %#v", len(expected), len(versions), versions)
	}
	for i, want := range expected {
		if versions[i] != want {
			t.Fatalf("version %d mismatch: got %s want %s (all: %#v)", i, versions[i], want, versions)
		}
	}
}
