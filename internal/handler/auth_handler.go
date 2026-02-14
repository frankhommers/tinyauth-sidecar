package handler

import (
	"net/http"

	"tinyauth-usermanagement/internal/config"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	cfg config.Config
}

func NewAuthHandler(cfg config.Config) *AuthHandler {
	return &AuthHandler{cfg: cfg}
}

func (h *AuthHandler) Register(r *gin.RouterGroup) {
	r.GET("/auth/check", h.Check)
	r.POST("/auth/logout", h.Logout)
}

func (h *AuthHandler) Check(c *gin.Context) {
	u, _ := c.Get("username")
	username, _ := u.(string)
	c.JSON(http.StatusOK, gin.H{"authenticated": true, "username": username})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	// Clear tinyauth session cookies — match the exact attributes tinyauth uses
	for _, cookie := range c.Request.Cookies() {
		if cookie.Name != "" {
			http.SetCookie(c.Writer, &http.Cookie{
				Name:     cookie.Name,
				Value:    "",
				MaxAge:   -1,
				Path:     "/",
				Domain:   "",       // let browser match
				Secure:   true,
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
			})
		}
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
