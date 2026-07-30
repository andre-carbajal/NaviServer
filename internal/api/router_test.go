package api

import (
	"net/http/httptest"
	"testing"
)

func TestWantsHTMLDocument(t *testing.T) {
	tests := []struct {
		name   string
		accept string
		want   bool
	}{
		{name: "browser navigation", accept: "text/html,application/xhtml+xml", want: true},
		{name: "json api request", accept: "application/json", want: false},
		{name: "missing accept header", accept: "", want: false},
		{name: "mixed accept header", accept: "application/json, text/html;q=0.9", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/servers/server-id", nil)
			req.Header.Set("Accept", tt.accept)
			if got := wantsHTMLDocument(req); got != tt.want {
				t.Fatalf("wantsHTMLDocument() = %v, want %v", got, tt.want)
			}
		})
	}
}
