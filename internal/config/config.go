package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Port                    string
	UsersFilePath           string
	SessionCookieName       string
	SessionSecret           string
	SessionTTLSeconds       int64
	ResetTokenTTLSeconds    int64
	SignupRequireApproval   bool
	SMTPHost                string
	SMTPPort                int
	SMTPUsername            string
	SMTPPassword            string
	SMTPFrom                string
	MailBaseURL             string
	TOTPIssuer              string
	TinyauthContainerName   string
	DockerSocketPath        string
	SecureCookie            bool
	CORSOrigins             []string
}

func Load() Config {
	return Config{
		Port:                  getEnv("PORT", "8080"),
		UsersFilePath:         getEnv("USERS_FILE_PATH", "/data/users.txt"),
		SessionCookieName:     getEnv("SESSION_COOKIE_NAME", "tinyauth_um_session"),
		SessionSecret:         getEnv("SESSION_SECRET", "dev-secret-change-me"),
		SessionTTLSeconds:     getEnvInt64("SESSION_TTL_SECONDS", 86400),
		ResetTokenTTLSeconds:  getEnvInt64("RESET_TOKEN_TTL_SECONDS", 3600),
		SignupRequireApproval: getEnvBool("SIGNUP_REQUIRE_APPROVAL", false),
		SMTPHost:              getEnv("SMTP_HOST", ""),
		SMTPPort:              getEnvInt("SMTP_PORT", 587),
		SMTPUsername:          getEnv("SMTP_USERNAME", ""),
		SMTPPassword:          getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:              getEnv("SMTP_FROM", "noreply@example.local"),
		MailBaseURL:           getEnv("MAIL_BASE_URL", "http://localhost:8080"),
		TOTPIssuer:            getEnv("TOTP_ISSUER", "tinyauth"),
		TinyauthContainerName: getEnv("TINYAUTH_CONTAINER_NAME", "tinyauth"),
		DockerSocketPath:      getEnv("DOCKER_SOCKET_PATH", "/var/run/docker.sock"),
		SecureCookie:          getEnvBool("SECURE_COOKIE", false),
		CORSOrigins:           parseCSV(getEnv("CORS_ORIGINS", "http://localhost:5173,http://localhost:8080")),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvInt64(key string, fallback int64) int64 {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		return strings.EqualFold(v, "1") || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes")
	}
	return fallback
}

// HeaderEntry is a key-value pair for HTTP headers.
type HeaderEntry struct {
	Key   string `toml:"key"`
	Value string `toml:"value"`
}

// WebhookConfig holds configuration for a generic webhook (password hook or SMS).
type WebhookConfig struct {
	Enabled       bool              `toml:"enabled"`
	URL           string            `toml:"url"`
	Method        string            `toml:"method"`
	ContentType   string            `toml:"content_type"`
	Body          string            `toml:"body"`
	Headers        []HeaderEntry `toml:"headers"`
	Timeout        int           `toml:"timeout"`
	SkipTLSVerify  bool          `toml:"skip_tls_verify"`
	FilterDomains  []string      `toml:"filter_domains"`
	FilterRoles    []string      `toml:"filter_roles"`
	FilterUsers   []string      `toml:"filter_users"`
}

// FileConfig represents the TOML config file structure.
type FileConfig struct {
	PasswordHooks []WebhookConfig `toml:"password_hooks"`
	SMS           WebhookConfig   `toml:"sms"`
}

// LoadFileConfig reads the TOML config file from CONFIG_PATH (default /data/config.toml).
// Returns an empty config if the file doesn't exist.
func LoadFileConfig() FileConfig {
	path := getEnv("CONFIG_PATH", "/data/config.toml")

	var fc FileConfig
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fc
	}

	if _, err := toml.DecodeFile(path, &fc); err != nil {
		log.Printf("[config] failed to parse %s: %v", path, err)
		return FileConfig{}
	}

	// Apply defaults
	for i := range fc.PasswordHooks {
		applyWebhookDefaults(&fc.PasswordHooks[i], "POST", "application/x-www-form-urlencoded", 10)
	}
	applyWebhookDefaults(&fc.SMS, "POST", "application/json", 15)

	log.Printf("[config] loaded %s", path)
	return fc
}

func applyWebhookDefaults(wc *WebhookConfig, method, contentType string, timeout int) {
	if wc.Method == "" {
		wc.Method = method
	}
	if wc.ContentType == "" {
		wc.ContentType = contentType
	}
	if wc.Timeout <= 0 {
		wc.Timeout = timeout
	}
}

func parseCSV(v string) []string {
	parts := strings.Split(v, ",")
	res := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			res = append(res, p)
		}
	}
	if len(res) == 0 {
		return []string{"*"}
	}
	return res
}
