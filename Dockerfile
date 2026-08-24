# syntax=docker/dockerfile:1

# Build stage: compile the Go binary in a controlled environment.
# This follows a multi-stage pattern that keeps the final runtime image small and
# improves security by excluding build tooling from the production artifact.
FROM golang:1.22-alpine AS builder

WORKDIR /src

# Copy dependency manifests first to leverage Docker layer caching.
COPY go.mod go.sum* ./
RUN go mod download

# Copy source and compile a statically linked binary.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/user-service .

# Runtime stage: minimal image with the binary and no shell or package manager.
FROM gcr.io/distroless/base-debian12:nonroot

WORKDIR /app

# Expose the application port in a container-friendly way.
EXPOSE 8080

# Copy only the final compiled artifact into the runtime image.
COPY --from=builder /out/user-service /app/user-service

# Set the runtime environment for the service.
ENV APP_ENV=production
ENV PORT=8080

ENTRYPOINT ["/app/user-service"]
