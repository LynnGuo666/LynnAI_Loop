package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthMiddlewareAcceptsBearerToken(t *testing.T) {
	handler := AuthMiddleware("secret")(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/v1/messages", nil)
	req.Header.Set("Authorization", "Bearer secret")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
}

func TestAuthMiddlewareAcceptsXAPIKey(t *testing.T) {
	handler := AuthMiddleware("secret")(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/v1/messages", nil)
	req.Header.Set("x-api-key", "secret")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
}

func TestAuthMiddlewareRejectsMissingToken(t *testing.T) {
	handler := AuthMiddleware("secret")(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/v1/messages", nil)
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusUnauthorized)
	}
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}
