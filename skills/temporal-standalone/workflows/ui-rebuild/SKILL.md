---
name: ui-rebuild
description: Rebuild da UI customizada após mudanças nos overlays Go/Svelte
---

# ui-rebuild

## Goal

Rebuildar a imagem Docker da UI customizada após alterações nos overlays (Go backend ou Svelte frontend).

## Scope

- Aplica-se a: qualquer mudança em `ui-custom/overlays/`
- NÃO cobre: mudanças no `upstream/` (git submodule — não editamos)

## Triggers

- "rebuild UI"
- "mudanças nos overlays"
- "aplicar mudanças no Go/Svelte"

## Inputs

- Nenhum obrigatório

## Invariants

- **SEMPRE** `docker compose build` antes de `up`
- Build é multi-stage: node → go → alpine (pode demorar 3-5 min)
- API keys in-memory serão **perdidas** no restart do container
- Não editar arquivos em `upstream/` — são do git submodule

## Procedure

1. **Verificar mudanças**
   ```bash
   cd ~/projects/temporal-standalone
   git status ui-custom/overlays/
   ```

2. **Build**
   ```bash
   docker compose build temporal-ui
   ```
   - Stage 1: Frontend (pnpm install + build:server) — ~2 min
   - Stage 2: Backend (go mod + make build-server) — ~1 min
   - Stage 3: Copy to alpine — instant

3. **Deploy**
   ```bash
   docker compose up -d temporal-ui
   ```

4. **Verificar**
   ```bash
   docker logs temporal-ui --tail 20
   curl -sf http://192.168.2.68:8080/api/v1/settings | head -1
   ```

5. **Smoke test**
   - Abrir `http://192.168.2.68:8080` no browser
   - Fazer login via Casdoor
   - Navegar para Settings > API Keys
   - Verificar se a página carrega

## Outputs

- Container `temporal-ui` rodando com código atualizado
- Logs sem erros

## Review Gate

- [ ] Container healthy
- [ ] `curl /api/v1/settings` retorna 200
- [ ] Login Casdoor funciona
- [ ] API Keys page carrega
- [ ] Logs sem Go panic ou Svelte errors

## References

- [Dockerfile.custom](../../../ui-custom/Dockerfile.custom)
