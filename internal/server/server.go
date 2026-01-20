package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ngenohkevin/hivedeck-agent/config"
)

// Server represents the HTTP server
type Server struct {
	cfg           *config.Config
	router        *gin.Engine
	handlers      *Handlers
	setupHandlers *SetupHandlers
	auth          *AuthService
	limiter       *RateLimiter
	httpServer    *http.Server
}

// New creates a new server instance
func New(cfg *config.Config) *Server {
	// Set Gin mode based on log level
	if cfg.LogLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	auth := NewAuthService(cfg.APIKey, cfg.JWTSecret)
	limiter := NewRateLimiter(cfg.RateLimitRPS)
	handlers := NewHandlers(cfg)
	setupHandlers := NewSetupHandlers(cfg)

	s := &Server{
		cfg:           cfg,
		router:        router,
		handlers:      handlers,
		setupHandlers: setupHandlers,
		auth:          auth,
		limiter:       limiter,
	}

	s.setupMiddleware()
	s.setupRoutes()

	return s
}

func (s *Server) setupMiddleware() {
	// Recovery middleware
	s.router.Use(RecoveryMiddleware())

	// Logger middleware
	s.router.Use(LoggerMiddleware())

	// CORS middleware
	s.router.Use(CORSMiddleware(s.cfg.AllowedOrigins))

	// Rate limiting
	s.router.Use(RateLimitMiddleware(s.limiter))
}

func (s *Server) setupRoutes() {
	// Health check (no auth)
	s.router.GET("/health", s.handlers.HealthCheck)

	// Setup routes (no auth required in setup mode)
	if s.cfg.SetupMode {
		setup := s.router.Group("/setup")
		{
			setup.GET("", s.setupHandlers.SetupPage)
			setup.POST("/generate", s.setupHandlers.GenerateKey)
			setup.POST("/save", s.setupHandlers.SaveKey)
		}
	}

	// API routes (require auth)
	api := s.router.Group("/api")
	api.Use(AuthMiddleware(s.auth))
	{
		// Server info
		api.GET("/info", s.handlers.GetInfo)

		// Metrics
		api.GET("/metrics", s.handlers.GetAllMetrics)
		api.GET("/metrics/cpu", s.handlers.GetCPUMetrics)
		api.GET("/metrics/memory", s.handlers.GetMemoryMetrics)
		api.GET("/metrics/disk", s.handlers.GetDiskMetrics)
		api.GET("/metrics/network", s.handlers.GetNetworkMetrics)

		// Processes
		api.GET("/processes", s.handlers.ListProcesses)
		api.POST("/processes/:pid/kill", s.handlers.KillProcess)

		// Services (systemd)
		api.GET("/services", s.handlers.ListServices)
		api.GET("/services/:name", s.handlers.GetService)
		api.POST("/services/:name/start", s.handlers.StartService)
		api.POST("/services/:name/stop", s.handlers.StopService)
		api.POST("/services/:name/restart", s.handlers.RestartService)

		// Logs
		api.GET("/logs", s.handlers.StreamLogs)
		api.GET("/logs/query", s.handlers.GetLogs)
		api.GET("/logs/:unit", s.handlers.GetUnitLogs)

		// Docker
		api.GET("/docker/containers", s.handlers.ListContainers)
		api.GET("/docker/containers/:id", s.handlers.GetContainer)
		api.POST("/docker/containers/:id/start", s.handlers.StartContainer)
		api.POST("/docker/containers/:id/stop", s.handlers.StopContainer)
		api.POST("/docker/containers/:id/restart", s.handlers.RestartContainer)
		api.GET("/docker/containers/:id/logs", s.handlers.GetContainerLogs)

		// Files
		api.GET("/files", s.handlers.ListDirectory)
		api.GET("/files/content", s.handlers.GetFileContent)
		api.GET("/files/diskusage", s.handlers.GetDiskUsage)

		// Tasks
		api.GET("/tasks", s.handlers.ListTasks)
		api.POST("/tasks/:name/run", s.handlers.RunTask)

		// Real-time events (SSE)
		api.GET("/events", s.handlers.StreamEvents)

		// Settings (authenticated)
		api.GET("/settings", s.setupHandlers.GetSettings)
		api.PUT("/settings", s.setupHandlers.UpdateSettings)
		api.POST("/settings/generate-key", s.setupHandlers.GenerateKey)
		api.POST("/settings/api-key", s.setupHandlers.SaveKey)
	}

	// Settings page (requires auth via query param)
	s.router.GET("/settings", s.setupHandlers.SettingsPage)
}

// Run starts the HTTP server
func (s *Server) Run() error {
	s.httpServer = &http.Server{
		Addr:         s.cfg.Addr(),
		Handler:      s.router,
		ReadTimeout:  s.cfg.ReadTimeout,
		WriteTimeout: s.cfg.WriteTimeout,
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("Shutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := s.httpServer.Shutdown(ctx); err != nil {
			log.Printf("Server forced to shutdown: %v", err)
		}
	}()

	log.Printf("Starting Hivedeck Agent on %s", s.cfg.Addr())

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}

	// Clean up
	if err := s.handlers.Close(); err != nil {
		log.Printf("Error closing handlers: %v", err)
	}

	log.Println("Server stopped")
	return nil
}

// Router returns the Gin router (for testing)
func (s *Server) Router() *gin.Engine {
	return s.router
}
