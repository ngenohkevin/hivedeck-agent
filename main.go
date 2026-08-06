package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/ngenohkevin/hivedeck-agent/config"
	"github.com/ngenohkevin/hivedeck-agent/internal/server"
)

// Build metadata, injected with -ldflags at build time (see Makefile / CI).
var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("hivedeck-agent %s (built %s)\n", Version, BuildTime)
		return
	}

	server.SetBuildInfo(Version, BuildTime)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Check if in setup mode
	if cfg.SetupMode {
		log.Printf("⚠️  No API key configured - starting in SETUP MODE")
		log.Printf("📋 Open http://%s/setup to configure the agent", cfg.Addr())
		log.Printf("🔒 Setup is only reachable from loopback, LAN or Tailscale")
		log.Printf("🔒 After setup, restart the agent to enable authentication")
	}

	// Create and run server
	srv := server.New(cfg)
	if err := srv.Run(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
