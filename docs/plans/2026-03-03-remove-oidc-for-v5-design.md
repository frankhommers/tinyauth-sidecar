# Remove OIDC Provider for Tinyauth v5 - Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Remove the sidecar's OIDC provider since tinyauth v5 now has integrated OIDC support, and update deployment configs for v5 compatibility.

**Architecture:** The sidecar keeps all non-OIDC functionality (user self-service, password webhooks, admin panel, Docker restart, SMS). The OIDC package, config, routes, and related dependencies are removed. Docker-compose files are updated to reference tinyauth v5.

**Tech Stack:** Go 1.24, Gin, TOML config, Docker Compose, Traefik

---

### Task 1: Remove OIDC package

**Files:**
- Delete: `internal/oidc/oidc.go`

**Step 1: Delete the OIDC package**

```bash
rm -rf internal/oidc/
```

**Step 2: Verify deletion**

```bash
ls internal/oidc/ 2>&1
# Expected: "No such file or directory"
```

**Step 3: Commit**

```bash
git add -A && git commit -m "remove: delete OIDC provider package (now in tinyauth v5)"
```

---

### Task 2: Remove OIDC config types from config.go

**Files:**
- Modify: `internal/config/config.go:163-188`

**Step 1: Remove OIDCConfig, OIDCClient types and OIDC field from FileConfig**

In `internal/config/config.go`, remove:
- `OIDCConfig` struct (lines 164-170)
- `OIDCClient` struct (lines 173-177)
- `OIDC OIDCConfig` field from `FileConfig` (line 188)
- Associated comments (lines 163, 172)

The `FileConfig` struct should become:

```go
type FileConfig struct {
	PasswordPolicy PasswordPolicy  `toml:"password_policy"`
	PasswordHooks  []WebhookConfig `toml:"password_hooks"`
	SMS            WebhookConfig   `toml:"sms"`
	Users          UsersConfig     `toml:"users"`
	SMTP           SMTPConfig          `toml:"smtp"`
	Email          EmailTemplateConfig `toml:"email"`
	UI             UIConfig            `toml:"ui"`
}
```

