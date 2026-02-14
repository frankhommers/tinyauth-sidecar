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
	"strings"
	"sync"
	"text/template"
	"time"
)

// PasswordTargetConfig represents a single webhook target for password syncing.
type PasswordTargetConfig struct {
	Name          string            `json:"name"`
	URL           string            `json:"url"`
	Method        string            `json:"method"`
	ContentType   string            `json:"content_type"`
	Body          string            `json:"body"`
	Headers       map[string]string `json:"headers"`
	SkipTLSVerify bool              `json:"skip_tls_verify"`
	Env           map[string]string `json:"env"`
}

// PasswordTargetProvider manages webhook-based password sync targets.
type PasswordTargetProvider struct {
	targets []PasswordTargetConfig
}

// NewPasswordTargetProvider parses PASSWORD_TARGETS from the environment.
// Returns nil if not configured.
func NewPasswordTargetProvider() *PasswordTargetProvider {
	raw := os.Getenv("PASSWORD_TARGETS")
	if raw == "" {
		return nil
	}

	var targets []PasswordTargetConfig
	if err := json.Unmarshal([]byte(raw), &targets); err != nil {
		log.Printf("[password-targets] failed to parse PASSWORD_TARGETS: %v", err)
		return nil
	}

	// Apply defaults
	for i := range targets {
		if targets[i].Method == "" {
			targets[i].Method = "POST"
		}
		if targets[i].ContentType == "" {
			targets[i].ContentType = "application/json"
		}
	}

	for _, t := range targets {
		if strings.HasPrefix(t.URL, "http://") {
			log.Printf("[password-targets] WARNING: target %q uses plain HTTP (not HTTPS): %s — passwords will be sent unencrypted!", t.Name, t.URL)
		}
	}

	log.Printf("[password-targets] loaded %d target(s)", len(targets))
	return &PasswordTargetProvider{targets: targets}
}

// SyncPassword calls all configured webhook targets in parallel.
// Returns a slice of errors (one per failed target). Successful targets return nil.
func (p *PasswordTargetProvider) SyncPassword(username, plainPassword, hashedPassword string) []error {
	if p == nil || len(p.targets) == 0 {
		return nil
	}

	var (
		mu   sync.Mutex
		errs []error
		wg   sync.WaitGroup
	)

	for _, target := range p.targets {
		wg.Add(1)
		go func(t PasswordTargetConfig) {
			defer wg.Done()
			if err := callPasswordTarget(t, username, plainPassword, hashedPassword); err != nil {
				log.Printf("[password-targets] %s failed: %v", t.Name, err)
				mu.Lock()
				errs = append(errs, fmt.Errorf("%s: %w", t.Name, err))
				mu.Unlock()
			} else {
				log.Printf("[password-targets] %s synced OK for user %s", t.Name, username)
			}
		}(target)
	}

	wg.Wait()
	return errs
}

func callPasswordTarget(t PasswordTargetConfig, username, plainPassword, hashedPassword string) error {
	data := buildTemplateData(t.Env, map[string]string{
		"Username":       username,
		"Password":       plainPassword,
		"HashedPassword": hashedPassword,
	})

	urlStr, err := executeTemplate("url", t.URL, data)
	if err != nil {
		return fmt.Errorf("template url: %w", err)
	}

	bodyStr, err := executeTemplate("body", t.Body, data)
	if err != nil {
		return fmt.Errorf("template body: %w", err)
	}

	req, err := http.NewRequest(t.Method, urlStr, bytes.NewBufferString(bodyStr))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}

	req.Header.Set("Content-Type", t.ContentType)

	for k, v := range t.Headers {
		headerVal, err := executeTemplate("header-"+k, v, data)
		if err != nil {
			return fmt.Errorf("template header %s: %w", k, err)
		}
		req.Header.Set(k, headerVal)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	if t.SkipTLSVerify {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// buildTemplateData merges env vars and explicit vars into a single map.
func buildTemplateData(env map[string]string, vars map[string]string) map[string]string {
	data := make(map[string]string, len(env)+len(vars))
	for k, v := range env {
		// Resolve env var references: if the value starts with "$", read from os.Getenv
		data[k] = v
	}
	for k, v := range vars {
		data[k] = v
	}
	return data
}

// executeTemplate parses and executes a Go text/template with the given data.
func executeTemplate(name, tmplStr string, data map[string]string) (string, error) {
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
