package handler

import (
	"net/http"

	"tinyauth-usermanagement/internal/config"
	"tinyauth-usermanagement/internal/service"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	auth *service.AuthService
	cfg  config.Config
}

func NewAuthHandler(cfg config.Config, auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth, cfg: cfg}
}

func (h *AuthHandler) Register(r *gin.RouterGroup) {
	r.POST("/auth/login", h.Login)
	r.POST("/auth/logout", h.Logout)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	token, err := h.auth.Login(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	c.SetCookie(h.cfg.SessionCookieName, token, int(h.cfg.SessionTTLSeconds), "/", "", h.cfg.SecureCookie, true)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	token, _ := c.Cookie(h.cfg.SessionCookieName)
	_ = h.auth.Logout(token)
	c.SetCookie(h.cfg.SessionCookieName, "", -1, "/", "", h.cfg.SecureCookie, true)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
