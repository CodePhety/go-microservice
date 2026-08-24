package config

import (
	"os"
	"strconv"
)

// Config holds the runtime values that affect how the service behaves.
// Designing configuration as a struct is a 12-factor principle because it keeps
// environment-specific behavior outside the code and makes deployments portable.
type Config struct {
	Port        int
	Environment string
}

// Load reads application configuration from environment variables.
// It provides reasonable defaults so local development can start with minimal setup
// while still allowing production deployments to override values using env vars.
func Load() Config {
	return Config{
		Port:        getEnvInt("PORT", 8080),
		Environment: getEnvString("APP_ENV", "development"),
	}
}

// getEnvString returns a string value from the environment or a fallback.
// This pattern keeps configuration centralized and easy to reason about during
// debugging and interviews, since every important setting has one source of truth.
func getEnvString(key string, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}

// getEnvInt converts an environment variable to an integer.
// If the value is missing or invalid, the fallback is used so the service can fail
// safely instead of crashing during startup.
func getEnvInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		parsed, err := strconv.Atoi(value)
		if err == nil {
			return parsed
		}
	}
	return fallback
}
