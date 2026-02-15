package service

import (
	"bytes"
	"fmt"
	"log"
	"net/smtp"
	"text/template"

	"tinyauth-usermanagement/internal/config"

	"github.com/jordan-wright/email"
)

type MailService struct{ cfg *config.Config }

func NewMailService(cfg *config.Config) *MailService { return &MailService{cfg: cfg} }

const defaultEmailBody = `Hello,

A password reset was requested for your account.

Click this link to reset your password:
{{.URL}}

Or use this token: {{.Token}}

If you did not request this, you can safely ignore this email.
`

type emailData struct {
	URL      string
	Token    string
	Username string
}

func (s *MailService) SendResetEmail(toEmail, token string) error {
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.cfg.MailBaseURL, token)
	if s.cfg.SMTPHost == "" {
		log.Printf("[mail disabled] reset token for %s: %s (%s)", toEmail, token, resetURL)
		return nil
	}

	data := emailData{
		URL:      resetURL,
		Token:    token,
		Username: toEmail,
	}

	// Render subject
	subject := s.cfg.EmailSubject
	if subject == "" {
		subject = "Password reset"
	}
	renderedSubject, err := renderTemplate("subject", subject, data)
	if err != nil {
		return fmt.Errorf("email subject template: %w", err)
	}

	// Render body
	bodyTmpl := s.cfg.EmailBody
	if bodyTmpl == "" {
		bodyTmpl = defaultEmailBody
	}
	renderedBody, err := renderTemplate("body", bodyTmpl, data)
	if err != nil {
		return fmt.Errorf("email body template: %w", err)
	}

	e := email.NewEmail()
	e.From = s.cfg.SMTPFrom
	e.To = []string{toEmail}
	e.Subject = renderedSubject
	e.Text = []byte(renderedBody)
	addr := fmt.Sprintf("%s:%d", s.cfg.SMTPHost, s.cfg.SMTPPort)
	auth := smtp.PlainAuth("", s.cfg.SMTPUsername, s.cfg.SMTPPassword, s.cfg.SMTPHost)
	return e.Send(addr, auth)
}

// SendTestEmail sends a simple test email to verify SMTP configuration.
func (s *MailService) SendTestEmail(toEmail string) error {
	if s.cfg.SMTPHost == "" {
		return fmt.Errorf("SMTP not configured")
	}
	e := email.NewEmail()
	e.From = s.cfg.SMTPFrom
	e.To = []string{toEmail}
	e.Subject = "TinyAuth — Test email"
	e.Text = []byte("This is a test email from TinyAuth Usermanagement.\n\nIf you received this, your email configuration is working correctly.")
	addr := fmt.Sprintf("%s:%d", s.cfg.SMTPHost, s.cfg.SMTPPort)
	auth := smtp.PlainAuth("", s.cfg.SMTPUsername, s.cfg.SMTPPassword, s.cfg.SMTPHost)
	return e.Send(addr, auth)
}

func renderTemplate(name, tmplStr string, data emailData) (string, error) {
	tmpl, err := template.New(name).Parse(tmplStr)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
