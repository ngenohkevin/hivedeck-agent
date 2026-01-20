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

	// Create and run server
	srv := server.New(cfg)
	if err := srv.Run(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
