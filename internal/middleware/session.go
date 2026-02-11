package middleware

import (
	"net/http"

	"tinyauth-usermanagement/internal/config"
	"tinyauth-usermanagement/internal/store"

	"github.com/gin-gonic/gin"
)

func SessionMiddleware(cfg config.Config, st *store.SQLiteStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(cfg.SessionCookieName)
		if err != nil || token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		var username string
		var expires int64
		err = st.DB.QueryRow(`SELECT username, expires_at FROM sessions WHERE token = ?`, token).Scan(&username, &expires)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Set("username", username)
		c.Next()
	}
}
