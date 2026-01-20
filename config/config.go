package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the agent
type Config struct {
	// Server settings
	Port         int
	Host         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration

	// Authentication
	APIKey    string
	JWTSecret string

	// Security
	AllowedOrigins []string
	RateLimitRPS   int

	// Features
	DockerEnabled bool

	// Logging
	LogLevel string

	// Allowed operations
	AllowedServices []string
	AllowedTasks    map[string]Task
}

// Task represents a pre-defined safe command
type Task struct {
	Name        string
	Command     string
	Description string
	Dangerous   bool
}

// DefaultTasks returns the pre-defined safe commands
func DefaultTasks() map[string]Task {
	return map[string]Task{
		"apt-update": {
			Name:        "apt-update",
			Command:     "apt update",
			Description: "Update package lists",
			Dangerous:   false,
		},
		"apt-upgrade": {
			Name:        "apt-upgrade",
			Command:     "apt upgrade -y",
			Description: "Upgrade packages",
			Dangerous:   false,
		},
		"df": {
			Name:        "df",
			Command:     "df -h",
			Description: "Check disk space",
			Dangerous:   false,
		},
		"free": {
			Name:        "free",
			Command:     "free -m",
			Description: "Check memory",
			Dangerous:   false,
		},
		"uptime": {
			Name:        "uptime",
			Command:     "uptime",
			Description: "System uptime",
			Dangerous:   false,
		},
		"who": {
			Name:        "who",
			Command:     "who",
			Description: "Logged-in users",
			Dangerous:   false,
		},
		"pi-temp": {
			Name:        "pi-temp",
			Command:     "vcgencmd measure_temp",
			Description: "Pi temperature",
			Dangerous:   false,
		},
		"reboot": {
			Name:        "reboot",
			Command:     "reboot",
			Description: "Reboot system",
			Dangerous:   true,
		},
	}
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	cfg := &Config{
		Port:           getEnvInt("PORT", 8091),
		Host:           getEnv("HOST", "0.0.0.0"),
		ReadTimeout:    time.Duration(getEnvInt("READ_TIMEOUT_SECONDS", 30)) * time.Second,
		WriteTimeout:   time.Duration(getEnvInt("WRITE_TIMEOUT_SECONDS", 300)) * time.Second,
		APIKey:         getEnv("API_KEY", ""),
		JWTSecret:      getEnv("JWT_SECRET", ""),
		AllowedOrigins: getEnvSlice("ALLOWED_ORIGINS", []string{"*"}),
		RateLimitRPS:   getEnvInt("RATE_LIMIT_RPS", 100),
		DockerEnabled:  getEnvBool("DOCKER_ENABLED", true),
		LogLevel:       getEnv("LOG_LEVEL", "info"),
		AllowedServices: getEnvSlice("ALLOWED_SERVICES", []string{
			"routerctl-agent",
			"hivedeck-agent",
			"docker",
			"nginx",
			"ssh",
			"tailscaled",
		}),
		AllowedTasks: DefaultTasks(),
	}

	// Validate required fields
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API_KEY is required")
	}

	if cfg.JWTSecret == "" {
		// Use API key as fallback for JWT secret
		cfg.JWTSecret = cfg.APIKey
	}

	return cfg, nil
}

// LoadWithDefaults loads config with defaults for testing
func LoadWithDefaults() *Config {
	return &Config{
		Port:            8091,
		Host:            "0.0.0.0",
		ReadTimeout:     30 * time.Second,
		WriteTimeout:    300 * time.Second,
		APIKey:          "test-api-key",
		JWTSecret:       "test-jwt-secret",
		AllowedOrigins:  []string{"*"},
		RateLimitRPS:    100,
		DockerEnabled:   true,
		LogLevel:        "info",
		AllowedServices: []string{"test-service"},
		AllowedTasks:    DefaultTasks(),
	}
}

// Addr returns the server address string
func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// IsServiceAllowed checks if a service can be managed
func (c *Config) IsServiceAllowed(service string) bool {
	for _, s := range c.AllowedServices {
		if s == service {
			return true
		}
	}
	return false
}

// GetTask returns a task by name if it exists
func (c *Config) GetTask(name string) (Task, bool) {
	task, ok := c.AllowedTasks[name]
	return task, ok
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}
