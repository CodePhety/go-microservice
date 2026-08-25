# User Management Microservice

This project is an example of a small Go microservice for managing users. It is intentionally narrow in scope, but intentionally rich in the kinds of architectural decisions and operational patterns.
It demonstrates:

- layered architecture and domain separation
- dependency injection and composition roots
- environment-driven configuration
- health and readiness endpoints
- graceful shutdown and lifecycle control
- Prometheus metrics and observability
- defensive validation and consistent API contracts
- concurrency-aware in-memory storage for a local demo

## Why this design is strong

### 12-factor alignment

1. Codebase: one service with a single, focused responsibility.
2. Dependencies: the service relies on explicit runtime dependencies rather than hidden state.
3. Config: environment variables control runtime behavior in `internal/config`.
4. Backing services: the repository abstraction isolates persistence concerns and can be replaced with a relational DB, cache, or remote service later.
5. Build, release, run: the binary is built once and runs with environment-specific config.
6. Processes: the service is stateless from the request-processing perspective; the in-memory store is just a local learning mechanism.
7. Port binding: port selection is externalized for deployment.
8. Concurrency: in-memory storage uses locking to protect shared state.
9. Disposability: shutdown is signal-aware and graceful.
10. Dev/prod parity: configuration defaults encourage repeatable local behavior while still being overrideable in production.
11. Logs: structured lifecycle and domain logging support debugging and tracing.
12. Admin processes: operations are easy to test in isolation and explain to teams.

### Senior design principles included

- Separation of concerns: HTTP, domain logic, and persistence are isolated.
- Single responsibility: each package owns one major concern.
- Explicit interfaces: the repository contract allows testing and extension.
- Defensive programming: validation and error handling happen before writes.
- Observability by default: Prometheus metrics provide operational insight.
- Secure-by-default mental model: path sanitization and request validation are basic but important design choices.
- Graceful degradation: the app does not crash during shutdown; it cleans up.
- Operational readiness: health and readiness endpoints are separated intentionally.

## Practical senior architecture reasoning

This is the kind of system a that can be described as using architecture language such as:

> The service has a clear composition root at startup, a domain layer that encapsulates business rules, and a transport layer that enforces API boundary behavior. The in-memory repository is intentionally not production-grade, but the abstraction makes it straightforward to swap in PostgreSQL or a distributed datastore later. The service also exposes Prometheus metrics and readiness signals so it can participate in container orchestration, autoscaling, and SRE tooling without needing custom logic outside the service.

## Project structure

- `main.go`: process lifecycle, composition, startup, shutdown
- `internal/config/config.go`: environment configuration
- `internal/user/model.go`: domain models and request/response contracts
- `internal/user/service.go`: business logic, validation, and repository orchestration
- `internal/server/server.go`: HTTP routing, request handling, and API contract behavior
- `internal/metrics/metrics.go`: Prometheus instrumentation and middleware

## API endpoints

### Infrastructure

- `GET /health` - liveness check
- `GET /ready` - readiness check
- `GET /metrics` - Prometheus metrics endpoint

### User CRUD

- `GET /users` - list users
- `POST /users` - create a user
- `GET /users/:id` - fetch a user
- `PUT /users/:id` - update one user
- `DELETE /users/:id` - delete one user

## Example curl commands

```bash
# health
curl http://localhost:8080/health

# readiness
curl http://localhost:8080/ready

# metrics
curl http://localhost:8080/metrics

# create user
curl -X POST http://localhost:8080/users \
  -H 'Content-Type: application/json' \
  -d '{"name":"Jane Doe","email":"jane@example.com","active":true}'

# list users
curl http://localhost:8080/users

# get user
curl http://localhost:8080/users/user-123

# update user
curl -X PUT http://localhost:8080/users/user-123 \
  -H 'Content-Type: application/json' \
  -d '{"name":"Jane Smith","email":"jane.smith@example.com","active":false}'

# delete user
curl -X DELETE http://localhost:8080/users/user-123
```

## Run locally

```bash
go mod tidy
go run .
```

With environment overrides:

```bash
PORT=9090 APP_ENV=production go run .
```

## Docker

Build the image:

```bash
docker build -t user-service:local .
```

Run it:

```bash
docker run --rm -p 8080:8080 -e APP_ENV=production -e PORT=8080 user-service:local
```

Or use Compose:

```bash
docker compose up --build
```

## Kubernetes

The manifests in `k8s/` provide a basic deployment pattern:

- namespace
- deployment with health and readiness probes
- service for cluster-internal routing
- ingress for external access

Apply them with:

```bash
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
kubectl apply -f k8s/ingress.yaml
```

## Prometheus and observability notes

This service exposes metrics for:

- HTTP request count by method/path/status
- HTTP request latency
- HTTP in-flight requests
- user-domain operations such as create, update, delete, and list
