# Connect OIDC Middleware with API Keys

## Status: DONE

## Problem

The API Keys code (`overlays/server/server/route/api_keys.go`) depends on `getOwnerIDFromContext()` to identify the API key owner, but the Temporal UI OIDC middleware **does not populate the context** with JWT claims.

```go
func getOwnerIDFromContext(c echo.Context) string {
    if user := c.Get("user"); user != nil {
        // Extract owner_id from claims...
    }
    return "default-user"  // Fallback - ALWAYS returns this!
}
```

Result: all API keys are created with `ownerId = "default-user"`.

## Solution Implemented

### AuthMiddleware (Option A - IMPLEMENTED)

Created `overlays/server/server/route/auth_middleware.go`:

1. **Extracts user from cookies**: Reads OIDC session cookies (user0, user1, etc.) and parses `UserResponse`
2. **Extracts user from Authorization header**: Validates JWT via `tokenVerifier` when Bearer token present
3. **Populates Echo context**: Sets `c.Set("user", userInfo)` with `UserInfo{Subject, Email, Name}`
4. **Debug logging**: Added logs for troubleshooting auth flow

### Changes Made

| File | Action |
|------|--------|
| `overlays/server/server/route/auth_middleware.go` | Created - AuthMiddleware with cookie/header extraction |
| `overlays/server/server/route/api_keys.go` | Modified `getOwnerIDFromContext()` to use UserInfo from context |
| `overlays/server/server/route/api.go` | Modified `RegisterAPIKeyRoutes()` to pass cfgProvider |
| `overlays/server/server/auth/auth.go` | Added `GetVerifier()` to expose OIDC verifier |
| `docker-compose.yml` | Removed volume mount that overrides built-in config |
| `docker-compose.override.yml` | Updated for custom UI build |

## Testing

```bash
# Rebuild and restart
docker compose build temporal-ui
docker compose up -d

# Test API Keys endpoint (unauthenticated - falls back to "default-user")
curl http://192.168.2.68:8080/api/v1/api-keys
# Returns: {"keys":[]}

# Check middleware logs
docker logs temporal-ui 2>&1 | grep AuthMiddleware
# Shows: [AuthMiddleware] Processing request: GET /api/v1/api-keys
# Shows: [AuthMiddleware] No user from cookies: ...
# Shows: [AuthMiddleware] No Authorization header found
```

## Next Steps

To fully test with authenticated user:
1. Login via UI (OIDC flow at `/auth/sso`)
2. Browser will have session cookies
3. API Keys will be associated with logged-in user's `sub` claim

## References

- `upstream/server/server/auth/auth.go` - OIDC validation
- `upstream/server/server/auth/oidc.go` - OIDC configuration
- `upstream/server/server/config/auth.go` - Auth config
