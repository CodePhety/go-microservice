package user

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"user-service/internal/metrics"
)

// ErrUserNotFound is returned when an operation asks for a user ID that is not in
// the current data store. This error is a domain-level signal, not a transport detail.
var ErrUserNotFound = errors.New("user not found")

// Repository defines the persistence contract.
// Separating the domain service from the data layer is a strong design principle
// because it ensures the business logic can be tested independently from storage.
type Repository interface {
	List(ctx context.Context) ([]User, error)
	Get(ctx context.Context, id string) (User, error)
	Create(ctx context.Context, user User) (User, error)
	Update(ctx context.Context, user User) (User, error)
	Delete(ctx context.Context, id string) error
}

// Service contains the business logic for the user domain.
// This layer acts as the orchestration boundary: it validates input, applies domain
// rules, and communicates with persistence without exposing storage details to handlers.
type Service struct {
	repo     Repository
	logger   *log.Logger
	metrics  *metrics.Metrics
}

// NewService creates a new application service with its dependency on a repository.
// Dependency injection makes the design easier to test and aligns with good senior
// design practices because external concerns are passed in rather than hard-coded.
func NewService(repo Repository, logger *log.Logger, telemetry *metrics.Metrics) *Service {
	logger = normalizeLogger(logger)
	if telemetry == nil {
		telemetry = metrics.New()
	}

	return &Service{
		repo:    repo,
		logger:  logger,
		metrics: telemetry,
	}
}

// List returns every user currently stored.
// The method exposes a stable operation that can be used by the HTTP layer for read
// endpoints and gives interview candidates a clean place to discuss pagination later.
func (s *Service) List(ctx context.Context) ([]User, error) {
	start := time.Now()
	s.logger.Printf("listing users")
	users, err := s.repo.List(ctx)
	if err != nil {
		s.metrics.RecordUserOperation("list", "error")
		return nil, err
	}
	s.metrics.RecordUserOperation("list", "success")
	s.logger.Printf("list users completed in %s", time.Since(start))
	return users, nil
}

// Get retrieves a single user by ID.
// This is a good example of a focused domain query: it translates an identifier into
// a single business object and returns a domain error if the record is absent.
func (s *Service) Get(ctx context.Context, id string) (User, error) {
	if id == "" {
		s.metrics.RecordUserOperation("get", "validation_error")
		return User{}, errors.New("user id is required")
	}

	start := time.Now()
	s.logger.Printf("retrieving user %s", id)
	user, err := s.repo.Get(ctx, id)
	if err != nil {
		s.metrics.RecordUserOperation("get", "error")
		return User{}, err
	}
	s.metrics.RecordUserOperation("get", "success")
	s.logger.Printf("get user %s completed in %s", id, time.Since(start))
	return user, nil
}

