.PHONY: all build run test test-race clean docker-up docker-down wire migrate-create help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GORUN=$(GOCMD) run
BINARY_NAME=server
MAIN_PATH=cmd/server/main.go

all: help

build: ## Compile the optimized production binary
	CGO_ENABLED=0 GOOS=linux $(GOBUILD) -ldflags="-s -w" -o $(BINARY_NAME) $(MAIN_PATH)

run: ## Run the application locally in development mode
	$(GORUN) $(MAIN_PATH)

test: ## Run all unit tests
	$(GOTEST) -v ./...

test-race: ## Run all unit tests with Go data race detector enabled
	$(GOTEST) -race -v ./...

clean: ## Remove build artifacts and clean local Go cache
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	$(GOCLEAN) -testcache

docker-up: ## Boot up all infrastructure and app using Docker Compose
	docker compose up --build

docker-down: ## Stop all container services and purge local volumes
	docker compose down -v

wire: ## Generate automated Google Wire dependency injection code
	cd internal/app && wire

migrate-create: ## Create a new SQL migration file (usage: make migrate-create name=description)
	@if [ -z "$(name)" ]; then \
		echo "Error: 'name' is required. Example: make migrate-create name=create_users_table"; \
		exit 1; \
	fi
	@mkdir -p database/migrations
	@filename="database/migrations/$$(date +%Y%m%d%H%M%S)_$(name).sql"; \
	echo "-- +goose Up" > $$filename; \
	echo "-- +goose StatementBegin" >> $$filename; \
	echo "-- +goose StatementEnd" >> $$filename; \
	echo "" >> $$filename; \
	echo "-- +goose Down" >> $$filename; \
	echo "-- +goose StatementBegin" >> $$filename; \
	echo "-- +goose StatementEnd" >> $$filename; \
	echo "Created new SQL migration: $$filename"

help: ## Print helper instructions for all available make commands
	@echo "Available commands in this Boilerplate Makefile:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
