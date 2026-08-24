# Principal-Level Architecture Walkthrough

## Executive summary

This service is a deliberately small user-management microservice, but it is designed to demonstrate the way a principal-level engineer thinks about architecture. The goal is not to show a production-ready identity platform. The goal is to show clear separation of concerns, cloud-native readiness, operational maturity, and the ability to explain trade-offs.

At a high level, the service is organized as:

- a transport layer that handles HTTP concerns
- a domain/service layer that owns validation and business rules
- a repository abstraction that isolates persistence concerns
- a metrics layer that plugs into Prometheus for runtime observability
- a small runtime configuration layer that follows 12-factor principles

This creates a service that is easy to evolve from a local demo into a production-grade platform.

---

## 1. Why this is a strong principal-level example

A principal-level answer is not just "it stores users." It is:

> This is a stateless API service designed with explicit boundaries, operational telemetry, and a clean domain model. It can be deployed behind a load balancer, scaled horizontally, and monitored using standard platform tools. The service separates transport, business logic, and persistence concerns so that each layer can evolve independently and be replaced without breaking the domain contract.

That is the kind of reasoning interviewers want to hear when discussing modern service architecture.

---

## 2. System context

The service receives requests from clients or upstream systems and exposes a small set of CRUD endpoints for users.

The runtime interaction model is straightforward:

1. A client sends HTTP to `/users`, `/users/:id`, `/ready`, `/health`, or `/metrics`.
2. The HTTP layer decodes the request and routes it.
3. The domain service validates the request.
4. The repository executes the write/read operation.
5. The app records metrics and returns a consistent HTTP response.

This is important because it demonstrates a clean request path, not just ad hoc controller logic.

---

## 3. Architectural boundaries

### Transport boundary

The HTTP package is responsible for:

- request decoding
- routing
- response shaping
- status-code selection
- HTTP-specific concerns such as invalid JSON handling and unsupported methods

It should not contain business rules like "a user must have a valid email" unless those rules are part of the HTTP boundary. In this service, validation lives primarily in the service layer.

### Domain boundary

The service layer is responsible for:

- validating business rules
- checking invariants
- generating identifiers
- orchestrating repository interactions
- surfacing domain-specific errors

This matters because the domain layer stays stable even if infrastructure changes.

### Persistence boundary

The repository interface abstracts storage implementation. That gives us a clean seam for future upgrades.

A senior architect would describe this as:

> We separate the domain from persistence so that the business logic remains independent from storage technology decisions. In a real system, we could replace an in-memory store with PostgreSQL, a document database, or a distributed cache without altering the domain contract.

---

## 4. Why the design is useful in interview conversations

A principal-level answer should show understanding of both technical and organizational concerns.

### Clear ownership

Each package owns a clear concern:

- `server`: HTTP contract and API behavior
- `user`: domain entities and business logic
- `config`: environment-based runtime configuration
- `metrics`: operational telemetry

This reduces accidental coupling and improves code maintainability.

### Testability

Because dependencies are injected, the service can be tested in isolation without spinning up a network stack. A senior architect will emphasize the value of testing the business behavior directly.

### Evolvability

The repository abstraction means persistence is replaceable. This is a crucial design conversation point: the business logic should not be tightly coupled to a particular datastore.

---

## 5. 12-factor alignment in more detail

### Factor 1: One codebase, many deploys

This service has one logical codebase and can run in local, staging, and production environments with different config values.

### Factor 2: Explicit dependencies

Dependencies are declared in Go modules. There are no hidden runtime assumptions beyond the standard environment variables.

### Factor 3: Configuration stored in the environment

The config package reads values such as `PORT` and `APP_ENV` from environment variables. That makes the service portable across environments.

### Factor 4: Treat backing services as attached resources

The repository abstraction makes data storage a replaceable dependency. In a real system, we would inject a database client rather than hard-coding data access in the domain layer.

### Factor 5: Build, release, run

This repository supports a build artifact and then runs that artifact with environment-specific parameters. That is exactly how containerized and platform-based deployment works.

### Factor 6: Stateless processes

The service is stateless at the application layer. Local in-memory storage exists only for practice and demonstration; it is not a production-grade persistence model.

### Factor 7: Port binding

The service binds to a configurable port, which is essential for deployment in containers or orchestrators.

### Factor 8: Concurrency

The repository uses mutexes for thread safety. This is a simple but important example of handling shared state correctly.

### Factor 9: Disposability

The startup and shutdown logic is intentionally graceful. Requests are allowed to drain as the process exits, which is the right operational pattern for cloud-native systems.

### Factor 10: Dev/prod parity

