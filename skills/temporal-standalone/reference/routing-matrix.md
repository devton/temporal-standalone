# Routing Matrix — Temporal Standalone

## Task → Workflow Mapping

| Task Intent | Route To | Danger Level |
|---|---|---|
| Habilitar Standalone Activities | `workflows/enable-feature/SKILL.md` | Medium (namespace capability via gRPC) |
| Habilitar qualquer feature/capability | `workflows/enable-feature/SKILL.md` | Medium |
| Corrigir API Keys (persistência) | `workflows/fix-api-keys/SKILL.md` | High (requer rebuild UI) |
| Corrigir auth/JWT | `workflows/fix-api-keys/SKILL.md` | High (requer rebuild UI) |
| Upgrade Temporal Server | `workflows/server-upgrade/SKILL.md` | Critical (schema migration) |
| Migrar schema | `workflows/server-upgrade/SKILL.md` | Critical |
| Rebuild UI custom | `workflows/ui-rebuild/SKILL.md` | Low (container restart) |
| Modificar overlays Go/Svelte | `workflows/ui-rebuild/SKILL.md` | Low |
| Criar/gerenciar namespaces | `workflows/namespace-ops/SKILL.md` | Low |
| Debug container crash | `workflows/server-upgrade/SKILL.md` (schema check) | Medium |

## Containers & Responsibilities

| Container | Image | Role | Restart Impact |
|---|---|---|---|
| `temporal-server` | `temporalio/server:1.31.0` | Workflow engine | Workflows pausam, retomam ao subir |
| `temporal-postgres` | `postgres:18` | Persistência | **NÃO restartar** — dados em volume |
| `temporal-casdoor` | `casbin/casdoor:3.49.0` | OIDC | Login quebra temporariamente |
| `temporal-ui` | `temporal-ui-custom:latest` | UI + API Keys | Keys in-memory **perdidas** no restart |
| `temporal-setup` | `temporalio/admin-tools` | Namespace setup | One-shot, re-roda a cada `docker compose up` |

## Port Mapping

| Host | Container | Protocol |
|---|---|---|
| `192.168.2.68:7233` | `temporal-server:7233` | gRPC |
| `192.168.2.68:8080` | `temporal-ui:8080` | HTTP |
| `192.168.2.68:8000` | `temporal-casdoor:8000` | HTTP |
| `192.168.2.68:5432` | `temporal-postgres:5432` | PostgreSQL |
