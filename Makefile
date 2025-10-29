# Hot Fixture Tool - Docker Operations
.PHONY: help build up down logs shell adduser renew-jwt backup clean

# Default target
help:
	@echo "Hot Fixture Tool Docker Commands"
	@echo ""
	@echo "Setup:"
	@echo "  build     Build Docker images"
	@echo "  up        Start all services"
	@echo "  down      Stop all services"
	@echo ""
	@echo "Administration:"
	@echo "  adduser EMAIL=user@example.com KEY=key.pem   Add user with public key"
	@echo "  renew-jwt                                     Renew JWT signing keys"
	@echo "  get-jwt                                       Get JWT public key"
	@echo ""
	@echo "Operations:"
	@echo "  logs      Follow service logs"
	@echo "  shell     Open shell in hfitd container"
	@echo "  status    Show service status"
	@echo ""
	@echo "Maintenance:"
	@echo "  backup    Backup databases"
	@echo "  clean     Clean up containers and volumes"
	@echo ""
	@echo "Examples:"
	@echo "  make up"
	@echo "  make adduser EMAIL=alice@company.com KEY=alice_key.pem"
	@echo "  make logs"

# Build Docker images
build:
	docker-compose build

# Start services
up:
	docker-compose up -d
	@echo "Services started. Check status with 'make status'"
	@echo "Add users with 'make adduser EMAIL=user@example.com KEY=key.pem'"

# Stop services
down:
	docker-compose down

# Follow logs
logs:
	docker-compose logs -f

# Show service status
status:
	docker-compose ps
	@echo ""
	@echo "Health checks:"
	@curl -s http://localhost:8080/health && echo " ✓ hfitd healthy" || echo " ✗ hfitd unhealthy"

# Open shell in hfitd container
shell:
	docker-compose exec hfitd sh

# Add user (requires EMAIL and KEY parameters)
adduser:
ifndef EMAIL
	@echo "Error: EMAIL parameter required"
	@echo "Usage: make adduser EMAIL=user@example.com KEY=key.pem"
	@exit 1
endif
ifndef KEY
	@echo "Error: KEY parameter required"
	@echo "Usage: make adduser EMAIL=user@example.com KEY=key.pem"
	@exit 1
endif
	@if [ -f "$(KEY)" ]; then \
		docker-compose exec hfitd ./hfitd-cli adduser "$(EMAIL)" "$$(cat $(KEY))"; \
	else \
		docker-compose exec hfitd ./hfitd-cli adduser "$(EMAIL)" "$(KEY)"; \
	fi

# Renew JWT keys
renew-jwt:
	docker-compose exec hfitd ./hfitd-cli renew-jwt

# Get JWT public key
get-jwt:
	docker-compose exec hfitd ./hfitd-cli get-jwt-public-key

# Backup Redis (user keys and JWT keys)
backup:
	@mkdir -p backups
	@echo "Backing up Redis..."
	docker-compose exec redis redis-cli --rdb /data/backup.rdb
	docker cp $$(docker-compose ps -q redis):/data/backup.rdb backups/redis-$(shell date +%Y%m%d-%H%M%S).rdb
	@echo "Redis backup saved to backups/ directory"

# Clean up everything
clean:
	docker-compose down -v --rmi local
	docker system prune -f

# Development targets (removed admin container references)
dev-up:
	docker-compose up -d

dev-shell:
	docker-compose exec hfitd sh

# Generate example keys for testing
gen-keys:
	@mkdir -p keys
	openssl genrsa -out keys/test_private.pem 2048
	openssl rsa -in keys/test_private.pem -pubout -out keys/test_public.pem
	@echo "Test keys generated in keys/ directory"
	@echo "Add test user: make adduser EMAIL=test@example.com KEY=keys/test_public.pem"

# Quick demo setup
demo: build up gen-keys
	@echo "Waiting for services to be ready..."
	@sleep 10
	@echo "Adding demo user..."
	@make adduser EMAIL=demo@hotfixtool.local KEY=keys/test_public.pem
	@echo ""
	@echo "🎉 Demo setup complete!"
	@echo ""
	@echo "Demo user: demo@hotfixtool.local"
	@echo "Private key: keys/test_private.pem"
	@echo "Public key: keys/test_public.pem"
	@echo ""
	@echo "API Base URL: http://localhost:8080"
	@echo "JWT Public Key: curl http://localhost:8080/.well-known/jwks.json"
	@echo ""
	@echo "Note: Configure DB_HOST, DB_USER, DB_PASSWORD, DB_NAME in docker-compose.yaml"
	@echo "      to connect to your existing database for data export operations"
	@echo ""
	@echo "Check logs: make logs"
	@echo "Stop demo: make down"