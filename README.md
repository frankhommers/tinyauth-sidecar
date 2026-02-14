# tinyauth-usermanagement

Companion sidecar app for [tinyauth](https://github.com/steveiliop56/tinyauth) (v4). It manages users stored in the shared plain text users file:

- `username:bcrypt_hash`
- `username:bcrypt_hash:totp_secret`

## Features

- Password reset by token (1h default) + email sender hook
- Signup flow with optional admin approval (`SIGNUP_REQUIRE_APPROVAL=true`) or fully disabled (`DISABLE_SIGNUP=true`)
- TOTP setup/enable/disable/recovery with copyable OTP URL
- Account profile + password change
- SSO auto-login from tinyauth session (no double login)
- In-memory sessions (no database required)
- Restarts tinyauth container after users file mutations (via Docker socket)
- React + Tailwind + shadcn/ui SPA embedded in Go binary (`embed.FS`)
- Serves under `/manage` base path (designed for Traefik PathPrefix routing)

## Quick start

See [`examples/docker-compose.yml`](examples/docker-compose.yml) for a complete Traefik setup with tinyauth + usermanagement.

```bash
sudo docker compose up -d
```

## Run locally (development)

```bash
cd frontend
pnpm install
pnpm build
cd ..
go mod tidy
go run .
```

## Environment variables

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | Server port |
| `USERS_FILE_PATH` | `/data/users.txt` | Path to tinyauth users file |
| `USERS_TOML` | `/users/users.toml` | Path to users TOML metadata |
| `TINYAUTH_CONTAINER_NAME` | `tinyauth` | Container to restart after user changes |
| `DOCKER_SOCKET_PATH` | `/var/run/docker.sock` | Docker socket path |
| `DISABLE_SIGNUP` | `false` | Disable signup (hides UI + blocks API) |
| `SIGNUP_REQUIRE_APPROVAL` | `false` | Require admin approval for signups |
| `TINYAUTH_VERIFY_URL` | `http://tinyauth:3000/api/auth/traefik` | Tinyauth forwardauth URL for SSO |
| `SESSION_COOKIE_NAME` | `tinyauth_um_session` | Session cookie name |
| `SESSION_SECRET` | `dev-secret-change-me` | Session signing secret |
| `SESSION_TTL_SECONDS` | `86400` | Session lifetime |
| `SECURE_COOKIE` | `false` | Send cookies over HTTPS only |
| `TOTP_ISSUER` | `tinyauth` | TOTP issuer name in authenticator apps |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM` | — | Email configuration |
| `MAIL_BASE_URL` | `http://localhost:8080` | Base URL for password reset emails |
| `CONFIG_PATH` | `/data/config.toml` | Webhook configuration file |
| `CORS_ORIGINS` | `*` | Allowed CORS origins |

## SSO (Single Sign-On)

When a user is logged in to tinyauth and visits `/manage`, usermanagement automatically creates a session by verifying the tinyauth session cookie via the forwardauth endpoint. No double login required.

Set `TINYAUTH_VERIFY_URL` to empty string to disable SSO.

## Webhook configuration (`config.toml`)

Password sync and SMS webhooks are configured via a TOML file. See [`config.example.toml`](config.example.toml) for a full example.

### Password change webhook

Called after any successful password change. Fire-and-forget.

```toml
[[password_hooks]]
enabled = true
url = "https://your-server.com/api/password"
method = "POST"
content_type = "application/x-www-form-urlencoded"
body = "user={{.User}}&domain={{.Domain}}&passwd={{.Password}}"
timeout = 10
```

**Template variables:** `{{.Email}}`, `{{.User}}` (before @), `{{.Domain}}` (after @), `{{.Password}}` (plaintext).

### SMS webhook

Used for SMS-based password reset codes.

```toml
[sms]
enabled = true
url = "https://api.sms-provider.com/send"
method = "POST"
content_type = "application/json"
body = '{"to":"{{.To}}","message":"{{.Message}}"}'
```

## API overview

All API routes are under `/manage/api/`.

**Public:**
- `POST /manage/api/auth/login`
- `POST /manage/api/auth/logout`
- `GET  /manage/api/auth/sso` — SSO auto-login check
- `POST /manage/api/password-reset/request`
- `POST /manage/api/password-reset/confirm`
- `POST /manage/api/signup` (disabled when `DISABLE_SIGNUP=true`)
- `GET  /manage/api/features` — runtime feature flags
- `GET  /manage/api/health`

**Authenticated:**
- `GET  /manage/api/account/profile`
- `POST /manage/api/account/change-password`
- `POST /manage/api/account/totp/setup`
- `POST /manage/api/account/totp/enable`
- `POST /manage/api/account/totp/disable`
- `POST /manage/api/account/totp/recover`
