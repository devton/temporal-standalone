.PHONY: up down down-v logs status ps restart clean

# Temporal Standalone Makefile

up:
	@echo "Starting Temporal stack..."
	docker compose up -d
	@echo "Waiting for services to be ready..."
	@sleep 10
	@docker compose ps

down:
	@echo "Stopping Temporal stack..."
	docker compose down

down-v:
	@echo "Stopping Temporal stack and removing volumes..."
	docker compose down -v

logs:
	docker compose logs -f

status:
	docker compose ps

ps: status

restart:
	@echo "Restarting Temporal stack..."
	docker compose restart
	@sleep 5
	docker compose ps

clean: down-v
	@echo "Cleaning up..."
	docker system prune -f
