package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"user-service/internal/config"
	"user-service/internal/metrics"
	"user-service/internal/user"
)

// Server wires the HTTP layer to the domain service.
// This package is responsible for request parsing and response shaping, while the
// user service remains focused on business rules and domain validation.
type Server struct {
	service *user.Service
	config  config.Config
	logger  *log.Logger
	metrics *metrics.Metrics
}

// New creates a new HTTP server adapter around a service instance.
// This is a classic separation-of-concerns pattern: transport concerns are isolated
// from business logic, while the logger remains available for request-level observability.
func New(service *user.Service, cfg config.Config, logger *log.Logger, telemetry *metrics.Metrics) *Server {
	if logger == nil || logger.Writer() == nil {
		logger = log.New(io.Discard, "", 0)
	}
	if telemetry == nil {
		telemetry = metrics.New()
	}
	return &Server{
		service: service,
		config:  cfg,
		logger:  logger,
		metrics: telemetry,
	}
}

// Router registers the endpoints for the user microservice.
// The route design keeps the service easy to explain in an interview: health, ready,
// and CRUD user operations are all clear and discoverable.
func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()

	// Prometheus scraping is deliberately exposed on a separate endpoint because it
	// should be considered a system concern, not part of the business API footprint.
	mux.Handle("/metrics", s.metrics.Handler())
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/ready", s.handleReady)
	mux.HandleFunc("/users", s.handleUsers)
	mux.HandleFunc("/users/", s.handleUserByID)

	return s.metrics.Middleware(mux)
}

// handleHealth returns a lightweight health check signal.
// Health endpoints are required by 12-factor principles because containers and load
// balancers need to tell whether the service is alive without depending on business logic.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	// Health checks are intentionally cheap and read-only. They tell a platform whether
	// the process is alive without attempting deep dependency verification.
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "user-service",
		"env":     s.config.Environment,
		"uptime":  time.Since(startTime).String(),
	})
}

// handleReady reports whether the service is ready to serve requests.
// Readiness checks are different from liveness checks because they can fail when the
// service has not finished its startup dependencies, even though it is still running.
func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	// Readiness is a stronger signal than liveness. It answers: "can this instance take
	// traffic right now?" It is often used in orchestration and load-balancing decisions.
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ready",
	})
}

// handleUsers dispatches list and create requests for the /users collection.
// This method demonstrates a common microservice design: collection endpoints are
// separated from resource-specific endpoints for clarity and good API semantics.
func (s *Server) handleUsers(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/users" {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleListUsers(w, r)
	case http.MethodPost:
		s.handleCreateUser(w, r)
	default:
		writeMethodNotAllowed(w)
	}
}

// handleUserByID dispatches GET, PUT, and DELETE requests for a single user.
// Resource-level routes allow the API to follow REST conventions and make the service
// easier to discuss in an interview because each action has a clear path and contract.
func (s *Server) handleUserByID(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, "/users/") {
		http.NotFound(w, r)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/users/")
	if id == "" || strings.Contains(id, "/") {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetUser(w, r, id)
	case http.MethodPut:
		s.handleUpdateUser(w, r, id)
	case http.MethodDelete:
		s.handleDeleteUser(w, r, id)
	default:
		writeMethodNotAllowed(w)
	}
}

// handleListUsers reads all users from the service and returns them as JSON.
func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	usersList, err := s.service.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.logger.Printf("serving %d users from list endpoint", len(usersList))
	writeJSON(w, http.StatusOK, usersList)
}

// handleGetUser fetches a single user and returns either the resource or not found.
func (s *Server) handleGetUser(w http.ResponseWriter, r *http.Request, id string) {
	userRecord, err := s.service.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, userRecord)
}

// handleCreateUser decodes the input JSON and creates a user through the domain service.
func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var payload user.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	created, err := s.service.Create(r.Context(), payload)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.logger.Printf("created user %s via API", created.ID)
	writeJSON(w, http.StatusCreated, created)
}

// handleUpdateUser updates an existing user and returns the new state.
func (s *Server) handleUpdateUser(w http.ResponseWriter, r *http.Request, id string) {
	var payload user.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	updated, err := s.service.Update(r.Context(), id, payload)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

// handleDeleteUser removes a user by ID and returns a success response when it succeeds.
func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request, id string) {
	if err := s.service.Delete(r.Context(), id); err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id})
}

// writeJSON writes a typed JSON response to the client.
// This helper keeps handlers consistent and avoids repetitive response logic.
func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, "failed to write response", http.StatusInternalServerError)
	}
}

// writeError standardizes error responses for all HTTP handlers.
// A consistent error payload supports client debugging and makes the API easier to
// consume in both tests and production systems.
func writeError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, user.ErrorResponse{Error: message})
}

// writeMethodNotAllowed sends the standard HTTP response for unsupported methods.
func writeMethodNotAllowed(w http.ResponseWriter) {
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

// startTime gives the HTTP layer a simple uptime signal for health reporting.
// This is a good conversation point in interviews because it demonstrates operational
// awareness beyond the happy path.
var startTime = time.Now()

// context.Background is used in this example only for the main lifecycle; the server
// itself uses request-scoped contexts in each handler, which is a best practice for
// cancellation and time-bound work.
var _ = context.Background
