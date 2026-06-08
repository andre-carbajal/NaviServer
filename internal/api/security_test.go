package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"naviserver/internal/config"
	"naviserver/internal/domain"

	"github.com/golang-jwt/jwt/v5"
)

func TestAuthMiddlewareRejectsSpoofedCLIHeaderWithoutToken(t *testing.T) {
	api := &Server{Config: &config.Config{CLIToken: "secure-cli-token"}}
	handler := api.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not run for unauthenticated CLI request")
	}), "admin", "jwt-secret")

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("X-NaviServer-Client", "CLI")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestAuthMiddlewareRejectsCLIHeaderWithWrongToken(t *testing.T) {
	api := &Server{Config: &config.Config{CLIToken: "secure-cli-token"}}
	handler := api.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not run for invalid CLI token")
	}), "admin", "jwt-secret")

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	req.Header.Set("X-NaviServer-Client", "CLI")
	req.Header.Set("X-NaviServer-CLI-Token", "wrong-token")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestAuthMiddlewareAcceptsCLIHeaderWithCorrectToken(t *testing.T) {
	api := &Server{Config: &config.Config{CLIToken: "secure-cli-token"}}
	handler := api.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := r.Context().Value(domain.UserContextKey).(map[string]string)
		if !ok {
			t.Fatal("expected CLI user context")
		}
		if user["id"] != "cli" || user["role"] != "admin" {
			t.Fatalf("unexpected user context: %#v", user)
		}
		w.WriteHeader(http.StatusNoContent)
	}), "admin", "jwt-secret")

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	req.Header.Set("X-NaviServer-Client", "CLI")
	req.Header.Set("X-NaviServer-CLI-Token", "secure-cli-token")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rr.Code)
	}
}

func TestAuthMiddlewareStillAcceptsJWTBearerToken(t *testing.T) {
	secret := "jwt-secret"
	api := &Server{Config: &config.Config{CLIToken: "secure-cli-token"}}
	handler := api.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := r.Context().Value(domain.UserContextKey).(map[string]string)
		if !ok {
			t.Fatal("expected JWT user context")
		}
		if user["id"] != "user-1" || user["role"] != "admin" {
			t.Fatalf("unexpected user context: %#v", user)
		}
		w.WriteHeader(http.StatusNoContent)
	}), "admin", secret)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": "user-1",
		"role":    "admin",
	})
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rr.Code)
	}
}

func TestIsAllowedOriginRejectsLocalhostPrefixBypass(t *testing.T) {
	api := &Server{Config: &config.Config{}}
	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)

	for _, origin := range []string{
		"http://localhost.attacker.com",
		"http://127.0.0.1.attacker.com",
	} {
		if api.isAllowedOrigin(origin, req) {
			t.Fatalf("expected origin %q to be rejected", origin)
		}
	}
}

func TestIsAllowedOriginAllowsExactLocalhostAndConfiguredOrigins(t *testing.T) {
	api := &Server{Config: &config.Config{API: config.APIConfig{
		AllowedOrigins: []string{"https://admin.example.com"},
	}}}
	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)

	for _, origin := range []string{
		"http://localhost:5173",
		"http://127.0.0.1:5173",
		"http://[::1]:5173",
		"https://admin.example.com",
	} {
		if !api.isAllowedOrigin(origin, req) {
			t.Fatalf("expected origin %q to be allowed", origin)
		}
	}
}

func TestIsAllowedOriginRejectsInvalidAndMismatchedOrigins(t *testing.T) {
	api := &Server{Config: &config.Config{}}
	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)

	for _, origin := range []string{
		"://bad-origin",
		"http://evil.example.com",
	} {
		if api.isAllowedOrigin(origin, req) {
			t.Fatalf("expected origin %q to be rejected", origin)
		}
	}
}

func TestIsAllowedOriginAllowsSameRequestHostOnly(t *testing.T) {
	api := &Server{Config: &config.Config{}}
	allowedReq := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	allowedReq.Host = "example.com:23008"
	if !api.isAllowedOrigin("http://example.com", allowedReq) {
		t.Fatal("expected exact request host origin to be allowed")
	}

	rejectedReq := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	rejectedReq.Host = "example.com"
	if api.isAllowedOrigin("http://example.com.attacker.com", rejectedReq) {
		t.Fatal("expected lookalike request host origin to be rejected")
	}
}
