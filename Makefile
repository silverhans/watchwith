.PHONY: dev dev-backend dev-frontend build up down

# Development
dev:
	@echo "Starting development servers..."
	@make dev-backend &
	@make dev-frontend

dev-backend:
	cd backend && go run ./cmd/server

dev-frontend:
	cd frontend && npm run dev

# Docker
build:
	docker compose build

up:
	docker compose up -d

down:
	docker compose down

# Utilities
lint-backend:
	cd backend && go vet ./...

lint-frontend:
	cd frontend && npx tsc --noEmit
