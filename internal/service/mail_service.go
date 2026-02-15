package service

import (
	"fmt"
	"log"
	"net/smtp"

	"tinyauth-usermanagement/internal/config"

	"github.com/jordan-wright/email"
)

type MailService struct{ cfg *config.Config }

func NewMailService(cfg *config.Config) *MailService { return &MailService{cfg: cfg} }

func (s *MailService) SendResetEmail(toEmail, token string) error {
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.cfg.MailBaseURL, token)
	if s.cfg.SMTPHost == "" {
		log.Printf("[mail disabled] reset token for %s: %s (%s)", toEmail, token, resetURL)
		return nil
	}
	e := email.NewEmail()
	e.From = s.cfg.SMTPFrom
	e.To = []string{toEmail}
	e.Subject = "Password reset request"
	e.Text = []byte("Reset your password: " + resetURL)
	addr := fmt.Sprintf("%s:%d", s.cfg.SMTPHost, s.cfg.SMTPPort)
	auth := smtp.PlainAuth("", s.cfg.SMTPUsername, s.cfg.SMTPPassword, s.cfg.SMTPHost)
	return e.Send(addr, auth)
}
