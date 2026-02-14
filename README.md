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

## Password change webhook

Optionally call a webhook after any successful password change (change, reset, signup). Useful for syncing passwords to external systems like DirectAdmin email hosting.

The webhook is fire-and-forget: if it fails, a warning is logged but the local password change still succeeds.

| Variable | Description | Default |
|---|---|---|
| `PASSWORD_HOOK_ENABLED` | Enable the webhook (`true`/`false`) | `false` |
| `PASSWORD_HOOK_URL` | Webhook URL (Go template, supports `{{.Email}}`, `{{.User}}`, `{{.Domain}}`, `{{.Password}}`) | — |
| `PASSWORD_HOOK_METHOD` | HTTP method | `POST` |
| `PASSWORD_HOOK_CONTENT_TYPE` | Content-Type header | `application/x-www-form-urlencoded` |
| `PASSWORD_HOOK_BODY` | Request body (Go template) | — |
| `PASSWORD_HOOK_HEADERS` | JSON object of extra headers (values are Go templates) | `{}` |
| `PASSWORD_HOOK_TIMEOUT` | Request timeout in seconds | `10` |
| `PASSWORD_HOOK_SKIP_TLS_VERIFY` | Skip TLS certificate verification | `false` |

**Template variables:** `{{.Email}}` (full email), `{{.User}}` (part before @), `{{.Domain}}` (part after @), `{{.Password}}` (new plaintext password).

### Example: DirectAdmin email password sync

```yaml
PASSWORD_HOOK_ENABLED: "true"
PASSWORD_HOOK_URL: "https://your-server.com:2222/CMD_API_EMAIL_PW"
PASSWORD_HOOK_BODY: "user={{.User}}&domain={{.Domain}}&passwd={{.Password}}"
PASSWORD_HOOK_HEADERS: '{"Authorization":"Basic base64-encoded-credentials"}'
```

## SMS via CM.com

The existing webhook SMS provider can be configured to use CM.com's Messages API — no code changes needed:

```yaml
SMS_ENABLED: "true"
SMS_WEBHOOK_URL: "https://gw.cmtelecom.com/v1.0/message"
SMS_WEBHOOK_CONTENT_TYPE: "application/json"
SMS_WEBHOOK_BODY: '{"messages":{"authentication":{"producttoken":"YOUR_TOKEN"},"msg":[{"from":{"number":"TinyAuth"},"to":[{"number":"{{.To}}"}],"body":{"type":"AUTO","content":"{{.Message}}"}}]}}'
```

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
