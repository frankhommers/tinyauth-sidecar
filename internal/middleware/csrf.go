package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	csrfCookieName = "csrf_token"
	csrfHeaderName = "X-CSRF-Token"
	csrfTokenLen   = 16 // 32 hex chars
)

// CSRFMiddleware implements double-submit cookie CSRF protection.
// On every response, it sets a non-HttpOnly CSRF cookie (so JS can read it).
// On POST/PUT/DELETE requests, it requires the token back as X-CSRF-Token header.
func CSRFMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Ensure a CSRF cookie exists
		token, err := c.Cookie(csrfCookieName)
		if err != nil || len(token) < csrfTokenLen*2 {
			token = generateCSRFToken()
			c.SetCookie(csrfCookieName, token, 86400, "/", "", false, false)
		}

		// Safe methods don't need validation
		if c.Request.Method == http.MethodGet ||
			c.Request.Method == http.MethodHead ||
			c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}

		// Validate: header must match cookie
		headerToken := c.GetHeader(csrfHeaderName)
		if headerToken == "" || headerToken != token {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "CSRF token mismatch"})
			return
		}

		c.Next()
	}
}

func generateCSRFToken() string {
	b := make([]byte, csrfTokenLen)
	if _, err := rand.Read(b); err != nil {
		// Fallback: this should never happen
		return "fallback-csrf-token-000000000000"
	}
	return hex.EncodeToString(b)
}
