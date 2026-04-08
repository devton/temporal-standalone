# Temporal Standalone

Ambiente Temporal completo com PostgreSQL, Casdoor (OIDC) e UI.

## Serviços

| Serviço | Porta | URL |
|---------|-------|-----|
| Temporal Server | 7233 | `localhost:7233` |
| Temporal UI | 8080 | http://localhost:8080 |
| Casdoor (OIDC) | 8000 | http://localhost:8000 |
| PostgreSQL | 5432 | `localhost:5432` |

## Iniciar

```bash
docker compose up -d
```

Aguarde todos os serviços ficarem healthy:

```bash
docker ps --format "table {{.Names}}\t{{.Status}}"
```

## Configurar OIDC (Casdoor)

### Opção A: Script Automático

Execute o script de configuração:

```bash
./scripts/setup-casdoor.sh
```

O script cria automaticamente:
- Organização `temporal`
- Aplicação `temporal-ui` com Client ID/Secret
- Usuário de teste `testuser` com senha `Temporal123!`

### Opção B: Configuração Manual

Acesse o Casdoor Admin UI e configure manualmente.

#### 1. Acessar Casdoor Admin

Abra http://localhost:8000

- **Usuário:** admin
- **Senha:** 123
- **Organização:** built-in

#### 2. Criar Organização

1. Vá em **Organizations** → **Add**
2. Preencha:
   - **Name:** `temporal`
   - **DisplayName:** Temporal
   - **Website:** `http://localhost:8080`
3. Clique em **Save**

#### 3. Criar Aplicação

1. Vá em **Applications** → **Add**
2. Preencha:
   - **Organization:** temporal
   - **Name:** `temporal-ui`
   - **Client ID:** `temporal-ui`
   - **Client secret:** `temporal-ui-secret`
   - **Redirect URLs:** 
     - `http://localhost:8080/auth/callback`
     - `http://localhost:8080`
   - **Token format:** JWT
   - **Expire in hours:** 168 (7 dias)
   - **Grant types:** authorization_code, refresh_token
3. Clique em **Save**

#### 4. Criar Usuário de Teste

1. Vá em **Users** → **Add**
2. Preencha:
   - **Organization:** temporal
   - **Name:** `testuser`
   - **Password:** `Temporal123!`
   - **Email:** `testuser@temporal.local`
   - **Email verified:** true
3. Clique em **Save**

#### 5. Testar Login

1. Acesse http://localhost:8080
2. Você será redirecionado para o Casdoor
3. Faça login com:
   - **Organization:** temporal
   - **Username:** testuser
   - **Password:** `Temporal123!`
4. Após login, será redirecionado de volta para a UI

## Namespaces

O namespace `default` está configurado com:

- **Retention:** 720h (30 dias)
- **History Archival:** Habilitado (`file:///tmp/temporal_archival/development`)
- **Visibility Archival:** Habilitado (`file:///tmp/temporal_vis_archival/development`)

### Verificar Namespace

```bash
docker exec temporal-server temporal operator namespace describe default
```

### Atualizar Namespace

```bash
# Habilitar archival (já feito pelo setup)
docker exec temporal-server temporal operator namespace update default \
  --history-archival-state enabled \
  --visibility-archival-state enabled \
  --retention 720h
```

## Usar com CLI

```bash
# Sem auth (server sem auth habilitado)
temporal workflow list --address localhost:7233

# Listar namespaces
temporal operator namespace list --address localhost:7233

# Executar workflow de teste
temporal workflow execute \
  --address localhost:7233 \
  --namespace default \
  --task-queue test \
  --type test \
  --input '"hello"'
```

## Usar com SDK

### Go

```go
import "go.temporal.io/sdk/client"

func main() {
    c, err := client.Dial(client.Options{
        HostPort:  "localhost:7233",
        Namespace: "default",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer c.Close()
    
    // Usar client...
}
```

### Python

```python
from temporalio.client import Client

async def main():
    client = await Client.connect(
        "localhost:7233",
        namespace="default"
    )
    
    # Usar client...
```

### TypeScript

```typescript
import { Client } from '@temporalio/client';

const client = new Client({
  address: 'localhost:7233',
  namespace: 'default',
});

// Usar client...
```

## Archival

Os workflows arquivados ficam em volumes Docker:

- **History:** `archive_data` → `/tmp/temporal_archival`
- **Visibility:** `archive_vis_data` → `/tmp/temporal_vis_archival`

### Verificar Archival

```bash
# Listar arquivos no container
docker exec temporal-server ls -la /tmp/temporal_archival/development/
docker exec temporal-server ls -la /tmp/temporal_vis_archival/development/
```

### MinIO (S3 Backend)

Para usar MinIO como backend S3:

1. Descomente as seções `minio` e `minio-init` no `docker-compose.yml`
2. Atualize o `dynamicconfig/docker.yaml`:

