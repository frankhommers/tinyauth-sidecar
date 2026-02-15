package middleware

import (
	"net/http"

	"tinyauth-usermanagement/internal/config"

	"github.com/gin-gonic/gin"
)

// SessionMiddleware validates requests by forwarding cookies to tinyauth's
// forwardauth endpoint on every request. No caching.
func SessionMiddleware(cfg *config.Config) gin.HandlerFunc {
	verifyURL := cfg.TinyauthVerifyURL

	return func(c *gin.Context) {
		if verifyURL == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "auth not configured"})
			return
		}

		cookieHeader := c.GetHeader("Cookie")
		if cookieHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		req, err := http.NewRequest("GET", verifyURL, nil)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "auth check failed"})
			return
		}
		req.Header.Set("Cookie", cookieHeader)
		req.Header.Set("X-Forwarded-Host", c.Request.Host)
		req.Header.Set("X-Forwarded-Uri", c.Request.URL.Path)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		remoteUser := resp.Header.Get("Remote-User")
		if remoteUser == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		c.Set("username", remoteUser)
		c.Next()
	}
}
