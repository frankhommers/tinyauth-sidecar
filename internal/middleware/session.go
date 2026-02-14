package middleware

import (
	"net/http"
	"sync"
	"time"

	"tinyauth-usermanagement/internal/config"

	"github.com/gin-gonic/gin"
)

type cacheEntry struct {
	username  string
	expiresAt time.Time
}

var (
	authCache sync.Map
	cacheTTL  = 30 * time.Second
)

// SessionMiddleware validates requests by forwarding cookies to tinyauth's
// forwardauth endpoint. Results are cached in-memory for 30 seconds.
func SessionMiddleware(cfg config.Config) gin.HandlerFunc {
	verifyURL := cfg.TinyauthVerifyURL

	return func(c *gin.Context) {
		if verifyURL == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "auth not configured"})
			return
		}

		// Build cache key from all cookie values
		cookieHeader := c.GetHeader("Cookie")
		if cookieHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		// Check cache
		if entry, ok := authCache.Load(cookieHeader); ok {
			ce := entry.(*cacheEntry)
			if time.Now().Before(ce.expiresAt) {
				c.Set("username", ce.username)
				c.Next()
				return
			}
			authCache.Delete(cookieHeader)
		}

		// Forward to tinyauth
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

		// Cache the result
		authCache.Store(cookieHeader, &cacheEntry{
			username:  remoteUser,
			expiresAt: time.Now().Add(cacheTTL),
		})

		c.Set("username", remoteUser)
		c.Next()
	}
}
