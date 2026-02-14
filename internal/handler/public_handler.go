package handler

import (
	"net/http"

	"tinyauth-usermanagement/internal/config"
	"tinyauth-usermanagement/internal/service"

	"github.com/gin-gonic/gin"
)

type PublicHandler struct {
	account *service.AccountService
	cfg     config.Config
}

func NewPublicHandler(account *service.AccountService, cfg config.Config) *PublicHandler {
	return &PublicHandler{account: account, cfg: cfg}
}

func (h *PublicHandler) Register(r *gin.RouterGroup) {
	r.POST("/password-reset/request", h.RequestReset)
	r.POST("/password-reset/confirm", h.ConfirmReset)
	r.POST("/signup", h.Signup)
	r.POST("/signup/approve", h.ApproveSignup)
	r.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })
	r.GET("/features", h.Features)
	r.POST("/auth/forgot-password-sms", h.ForgotPasswordSMS)
	r.POST("/auth/reset-password-sms", h.ResetPasswordSMS)
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
	if h.cfg.DisableSignup {
		c.JSON(http.StatusForbidden, gin.H{"error": "signups are disabled"})
		return
	}
	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Phone    string `json:"phone"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	status, err := h.account.SignupWithPhone(req.Username, req.Email, req.Password, req.Phone)
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

func (h *PublicHandler) Features(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"smsEnabled":    h.account.SMSEnabled(),
		"signupEnabled": !h.cfg.DisableSignup,
	})
}

func (h *PublicHandler) ForgotPasswordSMS(c *gin.Context) {
	var req struct {
		Phone string `json:"phone"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Phone == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "phone required"})
		return
	}
	_ = h.account.RequestSMSReset(req.Phone)
	// Always return OK to not leak whether phone exists
	c.JSON(http.StatusOK, gin.H{"ok": true, "message": "If a user is associated with this phone, a code was sent"})
}

func (h *PublicHandler) ResetPasswordSMS(c *gin.Context) {
	var req struct {
		Phone       string `json:"phone"`
		Code        string `json:"code"`
		NewPassword string `json:"newPassword"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Phone == "" || req.Code == "" || req.NewPassword == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "phone, code, and newPassword required"})
		return
	}
	if err := h.account.ResetPasswordSMS(req.Phone, req.Code, req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
