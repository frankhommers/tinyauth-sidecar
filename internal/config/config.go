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
	ResetTokenTTLSeconds    int64
	DisableSignup          bool
	SignupRequireApproval   bool
	SMTPHost                string
	SMTPPort                int
	SMTPUsername            string
	SMTPPassword            string
	SMTPFrom                string
	MailBaseURL             string
	TOTPIssuer              string
	TinyauthBaseURL         string
	TinyauthExternalURL     string
	TinyauthVerifyURL       string
	TinyauthLogoutURL       string
	TinyauthContainerName   string
	DockerSocketPath        string
	CORSOrigins             []string
	MinPasswordLength       int
	MinPasswordStrength     int
	UsernameIsEmail         bool
	EmailSubject            string
	EmailBody               string
	BackgroundImage         string
	Title                   string
	RestartMethod           string
}

func Load() *Config {
	baseURL := strings.TrimRight(getEnv("TINYAUTH_BASEURL", "http://tinyauth:3000"), "/")

	cfg := &Config{
		Port:                  getEnv("PORT", "8080"),
		UsersFilePath:         getEnv("USERS_FILE_PATH", "/data/users.txt"),
		ResetTokenTTLSeconds:  getEnvInt64("RESET_TOKEN_TTL_SECONDS", 3600),
		DisableSignup:         getEnvBool("DISABLE_SIGNUP", true),
		SignupRequireApproval: getEnvBool("SIGNUP_REQUIRE_APPROVAL", false),
		SMTPHost:              getEnv("SMTP_HOST", ""),
		SMTPPort:              getEnvInt("SMTP_PORT", 587),
		SMTPUsername:          getEnv("SMTP_USERNAME", ""),
		SMTPPassword:          getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:              getEnv("SMTP_FROM", "noreply@example.local"),
		MailBaseURL:           getEnv("MAIL_BASE_URL", "http://localhost:8080"),
		TOTPIssuer:            getEnv("TOTP_ISSUER", "tinyauth"),
		TinyauthBaseURL:      baseURL,
		TinyauthExternalURL:  getEnv("TINYAUTH_EXTERNAL_URL", ""),
		TinyauthVerifyURL:    getEnv("TINYAUTH_VERIFY_URL", baseURL+"/api/auth/traefik"),
		TinyauthLogoutURL:    getEnv("TINYAUTH_LOGOUT_URL", baseURL+"/api/auth/logout"),
		TinyauthContainerName: getEnv("TINYAUTH_CONTAINER_NAME", "tinyauth"),
		DockerSocketPath:      getEnv("DOCKER_SOCKET_PATH", "/var/run/docker.sock"),
		CORSOrigins:           parseCSV(getEnv("CORS_ORIGINS", "http://localhost:5173,http://localhost:8080")),
		MinPasswordLength:     getEnvInt("MIN_PASSWORD_LENGTH", 8),
		MinPasswordStrength:   getEnvInt("MIN_PASSWORD_STRENGTH", 3),
		UsernameIsEmail:       getEnvBool("USERNAME_IS_EMAIL", true),
		EmailSubject:          getEnv("EMAIL_SUBJECT", "Password reset"),
		EmailBody:             getEnv("EMAIL_BODY", ""),
		BackgroundImage:       getEnv("BACKGROUND_IMAGE", "/background.jpg"),
		Title:                getEnv("TITLE", ""),
		RestartMethod:        getEnv("TINYAUTH_RESTART_METHOD", "restart"),
	}

	return cfg
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

// PasswordPolicy configures password strength requirements.
type PasswordPolicy struct {
	MinLength   int `toml:"min_length"`
	MinStrength int `toml:"min_strength"`
}

// UsersConfig configures user-related behaviour.
type UsersConfig struct {
	UsernameIsEmail *bool `toml:"username_is_email"`
}

// UIConfig configures UI appearance.
type UIConfig struct {
	BackgroundImage string `toml:"background_image"`
	Title           string `toml:"title"`
}

// SMTPConfig holds SMTP settings from config.toml.
type SMTPConfig struct {
	Host     string `toml:"host"`
	Port     int    `toml:"port"`
	Username string `toml:"username"`
	Password string `toml:"password"`
	From     string `toml:"from"`
}

// EmailTemplateConfig holds email template settings.
type EmailTemplateConfig struct {
	Subject string `toml:"subject"`
	Body    string `toml:"body"`
}

// OIDCConfig holds OIDC provider settings from config.toml.
type OIDCConfig struct {
	Enabled   bool           `toml:"enabled"`
	IssuerURL string         `toml:"issuer_url"`
	LoginURL  string         `toml:"login_url"`
	KeyPath   string         `toml:"key_path"`
	Clients   []OIDCClient   `toml:"clients"`
}

// OIDCClient represents a registered OIDC client.
type OIDCClient struct {
	ID           string   `toml:"id"`
	Secret       string   `toml:"secret"`
	RedirectURIs []string `toml:"redirect_uris"`
}

// FileConfig represents the TOML config file structure.
type FileConfig struct {
	PasswordPolicy PasswordPolicy  `toml:"password_policy"`
	PasswordHooks  []WebhookConfig `toml:"password_hooks"`
	SMS            WebhookConfig   `toml:"sms"`
	Users          UsersConfig     `toml:"users"`
	SMTP           SMTPConfig          `toml:"smtp"`
	Email          EmailTemplateConfig `toml:"email"`
	UI             UIConfig            `toml:"ui"`
	OIDC           OIDCConfig          `toml:"oidc"`
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

// ApplyFileConfig applies FileConfig overrides onto the running Config.
// Only non-zero values override; env vars remain as fallback.
func (c *Config) ApplyFileConfig(fc FileConfig) {
	if fc.PasswordPolicy.MinLength > 0 {
		c.MinPasswordLength = fc.PasswordPolicy.MinLength
	}
	if fc.PasswordPolicy.MinStrength > 0 {
		c.MinPasswordStrength = fc.PasswordPolicy.MinStrength
	}
	if fc.Users.UsernameIsEmail != nil {
		c.UsernameIsEmail = *fc.Users.UsernameIsEmail
	}
	if fc.SMTP.Host != "" {
		c.SMTPHost = fc.SMTP.Host
	}
	if fc.SMTP.Port > 0 {
		c.SMTPPort = fc.SMTP.Port
	}
	if fc.SMTP.Username != "" {
		c.SMTPUsername = fc.SMTP.Username
	}
	if fc.SMTP.Password != "" {
		c.SMTPPassword = fc.SMTP.Password
	}
	if fc.SMTP.From != "" {
		c.SMTPFrom = fc.SMTP.From
	}
	if fc.Email.Subject != "" {
		c.EmailSubject = fc.Email.Subject
	}
	if fc.Email.Body != "" {
		c.EmailBody = fc.Email.Body
	}
	if fc.UI.BackgroundImage != "" {
		c.BackgroundImage = fc.UI.BackgroundImage
	}
	if fc.UI.Title != "" {
		c.Title = fc.UI.Title
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
