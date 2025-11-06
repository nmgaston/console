# Auth Configuration Fix

## Problem
The service was always returning 401 Unauthorized on Redfish endpoints despite `auth.disabled: true` in `config/config.yml`.

## Root Cause
The `.env` file is **not automatically loaded** by the application. The `cleanenv.ReadEnv()` function only reads **actual shell environment variables**, not `.env` files. Although `github.com/joho/godotenv` is in `go.mod`, it's never invoked.

## Solution
Export the `AUTH_DISABLED` environment variable before running the service:

```bash
export AUTH_DISABLED=true
go run cmd/app/main.go
```

Or for Docker/systemd:
```bash
# In docker-compose.yml
environment:
  - AUTH_DISABLED=true

# In systemd service file
Environment="AUTH_DISABLED=true"
```

## For CI/CD (GitHub Actions)
Update `.github/workflows/api-test.yml`:

```yaml
- name: Start Console Service
  run: |
    export AUTH_DISABLED=true
    go run cmd/app/main.go &
    sleep 5
```

## Alternative: Load .env Automatically
To make the `.env` file work automatically, add to `config/config.go` (before `cleanenv.ReadConfig`):

```go
import "github.com/joho/godotenv"

// Load .env file if it exists (ignore errors if file doesn't exist)
_ = godotenv.Load()
```

## Verification
Check the logs for these messages:
```
DEBUG: cfg.Auth.Disabled value (disabled=true)
DEBUG: Skipping JWT middleware creation (auth is DISABLED)
Redfish v1 routes setup complete without authentication
```

If you see "Redfish v1 routes protected setup complete" instead, auth is still enabled.
