# tinyauth-usermanagement

Companion sidecar for [tinyauth](https://github.com/steveiliop56/tinyauth) (v4). Manages users in the shared plain-text users file.

## Features

- Password reset via email or SMS
- Two-step verification (TOTP) setup/enable/disable with copyable OTP URL
- Account profile + password change + phone number
- Password change webhooks (pluggable, config-driven)
- SMS webhooks for reset codes (CM.com, generic webhook)
- Signup flow (optional, disabled by default)
- Restarts tinyauth container after user file changes (via Docker socket)
- React + Tailwind + shadcn/ui SPA embedded in Go binary
- Serves under `/manage` base path (Traefik PathPrefix routing)
- i18n: English + Nederlands

## Security

- **No separate session** — every authenticated request is validated via tinyauth's forwardauth endpoint
- **Rate limiting** — public endpoints (password reset, SMS) are rate-limited per IP
- **CSRF protection** — double-submit cookie pattern on all state-changing API requests
- **Security headers** — X-Content-Type-Options, X-Frame-Options, X-XSS-Protection, Referrer-Policy
- **TLS warnings** — logs warnings if password hook URLs use plain HTTP

## Architecture

```
Browser → Traefik → tinyauth (auth.example.com)
                  → usermanagement (auth.example.com/manage)

usermanagement validates authenticated requests by forwarding
cookies to tinyauth's forwardauth endpoint. No double login,
no separate session cookie, one source of truth.

Public pages (reset password, signup) are accessible without auth.
```

## Quick start

See `docker-compose.yml` for a complete Traefik setup.

```bash
sudo docker compose up -d
```

## Environment variables

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | Server port |
| `USERS_FILE_PATH` | `/data/users.txt` | Path to shared tinyauth users file |
| `USERS_TOML` | `/users/users.toml` | User metadata (names, roles, phone) |
| `TINYAUTH_BASEURL` | `http://tinyauth:3000` | Tinyauth base URL (derives verify + logout URLs) |
| `TINYAUTH_VERIFY_URL` | `{BASEURL}/api/auth/traefik` | Override: tinyauth forwardauth URL |
| `TINYAUTH_LOGOUT_URL` | `{BASEURL}/api/auth/logout` | Override: tinyauth logout URL |
| `TINYAUTH_CONTAINER_NAME` | `tinyauth` | Container to restart after user changes |
| `DOCKER_SOCKET_PATH` | `/var/run/docker.sock` | Docker socket path |
| `DISABLE_SIGNUP` | `true` | Disable signup (hides UI + blocks API) |
| `SIGNUP_REQUIRE_APPROVAL` | `false` | Require admin approval for signups |
| `USERNAME_IS_EMAIL` | `true` | When true, username must be a valid email address. When false, a separate email field is available |
| `TOTP_ISSUER` | `tinyauth` | Issuer name in authenticator apps |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM` | — | Email for password resets (see Email setup) |
| `MAIL_BASE_URL` | `http://localhost:8080` | Base URL in reset emails |
| `RESET_TOKEN_TTL_SECONDS` | `3600` | Password reset token validity |
| `CONFIG_PATH` | `/data/config.toml` | Webhook config file path |
| `CORS_ORIGINS` | `http://localhost:5173,http://localhost:8080` | Allowed CORS origins |

## Webhook configuration (`config.toml`)

Password sync and SMS webhooks are configured via a TOML file. See [`config.example.toml`](config.example.toml).

### Users configuration

```toml
[users]
username_is_email = true   # default: true; set to false for separate username + email
```

When `username_is_email = false`:
- Users have a separate email field in their profile
- Password reset looks up users by username OR email
- Reset emails are sent to the email field (not the username)

### Password change hooks (multiple supported)

Called after any successful password change. Fire-and-forget.
Filterable by domain, role, or user.

```toml
[[password_hooks]]
enabled = true
url = "https://your-server.com:2222/CMD_API_EMAIL_PW"
body = "user={{.User}}&domain={{.Domain}}&passwd={{.Password}}"
filter_domains = ["example.com"]
headers = [
  { key = "Authorization", value = "Basic xxx" }
]
```

**Template variables:** `{{.Email}}`, `{{.User}}` (before @), `{{.Domain}}` (after @), `{{.Password}}`, `{{.Role}}`

### SMS webhook

For SMS-based password reset codes (e.g., CM.com):

```toml
[sms]
enabled = true
url = "https://gw.cmtelecom.com/v1.0/message"
content_type = "application/json"
body = '{"messages":{"authentication":{"producttoken":"TOKEN"},...}}'
```

**Template variables:** `{{.To}}` (phone number), `{{.Message}}`

## API

All routes under `/manage/api/`.

**Public (no auth):**
- `POST /password-reset/request` — request reset via email
- `POST /password-reset/confirm` — confirm reset with token
- `POST /auth/forgot-password-sms` — request reset via SMS
- `POST /auth/reset-password-sms` — confirm SMS reset
- `POST /signup` — create account (if enabled)
- `POST /signup/approve` — approve pending signup
- `GET  /features` — runtime feature flags
- `GET  /health`

**Authenticated (validated via tinyauth):**
- `GET  /auth/check` — auth status
- `POST /auth/logout` — get tinyauth logout URL
- `GET  /account/profile`
- `POST /account/change-password`
- `POST /account/phone`
- `POST /account/email`
- `POST /account/totp/setup`
- `POST /account/totp/enable`
- `POST /account/totp/disable`
- `POST /account/totp/recover`

## Email setup

To enable email-based password resets, configure these environment variables:

```yaml
SMTP_HOST: smtp.example.com
SMTP_PORT: 587                    # default: 587
SMTP_USERNAME: noreply@example.com
SMTP_PASSWORD: your-smtp-password
SMTP_FROM: noreply@example.com
MAIL_BASE_URL: https://auth.example.com/manage  # base URL used in reset links
```

## Tinyauth config tip

Add a link to usermanagement's reset page in tinyauth's forgot password message:

```
FORGOT_PASSWORD_MESSAGE='<a href="https://auth.example.com/manage/reset-password">Forgot your password?</a>'
```
