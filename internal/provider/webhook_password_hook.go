package provider

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"text/template"
	"time"

	"tinyauth-usermanagement/internal/config"
)

// WebhookPasswordHook sends password changes to a configurable webhook.
type WebhookPasswordHook struct {
	cfg config.WebhookConfig
}

// NewWebhookPasswordHook creates a webhook-based password change hook from config.
// Returns nil if not enabled or URL is empty.
func NewWebhookPasswordHook(cfg config.WebhookConfig) PasswordChangeHook {
	if !cfg.Enabled || cfg.URL == "" {
		return nil
	}
	if cfg.Body == "" {
		log.Printf("[password-hook] enabled but body is empty")
		return nil
	}
	log.Printf("[password-hook] webhook configured: %s %s", cfg.Method, cfg.URL)
	return &WebhookPasswordHook{cfg: cfg}
}

// passwordHookData is the template data for password hook templates.
type passwordHookData struct {
	Email    string
	User     string
	Domain   string
	Password string
}

func (h *WebhookPasswordHook) OnPasswordChanged(email, newPassword string) error {
	user, domain := email, ""
	if parts := strings.SplitN(email, "@", 2); len(parts) == 2 {
		user, domain = parts[0], parts[1]
	}

	data := passwordHookData{
		Email:    email,
		User:     user,
		Domain:   domain,
		Password: newPassword,
	}

	urlStr, err := execTmpl("url", h.cfg.URL, data)
	if err != nil {
		return fmt.Errorf("template url: %w", err)
	}

	bodyStr, err := execTmpl("body", h.cfg.Body, data)
	if err != nil {
		return fmt.Errorf("template body: %w", err)
	}

	req, err := http.NewRequest(h.cfg.Method, urlStr, bytes.NewBufferString(bodyStr))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", h.cfg.ContentType)

	for k, v := range h.cfg.Headers {
		headerVal, err := execTmpl("header-"+k, v, data)
		if err != nil {
			return fmt.Errorf("template header %s: %w", k, err)
		}
		req.Header.Set(k, headerVal)
	}

	client := &http.Client{Timeout: time.Duration(h.cfg.Timeout) * time.Second}
	if h.cfg.SkipTLSVerify {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	log.Printf("[password-hook] synced for %s (HTTP %d)", email, resp.StatusCode)
	return nil
}

func execTmpl(name, tmplStr string, data interface{}) (string, error) {
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
