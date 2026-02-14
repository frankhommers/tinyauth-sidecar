# tinyauth-usermanagement

Companion sidecar for [tinyauth](https://github.com/steveiliop56/tinyauth) (v4). Manages users in the shared plain text users file.

## Features

- **No separate session** — authenticates every request via tinyauth's forwardauth endpoint
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
| `TINYAUTH_VERIFY_URL` | — | Tinyauth forwardauth URL (required) |
| `TINYAUTH_CONTAINER_NAME` | `tinyauth` | Container to restart after user changes |
| `DOCKER_SOCKET_PATH` | `/var/run/docker.sock` | Docker socket path |
| `DISABLE_SIGNUP` | `false` | Disable signup (hides UI + blocks API) |
| `TOTP_ISSUER` | `tinyauth` | Issuer name in authenticator apps |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM` | — | Email for password resets |
| `MAIL_BASE_URL` | `http://localhost:8080` | Base URL in reset emails |
| `CONFIG_PATH` | `/data/config.toml` | Webhook config file path |

## Webhook configuration (`config.toml`)

Password sync and SMS webhooks are configured via a TOML file. See [`config.example.toml`](config.example.toml).

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
- `GET  /features` — runtime feature flags
- `GET  /health`

**Authenticated (validated via tinyauth):**
- `GET  /auth/check` — auth status
- `POST /auth/logout` — logout
- `GET  /account/profile`
- `POST /account/change-password`
- `POST /account/phone`
- `POST /account/totp/setup`
- `POST /account/totp/enable`
- `POST /account/totp/disable`
- `POST /account/totp/recover`

## Tinyauth config tip

Add a link to usermanagement's reset page in tinyauth's forgot password message:

```
TINYAUTH_UI_FORGOTPASSWORDMESSAGE='<a href="https://auth.example.com/manage/reset-password">Forgot your password?</a>'
```
