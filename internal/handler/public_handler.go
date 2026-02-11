package handler

import (
	"encoding/base64"
	"net/http"

	"tinyauth-usermanagement/internal/service"

	"github.com/gin-gonic/gin"
)

type PublicHandler struct{ account *service.AccountService }

func NewPublicHandler(account *service.AccountService) *PublicHandler { return &PublicHandler{account: account} }

func (h *PublicHandler) Register(r *gin.RouterGroup) {
	r.POST("/password-reset/request", h.RequestReset)
	r.POST("/password-reset/confirm", h.ConfirmReset)
	r.POST("/signup", h.Signup)
	r.POST("/signup/approve", h.ApproveSignup)
	r.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })
	_ = base64.StdEncoding
}

func (h *PublicHandler) RequestReset(c *gin.Context) {
	var req struct{ Username string `json:"username"` }
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_ = h.account.RequestPasswordReset(req.Username)
	c.JSON(http.StatusOK, gin.H{"ok": true, "message": "If user exists, reset email sent"})
}

func (h *PublicHandler) ConfirmReset(c *gin.Context) {
	var req struct {
		Token       string `json:"token"`
		NewPassword string `json:"newPassword"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.account.ResetPassword(req.Token, req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *PublicHandler) Signup(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	status, err := h.account.Signup(req.Username, req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "status": status})
}

func (h *PublicHandler) ApproveSignup(c *gin.Context) {
	var req struct{ ID string `json:"id"` }
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.account.ApproveSignup(req.ID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
