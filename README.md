# Golang Modular Monolith Clean Architecture Boilerplate

[English README](README.md) | [Tiếng Việt](README_VI.md) | [Concurrency Guide (EN)](CONCURRENCY.md) | [Hướng Dẫn Concurrency (VI)](CONCURRENCY_VI.md)

This boilerplate provides a senior-grade, production-ready foundation designed for high-performance scale and rapid development. Built using Go 1.26, it implements a highly-decoupled Modular Monolith layout aligned strictly with Clean Architecture and SOLID principles, enabling painless transitions to microservices in the future.


Included Out-of-the-Box:
- Web Framework: Gin
- Database ORM: GORM (PostgreSQL)
- Schema Migrations: Goose (embedded into compile binary)
- Cache storage: Redis
- Event messaging: Kafka (KRaft mode confluent container)
- Dependency Injection: Manual wiring (with a complete Google Wire automated DI backup template)
- Structured logging: Uber Zap
- Error Management: Centralized Domain error mapping
- Tracing: Request ID propagation middleware
- Environments: Viper configuration with .env file support (via godotenv)

---

## Architectural Flows

Clean Architecture enforces that dependency rules flow strictly inward, while control flow travels through layers.

### Request-Response Flow
```
HTTP Request  -> Delivery (Gin Handler) -> Usecase (Logic) -> Repository (GORM) -> PostgreSQL DB
HTTP Response <- Delivery (Gin Handler) <- Usecase (Logic) <- Repository (GORM) <- PostgreSQL DB
```

### Dependency Flow (Strict Dependency Inversion)
Outer layers depend strictly on abstract interfaces defined in the inner Domain layer, preventing framework or database engine details from contaminating core business logic.
```
[Delivery Layer]       [Usecase Layer]        [Domain Layer]        [Repository Layer]
ProductHandler  --->  ProductUsecase (I)  <--  productUsecase (Impl)
                             |
                             v
                      ProductRepository (I) <-- postgresProductRepo (Impl)
```
Note: (I) represents an Interface, and (Impl) represents the physical structural Implementation.

---

## Directory Structure

```
.
├── cmd/
│   └── server/
│       └── main.go         # Application entrypoint (bootstrap, signals, graceful shutdown)
├── config/
│   └── config.yaml         # Configuration file with default key values
├── database/
│   ├── migrations/         # Plain SQL database migration scripts (Goose format)
│   └── migrations.go       # Compiled binary SQL file embedder using go:embed
├── internal/
│   ├── app/
│   │   ├── app.go          # Application container (Manual DI assembly, server lifecycles)
│   │   ├── middleware.go   # Gin Middlewares (Zap logging, recovery, CORS, RequestID)
│   │   └── wire.go         # Google Wire automated dependency injection graph backup
│   └── modules/
│       └── product/        # Self-contained domain module (can be extracted to microservice)
│           ├── delivery/   # HTTP Transports (Gin Handlers, request/response DTOs)
│           ├── domain/     # Pure Business Entities, invariants, and Interfaces (0 imports)
│           ├── repository/ # Database adapters (GORM implementation mapping structures)
│           └── usecase/    # Domain Business Logic implementations
├── pkg/
│   ├── apperr/             # Structured Domain Errors
│   ├── config/             # Config parser using Viper with godotenv overrides
│   ├── database/           # Postgres & Redis connectors and health check routines
│   ├── logger/             # High performance structured logger wrapper around Zap
│   ├── messaging/          # Kafka Producer/Consumer wrappers (segmentio)
│   └── response/           # Unified API success and error JSON response mappings
├── Dockerfile              # Multi-stage production container builder (Alpine, non-root)
├── docker-compose.yml      # Orchestrates Postgres, Redis, Kafka, and the App locally
├── go.mod                  # Core Go module declarations
└── .env.example            # Environment overrides file template for local developer use
```

---

## Configuration and Environments

The boilerplate uses a hybrid configuration management system. Viper loads configurations in the following priority order:
1. Environment variables (OS system level)
2. Local `.env` file (loaded via `godotenv` on startup, useful for local developer overrides)
3. Default `config/config.yaml` file

### Overriding Settings via Environment Variables
Environment variables override YAML settings automatically. Simply map the nested path using underscores.
Example:
- `database.password` in YAML becomes `DATABASE_PASSWORD` as an environment variable.
- `kafka.brokers` in YAML becomes `KAFKA_BROKERS` as an environment variable.

A complete list of options can be found inside `.env.example`.

---

## Setup and Installation

### Prerequisites
- Go 1.26 or higher (if running locally)
- Docker and Docker Compose

### 1. Running with Docker Compose (Recommended)
Docker Compose spins up all infrastructure (Postgres, Redis, Kafka KRaft) and builds the local Go binary into a secure container automatically.

Run the following command at the project root:
```bash
docker compose up --build
```
Once healthy:
- The HTTP server starts listening at `http://localhost:8080`
- Goose runs all database migrations automatically
- Postgres runs at `localhost:5432`
- Redis runs at `localhost:6379`
- Kafka brokers are open at `localhost:9092`

### 2. Running Locally for Development
To debug or run the application directly on your local system:

Step 1: Spin up only the infrastructure dependencies using Docker:
```bash
docker compose up -d postgres redis kafka
```

Step 2: Create your local environment variable overrides file:
```bash
cp .env.example .env
```
Note: If your local Postgres, Redis, or Kafka ports or credentials differ from `.env.example` defaults, update them in `.env`.

