package handler

import (
	"net/http"

	"tinyauth-sidecar/internal/config"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	cfg *config.Config
}

func NewAuthHandler(cfg *config.Config) *AuthHandler {
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
	c.JSON(http.StatusOK, gin.H{"ok": true, "redirectUrl": h.cfg.TinyauthLogoutURL})
}