The application uses the same runtime model across environments. The difference is configuration, not code structure.

### Factor 11: Logs as event streams

Logs are emitted for lifecycle and business events. This supports debugging and relies less on local console debugging during outages.

### Factor 12: Admin processes as one-off tasks

The service is structured so maintenance and operational actions can be handled as separate, well-defined processes rather than embedded in the request lifecycle.

---

## 6. Observability architecture

This project intentionally includes Prometheus metrics because it demonstrates a mature mindset.

### What is measured

- HTTP request counts by path, method, and status
- request latency distribution
- in-flight requests
- business-level user operations like create, read, update, delete

This shows you are thinking about both system health and product behavior.

### Why this matters

A senior engineer should be able to say:

> Metrics are not optional. They are part of the service contract. Without them, teams operate in the dark during incidents and cannot honestly estimate capacity or performance.

This kind of statement is exactly what senior and principal interviews are looking for.

---

## 7. Reliability and resiliency thinking

### Graceful shutdown

The application listens for `SIGINT` and `SIGTERM` and then performs a controlled shutdown. This is important in container environments, orchestrators, and service migrations.

### Validation before mutation

The service validates user payloads before writing to storage. This prevents partially invalid data from entering the system and helps keep the domain consistent.

### Defensive response handling

HTTP responses and error payloads are consistent. That matters because clients need predictable behavior, and operators need meaningful errors when incidents happen.

---

## 8. Design trade-offs and honest discussion

A strong interview answer is not only about what is good; it also includes awareness of trade-offs.

### Trade-off: in-memory repository

This is intentionally simple for demonstration, but it is not production-grade. It avoids the complexity of a real datastore while still proving the architecture pattern.

A principal-level answer would say:

> For education and demonstration, an in-memory store is useful because it removes external dependencies. In production, I would replace it with a durable database like PostgreSQL and make the storage layer explicitly transactional and observable.

### Trade-off: single service responsibility

This service is intentionally narrow. It does not include authentication, authorization, auditing, queueing, or distributed transaction management. That is a strength because it keeps the architecture teachable.

### Trade-off: direct HTTP API

The service exposes a REST-like API and not a full event-driven architecture. That is appropriate for the scope of the project, but it would not be the right choice for a multi-region, highly decoupled, asynchronous platform.

---

## 9. How to talk about this in an interview

Here is a more advanced answer you can practice:

> This service demonstrates a layered microservice architecture with explicit boundaries between transport, domain logic, and persistence. It follows 12-factor principles by externalizing configuration, exposing liveness and readiness, and treating dependencies as attached resources. The application uses dependency injection to keep service behavior testable and the repository abstraction allows persistence to evolve without rewriting the domain layer. Prometheus instrumentation gives us operational visibility into latency, request volume, and domain activity, which is essential for production operations and SRE practices. The design is intentionally small, but it reflects the way I think about service architecture at scale: strong boundaries, low coupling, observability, graceful lifecycle management, and clear operational trade-offs.

That answer demonstrates senior-level architectural thinking rather than just coding ability.

---

## 10. What a stronger production version would look like

If this were moved from a learning project into a production system, the next evolution would likely include:

- PostgreSQL for durable persistence
- migrations and schema management
- structured logging in JSON format
- OpenTelemetry tracing
- rate limiting and request validation middleware
- authentication and authorization
- metrics and alert dashboards
- distributed tracing across upstream dependencies
- CI/CD pipeline and deployment automation
- secrets management and zero-trust patterns

That discussion is especially valuable in a principal interview because it shows you understand what is necessary to move from demo architecture to enterprise-grade design.

---

## 11. One-minute interview summary

If you need a short, crisp summary:

> This is a small Go microservice designed around domain-driven boundaries, 12-factor principles, and production readiness. It can be scaled horizontally, observed with Prometheus, and evolved without tightly coupling business logic to transport or persistence concerns. It demonstrates the kind of architectural discipline I bring to real-world systems: explicit contracts, operational awareness, and a focus on maintainability and resilience.

---

## 12. Suggested practice prompts

Use these prompts while rehearsing:

1. What are the major responsibilities of each layer in this service?
2. How would you evolve the in-memory repository to PostgreSQL?
3. How does readiness differ from liveness?
4. Why is Prometheus useful here?
5. What would you change for production security?
6. How would you scale this service horizontally?
7. What are the failure modes if the repository becomes slow or unavailable?
8. How would you add tracing and correlation IDs?
9. What does this service do well for 12-factor compliance?
10. What trade-offs did you make to keep the service simple enough for learning?

These are the kinds of conversation starters that feel natural in a principal-level architectural interview.
