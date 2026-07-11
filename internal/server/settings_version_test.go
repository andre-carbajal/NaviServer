package server

import "testing"

func TestIsFutureVersion(t *testing.T) {
	tests := []struct {
		name       string
		current    string
		target     string
		isFuture   bool
		comparable bool
	}{
		{
			name:       "higher patch is future",
			current:    "1.21.1",
			target:     "1.21.2",
			isFuture:   true,
			comparable: true,
		},
		{
			name:       "same version is not future",
			current:    "1.21.2",
			target:     "1.21.2",
			isFuture:   false,
			comparable: true,
		},
		{
			name:       "lower version is not future",
			current:    "1.21.3",
			target:     "1.21.1",
			isFuture:   false,
			comparable: true,
		},
		{
			name:       "newer Bedrock preview build",
			current:    "1.26.40-preview.29",
			target:     "1.26.40-preview.30",
			isFuture:   true,
			comparable: true,
		},
		{
			name:       "Bedrock release supersedes matching preview",
			current:    "1.26.40-preview.30",
			target:     "1.26.40",
			isFuture:   true,
			comparable: true,
		},
		{
			name:       "Bedrock preview is older than matching release",
			current:    "1.26.40",
			target:     "1.26.40-preview.30",
			isFuture:   false,
			comparable: true,
		},
		{
			name:       "invalid version not comparable",
			current:    "latest",
			target:     "1.21.4",
			isFuture:   false,
			comparable: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			future, comparable := isFutureVersion(tc.target, tc.current)
			if future != tc.isFuture {
				t.Fatalf("future mismatch: got %v want %v", future, tc.isFuture)
			}
			if comparable != tc.comparable {
				t.Fatalf("comparable mismatch: got %v want %v", comparable, tc.comparable)
			}
		})
	}
}
