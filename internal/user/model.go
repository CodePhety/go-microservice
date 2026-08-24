package user

// User represents the core domain entity for this service.
// It is intentionally small and focused because a good domain model should express
// the business concept clearly without spilling transport or database concerns into it.
type User struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Active    bool   `json:"active"`
	CreatedAt string `json:"created_at"`
}

// CreateUserRequest is the incoming payload for creating a user.
// This shape keeps API contracts explicit and allows validation before the request
// reaches the repository layer.
type CreateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Active bool `json:"active"`
}

// UpdateUserRequest is the request body used when modifying an existing user.
// It mirrors the write domain in a way that keeps partial updates easy to reason about.
type UpdateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Active bool `json:"active"`
}

// ErrorResponse is a consistent payload returned for failed requests.
// Centralizing error payloads helps API clients build predictable error handling
// and makes interviews easier because the contract is easy to explain.
type ErrorResponse struct {
	Error string `json:"error"`
}
