# Temporal Standalone

Deploy standalone do Temporal com PostgreSQL, MinIO para archive e UI com autenticação.

## Arquitetura

```
┌─────────────────────────────────────────────────────────────────┐
│                     Temporal Standalone                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────┐     ┌──────────────┐     ┌──────────────┐   │
│  │  Temporal UI │────▶│   Temporal    │────▶│  PostgreSQL  │   │
│  │   :8080      │     │   Server      │     │   :5432      │   │
│  │              │     │   :7233       │     │              │   │
│  └──────────────┘     └──────┬───────┘     └──────────────┘   │
│                              │                                  │
│                              ▼                                  │
│                       ┌──────────────┐                        │
│                       │    MinIO     │                        │
│                       │  (Archive)   │                        │
│                       │ :9000, :9001 │                        │
│                       └──────────────┘                        │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## Componentes

| Serviço | Porta | Descrição |
|---------|-------|-----------|
| Temporal Server | 7233 | Motor de workflows |
| Temporal UI | 8080 | Interface web |
| PostgreSQL | 5432 | Banco de dados |
| MinIO | 9000, 9001 | Object storage para archive |
| Admin Tools | - | CLI temporal (profile: admin) |

## Requisitos

- Docker Engine 20.10+
- Docker Compose 2.0+
- 4GB+ RAM disponível

## Quick Start

```bash
# 1. Clonar/copiar o projeto
cd temporal-standalone

# 2. Criar arquivo .env
cp .env.example .env

# 3. Editar senhas no .env
nano .env

# 4. Iniciar serviços
docker compose up -d

# 5. Verificar status
docker compose ps

# 6. Acessar UI
open http://localhost:8080
```

## Configuração de Autenticação

### Modo Local (Desenvolvimento)

No `.env`:
```
AUTH_TYPE=local
```

Qualquer usuário pode acessar sem autenticação.

### Modo OIDC/OAuth (Produção)

Configure um provider OIDC (Auth0, Keycloak, Okta, etc):

```
AUTH_TYPE=oidc
AUTH_PROVIDER_URL=https://your-tenant.auth0.com
AUTH_CLIENT_ID=your-client-id
AUTH_CLIENT_SECRET=your-client-secret
AUTH_CALLBACK_URL=http://localhost:8080/auth/callback
AUTH_SCOPES=openid email profile
AUTH_ISSUER=https://your-tenant.auth0.com
```

#### Providers Suportados

**Auth0:**
```
AUTH_PROVIDER_URL=https://your-tenant.auth0.com
AUTH_CLIENT_ID=...
AUTH_CLIENT_SECRET=...
AUTH_CALLBACK_URL=http://localhost:8080/auth/callback
```

**Keycloak:**
```
AUTH_PROVIDER_URL=https://keycloak.example.com/realms/your-realm
AUTH_CLIENT_ID=temporal-ui
AUTH_CLIENT_SECRET=...
AUTH_CALLBACK_URL=http://localhost:8080/auth/callback
```

**Okta:**
```
AUTH_PROVIDER_URL=https://your-org.okta.com/oauth2/default
AUTH_CLIENT_ID=...
AUTH_CLIENT_SECRET=...
AUTH_CALLBACK_URL=http://localhost:8080/auth/callback
```

## Retenção de Dados

Configurada para **30 dias** por padrão.

### Personalizar Retenção

Edite `config/dynamicconfig/development.yaml`:

```yaml
history.retention:
  - value: "2592000s"  # 30 dias
    constraints: {}
```

Valores comuns:
- 7 dias: `604800s`
- 30 dias: `2592000s`
- 90 dias: `7776000s`

## Archive

O archive está habilitado e usa MinIO como backend.

### Acessar MinIO Console

```bash
# URL: http://localhost:9001
# User: temporal
# Password: (definido no .env como MINIO_PASSWORD)
```

### Verificar Archive

```bash
# Usando mc (MinIO Client)
docker run --rm -it --network temporal-network \
  minio/mc alias set myminio http://minio:9000 temporal temporal_archive_password_change_me

# Listar arquivos arquivados
docker run --rm -it --network temporal-network \
  minio/mc ls myminio/temporal-archive/
```

## Namespaces

### Criar Namespace

```bash
# Usando temporal CLI
docker compose exec temporal-admin-tools temporal operator namespace create \
  --retention 30d \
  --description "Namespace de produção" \
  producao
