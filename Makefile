.PHONY: run build test clean migrate docker

# Default target
all: build

# Run the server in development mode
run:
	go run cmd/server/main.go

# Build the binary
build:
	go build -o bin/kavach cmd/server/main.go

# Run tests
test:
	go test ./... -v

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f kavach

# Run database migrations (requires psql)
migrate:
	@echo "Running migrations..."
	psql $${DATABASE_URL} -f migrations/001_initial_schema.sql
	@echo "Migrations complete."

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run ./...

# Generate go.sum
deps:
	go mod tidy
	go mod download

# Docker build
docker:
	docker build -t kavach:latest .

# Docker run
docker-run:
	docker run -p 8080:8080 --env-file .env kavach:latest

# Dev with hot reload (requires air)
dev:
	air

# Show all routes
routes:
	@echo "HTML Pages:"
	@echo "  GET  /              Dashboard"
	@echo "  GET  /login         Login page"
	@echo "  GET  /signup        Signup page"
	@echo "  GET  /tokens        Tokens list"
	@echo "  GET  /tokens/new    Create token"
	@echo "  GET  /alerts        Alert history"
	@echo "  GET  /attackers     Attacker profiles"
	@echo "  GET  /attackers/:id Attacker detail"
	@echo "  GET  /integrations  Integration settings"
	@echo "  GET  /settings      User settings"
	@echo ""
	@echo "API:"
	@echo "  POST /api/v1/auth/signup    Register"
	@echo "  POST /api/v1/auth/login     Login"
	@echo "  POST /api/v1/auth/logout    Logout"
	@echo "  GET  /api/v1/auth/me        Current user"
	@echo "  GET  /api/v1/tokens         List tokens"
	@echo "  POST /api/v1/tokens         Create token"
	@echo "  GET  /api/v1/stats          Dashboard stats"
	@echo "  GET  /api/v1/alerts         List alerts"
	@echo "  GET  /api/v1/attackers      List attackers"
	@echo ""
	@echo "Token Triggers:"
	@echo "  GET  /t/:id         Trigger token (URL type)"
	@echo "  GET  /t/:id/doc     Trigger token (document)"
	@echo "  GET  /t/:id/key     Trigger token (API key)"
