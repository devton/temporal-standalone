---
name: enable-feature
description: Habilitar namespace capabilities (Standalone Activities, etc) via gRPC
---

# enable-feature

## Goal

Habilitar features/capabilities de namespace no Temporal Standalone (ex: Standalone Activities).

## Scope

- Aplica-se a: habilitar qualquer namespace capability via gRPC UpdateNamespace
- NÃO cobre: dynamic config changes (que usam arquivo yaml)

## Triggers

- "habilitar standalone activities"
- "ativar feature X no namespace"
- "enable standalone activity"

## Inputs

- `namespace`: namespace alvo (default: `default`)
- `capability`: nome da capability (ex: `standaloneActivities`)
- `method`: grpcurl | go-script | cli (default: `grpcurl`)

## Invariants

- NÃO restartar o server para habilitar capabilities
- NÃO modificar dynamic config para features que são namespace capabilities
- Sempre verificar se a capability já está habilitada antes de tentar habilitar
- Testar com `temporal operator namespace describe` após habilitar

## Procedure

1. **Verificar pré-requisitos**
   ```bash
   # Server version >= required
   docker exec temporal-server temporal-server version 2>/dev/null || \
     docker inspect temporal-server --format '{{.Config.Image}}'
   ```

2. **Verificar estado atual**
   ```bash
   docker exec temporal-setup temporal operator namespace describe <namespace> \
     --address temporal:7233 2>&1
   ```

3. **Habilitar via gRPC**
   ```bash
   # Tentativa via grpcurl
   grpcurl -plaintext \
     -d '{"namespace":"<namespace>","updateMask":{"paths":["postUpdateSpec"]},"postUpdateSpec":{"capabilities":{"<capability>":true}}}' \
     192.168.2.68:7233 \
     temporal.api.workflowservice.v1.WorkflowService/UpdateNamespace
   ```

4. **Verificar resultado**
   ```bash
   docker exec temporal-setup temporal operator namespace describe <namespace> \
     --address temporal:7233 2>&1 | grep -i <capability>
   ```

5. **Testar a feature**
   ```bash
   # Para Standalone Activities:
   docker exec temporal-setup temporal activity list \
     --address temporal:7233 --namespace <namespace>
   ```

## Outputs

- Capability habilitada no namespace
- Verificação via `namespace describe` confirmada
- Log do resultado

## Review Gate

- [ ] `namespace describe` mostra a capability habilitada
- [ ] Comando de teste da feature funciona (ex: `temporal activity list`)
- [ ] Nenhum container crashou
- [ ] Workflows existentes continuam rodando

## References

- [standalone-activities.md](../../reference/standalone-activities.md)
- [routing-matrix.md](../../reference/routing-matrix.md)