```

### Listar Namespaces

```bash
docker compose exec temporal-admin-tools temporal operator namespace list
```

### Namespace com Archive

```bash
docker compose exec temporal-admin-tools temporal operator namespace create \
  --retention 30d \
  --archive \
  --description "Namespace com archive" \
  arquivado
```

## Uso do Admin Tools

O container `temporal-admin-tools` oferece a CLI completa:

```bash
# Iniciar container de admin
docker compose --profile admin up -d temporal-admin-tools

# Executar comandos
docker compose exec temporal-admin-tools temporal workflow list
docker compose exec temporal-admin-tools temporal workflow describe -w <workflow-id>
docker compose exec temporal-admin-tools temporal workflow cancel -w <workflow-id>
```

## Monitoramento

### Health Checks

```bash
# PostgreSQL
docker compose exec postgresql pg_isready -U temporal

# Temporal Server
docker compose exec temporal temporal operator cluster health

# MinIO
curl http://localhost:9000/minio/health/live
```

### Logs

```bash
# Todos os serviços
docker compose logs -f

# Serviço específico
docker compose logs -f temporal
docker compose logs -f temporal-ui
```

## Backup e Restore

### Backup PostgreSQL

```bash
docker compose exec postgresql pg_dump -U temporal temporal > backup_temporal.sql
docker compose exec postgresql pg_dump -U temporal temporal_visibility > backup_visibility.sql
```

### Restore PostgreSQL

```bash
cat backup_temporal.sql | docker compose exec -T postgresql psql -U temporal temporal
cat backup_visibility.sql | docker compose exec -T postgresql psql -U temporal temporal_visibility
```

### Backup MinIO (Archive)

```bash
docker run --rm -v minio_data:/data -v $(pwd)/backup:/backup \
  alpine tar czf /backup/archive-backup.tar.gz -C /data .
```

## Escalabilidade

### Aumentar History Shards

Edite `config/config.yaml`:

```yaml
persistence:
  numHistoryShards: 16  # Padrão: 4
```

**Importante:** Mudança requer recriação do banco de dados.

### Múltiplas Réplicas

```yaml
services:
  temporal:
    deploy:
      replicas: 3
```

## Troubleshooting

### Erro: "connection refused"

```bash
# Verificar se PostgreSQL está pronto
docker compose logs postgresql | grep "ready to accept connections"

# Reiniciar Temporal
docker compose restart temporal
```

### UI não conecta

```bash
# Verificar variável TEMPORAL_ADDRESS
docker compose exec temporal-ui env | grep TEMPORAL

# Verificar conectividade
docker compose exec temporal-ui ping temporal
```

### Archive não funciona

```bash
# Verificar bucket existe
docker compose exec temporal curl http://minio:9000/minio/health/live

# Verificar configuração
docker compose exec temporal cat /etc/temporal/config/config.yaml | grep -A10 archival
```

## Variáveis de Ambiente

| Variável | Padrão | Descrição |
|----------|--------|-----------|
| POSTGRES_PASSWORD | temporal_password | Senha do PostgreSQL |
| MINIO_USER | temporal | Usuário MinIO |
| MINIO_PASSWORD | temporal_archive_password | Senha MinIO |
| AUTH_TYPE | local | Tipo de autenticação |
| LOG_LEVEL | info | Nível de log |

## Produção

### Checklist

- [ ] Alterar todas as senhas padrão
- [ ] Configurar autenticação OIDC
- [ ] Configurar TLS/SSL
- [ ] Configurar backup automatizado
- [ ] Configurar alertas de monitoramento
- [ ] Aumentar recursos conforme carga
- [ ] Revisar retenção de dados

### TLS

Para produção, configure TLS:

1. Gerar certificados
2. Montar em `/etc/temporal/tls`
3. Configurar no `config.yaml`

### Recursos Recomendados

| Componente | CPU | Memória |
|------------|-----|---------|
| Temporal Server | 2 | 4GB |
| PostgreSQL | 2 | 4GB |
| MinIO | 1 | 2GB |
| Temporal UI | 0.5 | 512MB |

## Referências

- [Documentação Oficial](https://docs.temporal.io)
- [Temporal Server Config](https://github.com/temporalio/temporal/tree/main/config)
- [Docker Compose Reference](https://docs.temporal.io/cluster-deployment-guide#docker-compose)
- [Archival Guide](https://docs.temporal.io/cluster-deployment-guide#archival)
