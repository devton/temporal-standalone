---
name: fix-api-keys
description: Corrigir 3 bugs críticos na implementação de API Keys (persistência, JWT verification, server auth)
---

# fix-api-keys

## Goal

Corrigir a implementação de API Keys do Temporal UI customizado, resolvendo 3 bugs: store in-memory, JWT sem verificação, e keys que não autenticam no Temporal Server.

## Scope

- Aplica-se a: `ui-custom/overlays/server/server/route/api_keys.go` e `auth_middleware.go`
- NÃO cobre: mudanças no Temporal Server ou Casdoor

## Triggers

- "corrigir API keys"
- "persistir API keys"
- "bug no auth"
- "JWT sem verificação"

## Inputs

- `fixTarget`: qual bug corrigir primeiro (default: `all` = Bug#1 → Bug#2 → Bug#3)
- `dbURL`: PostgreSQL connection string (default: do compose env)

## Invariants

- NÃO quebrar o fluxo OIDC existente (cookies + Casdoor)
- NÃO modificar o Temporal Server
- SEMPRE rebuild UI após mudanças: `docker compose build temporal-ui && docker compose up -d temporal-ui`
- SEMPRE testar com usuário logado via Casdoor após rebuild
- Fazer backup dos arquivos antes de modificar

## Procedure

### Bug #1: In-Memory → PostgreSQL

1. **Criar migration SQL**
   - Criar `scripts/migrations/001_create_api_keys.sql`
   - Tabela: `api_keys` com columns id, name, description, key_id, key_secret, owner_id, created_at, expires_at, last_used_at

2. **Refatorar `api_keys.go`**
   - Substituir `var apiKeysStore map[string]*APIKey` por `type APIKeyStore struct { db *sql.DB }`
   - `NewAPIKeyStore(dbURL string)` — init + auto-migrate
   - `ListAPIKeys` → `SELECT * FROM api_keys WHERE owner_id = $1`
   - `CreateAPIKey` → `INSERT INTO api_keys ...`
   - `DeleteAPIKey` → `DELETE FROM api_keys WHERE id = $1 AND owner_id = $2`
   - Adicionar `_ "github.com/lib/pq"` nos imports

3. **Atualizar `api.go`**
   - Init store em `SetAPIRoutes`: `store, err := NewAPIKeyStore(os.Getenv("API_KEYS_DB_URL"))`
   - Passar store para handlers ao invés de var global

4. **Atualizar compose override**
   - Adicionar env var `API_KEYS_DB_URL`

5. **Rebuild + test**
   ```bash
   docker compose build temporal-ui && docker compose up -d temporal-ui
   # Test: criar key, restart container, verificar se key persiste
   ```

### Bug #2: JWT Verification

1. **Criar `auth_jwt.go`** (novo arquivo no mesmo package)
   - `verifyAPIKeyJWT(tokenString string) (*UserInfo, error)` — HS256 com JWT_SECRET
   - `verifyOIDCToken(tokenString string) (*UserInfo, error)` — JWKS do Casdoor

2. **Atualizar `auth_middleware.go`**
   - Cookies → manter `parseUnverifiedJWT` (server-set, safe)
   - Authorization-Extras → `verifyOIDCToken`
   - Authorization Bearer → detectar issuer:
     - `temporal-standalone` → `verifyAPIKeyJWT`
     - outro → `verifyOIDCToken`

### Bug #3: Server Auth (decisão pendente)

Escolher entre Opção A (mTLS), B (UI Proxy) ou C (HS256 no server). Ver [api-keys-bugs.md](../../reference/api-keys-bugs.md).

## Outputs

- API keys persistidas em PostgreSQL
- JWT verification real (HS256 + JWKS)
- API keys funcional para o propósito definido

## Review Gate

- [ ] Criar API key → restart container → key ainda existe
- [ ] JWT forjado (claims falsas) → rejeitado (401)
- [ ] OIDC login via Casdoor continua funcionando
- [ ] Authorization Bearer com API key JWT funciona
- [ ] Frontend Settings > API Keys carrega sem erros
- [ ] `docker logs temporal-ui --tail 20` sem erros

## References

- [api-keys-bugs.md](../../reference/api-keys-bugs.md) — Detalhamento completo dos 3 bugs
- [routing-matrix.md](../../reference/routing-matrix.md)
