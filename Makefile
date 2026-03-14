IMAGE := dstathis/hermits-website
TAG   := latest

.PHONY: help test test-unit image push run stop logs clean lint vet

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

test: ## Run full test suite (requires Docker)
	./scripts/test.sh

test-unit: ## Run unit tests only (no database needed)
	go test -v ./internal/middleware/... ./internal/config/... ./internal/mail/...

lint: ## Run go vet
	go vet ./...

image: ## Build Docker image
	docker build -t $(IMAGE):$(TAG) .

push: image ## Build and push image to Docker Hub
	docker push $(IMAGE):$(TAG)

run: ## Start all services with docker compose
	docker compose up -d

stop: ## Stop all services
	docker compose down

logs: ## Tail service logs
	docker compose logs -f

clean: ## Stop services and remove volumes
	docker compose down -v

seed: ## Create an admin user (username: admin, password: admin)
	docker compose exec app ./seed admin admin
