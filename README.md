# tinyauth-usermanagement

Companion sidecar app for [tinyauth](https://github.com/steveiliop56/tinyauth). It manages users stored in the shared plain text users file:

- `username:bcrypt_hash`
- `username:bcrypt_hash:totp_secret`

## Features (scaffolded + functional baseline)

- Password reset by token (1h default) + email sender hook
- Signup flow with optional admin approval (`SIGNUP_REQUIRE_APPROVAL=true`)
- TOTP setup/enable/disable/recovery endpoints
- Account profile + password change
- SQLite for sessions, reset tokens, pending signups
- Restarts tinyauth container after users file mutations (via Docker socket)
- React + MUI SPA embedded in Go binary (`embed.FS`)

## Run locally

```bash
cd frontend
pnpm install
pnpm build
cd ..
go mod tidy
go run .
```

## Docker compose

```bash
sudo docker compose up --build
```

> Use `sudo` for docker commands (user not in docker group).

## Important environment variables

- `USERS_FILE_PATH` (default `/data/users.txt`)
- `SQLITE_PATH` (default `/data/usermanagement.db`)
- `SESSION_COOKIE_NAME` (default `tinyauth_um_session`)
- `RESET_TOKEN_TTL_SECONDS` (default `3600`)
- `SIGNUP_REQUIRE_APPROVAL` (default `false`)
- `TINYAUTH_CONTAINER_NAME` (default `tinyauth`)
- `DOCKER_SOCKET_PATH` (default `/var/run/docker.sock`)
- `SMTP_*` vars for mail
- `CONFIG_PATH` (default `/data/config.toml`) — webhook configuration file

## Webhook configuration (`config.toml`)

Password sync and SMS webhooks are configured via a TOML file. See [`config.example.toml`](config.example.toml) for a full example.

Set the path via `CONFIG_PATH` env var (default: `/data/config.toml`).

### Password change webhook

Called after any successful password change (change, reset, signup). Fire-and-forget: failures are logged but don't affect the local password change.

```toml
[password_hook]
enabled = true
url = "https://your-server.com:2222/CMD_API_EMAIL_PW"
method = "POST"
content_type = "application/x-www-form-urlencoded"
body = "user={{.User}}&domain={{.Domain}}&passwd={{.Password}}"
timeout = 10

[password_hook.headers]
Authorization = "Basic base64-encoded-credentials"
```

**Template variables:** `{{.Email}}` (full email), `{{.User}}` (part before @), `{{.Domain}}` (part after @), `{{.Password}}` (new plaintext password).

### SMS webhook

Used for sending SMS messages (e.g. password reset codes). If configured in `config.toml`, it takes precedence over `SMS_WEBHOOK_*` env vars.

```toml
[sms]
enabled = true
url = "https://gw.cmtelecom.com/v1.0/message"
method = "POST"
content_type = "application/json"
body = '{"messages":{"authentication":{"producttoken":"xxx"},"msg":[{"from":{"number":"TinyAuth"},"to":[{"number":"{{.To}}"}],"body":{"type":"AUTO","content":"{{.Message}}"}}]}}'
```

**Template variables:** `{{.To}}` (phone number), `{{.Message}}` (SMS text).

## API overview

Public:
- `POST /api/auth/login`
- `POST /api/auth/logout`
- `POST /api/password-reset/request`
- `POST /api/password-reset/confirm`
- `POST /api/signup`
- `POST /api/signup/approve`

Authenticated:
- `GET /api/account/profile`
- `POST /api/account/change-password`
- `POST /api/account/totp/setup`
- `POST /api/account/totp/enable`
- `POST /api/account/totp/disable`
- `POST /api/account/totp/recover`

## Notes

- `signup/approve` is intentionally bare-bones scaffold endpoint. Add proper admin auth before production use.
- Recovery key flow is scaffolded with a placeholder format `RECOVERY-<username>`.
