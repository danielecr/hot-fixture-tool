# Hot Fixture Tool Daemon - Makefile
# Copyright (c) 2025 Daniele Cruciani

.PHONY: help build swagger-gen swagger-serve clean localtest-with-redis-docker redis-start redis-stop redis-status test-swagger

# Default target
help:
	@echo "Available targets:"
	@echo "  build        - Build the application"
	@echo "  swagger-gen  - Generate swagger documentation"
	@echo "  swagger-serve - Serve swagger documentation locally"
	@echo "  localtest-with-redis-docker - Start Redis in Docker and run service for testing"
	@echo "  redis-start  - Start Redis container"
	@echo "  redis-stop   - Stop Redis container"
	@echo "  redis-status - Check Redis container status"
	@echo "  clean        - Clean build artifacts"

# Build the application
build: swagger-gen
	go build ./cmd/hfitd
	go build ./cmd/hfitd-cli

# Generate swagger documentation
swagger-gen:
	~/go/bin/swag init -g api/api.go
	@echo "Fixing swagger compatibility issues..."
	@sed -i '' 's/LeftDelim:.*"{{",//' docs/docs.go
	@sed -i '' 's/RightDelim:.*"}}",//' docs/docs.go

# Serve swagger documentation locally  
swagger-serve:
	@echo "Swagger UI available at: http://localhost:8080/swagger/"
	@echo "Starting development server..."
	./hfitd

# Clean build artifacts
clean:
	rm -f hfitd hfitd-cli
	rm -rf docs/

# Development workflow - regenerate docs and build
dev: swagger-gen build

# Check if swagger binary exists
check-swag:
	@which ~/go/bin/swag > /dev/null || (echo "swag not found. Install with: go install github.com/swaggo/swag/cmd/swag@latest" && exit 1)

# Redis Docker container management
redis-start:
	@echo "Starting Redis container for local testing..."
	@docker run -d --name hfitd-redis \
		-p 6379:6379 \
		--rm \
		redis:7-alpine \
		redis-server --requirepass "" || \
		(echo "Redis container already running or failed to start" && docker ps | grep hfitd-redis)

redis-stop:
	@echo "Stopping Redis container..."
	@docker stop hfitd-redis 2>/dev/null || echo "Redis container not running"

redis-status:
	@echo "Redis container status:"
	@docker ps | grep hfitd-redis || echo "Redis container not running"
	@echo "Testing Redis connection:"
	@docker exec hfitd-redis redis-cli ping 2>/dev/null || echo "Cannot connect to Redis"

# Local testing with Redis in Docker
localtest-with-redis-docker: build redis-start
	@echo "Waiting for Redis to be ready..."
	@sleep 3
	@echo "Starting Hot Fixture Tool Daemon..."
	@echo "Swagger UI will be available at: http://localhost:8080/swagger/"
	@echo "API endpoints available at: http://localhost:8080/api/"
	@echo "Press Ctrl+C to stop the service and Redis container"
	@trap 'make redis-stop' EXIT INT TERM; \
	SERVER_ADDRESS=:8080 \
	DBMS_PROVIDERS= \
	REDIS_URL=redis://localhost:6379 \
	JWT_SECRET=development-jwt-secret-for-testing \
	./hfitd

# Test Swagger endpoints
test-swagger: 
	@echo "Testing Swagger JSON endpoint..."
	@curl -s http://localhost:8080/swagger.json | jq '.info.title' || echo "Failed to get Swagger JSON"
	@echo "Testing Swagger UI endpoint..."
	@curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/swagger/ || echo "Failed to get Swagger UI"
	@echo "Swagger endpoints:"
	@echo "  JSON: http://localhost:8080/swagger.json"
	@echo "  YAML: http://localhost:8080/swagger.yaml"  
	@echo "  UI:   http://localhost:8080/swagger/"