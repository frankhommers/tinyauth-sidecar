package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"tinyauth-usermanagement/internal/config"
	"tinyauth-usermanagement/internal/store"

	"github.com/gin-gonic/gin"
)

// TinyauthSSO checks if the user has a valid tinyauth session and auto-creates
// a usermanagement session if they don't already have one.
// This is called on page load via a dedicated endpoint.
func TinyauthSSO(cfg config.Config, st *store.Store) gin.HandlerFunc {
	tinyauthURL := cfg.TinyauthVerifyURL // e.g. "http://tinyauth:3000/api/auth/traefik"

	return func(c *gin.Context) {
		if tinyauthURL == "" {
			c.JSON(http.StatusOK, gin.H{"authenticated": false, "reason": "sso not configured"})
			return
		}

		// Already have a valid usermanagement session?
		token, _ := c.Cookie(cfg.SessionCookieName)
		if token != "" {
			username, expiresAt, err := st.GetSession(token)
			if err == nil && username != "" && time.Now().Unix() <= expiresAt {
				c.JSON(http.StatusOK, gin.H{"authenticated": true, "username": username})
				return
			}
		}

		// Forward cookies to tinyauth's verify endpoint
		req, err := http.NewRequest("GET", tinyauthURL, nil)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"authenticated": false})
			return
		}
		// Copy all cookies from the original request
		for _, cookie := range c.Request.Cookies() {
			req.AddCookie(cookie)
		}
		req.Header.Set("X-Forwarded-Host", c.Request.Host)
		req.Header.Set("X-Forwarded-Uri", c.Request.URL.Path)

		resp, err := http.DefaultClient.Do(req)
		if err != nil || resp.StatusCode != 200 {
			c.JSON(http.StatusOK, gin.H{"authenticated": false})
			return
		}
		defer resp.Body.Close()

		// Tinyauth returns the username in the Remote-User header
		remoteUser := resp.Header.Get("Remote-User")
		if remoteUser == "" {
			c.JSON(http.StatusOK, gin.H{"authenticated": false})
			return
		}

		// Auto-create a usermanagement session
		tokenBytes := make([]byte, 32)
		if _, err := rand.Read(tokenBytes); err != nil {
			c.JSON(http.StatusOK, gin.H{"authenticated": false})
			return
		}
		sessionToken := hex.EncodeToString(tokenBytes)
		now := time.Now().Unix()
		err = st.CreateSession(sessionToken, remoteUser, now, now+cfg.SessionTTLSeconds)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"authenticated": false})
			return
		}

		c.SetCookie(cfg.SessionCookieName, sessionToken, int(cfg.SessionTTLSeconds), "/", "", cfg.SecureCookie, true)
		c.JSON(http.StatusOK, gin.H{"authenticated": true, "username": remoteUser})
	}
}
