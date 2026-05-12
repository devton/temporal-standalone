---
name: namespace-ops
description: Operações de namespace — criar, listar, descrever, atualizar via CLI/gRPC
---

# namespace-ops

## Goal

Gerenciar namespaces no Temporal Standalone: listar, criar, descrever, atualizar configurações.

## Scope

- Aplica-se a: CRUD de namespaces
- NÃO cobre: namespace capabilities (ver `enable-feature`)

## Triggers

- "criar namespace"
- "listar namespaces"
- "descrever namespace"
- "namespace operations"

## Inputs

- `action`: list | describe | create | update (default: `list`)
- `namespace`: nome do namespace (quando aplicável)

## Invariants

- **SEMPRE** usar `temporal-setup` (admin-tools) como container CLI — `temporal-server` não tem CLI
- Namespace padrão DOU: `3b6ae4b6-3dc9-4321-89be-f83a0427d3e9`
- Namespaces auto-criados pelo `NamespaceEnsurer` usam UUID como nome
- Retention padrão: 30 dias

## Procedure

### List

```bash
docker exec temporal-setup temporal operator namespace list \
  --address temporal:7233
```

### Describe

```bash
docker exec temporal-setup temporal operator namespace describe <name> \
  --address temporal:7233
```

### Create

```bash
docker exec temporal-setup temporal operator namespace create <name> \
  --address temporal:7233 \
  --retention 30d \
  --description "Description here"
```

Ou via API customizada (auto-cria com UUID):
```bash
curl -X POST http://192.168.2.68:8080/api/v1/user/ensure-namespace \
  -H "Authorization-Extras: <id-token>"
```

### Update

```bash
docker exec temporal-setup temporal operator namespace update <name> \
  --address temporal:7233 \
  --retention 30d
```

## Outputs

- Namespace criado/alterado conforme solicitado
- Output do comando CLI

## Review Gate

- [ ] Namespace aparece no `list`
- [ ] `describe` mostra configurações corretas
- [ ] UI mostra o namespace

## References

- [routing-matrix.md](../../reference/routing-matrix.md)