**Step 2: Verify it compiles (will fail due to main.go references, that's expected)**

```bash
go vet ./internal/config/
# Expected: PASS
```

**Step 3: Commit**

```bash
git add internal/config/config.go && git commit -m "remove: OIDC config types from config package"
```

---

### Task 3: Remove OIDC wiring from main.go

**Files:**
- Modify: `main.go:14,104-127`

**Step 1: Remove OIDC import and route setup**

Remove the import:
```go
"tinyauth-sidecar/internal/oidc"
```

Remove the OIDC block (lines 104-127):
```go
// OIDC provider (optional, enabled via config.toml)
if fileCfg.OIDC.Enabled {
    ...
}
```

**Step 2: Verify the full project compiles**

```bash
go build ./...
# Expected: success
```

**Step 3: Commit**

```bash
git add main.go && git commit -m "remove: OIDC provider wiring from main.go"
```

---

### Task 4: Remove golang-jwt dependency

**Files:**
- Modify: `go.mod`

**Step 1: Remove unused dependency**

```bash
go mod tidy
```

**Step 2: Verify golang-jwt is gone**

```bash
grep golang-jwt go.mod
# Expected: no output (dependency removed)
```

**Step 3: Verify build still works**

```bash
go build ./...
# Expected: success
```

**Step 4: Commit**

```bash
git add go.mod go.sum && git commit -m "chore: go mod tidy, remove unused golang-jwt dependency"
```

---

### Task 5: Update store comments

**Files:**
- Modify: `internal/store/toml_store.go:198,210`

**Step 1: Update comments that reference OIDC**

Change:
```go
// LookupName returns the display name for a user (for OIDC claims).
```
To:
```go
// LookupName returns the display name for a user.
```

Change:
```go
// LookupEmail returns the email address for a user (for OIDC claims).
```
To:
```go
// LookupEmail returns the email address for a user.
```

**Step 2: Commit**

```bash
git add internal/store/toml_store.go && git commit -m "docs: remove OIDC references from store comments"
```

---

### Task 6: Remove OIDC section from config.example.toml

**Files:**
- Modify: `config.example.toml:60-72`

**Step 1: Remove the entire OIDC section**

Remove lines 60-72 (from `# OIDC Provider` to the end of file).

**Step 2: Commit**

```bash
git add config.example.toml && git commit -m "remove: OIDC section from config.example.toml"
```

---

### Task 7: Update docker-compose.yml for tinyauth v5

**Files:**
- Modify: `docker-compose.yml`

**Step 1: Update tinyauth image to v5**

Change:
```yaml
image: ghcr.io/steveiliop56/tinyauth:v4
```
To:
```yaml
image: ghcr.io/steveiliop56/tinyauth:v5
```

**Step 2: Update tinyauth environment variables to v5 format**

The v5 env vars use `TINYAUTH_` prefix with section namespacing. Update:
- `TINYAUTH_APPURL` stays (already v5 format)
- `TINYAUTH_AUTH_USERSFILE` stays (already v5 format)
- `TINYAUTH_DISABLEANALYTICS` -> `TINYAUTH_ANALYTICS_ENABLED: "false"`
- `TINYAUTH_AUTH_SECURECOOKIE` stays (already v5 format)
- `TINYAUTH_UI_FORGOTPASSWORDMESSAGE` stays (already v5 format)

**Step 3: Commit**

```bash
git add docker-compose.yml && git commit -m "chore: update docker-compose.yml for tinyauth v5"
```

---

### Task 8: Update docker-compose.example.yml for tinyauth v5

**Files:**
- Modify: `docker-compose.example.yml`

**Step 1: Update tinyauth image to v5**

Change image from `v4` to `v5`.

**Step 2: Update tinyauth environment variables to v5 format**

Migrate all env vars to v5 naming:
- `APP_URL` -> `TINYAUTH_APPURL`
- `DATABASE_PATH` -> `TINYAUTH_DATABASE_PATH`
- `DISABLE_ANALYTICS` -> `TINYAUTH_ANALYTICS_ENABLED: "false"`
- `LOG_LEVEL` -> `TINYAUTH_LOG_LEVEL`
- `PORT` -> `TINYAUTH_SERVER_PORT`
- `ADDRESS` -> `TINYAUTH_SERVER_ADDRESS`
- `USERS_FILE` -> `TINYAUTH_AUTH_USERSFILE`
- `SECURE_COOKIE` -> `TINYAUTH_AUTH_SECURECOOKIE`
- `SESSION_EXPIRY` -> `TINYAUTH_AUTH_SESSIONEXPIRY`
- `LOGIN_TIMEOUT` -> `TINYAUTH_AUTH_LOGINTIMEOUT`
- `LOGIN_MAX_RETRIES` -> `TINYAUTH_AUTH_LOGINMAXRETRIES`
- `APP_TITLE` -> `TINYAUTH_UI_TITLE`
- `FORGOT_PASSWORD_MESSAGE` -> `TINYAUTH_UI_FORGOTPASSWORDMESSAGE`

**Step 3: Remove `/oidc` from Traefik routing rule**

Change sidecar Traefik rule from:
```yaml
Host(`auth.hommers.nl`) && (PathPrefix(`/manage`) || PathPrefix(`/oidc`))
```
To:
```yaml
Host(`auth.hommers.nl`) && PathPrefix(`/manage`)
```

**Step 4: Remove experimental config file command (v5 has native config)**

Remove:
```yaml
command: ["--experimental.configFile=/data/tinyauth.yaml"]
```

**Step 5: Commit**

```bash
git add docker-compose.example.yml && git commit -m "chore: update docker-compose.example.yml for tinyauth v5, remove OIDC routing"
```

---

### Task 9: Update README.md

**Files:**
- Modify: `README.md`

**Step 1: Update version reference**

Change line 3 from "tinyauth (v4)" to "tinyauth (v5)".

**Step 2: Remove any OIDC mentions (if present)**

Check for and remove any OIDC references in the README. Currently there are none in the feature list or API docs, but verify.

**Step 3: Commit**

```bash
git add README.md && git commit -m "docs: update README for tinyauth v5"
```

---

### Task 10: Final verification

**Step 1: Verify clean build**

```bash
go build ./...
```

**Step 2: Verify no remaining OIDC references in Go code**

```bash
grep -r "oidc\|OIDC" --include="*.go" .
# Expected: no output (or only in go.sum which is OK)
```

**Step 3: Verify no remaining OIDC references in config files**

```bash
grep -r "oidc\|OIDC" --include="*.toml" --include="*.yml" --include="*.yaml" .
# Expected: no output
```
