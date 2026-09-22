package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestCorsMiddlewareAllowsUploadChunkHeaders(t *testing.T) {
	api := &Server{}
	handler := api.corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("preflight should not reach the wrapped handler")
	}))

	req := httptest.NewRequest(http.MethodOptions, "/uploads/upload-1/chunk", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", http.MethodPut)
	req.Header.Set("Access-Control-Request-Headers", "content-type,content-range")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("preflight status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("allow origin = %q, want %q", got, "http://localhost:5173")
	}
	if got := strings.ToLower(rr.Header().Get("Access-Control-Allow-Headers")); !strings.Contains(got, "content-range") {
		t.Fatalf("allow headers = %q, want Content-Range", got)
	}
}
