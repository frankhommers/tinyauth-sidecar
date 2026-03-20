package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"tinyauth-sidecar/internal/config"

	"github.com/gin-gonic/gin"
)

func TestSessionMiddlewareForwardsTraefikHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	verifyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Cookie"); got != "session=abc" {
			t.Fatalf("expected Cookie header session=abc, got %q", got)
		}
		if got := r.Header.Get("X-Forwarded-Host"); got != "auth.hommers.nl" {
			t.Fatalf("expected X-Forwarded-Host auth.hommers.nl, got %q", got)
		}
		if got := r.Header.Get("X-Forwarded-Uri"); got != "/manage/api/auth/check?foo=bar" {
			t.Fatalf("expected X-Forwarded-Uri /manage/api/auth/check?foo=bar, got %q", got)
		}
		if got := r.Header.Get("X-Forwarded-Method"); got != http.MethodGet {
			t.Fatalf("expected X-Forwarded-Method GET, got %q", got)
		}
		if got := r.Header.Get("X-Forwarded-Proto"); got != "https" {
			t.Fatalf("expected X-Forwarded-Proto https, got %q", got)
		}
		if got := r.Header.Get("X-Forwarded-For"); got != "198.51.100.10" {
			t.Fatalf("expected X-Forwarded-For 198.51.100.10, got %q", got)
		}

		w.Header().Set("Remote-User", "frank")
		w.WriteHeader(http.StatusOK)
	}))
	defer verifyServer.Close()

	r := gin.New()
	r.Use(SessionMiddleware(&config.Config{TinyauthVerifyURL: verifyServer.URL}))
	r.GET("/manage/api/auth/check", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/manage/api/auth/check?foo=bar", nil)
	req.Host = "auth.hommers.nl"
	req.Header.Set("Cookie", "session=abc")
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-For", "198.51.100.10")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, w.Code)
	}
}
