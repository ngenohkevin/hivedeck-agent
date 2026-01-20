package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// GenerateAPIKey generates a secure random API key
func GenerateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

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
	AllowedPaths    []string

	// Setup mode
	SetupMode bool
	EnvFile   string
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
	// Determine .env file path
	envFile := getEnvFile()

	// Load .env file if it exists
	_ = godotenv.Load(envFile)

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
		AllowedPaths: getEnvSlice("ALLOWED_PATHS", []string{
			"/var/log",
			"/etc",
			"/home",
			"/opt",
			"/tmp",
		}),
		SetupMode: false,
		EnvFile:   envFile,
	}

	// Check if API key is configured
	if cfg.APIKey == "" {
		cfg.SetupMode = true
		return cfg, nil
	}

	if cfg.JWTSecret == "" {
		// Use API key as fallback for JWT secret
		cfg.JWTSecret = cfg.APIKey
	}

	return cfg, nil
}

// getEnvFile returns the path to the .env file
func getEnvFile() string {
	// Check if running from a specific directory
	if envFile := os.Getenv("ENV_FILE"); envFile != "" {
		return envFile
	}

	// Try to find .env in current directory or executable directory
	if _, err := os.Stat(".env"); err == nil {
		return ".env"
	}

	// Get executable directory
	exe, err := os.Executable()
	if err == nil {
		dir := strings.TrimSuffix(exe, "/hivedeck-agent")
		envPath := dir + "/.env"
		if _, err := os.Stat(envPath); err == nil {
			return envPath
		}
	}

	return ".env"
}

// SaveAPIKey saves the API key to the .env file
func (c *Config) SaveAPIKey(apiKey string) error {
	updates := map[string]string{"API_KEY": apiKey}
	if err := UpdateEnvFile(c.EnvFile, updates); err != nil {
		return err
	}

	// Update config
	c.APIKey = apiKey
	c.JWTSecret = apiKey
	c.SetupMode = false

	return nil
}

// UpdateEnvFile updates or adds environment variables in a .env file
func UpdateEnvFile(envFile string, updates map[string]string) error {
	// Read existing .env content
	existingContent := ""
	if data, err := os.ReadFile(envFile); err == nil {
		existingContent = string(data)
	}

	// Parse existing lines
	lines := strings.Split(existingContent, "\n")
	found := make(map[string]bool)

	// Update existing keys
	for i, line := range lines {
		for key, value := range updates {
			if strings.HasPrefix(line, key+"=") {
				lines[i] = key + "=" + value
				found[key] = true
				break
			}
		}
	}

	// Add missing keys at the beginning
	var newLines []string
	for key, value := range updates {
		if !found[key] {
			newLines = append(newLines, key+"="+value)
		}
	}
	if len(newLines) > 0 {
		lines = append(newLines, lines...)
	}

	// Remove empty lines at the end
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}

	// Write back to .env file
	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(envFile, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write .env file: %w", err)
	}

	return nil
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
		AllowedPaths:    []string{"/tmp", "/var/log"},
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
