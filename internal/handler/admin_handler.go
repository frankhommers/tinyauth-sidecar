package handler

import (
	"fmt"
	"net/http"
	"net/smtp"

	"tinyauth-usermanagement/internal/config"
	"tinyauth-usermanagement/internal/provider"
	"tinyauth-usermanagement/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/jordan-wright/email"
)

type AdminHandler struct {
	cfg      config.Config
	sms      provider.SMSProvider
	usersSvc *service.UserFileService
}

func NewAdminHandler(cfg config.Config, sms provider.SMSProvider, usersSvc *service.UserFileService) *AdminHandler {
	return &AdminHandler{cfg: cfg, sms: sms, usersSvc: usersSvc}
}

func (h *AdminHandler) Register(r *gin.RouterGroup) {
	r.POST("/admin/test-email", h.TestEmail)
	r.POST("/admin/test-sms", h.TestSMS)
	r.GET("/admin/status", h.Status)
}

func (h *AdminHandler) TestEmail(c *gin.Context) {
	var req struct {
		To string `json:"to"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.To == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing 'to' field"})
		return
	}

	if h.cfg.SMTPHost == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "SMTP not configured"})
		return
	}

	e := email.NewEmail()
	e.From = h.cfg.SMTPFrom
	e.To = []string{req.To}
	e.Subject = "TinyAuth Test Email"
	e.Text = []byte("This is a test email from TinyAuth User Management.")
	addr := fmt.Sprintf("%s:%d", h.cfg.SMTPHost, h.cfg.SMTPPort)
	auth := smtp.PlainAuth("", h.cfg.SMTPUsername, h.cfg.SMTPPassword, h.cfg.SMTPHost)
	if err := e.Send(addr, auth); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *AdminHandler) TestSMS(c *gin.Context) {
	var req struct {
		To string `json:"to"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.To == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing 'to' field"})
		return
	}

	if h.sms == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "SMS not configured"})
		return
	}

	if err := h.sms.SendSMS(req.To, "TinyAuth test SMS"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *AdminHandler) Status(c *gin.Context) {
	userCount := 0
	if users, err := h.usersSvc.ReadAll(); err == nil {
		userCount = len(users)
	}

	c.JSON(http.StatusOK, gin.H{
		"email":           h.cfg.SMTPHost != "",
		"sms":             h.sms != nil,
		"usernameIsEmail": h.cfg.UsernameIsEmail,
		"userCount":       userCount,
	})
}
