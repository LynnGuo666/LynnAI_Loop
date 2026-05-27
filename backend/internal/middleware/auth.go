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
			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") {
				http.Error(w, `{"error":{"type":"auth_error","message":"missing or invalid authorization header"}}`, http.StatusUnauthorized)
				return
			}
			providedToken := strings.TrimPrefix(auth, "Bearer ")
			if sha256Hash(providedToken) != expectedHash {
				http.Error(w, `{"error":{"type":"auth_error","message":"invalid token"}}`, http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func sha256Hash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