Step 3: Run the database migration script programmatically (automatically run on boot) and start the application:
```bash
go run cmd/server/main.go
```

### High Resilience & Graceful Fallback (Running without Redis/Kafka)

This boilerplate features a senior-grade high-resilience and graceful fallback design. The application starts up and runs even if the optional services (Redis and Kafka) are down.

- **PostgreSQL (Required)**: If Postgres is down, the application logs a Fatal error and stops, since the demo product entity requires a persistent store to operate.
- **Redis (Optional)**: On boot, the app attempts to ping Redis. If the connection fails, the app logs a Warning (`Redis connection failed. Features requiring cache may fail.`) but continues running. The internal `Redis` pointer is set to `nil`.
- **Kafka (Optional)**: If the Kafka Producer initialization fails, the app logs a Warning (`Kafka Producer initialization failed. Event publishing disabled.`) but continues running. The internal `Producer` pointer is set to `nil`.
- **Health check (/health)**: The endpoint checks if `redis != nil` dynamically. If Redis is down, it reports `"redis": "down"` while responding with HTTP 200 OK.

#### How to run only PostgreSQL and App in Docker Compose

If you want to disable Redis and Kafka to save local compute resources, perform these adjustments:

1. Open `docker-compose.yml` and find the `app` service configuration.
2. Modify the `depends_on` block to depend ONLY on `postgres`:
   ```yaml
   app:
     build:
       context: .
       dockerfile: Dockerfile
     container_name: boilerplate-app
     ports:
       - "8080:8080"
     depends_on:
       postgres:
         condition: service_healthy
   ```
3. Comment out or delete the `redis` and `kafka` service blocks, as well as the `redis_data` volume block at the bottom.
4. Run the boot command:
   ```bash
   docker compose up --build
   ```
   The application container will launch instantly with only PostgreSQL running, gracefully bypassing the optional Redis and Kafka subsystems on startup.

---

## Core Operational Mechanics

### Centralized Error Handling
To keep the core business domain clean, usecases do not return web status codes. Instead, they return domain-specific errors via `pkg/apperr`:

- `apperr.NewValidationError(...)` -> Mapped to HTTP 400 Bad Request
- `apperr.NewNotFoundError(...)` -> Mapped to HTTP 404 Not Found
- `apperr.NewConflictError(...)` -> Mapped to HTTP 409 Conflict
- `apperr.NewUnauthorizedError(...)` -> Mapped to HTTP 401 Unauthorized
- `apperr.NewForbiddenError(...)` -> Mapped to HTTP 403 Forbidden
- `apperr.NewInternalError(...)` -> Mapped to HTTP 500 Internal Server Error

The delivery layer intercepts these errors and outputs a uniform JSON response format:
```json
{
  "success": false,
  "error": {
    "code": "PRODUCT_SKU_EXISTS",
    "message": "product with SKU SKU123 already exists"
  }
}
```

### Telemetry and Logging
- Request ID Middleware: Automatically generates or propagates an `X-Request-ID` header. Every log entry processed during a request carries this ID.
- GORM Query Logger: Integrates GORM queries into Zap logging automatically. Prints execution latency, rows affected, and marks warnings for slow queries (>200ms).
- Panic Recovery: The `RecoveryMiddleware` captures runtime panics, logs full structured stack traces, and prevents server crashes, responding to the client with a secure 500 JSON payload.

### Programmatic DB Migrations
Database modifications are managed cleanly via Goose:
- Add a new `.sql` file in `database/migrations/` using the name structure `0000X_description.sql`.
- Migration headers must specify UP and DOWN blocks:
  ```sql
  -- +goose Up
  CREATE TABLE demo (...);

  -- +goose Down
  DROP TABLE demo;
  ```
- Because the migrations directory is compiled into the Go binary using `go:embed` inside `database/migrations.go`, the compiled binary does not need external directory mounts inside production containers to run schema migrations on boot.

---

## Implementing a New Module (SOLID Guidelines)

When creating a new module (e.g. `order` or `customer`):

1. Create the module folder under `internal/modules/[module_name]`.
2. Define the pure entities and interface definitions in `domain/`:
   - `domain/entity.go`: Declarations of the business model and methods validating invariants.
   - `domain/repository.go`: Database operations interfaces.
   - `domain/usecase.go`: Business operation interfaces.
3. Write business workflows in `usecase/usecase.go` depending only on the Repository interface.
4. Implement storage code in `repository/postgres_repository.go` using GORM.
5. Create endpoints in `delivery/http/handler.go` utilizing Gin, routing validation requests from `request.go` and serializing responses in `response.go`.
6. Bind the module in `internal/app/app.go` inside `NewApp` manually.

---

## Activating Google Wire DI Backup

If manual wiring inside `internal/app/app.go` becomes too verbose as the application expands:

Step 1: Install the Wire CLI tool:
```bash
go install github.com/google/wire/cmd/wire@latest
```

Step 2: Generate the injection graph automatically:
```bash
cd internal/app
wire
```
This reads the graph defined inside `internal/app/wire.go` and generates `wire_gen.go`.

Step 3: Modify `cmd/server/main.go` to invoke the automated initializer:
```go
// Replace:
appInstance, err := app.NewApp(cfg, log)

// With:
appInstance, err := app.InitializeApp(cfg, log)
```
The application will execute using fully automated compile-time dependency injection.
