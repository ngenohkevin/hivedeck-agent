package main

import (
	"log"

	"github.com/ngenohkevin/hivedeck-agent/config"
	"github.com/ngenohkevin/hivedeck-agent/internal/server"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Check if in setup mode
	if cfg.SetupMode {
		log.Printf("⚠️  No API key configured - starting in SETUP MODE")
		log.Printf("📋 Open http://%s/setup to configure the agent", cfg.Addr())
		log.Printf("🔒 After setup, restart the agent to enable authentication")
	}

	// Create and run server
	srv := server.New(cfg)
	if err := srv.Run(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
