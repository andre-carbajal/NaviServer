package jvm

import "testing"

func TestResolveJavaVersion(t *testing.T) {
	tests := []struct {
		name       string
		mcVersion  string
		configured int
		want       int
	}{
		{name: "automatic Java 8", mcVersion: "1.17.1", want: 8},
		{name: "automatic Java 17", mcVersion: "1.20.4", want: 17},
		{name: "automatic Java 21", mcVersion: "1.20.5", want: 21},
		{name: "automatic Java 25", mcVersion: "26.1", want: 25},
		{name: "configured override", mcVersion: "1.21.1", configured: 8, want: 8},
		{name: "configured newer override", mcVersion: "1.17.1", configured: 25, want: 25},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ResolveJavaVersion(tc.mcVersion, tc.configured); got != tc.want {
				t.Fatalf("ResolveJavaVersion(%q, %d) = %d, want %d", tc.mcVersion, tc.configured, got, tc.want)
			}
		})
	}
}

func TestIsSupportedJavaVersion(t *testing.T) {
	for _, version := range []int{0, 8, 17, 21, 25} {
		if !IsSupportedJavaVersion(version) {
			t.Fatalf("expected Java %d to be supported", version)
		}
	}

	for _, version := range []int{-1, 11, 22} {
		if IsSupportedJavaVersion(version) {
			t.Fatalf("expected Java %d to be unsupported", version)
		}
	}
}
