package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type ContextKey string

const (
	UserContextKey ContextKey = "user"
)

func (api *Server) AuthMiddleware(next http.Handler, requiredRole string, secretKey string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Naviger-Client") == "CLI" {
			remoteAddr := r.RemoteAddr
			if strings.HasPrefix(remoteAddr, "127.0.0.1") || strings.HasPrefix(remoteAddr, "[::1]") {
				ctx := context.WithValue(r.Context(), UserContextKey, map[string]string{
					"id":   "cli",
					"role": "admin",
				})
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		var tokenString string
		cookie, err := r.Cookie("token")
		if err == nil {
			tokenString = cookie.Value
		}

		if tokenString == "" {
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" {
				parts := strings.Split(authHeader, " ")
				if len(parts) == 2 && parts[0] == "Bearer" {
					tokenString = parts[1]
				}
			}
		}

		if tokenString == "" {
			tokenString = r.URL.Query().Get("token")
		}

		if tokenString == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(secretKey), nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, "Invalid token claims", http.StatusUnauthorized)
			return
		}

		userID, ok := claims["user_id"].(string)
		if !ok {
			http.Error(w, "Invalid user ID in token", http.StatusUnauthorized)
			return
		}

		role, ok := claims["role"].(string)
		if !ok {
			http.Error(w, "Invalid role in token", http.StatusUnauthorized)
			return
		}

		if requiredRole != "" && role != requiredRole {
			if requiredRole == "admin" && role != "admin" {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
		}

		ctx := context.WithValue(r.Context(), UserContextKey, map[string]string{
			"id":   userID,
			"role": role,
		})

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