// Create validates a creation request and stores a new user.
// This is the primary write operation and is where domain-level validation belongs.
func (s *Service) Create(ctx context.Context, req CreateUserRequest) (User, error) {
	if req.Name == "" {
		s.metrics.RecordUserOperation("create", "validation_error")
		return User{}, errors.New("name is required")
	}
	if req.Email == "" {
		s.metrics.RecordUserOperation("create", "validation_error")
		return User{}, errors.New("email is required")
	}

	nextUser := User{
		ID:        generateUserID(),
		Name:      req.Name,
		Email:     req.Email,
		Active:    req.Active,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	start := time.Now()
	s.logger.Printf("creating user %s", nextUser.ID)
	created, err := s.repo.Create(ctx, nextUser)
	if err != nil {
		s.metrics.RecordUserOperation("create", "error")
		return User{}, err
	}
	s.metrics.RecordUserOperation("create", "success")
	s.logger.Printf("create user %s completed in %s", created.ID, time.Since(start))
	return created, nil
}

// Update changes the fields of an existing user.
// The method validates the target identifier and then applies the write operation to
// the current domain object, making the business rule clear to future maintainers.
func (s *Service) Update(ctx context.Context, id string, req UpdateUserRequest) (User, error) {
	if id == "" {
		s.metrics.RecordUserOperation("update", "validation_error")
		return User{}, errors.New("user id is required")
	}
	if req.Name == "" {
		s.metrics.RecordUserOperation("update", "validation_error")
		return User{}, errors.New("name is required")
	}
	if req.Email == "" {
		s.metrics.RecordUserOperation("update", "validation_error")
		return User{}, errors.New("email is required")
	}

	start := time.Now()
	existing, err := s.repo.Get(ctx, id)
	if err != nil {
		s.metrics.RecordUserOperation("update", "error")
		return User{}, err
	}

	existing.Name = req.Name
	existing.Email = req.Email
	existing.Active = req.Active

	s.logger.Printf("updating user %s", id)
	updated, err := s.repo.Update(ctx, existing)
	if err != nil {
		s.metrics.RecordUserOperation("update", "error")
		return User{}, err
	}
	s.metrics.RecordUserOperation("update", "success")
	s.logger.Printf("update user %s completed in %s", id, time.Since(start))
	return updated, nil
}

// Delete removes a user by ID.
// This is an example of a command-style operation: it changes state and returns an
// error only when the business operation cannot be completed.
func (s *Service) Delete(ctx context.Context, id string) error {
	if id == "" {
		s.metrics.RecordUserOperation("delete", "validation_error")
		return errors.New("user id is required")
	}

	start := time.Now()
	s.logger.Printf("deleting user %s", id)
	if err := s.repo.Delete(ctx, id); err != nil {
		s.metrics.RecordUserOperation("delete", "error")
		return err
	}
	s.metrics.RecordUserOperation("delete", "success")
	s.logger.Printf("delete user %s completed in %s", id, time.Since(start))
	return nil
}

// generateUserID creates a unique identifier using the current timestamp.
// This is intentionally simple for interview practice and keeps the service easy to
// understand, but in production you may prefer a UUID library for better portability.
func generateUserID() string {
	return fmt.Sprintf("user-%d", time.Now().UnixNano())
}

// InMemoryRepository stores user data in-process.
// This is a practical learning pattern for interviews because it demonstrates
// persistence abstraction without hiding the fact that production should use a real DB.
type InMemoryRepository struct {
	mu    sync.RWMutex
	users map[string]User
	logger *log.Logger
}

// NewInMemoryRepository initializes an empty in-memory data store.
// The logger is optional; if not provided, it falls back to a quiet logger to avoid nil dereferences.
func NewInMemoryRepository(logger *log.Logger) *InMemoryRepository {
	logger = normalizeLogger(logger)

	return &InMemoryRepository{
		users: map[string]User{},
		logger: logger,
	}
}

func normalizeLogger(logger *log.Logger) *log.Logger {
	if logger == nil || logger.Writer() == nil {
		return log.New(io.Discard, "", 0)
	}
	return logger
}

// List returns all current users while holding a read lock.
// Read locks allow concurrent access for read-heavy workloads and demonstrate a basic
// concurrency pattern that senior engineers expect when discussing thread safety.
func (r *InMemoryRepository) List(ctx context.Context) ([]User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	users := make([]User, 0, len(r.users))
	for _, user := range r.users {
		users = append(users, user)
	}
	return users, nil
}

// Get finds a single user by ID.
// This method returns a domain error when the record is missing so callers can treat
// "not found" consistently across the application.
func (r *InMemoryRepository) Get(ctx context.Context, id string) (User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, found := r.users[id]
	if !found {
		return User{}, ErrUserNotFound
	}
	return user, nil
}

// Create inserts a new user into the in-memory map.
// The method is intentionally simple, but it illustrates what a repository contract
// looks like in a layered architecture.
func (r *InMemoryRepository) Create(ctx context.Context, user User) (User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.users[user.ID] = user
	r.logger.Printf("stored user %s", user.ID)
	return user, nil
}

// Update replaces an existing user record.
// Because the repository uses a mutex, it remains safe even when multiple requests are
// trying to mutate the same store concurrently.
func (r *InMemoryRepository) Update(ctx context.Context, user User) (User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.users[user.ID]; !exists {
		return User{}, ErrUserNotFound
	}

	r.users[user.ID] = user
	return user, nil
}

// Delete removes a user from storage.
// This holds a write lock so it remains consistent with the rest of the repository.
func (r *InMemoryRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.users[id]; !exists {
		return ErrUserNotFound
	}

	delete(r.users, id)
	return nil
}
