package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadWithDefaults(t *testing.T) {
	cfg := LoadWithDefaults()

	assert.NotNil(t, cfg)
	assert.Equal(t, 8091, cfg.Port)
	assert.Equal(t, "0.0.0.0", cfg.Host)
	assert.Equal(t, "test-api-key", cfg.APIKey)
	assert.NotEmpty(t, cfg.AllowedTasks)
}

func TestLoadMissingAPIKey(t *testing.T) {
	// Clear environment
	os.Unsetenv("API_KEY")

	_, err := Load()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API_KEY is required")
}

func TestLoadWithEnvVars(t *testing.T) {
	// Set environment variables
	os.Setenv("API_KEY", "my-test-key")
	os.Setenv("PORT", "9000")
	os.Setenv("HOST", "127.0.0.1")
	os.Setenv("LOG_LEVEL", "debug")
	defer func() {
		os.Unsetenv("API_KEY")
		os.Unsetenv("PORT")
		os.Unsetenv("HOST")
		os.Unsetenv("LOG_LEVEL")
	}()

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "my-test-key", cfg.APIKey)
	assert.Equal(t, 9000, cfg.Port)
	assert.Equal(t, "127.0.0.1", cfg.Host)
	assert.Equal(t, "debug", cfg.LogLevel)
}

func TestConfigAddr(t *testing.T) {
	cfg := LoadWithDefaults()
	assert.Equal(t, "0.0.0.0:8091", cfg.Addr())
}

func TestIsServiceAllowed(t *testing.T) {
	cfg := LoadWithDefaults()
	cfg.AllowedServices = []string{"nginx", "docker", "ssh"}

	assert.True(t, cfg.IsServiceAllowed("nginx"))
	assert.True(t, cfg.IsServiceAllowed("docker"))
	assert.False(t, cfg.IsServiceAllowed("mysql"))
}

func TestGetTask(t *testing.T) {
	cfg := LoadWithDefaults()

	task, ok := cfg.GetTask("df")
	assert.True(t, ok)
	assert.Equal(t, "df -h", task.Command)
	assert.False(t, task.Dangerous)

	task, ok = cfg.GetTask("reboot")
	assert.True(t, ok)
	assert.True(t, task.Dangerous)

	_, ok = cfg.GetTask("nonexistent")
	assert.False(t, ok)
}

func TestDefaultTasks(t *testing.T) {
	tasks := DefaultTasks()

	assert.NotEmpty(t, tasks)
	assert.Contains(t, tasks, "df")
	assert.Contains(t, tasks, "free")
	assert.Contains(t, tasks, "uptime")
	assert.Contains(t, tasks, "reboot")

	// Verify reboot is marked as dangerous
	assert.True(t, tasks["reboot"].Dangerous)
	assert.False(t, tasks["df"].Dangerous)
}
