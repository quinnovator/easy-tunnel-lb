package config

import (
	"os"
)

// Config holds the configuration for the easy-tunnel-lb agent
type Config struct {
	ServerURL     string
	APIKey        string
	LogLevel      string
	WatchInterval int
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	config := &Config{
		ServerURL:     getEnvOrDefault("SERVER_URL", "http://localhost:8080"),
		APIKey:        getEnvOrDefault("API_KEY", ""),
		LogLevel:      getEnvOrDefault("LOG_LEVEL", "info"),
		WatchInterval: 30, // Default 30 seconds
	}

	if config.APIKey == "" {
		return nil, ErrMissingAPIKey
	}

	return config, nil
}

// getEnvOrDefault retrieves an environment variable or returns a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// Error types for configuration
var (
	ErrMissingAPIKey = ConfigError("API_KEY environment variable is required")
)

// ConfigError represents a configuration error
type ConfigError string

func (e ConfigError) Error() string {
	return string(e)
} 