```yaml
history.archival:
  - value:
      enableReadFromArchival: true
      enableWriteToArchival: true
      uri: "s3://temporal-archive/"
      s3:
        endpoint: "minio:9000"
        accessKey: "temporal"
        secretKey: "temporal_archive_secret"
        region: "us-east-1"
        usePathStyleAccess: true
```

## JWT Token Manual

Para gerar um token JWT manual (para testes com CLI):

```bash
./scripts/generate-jwt.sh
```

O token será salvo em `config/jwt/token.txt`.

### Usar Token

```bash
export TEMPORAL_CLI_AUTH_TOKEN=$(cat config/jwt/token.txt)
temporal workflow list --address localhost:7233
```

## Arquitetura

```
┌─────────────────────────────────────────────────────────────┐
│                     Docker Network                           │
│                    temporal-network                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐│
│  │  PostgreSQL  │  │   Casdoor    │  │   Temporal Server    ││
│  │    :5432     │  │    :8000     │  │       :7233          ││
│  │  (database)  │  │    (OIDC)    │  │   (auto-setup)       ││
│  └──────────────┘  └──────────────┘  └──────────────────────┘│
│                                            ┌──────────────────┤│
│                                            │  Temporal UI     ││
│                                            │     :8080       ││
│                                            │  (OIDC client)  ││
│                                            └──────────────────┘│
└─────────────────────────────────────────────────────────────┘
```

## Troubleshooting

### UI não redireciona para login

1. Verifique se Auth está habilitado:
   ```bash
   curl http://localhost:8080/api/v1/settings | jq .Auth
   ```
2. Verifique os logs: `docker logs temporal-ui`
3. Reinicie a UI: `docker compose restart temporal-ui`

### Erro "discovery failed" no login

1. Verifique se o Casdoor está acessível:
   ```bash
   curl http://localhost:8000/.well-known/openid-configuration
   ```
2. Verifique se a aplicação existe no Casdoor
3. Verifique se o Client ID/Secret estão corretos

### Casdoor não inicia

1. Verifique se o banco `casdoor` foi criado:
   ```bash
   docker exec temporal-postgres psql -U postgres -l | grep casdoor
   ```
2. Se não existir, crie manualmente:
   ```bash
   docker exec temporal-postgres psql -U postgres -c "CREATE DATABASE casdoor;"
   ```

### Server não inicia

1. Verifique os logs:
   ```bash
   docker logs temporal-server 2>&1 | tail -50
   ```
2. Problemas comuns:
   - Banco não está pronto: aguarde mais tempo
   - JWT key source inválido: verifique a URL do Casdoor
   - Auth habilitado sem configuração completa: desabilite auth no server

### Workflow arquivado não aparece

1. Verifique se o archival está habilitado no namespace
2. Verifique se o workflow foi fechado há mais tempo que o retention
3. Verifique os arquivos de archival no container

## Configuração Avançada

### Habilitar Auth no Server

Para habilitar autenticação no servidor Temporal (não apenas na UI):

1. Atualize `dynamicconfig/docker.yaml`:

```yaml
frontend.auth:
  - value:
      jwtKeySource:
        - keySourceKind: "jwks"
          url: "http://casdoor:8000/.well-known/jwks"
      claimMapper:
        name: "jwt"
      authorizer:
        name: "default"
```

2. Adicione ao `docker-compose.yml`:

```yaml
environment:
  - TEMPORAL_JWT_KEY_SOURCE1=http://casdoor:8000/.well-known/jwks
```

**Atenção:** Isso pode bloquear workflows internos do sistema. Use com cuidado.

### Múltiplos Namespaces

```bash
# Criar namespace de produção
temporal operator namespace create production \
  --retention 720h \
  --history-archival-state enabled \
  --visibility-archival-state enabled
```

### Backup do PostgreSQL

```bash
# Backup
docker exec temporal-postgres pg_dump -U temporal temporal > backup_temporal.sql
docker exec temporal-postgres pg_dump -U temporal temporal_visibility > backup_visibility.sql

# Restore
cat backup_temporal.sql | docker exec -i temporal-postgres psql -U temporal
cat backup_visibility.sql | docker exec -i temporal-postgres psql -U temporal
```

## Licença

MIT

## Nota sobre Rede

Por padrão, todos os serviços usam `localhost`. Se você precisar acessar de outra máquina na rede, altere as URLs nos seguintes arquivos:

- `docker-compose.yml` (variáveis de ambiente)
- `config/ui/docker.yaml` (OIDC provider URLs)
- `scripts/setup-casdoor.sh` (URLs do Casdoor e UI)

Exemplo com IP `192.168.1.100`:

```bash
# Atualizar todas as referências
sed -i 's/localhost/192.168.1.100/g' docker-compose.yml config/ui/docker.yaml scripts/setup-casdoor.sh
```

Após alterar, reinicie os serviços:

```bash
docker compose down && docker compose up -d
```
