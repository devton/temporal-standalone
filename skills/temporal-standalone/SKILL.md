---
name: temporal-standalone
description: Temporal Standalone — mini cloud local com Server 1.31.0, UI custom (overlays), Casdoor OIDC, PostgreSQL 18. Gerenciamento de API Keys, Standalone Activities, namespace operations.
version: 1.0.0
category: devops
---

# temporal-standalone

## Goal

Prover documentação agentic-friendly e executável para o ambiente Temporal Standalone, permitindo que qualquer agent operações (deploy, debug, migração, feature enablement) com confiança e zero-downtime.

## Orchestrator

### Classificação de Tarefas

| Intent | Workflow |
|---|---|
| "habilitar standalone activities", "ativar feature X" | → `workflows/enable-feature/SKILL.md` |
| "corrigir API keys", "persistir API keys", "bug auth" | → `workflows/fix-api-keys/SKILL.md` |
| "fazer upgrade do server", "migrar schema" | → `workflows/server-upgrade/SKILL.md` |
| "rebuild UI", "mudanças nos overlays" | → `workflows/ui-rebuild/SKILL.md` |
| "criar namespace", "listar namespaces" | → `workflows/namespace-ops/SKILL.md` |

### Validação (Close Gate)

Após qualquer operação:
1. `docker ps` — todos containers healthy
2. `docker logs <container> --tail 10` — sem erros
3. `temporal operator cluster health` — server respondendo
4. Se UI envolvido: `curl -sf http://192.168.2.68:8080/api/v1/settings` → 200

## Tech Stack

- Temporal Server 1.31.0 (`temporalio/server`)
- UI Server v2.49.x (`temporalio/ui` + overlays Go/Svelte)
- Casdoor 3.49.0 (OIDC)
- PostgreSQL 18
- Docker Compose

## Constraints

- **NUNCA** usar `temporalio/auto-setup` — re-roda schema migration a cada restart
- **NUNCA** modificar schema do PostgreSQL manualmente — usar `temporal-sql-tool`
- Schema migration SEMPRE via `temporal-sql-tool` no `admin-tools`
- Container `temporal-server` NÃO tem CLI — usar `temporal-setup` (admin-tools)
- Dynamic config só aceita keys oficiais — validar em docs antes de adicionar
- UI rebuild necessário após qualquer mudança em `ui-custom/overlays/`
- API Keys são in-memory (perde no restart) — bug conhecido, workflow `fix-api-keys`

## Architecture

```
192.168.2.68
├── :7233  Temporal Server (gRPC)
├── :8080  Temporal UI (custom overlays)
├── :8000  Casdoor (OIDC)
└── :5432  PostgreSQL
    ├── temporal          (workflow history, namespace metadata)
    ├── temporal_visibility (indexed search)
    └── casdoor           (OIDC users)
```

## Project Structure

```
~/projects/temporal-standalone/
├── docker-compose.yml           # Infra principal
├── docker-compose.override.yml  # UI custom build + env vars
├── .env                         # Config local (NÃO commitar)
├── config/temporal/dynamicconfig/docker.yaml
├── ui-custom/
│   ├── upstream/                # Git submodule (temporalio/ui)
│   ├── overlays/                # Customizações
│   │   ├── server/server/route/ # Go backend (api_keys, auth, namespace)
│   │   └── src/                 # Svelte frontend (api-keys page, menu)
│   └── Dockerfile.custom        # Build: node → go → alpine
├── scripts/                     # init-db.sh, setup-namespaces.sh
└── skills/temporal-standalone/  # Esta skill
```

## Internal Workflows

- [enable-feature](workflows/enable-feature/SKILL.md) — Habilitar namespace capabilities (Standalone Activities, etc)
- [fix-api-keys](workflows/fix-api-keys/SKILL.md) — Corrigir 3 bugs: in-memory store, JWT unverified, no server auth
- [server-upgrade](workflows/server-upgrade/SKILL.md) — Upgrade server + schema migration
- [ui-rebuild](workflows/ui-rebuild/SKILL.md) — Rebuild UI custom após mudanças nos overlays
- [namespace-ops](workflows/namespace-ops/SKILL.md) — CRUD namespaces via CLI/gRPC

## References

- [routing-matrix.md](reference/routing-matrix.md) — Mapa completo de rotas
- [api-keys-bugs.md](reference/api-keys-bugs.md) — Detalhamento dos 3 bugs e blueprints de correção
- [standalone-activities.md](reference/standalone-activities.md) — Feature e como habilitar
