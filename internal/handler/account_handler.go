package handler

import (
	"encoding/base64"
	"net/http"

	"tinyauth-sidecar/internal/service"

	"github.com/gin-gonic/gin"
)

type AccountHandler struct{ account *service.AccountService }

func NewAccountHandler(account *service.AccountService) *AccountHandler { return &AccountHandler{account: account} }

func (h *AccountHandler) Register(r *gin.RouterGroup) {
	r.GET("/account/profile", h.Profile)
	r.POST("/account/change-password", h.ChangePassword)
	r.POST("/account/phone", h.UpdatePhone)
	r.POST("/account/email", h.UpdateEmail)
	r.POST("/account/totp/setup", h.TotpSetup)
	r.POST("/account/totp/enable", h.TotpEnable)
	r.POST("/account/totp/disable", h.TotpDisable)
	r.POST("/account/totp/recover", h.TotpRecover)
}

func username(c *gin.Context) string {
	u, _ := c.Get("username")
	v, _ := u.(string)
	return v
}

func (h *AccountHandler) Profile(c *gin.Context) {
	p, err := h.account.Profile(username(c))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *AccountHandler) ChangePassword(c *gin.Context) {
	var req struct {
		OldPassword string `json:"oldPassword"`
		NewPassword string `json:"newPassword"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.account.ChangePassword(username(c), req.OldPassword, req.NewPassword, c.ClientIP()); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *AccountHandler) UpdatePhone(c *gin.Context) {
	var req struct {
		Phone string `json:"phone"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.account.SetPhone(username(c), req.Phone); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *AccountHandler) UpdateEmail(c *gin.Context) {
	var req struct {
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.account.SetEmail(username(c), req.Email); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *AccountHandler) TotpSetup(c *gin.Context) {
	secret, otpURL, pngBytes, err := h.account.TotpSetup(username(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"secret": secret,
		"otpUrl": otpURL,
		"qrPng":  "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes),
	})
}

func (h *AccountHandler) TotpEnable(c *gin.Context) {
	var req struct {
		Secret string `json:"secret"`
		Code   string `json:"code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.account.TotpEnable(username(c), req.Secret, req.Code); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *AccountHandler) TotpDisable(c *gin.Context) {
	var req struct{ Password string `json:"password"` }
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.account.TotpDisable(username(c), req.Password); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *AccountHandler) TotpRecover(c *gin.Context) {
	var req struct {
		RecoveryKey string `json:"recoveryKey"`
		Secret      string `json:"secret"`
		Code        string `json:"code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.account.TotpRecover(username(c), req.RecoveryKey, req.Secret, req.Code); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
