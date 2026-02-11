package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port                    string
	UsersFilePath           string
	DatabasePath            string
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
		DatabasePath:          getEnv("SQLITE_PATH", "/data/usermanagement.db"),
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
