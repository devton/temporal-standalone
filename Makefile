.PHONY: up down logs status restart clean backup restore init create-namespace help

# Variáveis
COMPOSE=docker compose
TEMPORAL_CLI=docker compose exec temporal-admin-tools temporal

help:
	@echo "Temporal Standalone - Comandos disponíveis:"
	@echo ""
	@echo "  make up           - Iniciar todos os serviços"
	@echo "  make down         - Parar todos os serviços"
	@echo "  make logs         - Ver logs (Ctrl+C para sair)"
	@echo "  make status       - Status dos containers"
	@echo "  make restart      - Reiniciar serviços"
	@echo "  make clean        - Parar e remover volumes (CUIDADO!)"
	@echo "  make backup       - Backup do banco de dados"
	@echo "  make init         - Criar arquivo .env se não existir"
	@echo "  make create-ns    - Criar namespace 'dev'"
	@echo "  make health       - Verificar saúde dos serviços"
	@echo ""

up:
	$(COMPOSE) up -d
	@echo "Aguardando serviços ficarem prontos..."
	@sleep 10
	$(MAKE) health

down:
	$(COMPOSE) down

logs:
	$(COMPOSE) logs -f

status:
	$(COMPOSE) ps

restart:
	$(COMPOSE) restart

clean:
	@echo "ATENÇÃO: Isso vai apagar TODOS os dados!"
	@read -p "Continuar? (y/N) " confirm; \
	if [ "$$confirm" = "y" ]; then \
		$(COMPOSE) down -v; \
		echo "Volumes removidos."; \
	else \
		echo "Operação cancelada."; \
	fi

backup:
	@mkdir -p backups
	@echo "Criando backup do PostgreSQL..."
	$(COMPOSE) exec postgresql pg_dump -U temporal temporal > backups/temporal_$$(date +%Y%m%d_%H%M%S).sql
	$(COMPOSE) exec postgresql pg_dump -U temporal temporal_visibility > backups/visibility_$$(date +%Y%m%d_%H%M%S).sql
	@echo "Backup concluído em backups/"

init:
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo ".env criado a partir de .env.example"; \
		echo "Edite o arquivo .env com suas configurações"; \
	else \
		echo ".env já existe"; \
	fi

create-ns:
	$(COMPOSE) --profile admin up -d temporal-admin-tools
	@sleep 2
	$(TEMPORAL_CLI) operator namespace create --retention 30d dev

health:
	@echo "=== PostgreSQL ==="
	$(COMPOSE) exec postgresql pg_isready -U temporal || echo "PostgreSQL não está pronto"
	@echo ""
	@echo "=== MinIO ==="
	curl -s http://localhost:9000/minio/health/live > /dev/null && echo "MinIO: OK" || echo "MinIO: NÃO RESPONDE"
	@echo ""
	@echo "=== Temporal Server ==="
	$(COMPOSE) exec temporal temporal operator cluster health 2>/dev/null || echo "Temporal Server: Iniciando..."
	@echo ""
	@echo "=== Temporal UI ==="
	curl -s http://localhost:8080 > /dev/null && echo "Temporal UI: http://localhost:8080" || echo "Temporal UI: Iniciando..."

admin:
	$(COMPOSE) --profile admin up -d temporal-admin-tools
	@echo "Admin tools disponível. Use: docker compose exec temporal-admin-tools temporal <comando>"

shell:
	$(COMPOSE) --profile admin up -d temporal-admin-tools
	$(COMPOSE) exec temporal-admin-tools bash
