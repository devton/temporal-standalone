---
name: server-upgrade
description: Upgrade Temporal Server + schema migration com zero-downtime
---

# server-upgrade

## Goal

Atualizar o Temporal Server para uma nova versão, incluindo schema migration do PostgreSQL, sem perda de dados e com mínimo downtime.

## Scope

- Aplica-se a: upgrade do `temporalio/server`, schema migration
- NÃO cobre: UI upgrades (ver `ui-rebuild`), Casdoor upgrades

## Triggers

- "upgrade temporal server"
- "migrar schema"
- "atualizar versão do temporal"

## Inputs

- `targetVersion`: versão alvo (ex: `1.32.0`)
- `schemaFrom`: versão atual do schema (default: verificar automaticamente)

## Invariants

- **SEMPRE** fazer backup do PostgreSQL antes de migrar
- **NUNCA** usar `temporalio/auto-setup` para migration
- Schema migration via `temporal-sql-tool` no `admin-tools`
- Preservar dados — containers PostgreSQL NÃO são recriados
- Verificar compatibility matrix antes de pular versões
- Fazer upgrade incremental (uma major/minor por vez)

## Procedure

1. **Pre-flight checks**
   ```bash
   # Verificar versão atual
   docker inspect temporal-server --format '{{.Config.Image}}'
   # Verificar schema version
   docker exec temporal-postgres psql -U temporal -d temporal -c "SELECT * FROM schema_version"
   # Verificar health
   docker ps --format "table {{.Names}}\t{{.Status}}"
   ```

2. **Backup PostgreSQL**
   ```bash
   docker exec temporal-postgres pg_dumpall -U temporal > ~/temporal-backup-$(date +%Y%m%d).sql
   ```

3. **Pull nova imagem**
   ```bash
   docker compose pull temporal
   ```

4. **Parar server (graceful)**
   ```bash
   docker stop temporal-server
   docker rm temporal-server
   ```

5. **Schema migration — temporal DB**
   ```bash
   docker run --rm --network temporal-network \
     -e DB=postgres12 -e DB_PORT=5432 \
     -e POSTGRES_USER=temporal -e POSTGRES_PWD=temporal \
     -e POSTGRES_SEEDS=postgresql -e DBNAME=temporal \
     temporalio/admin-tools:latest \
     temporal-sql-tool update-schema \
       -d /etc/temporal/schema/postgresql/v12/temporal/versioned
   ```

6. **Schema migration — visibility DB**
   ```bash
   docker run --rm --network temporal-network \
     -e DB=postgres12 -e DB_PORT=5432 \
     -e POSTGRES_USER=temporal -e POSTGRES_PWD=temporal \
     -e POSTGRES_SEEDS=postgresql -e DBNAME=temporal_visibility \
     temporalio/admin-tools:latest \
     temporal-sql-tool update-schema \
       -d /etc/temporal/schema/postgresql/v12/visibility/versioned
   ```

7. **Recriar container**
   ```bash
   docker compose create temporal && docker start temporal-server
   ```

8. **Verificar health**
   ```bash
   docker ps --format "table {{.Names}}\t{{.Status}}"
   docker logs temporal-server --tail 20
   docker exec temporal-setup temporal operator cluster health --address temporal:7233
   ```

## Outputs

- Server rodando na nova versão
- Schema migrado
- Workflows existentes intactos

## Review Gate

- [ ] Server container healthy
- [ ] `temporal operator cluster health` OK
- [ ] Schema version correta no PostgreSQL
- [ ] Workflows existentes visíveis no UI
- [ ] Namespaces intactos
- [ ] Casdoor ainda funcionando (login)

## References

- Temporal upgrade docs: https://docs.temporal.io/cluster-deployment#upgrade
