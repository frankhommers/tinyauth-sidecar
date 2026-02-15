package handler

import (
	"log"
	"net/http"

	"tinyauth-usermanagement/internal/config"
	"tinyauth-usermanagement/internal/provider"
	"tinyauth-usermanagement/internal/service"
	"tinyauth-usermanagement/internal/store"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	cfg       *config.Config
	mail      *service.MailService
	sms       provider.SMSProvider
	usersSvc  *service.UserFileService
	store     *store.Store
	dockerSvc *service.DockerService
}

func NewAdminHandler(cfg *config.Config, mail *service.MailService, sms provider.SMSProvider, usersSvc *service.UserFileService, st *store.Store, dockerSvc *service.DockerService) *AdminHandler {
	return &AdminHandler{cfg: cfg, mail: mail, sms: sms, usersSvc: usersSvc, store: st, dockerSvc: dockerSvc}
}

// isAdmin checks whether the authenticated user has role "admin".
func (h *AdminHandler) isAdmin(c *gin.Context) bool {
	u, _ := c.Get("username")
	username, _ := u.(string)
	if username == "" {
		return false
	}
	meta := h.store.GetUserMeta(username)
	return meta != nil && meta.Role == "admin"
}

// requireAdmin is middleware that returns 403 if the user is not an admin.
func (h *AdminHandler) requireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !h.isAdmin(c) {
			c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func (h *AdminHandler) Register(r *gin.RouterGroup) {
	admin := r.Group("", h.requireAdmin())
	admin.POST("/admin/test-email", h.TestEmail)
	admin.POST("/admin/test-sms", h.TestSMS)
	admin.GET("/admin/status", h.Status)
	admin.POST("/admin/reload-config", h.ReloadConfig)
	admin.POST("/admin/restart-tinyauth", h.RestartTinyauth)
	admin.GET("/admin/tinyauth-health", h.TinyauthHealth)
}

func (h *AdminHandler) TestEmail(c *gin.Context) {
	var req struct {
		To string `json:"to"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.To == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing 'to' field"})
		return
	}

	if err := h.mail.SendTestEmail(req.To); err != nil {
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

func (h *AdminHandler) ReloadConfig(c *gin.Context) {
	fileCfg := config.LoadFileConfig()
	h.cfg.ApplyFileConfig(fileCfg)
	log.Printf("[admin] config.toml reloaded by %s", username(c))
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *AdminHandler) RestartTinyauth(c *gin.Context) {
	log.Printf("[admin] tinyauth restart requested by %s", username(c))
	h.dockerSvc.RestartTinyauth()
	c.JSON(http.StatusOK, gin.H{"ok": true, "message": "restart initiated"})
}

func (h *AdminHandler) TinyauthHealth(c *gin.Context) {
	running, err := h.dockerSvc.IsTinyauthRunning()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"running": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"running": running})
}
