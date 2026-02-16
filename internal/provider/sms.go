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
	"text/template"
	"strings"
	"time"

	"tinyauth-sidecar/internal/config"
)

// SMSProvider is the interface for sending SMS messages.
type SMSProvider interface {
	SendSMS(to, message string) error
}

// WebhookSMSConfig holds the configuration for webhook-based SMS.
type WebhookSMSConfig struct {
	URL           string
	Method        string
	ContentType   string
	Body          string
	Headers       map[string]string
	Env           map[string]string
	SkipTLSVerify bool
}

// WebhookSMSProvider sends SMS via a configurable webhook.
type WebhookSMSProvider struct {
	config WebhookSMSConfig
}

// NewWebhookSMSProvider creates a WebhookSMSProvider from environment variables.
// Returns nil if SMS is not configured/enabled.
func NewWebhookSMSProvider() SMSProvider {
	enabled := os.Getenv("SMS_ENABLED")
	if enabled == "" || (enabled != "1" && enabled != "true" && enabled != "yes") {
		return nil
	}

	url := os.Getenv("SMS_WEBHOOK_URL")
	if url == "" {
		log.Printf("[sms] SMS_ENABLED but SMS_WEBHOOK_URL not set")
		return nil
	}

	method := os.Getenv("SMS_WEBHOOK_METHOD")
	if method == "" {
		method = "POST"
	}

	contentType := os.Getenv("SMS_WEBHOOK_CONTENT_TYPE")
	if contentType == "" {
		contentType = "application/json"
	}

	body := os.Getenv("SMS_WEBHOOK_BODY")
	if body == "" {
		log.Printf("[sms] SMS_WEBHOOK_BODY not set")
		return nil
	}

	var headers map[string]string
	if raw := os.Getenv("SMS_WEBHOOK_HEADERS"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &headers); err != nil {
			log.Printf("[sms] failed to parse SMS_WEBHOOK_HEADERS: %v", err)
		}
	}

	var env map[string]string
	if raw := os.Getenv("SMS_WEBHOOK_ENV"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &env); err != nil {
			log.Printf("[sms] failed to parse SMS_WEBHOOK_ENV: %v", err)
		}
	}

	skipTLS := false
	if v := os.Getenv("SMS_WEBHOOK_SKIP_TLS_VERIFY"); v == "1" || v == "true" || v == "yes" {
		skipTLS = true
	}

	log.Printf("[sms] webhook SMS provider configured: %s %s", method, url)
	return &WebhookSMSProvider{
		config: WebhookSMSConfig{
			URL:           url,
			Method:        method,
			ContentType:   contentType,
			Body:          body,
			Headers:       headers,
			Env:           env,
			SkipTLSVerify: skipTLS,
		},
	}
}

// NewWebhookSMSProviderFromConfig creates a WebhookSMSProvider from a WebhookConfig (TOML).
// Returns nil if not enabled.
func NewWebhookSMSProviderFromConfig(cfg config.WebhookConfig) SMSProvider {
	if !cfg.Enabled || cfg.URL == "" {
		return nil
	}
	if cfg.Body == "" {
		log.Printf("[sms] config enabled but body is empty")
		return nil
	}
	log.Printf("[sms] webhook SMS provider configured from config.toml: %s %s", cfg.Method, cfg.URL)
	return &WebhookSMSProvider{
		config: WebhookSMSConfig{
			URL:           cfg.URL,
			Method:        cfg.Method,
			ContentType:   cfg.ContentType,
			Body:          cfg.Body,
			Headers:       headerEntriesToMap(cfg.Headers),
			SkipTLSVerify: cfg.SkipTLSVerify,
		},
	}
}

// SendSMS sends an SMS message via the configured webhook.
func (p *WebhookSMSProvider) SendSMS(to, message string) error {
	data := buildTemplateData(p.config.Env, map[string]string{
		"To":      to,
		"Message": message,
	})

	urlStr, err := executeSMSTemplate("url", p.config.URL, data)
	if err != nil {
		return fmt.Errorf("template url: %w", err)
	}

	bodyStr, err := executeSMSTemplate("body", p.config.Body, data)
	if err != nil {
		return fmt.Errorf("template body: %w", err)
	}

	req, err := http.NewRequest(p.config.Method, urlStr, bytes.NewBufferString(bodyStr))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}

	req.Header.Set("Content-Type", p.config.ContentType)

	for k, v := range p.config.Headers {
		headerVal, err := executeSMSTemplate("header-"+k, v, data)
		if err != nil {
			return fmt.Errorf("template header %s: %w", k, err)
		}
		req.Header.Set(k, headerVal)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	if p.config.SkipTLSVerify {
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

	log.Printf("[sms] sent SMS to %s via webhook (HTTP %d)", to, resp.StatusCode)
	return nil
}

func executeSMSTemplate(name, tmplStr string, data map[string]string) (string, error) {
	tmpl, err := template.New(name).Funcs(template.FuncMap{
		"jsonEscape":          jsonEscape,
		"digitsOnly":          digitsOnly,
		"replace": strings.ReplaceAll,
	}).Parse(tmplStr)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func headerEntriesToMap(entries []config.HeaderEntry) map[string]string {
	if len(entries) == 0 {
		return nil
	}
	m := make(map[string]string, len(entries))
	for _, e := range entries {
		m[e.Key] = e.Value
	}
	return m
}
