package handlers

import (
	"net/http"

	"naviger/internal/domain"
	"naviger/internal/storage"
)

func CheckPermission(store *storage.GormStore, r *http.Request, serverID string, check func(*domain.Permission) bool) bool {
	userCtx := r.Context().Value(domain.UserContextKey)
	if userCtx == nil {
		return false
	}
	claims, ok := userCtx.(map[string]string)
	if !ok {
		return false
	}

	role := claims["role"]
	if role == "admin" {
		return true
	}

	userID := claims["id"]
	perms, err := store.GetPermissions(userID)
	if err != nil {
		return false
	}

	for _, p := range perms {
		if p.ServerID == serverID {
			return check(&p)
		}
	}
	return false
}

func GetUserClaims(r *http.Request) (map[string]string, bool) {
	userCtx := r.Context().Value(domain.UserContextKey)
	if userCtx == nil {
		return nil, false
	}
	claims, ok := userCtx.(map[string]string)
	return claims, ok
}
