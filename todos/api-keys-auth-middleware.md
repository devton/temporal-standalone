# Connect OIDC Middleware with API Keys

## Status: TODO

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

## Solution

### Option A: Create custom middleware (RECOMMENDED)

1. Create file `overlays/server/server/route/api_keys_middleware.go`:
   - Extract JWT from `Authorization` header or cookie
   - Validate via `tokenVerifier` (already exists in auth.go)
   - Populate context with claims: `c.Set("user", claims)`
   - Extract `sub` (subject) as `ownerId`

2. Register middleware on API Keys routes:
   ```go
   func RegisterAPIKeyRoutes(e *echo.Group) {
       api := e.Group("/api-keys", AuthMiddleware)
       // ...
   }
   ```

### Option B: Read claims directly from header/cookie

Modify `getOwnerIDFromContext()` to:
- Read `Authorization-Extras` header (where frontend puts ID token)
- Decode JWT without validating (or validate inline)
- Extract `sub` claim

**Disadvantage:** duplicates validation logic.

## Files Involved

| File | Action |
|------|--------|
| `overlays/server/server/route/api_keys.go` | Modify `getOwnerIDFromContext()` |
| `overlays/server/server/route/api_keys_middleware.go` | Create (new) |
| `overlays/server/server/route/api.go` | Check how to register middleware |

## Dependencies

- [ ] Check how Temporal UI registers middlewares
- [ ] Check if `tokenVerifier` is globally accessible
- [ ] Test with real JWT token from Casdoor

## References

- `upstream/server/server/auth/auth.go` - OIDC validation
- `upstream/server/server/auth/oidc.go` - OIDC configuration
- `upstream/server/server/config/auth.go` - Auth config
