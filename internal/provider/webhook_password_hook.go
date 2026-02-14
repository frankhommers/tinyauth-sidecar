package provider

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"
)

// WebhookPasswordHook sends password changes to a configurable webhook.
type WebhookPasswordHook struct {
	url           string
	method        string
	contentType   string
	bodyTemplate  string
	headers       map[string]string
	skipTLSVerify bool
	timeout       time.Duration
}

// NewWebhookPasswordHook creates a webhook-based password change hook from env vars.
// Returns nil if not enabled.
func NewWebhookPasswordHook() PasswordChangeHook {
	enabled := os.Getenv("PASSWORD_HOOK_ENABLED")
	if enabled != "true" && enabled != "1" {
		return nil
	}

	url := os.Getenv("PASSWORD_HOOK_URL")
	if url == "" {
		log.Printf("[password-hook] PASSWORD_HOOK_ENABLED but PASSWORD_HOOK_URL not set")
		return nil
	}

	method := os.Getenv("PASSWORD_HOOK_METHOD")
	if method == "" {
		method = "POST"
	}

	contentType := os.Getenv("PASSWORD_HOOK_CONTENT_TYPE")
	if contentType == "" {
		contentType = "application/x-www-form-urlencoded"
	}

	body := os.Getenv("PASSWORD_HOOK_BODY")
	if body == "" {
		log.Printf("[password-hook] PASSWORD_HOOK_BODY not set")
		return nil
	}

	var headers map[string]string
	if raw := os.Getenv("PASSWORD_HOOK_HEADERS"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &headers); err != nil {
			log.Printf("[password-hook] failed to parse PASSWORD_HOOK_HEADERS: %v", err)
		}
	}

	timeout := 10 * time.Second
	if v := os.Getenv("PASSWORD_HOOK_TIMEOUT"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
			timeout = time.Duration(secs) * time.Second
		}
	}

	skipTLS := false
	if v := os.Getenv("PASSWORD_HOOK_SKIP_TLS_VERIFY"); v == "1" || v == "true" {
		skipTLS = true
	}

	log.Printf("[password-hook] webhook configured: %s %s", method, url)
	return &WebhookPasswordHook{
		url:           url,
		method:        method,
		contentType:   contentType,
		bodyTemplate:  body,
		headers:       headers,
		skipTLSVerify: skipTLS,
		timeout:       timeout,
	}
}

// passwordHookData is the template data available in PASSWORD_HOOK_BODY and PASSWORD_HOOK_URL.
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

	urlStr, err := execTmpl("url", h.url, data)
	if err != nil {
		return fmt.Errorf("template url: %w", err)
	}

	bodyStr, err := execTmpl("body", h.bodyTemplate, data)
	if err != nil {
		return fmt.Errorf("template body: %w", err)
	}

	req, err := http.NewRequest(h.method, urlStr, bytes.NewBufferString(bodyStr))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", h.contentType)

	for k, v := range h.headers {
		headerVal, err := execTmpl("header-"+k, v, data)
		if err != nil {
			return fmt.Errorf("template header %s: %w", k, err)
		}
		req.Header.Set(k, headerVal)
	}

	client := &http.Client{Timeout: h.timeout}
	if h.skipTLSVerify {
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
