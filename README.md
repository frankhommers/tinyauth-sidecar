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
