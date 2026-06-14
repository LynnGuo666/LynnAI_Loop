package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
)

func AuthMiddleware(token string) func(http.Handler) http.Handler {
	expectedHash := sha256Hash(token)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			providedToken := tokenFromRequest(r)
			if providedToken == "" {
				http.Error(w, `{"error":{"type":"auth_error","message":"missing admin token"}}`, http.StatusUnauthorized)
				return
			}
			if sha256Hash(providedToken) != expectedHash {
				http.Error(w, `{"error":{"type":"auth_error","message":"invalid token"}}`, http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func tokenFromRequest(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	}
	if key := strings.TrimSpace(r.Header.Get("x-api-key")); key != "" {
		return key
	}
	return strings.TrimSpace(r.Header.Get("x-goog-api-key"))
}

func sha256Hash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
