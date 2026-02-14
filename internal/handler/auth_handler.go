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
	// We can't clear tinyauth's cookie (httpOnly, different domain potentially).
	// Return the tinyauth logout URL so the frontend can redirect there.
	logoutURL := h.cfg.TinyauthLogoutURL
	if logoutURL == "" {
		// Default: just tell the frontend to redirect to the login page
		logoutURL = "/manage"
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "redirectUrl": logoutURL})
}
