package handlers

import (
	"bytes"
	"encoding/json"
	"naviserver/internal/config"
	"naviserver/internal/domain"
	"naviserver/internal/storage"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func newTestAuthHandler(t *testing.T) (*AuthHandler, *storage.GormStore) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "auth.db")
	store, err := storage.NewGormStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	handler := &AuthHandler{
		BaseHandler: &BaseHandler{
			Store: store,
			Config: &config.Config{
				JWTSecret: "test-secret",
			},
		},
	}

	return handler, store
}

func seedAuthUser(t *testing.T, store *storage.GormStore, username, password string) *domain.User {
	t.Helper()

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	user := &domain.User{
		ID:       "user-1",
		Username: username,
		Password: string(hashedPassword),
		Role:     "admin",
	}

	if err := store.CreateUser(user); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	return user
}

func TestHandleLoginSetsCookieAndReturnsUser(t *testing.T) {
	handler, store := newTestAuthHandler(t)
	user := seedAuthUser(t, store, "andre", "secret123")

	body, _ := json.Marshal(LoginRequest{
		Username: "andre",
		Password: "secret123",
	})

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.HandleLogin(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d (%s)", rec.Code, rec.Body.String())
	}

	cookies := rec.Result().Cookies()
	if len(cookies) == 0 || cookies[0].Name != "token" || cookies[0].Value == "" {
		t.Fatalf("expected token cookie to be set, got %#v", cookies)
	}

	var response LoginResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.User == nil || response.User.Username != user.Username {
		t.Fatalf("expected user %q in response, got %#v", user.Username, response.User)
	}

	if response.Token != "" {
		t.Fatalf("expected empty token field, got %q", response.Token)
	}
}

func TestHandleLoginRejectsInvalidPassword(t *testing.T) {
	handler, store := newTestAuthHandler(t)
	seedAuthUser(t, store, "andre", "secret123")

	body, _ := json.Marshal(LoginRequest{
		Username: "andre",
		Password: "wrong-password",
	})

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.HandleLogin(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}

func TestHandleSetupCreatesAdminUserAndCookie(t *testing.T) {
	handler, store := newTestAuthHandler(t)

	body, _ := json.Marshal(RegisterRequest{
		Username: "owner",
		Password: "setup-password",
	})

	req := httptest.NewRequest(http.MethodPost, "/auth/setup", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.HandleSetup(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d (%s)", rec.Code, rec.Body.String())
	}

	users, err := store.ListUsers()
	if err != nil {
		t.Fatalf("failed to list users: %v", err)
	}

	if len(users) != 1 || users[0].Username != "owner" || users[0].Role != "admin" {
		t.Fatalf("unexpected users after setup: %#v", users)
	}

	cookies := rec.Result().Cookies()
	if len(cookies) == 0 || cookies[0].Name != "token" {
		t.Fatalf("expected setup to set auth cookie, got %#v", cookies)
	}
}

func TestHandleSetupRejectsInvalidUsername(t *testing.T) {
	handler, _ := newTestAuthHandler(t)

	body, _ := json.Marshal(RegisterRequest{
		Username: "has space",
		Password: "setup-password",
	})

	req := httptest.NewRequest(http.MethodPost, "/auth/setup", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.HandleSetup(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "cannot contain spaces") {
		t.Fatalf("expected validation message, got %q", rec.Body.String())
	}
}
