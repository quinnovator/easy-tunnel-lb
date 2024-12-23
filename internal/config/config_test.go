package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		expectError bool
		expected    *Config
	}{
		{
			name: "valid config",
			envVars: map[string]string{
				"SERVER_URL": "https://example.com",
				"API_KEY":    "test-key",
				"LOG_LEVEL":  "debug",
			},
			expectError: false,
			expected: &Config{
				ServerURL:     "https://example.com",
				APIKey:        "test-key",
				LogLevel:      "debug",
				WatchInterval: 30,
			},
		},
		{
			name:        "missing API key",
			envVars:     map[string]string{},
			expectError: true,
			expected:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			os.Clearenv()

			// Set test environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			// Run test
			cfg, err := LoadConfig()

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, cfg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, cfg)
			}
		})
	}
} 