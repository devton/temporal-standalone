# API Keys — Bug Report & Fix Blueprints

## Bug #1: In-Memory Store (CRITICAL)

**File**: `ui-custom/overlays/server/server/route/api_keys.go:46`
```go
var apiKeysStore = make(map[string]*APIKey)
```

**Impact**: Todas as API keys são perdidas a cada restart do `temporal-ui`.

### Fix: PostgreSQL Persistence

#### Step 1 — Migration SQL

```sql
-- scripts/migrations/001_create_api_keys.sql
CREATE TABLE IF NOT EXISTS api_keys (
    id           VARCHAR(64) PRIMARY KEY,
    name         VARCHAR(255) NOT NULL,
    description  TEXT,
    key_id       VARCHAR(64) NOT NULL UNIQUE,
    key_secret   TEXT NOT NULL,
    owner_id     VARCHAR(255) NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at   TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ
);
CREATE INDEX idx_api_keys_owner ON api_keys(owner_id);
```

#### Step 2 — Refactor `api_keys.go`

Replace `var apiKeysStore` with:

```go
type APIKeyStore struct {
    db *sql.DB
}

func NewAPIKeyStore(dbURL string) (*APIKeyStore, error) {
    db, err := sql.Open("postgres", dbURL)
    if err != nil {
        return nil, err
    }
    if err := runMigration(db); err != nil {
        return nil, err
    }
    return &APIKeyStore{db: db}, nil
}
```

All CRUD methods (`ListAPIKeys`, `CreateAPIKey`, `DeleteAPIKey`) switch from map operations to SQL queries.

#### Step 3 — Wire in `api.go`

```go
// In SetAPIRoutes, before RegisterAPIKeyRoutes:
store, err := NewAPIKeyStore(os.Getenv("API_KEYS_DB_URL"))
RegisterAPIKeyRoutesWithStore(route, cfgProvider, store)
```

#### Step 4 — Docker compose override

```yaml
environment:
  - API_KEYS_DB_URL=postgres://temporal:temporal@postgresql:5432/temporal?sslmode=disable
```

---

## Bug #2: JWT Without Signature Verification (HIGH)

**File**: `ui-custom/overlays/server/server/route/auth_middleware.go:113-156`

```go
func parseUnverifiedJWT(tokenString string) (*UserInfo, error) {
    // Only base64 decodes payload — NO signature verification
}
```

**Impact**: Any forged JWT with valid `sub`/`email` claims passes authentication. An attacker can craft arbitrary tokens.

### Fix: Differentiated Token Verification

The middleware must distinguish 3 token types:

1. **OIDC session cookies** — set by server's own OIDC flow (cookies are httpOnly, server-side). Safe to parse without verify since only the server sets them.

2. **OIDC tokens (Authorization-Extras / Bearer)** — Must verify via JWKS from Casdoor:
   ```go
   // Fetch JWKS from http://temporal-casdoor:8000/.well-known/jwks
   // Use jwt.Parse() with keyfunc that fetches Casdoor's public keys
   ```

3. **API Key JWTs (Bearer)** — Must verify via HS256 with JWT_SECRET:
   ```go
   // Check issuer == "temporal-standalone"
   // Verify with jwt.Parse(token, keyfunc returning JWT_SECRET)
   ```

### Suggested Flow

```
Request arrives
├── Has OIDC cookies? → parseUnverifiedJWT (ok, server-set)
├── Has Authorization-Extras? → verifyOIDCToken (JWKS)
├── Has Authorization Bearer?
│   ├── issuer == "temporal-standalone"? → verifyAPIKeyToken (HS256 + JWT_SECRET)
│   ├── looks like OIDC? → verifyOIDCToken (JWKS)
│   └── unknown → 401
└── None → 401
```

---

## Bug #3: API Keys Don't Authenticate to Temporal Server (WON'T FIX)

**Impact**: API keys signed with `JWT_SECRET` (HS256) are accepted by the UI server but NOT by Temporal Server (`:7233`), which validates via Casdoor JWKS (RS256).

**Decision (2025-05-26): UI-only.** API keys are designed for the Temporal UI HTTP API only. They do NOT grant access to the Temporal Server gRPC endpoint (`:7233`). Rationale:

1. Adding HS256 as a second JWT key source on Temporal Server conflicts with existing Casdoor RS256 auth
2. UI proxy layer adds complexity and latency unnecessary for a local dev standalone
3. The UI server already proxies all Temporal API calls with valid OIDC tokens
4. Temporal Cloud itself uses API keys for HTTP API, not for direct gRPC — same pattern

**Resolution**: Frontend updated to document this scope. Bug #3 closed as "by design".
