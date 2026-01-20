package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ngenohkevin/hivedeck-agent/config"
	"github.com/ngenohkevin/hivedeck-agent/internal/cache"
	"github.com/ngenohkevin/hivedeck-agent/internal/docker"
	"github.com/ngenohkevin/hivedeck-agent/internal/files"
	"github.com/ngenohkevin/hivedeck-agent/internal/process"
	"github.com/ngenohkevin/hivedeck-agent/internal/system"
	"github.com/ngenohkevin/hivedeck-agent/internal/systemd"
	"github.com/ngenohkevin/hivedeck-agent/internal/tasks"
)

// Handlers holds all HTTP handlers
type Handlers struct {
	cfg            *config.Config
	cache          *cache.MetricsCache
	metricsCollector *system.Collector
	processManager *process.Manager
	serviceManager *systemd.Manager
	journalReader  *systemd.JournalReader
	dockerManager  *docker.Manager
	fileBrowser    *files.Browser
	taskManager    *tasks.Manager
}

// NewHandlers creates a new handlers instance
func NewHandlers(cfg *config.Config) *Handlers {
	h := &Handlers{
		cfg:              cfg,
		cache:            cache.NewMetricsCache(),
		metricsCollector: system.NewCollector(),
		processManager:   process.NewManager(),
		serviceManager:   systemd.NewManager(cfg.AllowedServices),
		journalReader:    systemd.NewJournalReader(),
		fileBrowser:      files.NewBrowser(nil),
		taskManager:      tasks.NewManager(cfg.AllowedTasks),
	}

	// Initialize Docker if enabled
	if cfg.DockerEnabled {
		dockerMgr, err := docker.NewManager()
		if err == nil {
			h.dockerManager = dockerMgr
		}
	}

	return h
}

// HealthCheck handles GET /health
func (h *Handlers) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"timestamp": time.Now().UTC(),
		"version":   "1.0.0",
	})
}

// GetInfo handles GET /api/info
func (h *Handlers) GetInfo(c *gin.Context) {
	hostInfo, err := system.GetHostInfo()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"hostname":  hostInfo.Hostname,
		"os":        hostInfo.OS,
		"platform":  hostInfo.Platform,
		"kernel":    hostInfo.KernelVersion,
		"arch":      hostInfo.KernelArch,
		"uptime":    hostInfo.UptimeHuman,
		"agent":     "hivedeck-agent",
		"version":   "1.0.0",
	})
}

// GetAllMetrics handles GET /api/metrics
func (h *Handlers) GetAllMetrics(c *gin.Context) {
	cached, found := h.cache.Get(cache.KeyAll)
	if found {
		c.JSON(http.StatusOK, cached)
		return
	}

	metrics, err := h.metricsCollector.GetAllMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.cache.Set(cache.KeyAll, metrics)
	c.JSON(http.StatusOK, metrics)
}

// GetCPUMetrics handles GET /api/metrics/cpu
func (h *Handlers) GetCPUMetrics(c *gin.Context) {
	cached, found := h.cache.Get(cache.KeyCPU)
	if found {
		c.JSON(http.StatusOK, cached)
		return
	}

	cpu, err := h.metricsCollector.GetCPUInfo()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.cache.Set(cache.KeyCPU, cpu)
	c.JSON(http.StatusOK, cpu)
}

// GetMemoryMetrics handles GET /api/metrics/memory
func (h *Handlers) GetMemoryMetrics(c *gin.Context) {
	cached, found := h.cache.Get(cache.KeyMemory)
	if found {
		c.JSON(http.StatusOK, cached)
		return
	}

	memory, err := h.metricsCollector.GetMemoryInfo()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.cache.Set(cache.KeyMemory, memory)
	c.JSON(http.StatusOK, memory)
}

// GetDiskMetrics handles GET /api/metrics/disk
func (h *Handlers) GetDiskMetrics(c *gin.Context) {
	cached, found := h.cache.Get(cache.KeyDisk)
	if found {
		c.JSON(http.StatusOK, cached)
		return
	}

	disk, err := h.metricsCollector.GetDiskInfo()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.cache.Set(cache.KeyDisk, disk)
	c.JSON(http.StatusOK, disk)
}

// GetNetworkMetrics handles GET /api/metrics/network
func (h *Handlers) GetNetworkMetrics(c *gin.Context) {
	cached, found := h.cache.Get(cache.KeyNetwork)
	if found {
		c.JSON(http.StatusOK, cached)
		return
	}

	network, err := h.metricsCollector.GetNetworkInfo()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.cache.Set(cache.KeyNetwork, network)
	c.JSON(http.StatusOK, network)
}

// ListProcesses handles GET /api/processes
func (h *Handlers) ListProcesses(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 50
	}

	processes, err := h.processManager.ListTop(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, processes)
}

// KillProcess handles POST /api/processes/:pid/kill
func (h *Handlers) KillProcess(c *gin.Context) {
	pidStr := c.Param("pid")
	pid, err := strconv.ParseInt(pidStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pid"})
		return
	}

	var req process.KillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Signal = 15 // Default to SIGTERM
	}

	result, err := h.processManager.Kill(int32(pid), req.Signal)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !result.Success {
		c.JSON(http.StatusForbidden, result)
		return
	}

	c.JSON(http.StatusOK, result)
}

// ListServices handles GET /api/services
func (h *Handlers) ListServices(c *gin.Context) {
	services, err := h.serviceManager.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, services)
}

// GetService handles GET /api/services/:name
func (h *Handlers) GetService(c *gin.Context) {
	name := c.Param("name")

	service, err := h.serviceManager.Get(c.Request.Context(), name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, service)
}

// StartService handles POST /api/services/:name/start
func (h *Handlers) StartService(c *gin.Context) {
	name := c.Param("name")

	result, err := h.serviceManager.Start(c.Request.Context(), name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !result.Success {
		c.JSON(http.StatusForbidden, result)
		return
	}

	c.JSON(http.StatusOK, result)
}

// StopService handles POST /api/services/:name/stop
func (h *Handlers) StopService(c *gin.Context) {
	name := c.Param("name")

	result, err := h.serviceManager.Stop(c.Request.Context(), name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !result.Success {
		c.JSON(http.StatusForbidden, result)
		return
	}

	c.JSON(http.StatusOK, result)
}

// RestartService handles POST /api/services/:name/restart
func (h *Handlers) RestartService(c *gin.Context) {
	name := c.Param("name")

	result, err := h.serviceManager.Restart(c.Request.Context(), name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !result.Success {
		c.JSON(http.StatusForbidden, result)
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetLogs handles GET /api/logs/query
func (h *Handlers) GetLogs(c *gin.Context) {
	query := systemd.JournalQuery{
		Unit:     c.Query("unit"),
		Priority: -1,
		Lines:    100,
	}

	if prio := c.Query("priority"); prio != "" {
		if p, err := strconv.Atoi(prio); err == nil {
			query.Priority = p
		}
	}

	if lines := c.Query("lines"); lines != "" {
		if l, err := strconv.Atoi(lines); err == nil {
			query.Lines = l
		}
	}

	query.Since = c.Query("since")
	query.Until = c.Query("until")

	logs, err := h.journalReader.Query(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, logs)
}

// GetUnitLogs handles GET /api/logs/:unit
func (h *Handlers) GetUnitLogs(c *gin.Context) {
	unit := c.Param("unit")
	lines := 100
	if l := c.Query("lines"); l != "" {
		if n, err := strconv.Atoi(l); err == nil {
			lines = n
		}
	}

	logs, err := h.journalReader.GetRecentLogs(c.Request.Context(), unit, lines)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"unit":    unit,
		"entries": logs,
	})
}

// StreamLogs handles GET /api/logs (SSE)
func (h *Handlers) StreamLogs(c *gin.Context) {
	unit := c.Query("unit")

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	entryChan := make(chan systemd.JournalEntry, 100)

	if err := h.journalReader.Follow(ctx, unit, entryChan); err != nil {
		c.SSEvent("error", gin.H{"error": err.Error()})
		return
	}

	c.Stream(func(w io.Writer) bool {
		select {
		case entry := <-entryChan:
			data, _ := json.Marshal(entry)
			c.SSEvent("log", string(data))
			return true
		case <-ctx.Done():
			return false
		}
	})
}

// StreamEvents handles GET /api/events (SSE metrics)
func (h *Handlers) StreamEvents(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	ctx := c.Request.Context()

	c.Stream(func(w io.Writer) bool {
		select {
		case <-ticker.C:
			metrics, err := h.metricsCollector.GetAllMetrics()
			if err != nil {
				c.SSEvent("error", gin.H{"error": err.Error()})
				return true
			}
			data, _ := json.Marshal(metrics)
			c.SSEvent("metrics", string(data))
			return true
		case <-ctx.Done():
			return false
		}
	})
}

// Docker handlers

// ListContainers handles GET /api/docker/containers
func (h *Handlers) ListContainers(c *gin.Context) {
	if h.dockerManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "docker not available"})
		return
	}

	all := c.Query("all") == "true"

	containers, err := h.dockerManager.ListContainers(c.Request.Context(), all)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, containers)
}

// GetContainer handles GET /api/docker/containers/:id
func (h *Handlers) GetContainer(c *gin.Context) {
	if h.dockerManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "docker not available"})
		return
	}

	id := c.Param("id")

	container, err := h.dockerManager.GetContainer(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, container)
}

// StartContainer handles POST /api/docker/containers/:id/start
func (h *Handlers) StartContainer(c *gin.Context) {
	if h.dockerManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "docker not available"})
		return
	}

	id := c.Param("id")

	result, err := h.dockerManager.StartContainer(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// StopContainer handles POST /api/docker/containers/:id/stop
func (h *Handlers) StopContainer(c *gin.Context) {
	if h.dockerManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "docker not available"})
		return
	}

	id := c.Param("id")

	result, err := h.dockerManager.StopContainer(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// RestartContainer handles POST /api/docker/containers/:id/restart
func (h *Handlers) RestartContainer(c *gin.Context) {
	if h.dockerManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "docker not available"})
		return
	}

	id := c.Param("id")

	result, err := h.dockerManager.RestartContainer(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetContainerLogs handles GET /api/docker/containers/:id/logs
func (h *Handlers) GetContainerLogs(c *gin.Context) {
	if h.dockerManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "docker not available"})
		return
	}

	id := c.Param("id")

	opts := docker.LogOptions{
		Tail:       c.DefaultQuery("tail", "100"),
		Since:      c.Query("since"),
		Until:      c.Query("until"),
		Timestamps: c.Query("timestamps") == "true",
	}

	logs, err := h.dockerManager.GetContainerLogs(c.Request.Context(), id, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":   id,
		"logs": logs,
	})
}

// File browser handlers

// ListDirectory handles GET /api/files
func (h *Handlers) ListDirectory(c *gin.Context) {
	path := c.DefaultQuery("path", "/")

	listing, err := h.fileBrowser.ListDirectory(path)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "access denied: path not in allowed list" {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, listing)
}

// GetFileContent handles GET /api/files/content
func (h *Handlers) GetFileContent(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path is required"})
		return
	}

	content, err := h.fileBrowser.ReadFile(path)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "access denied: path not in allowed list" {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, content)
}

// GetDiskUsage handles GET /api/files/diskusage
func (h *Handlers) GetDiskUsage(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path is required"})
		return
	}

	usage, err := h.fileBrowser.GetDiskUsage(path)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "access denied: path not in allowed list" {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, usage)
}

// Task handlers

// ListTasks handles GET /api/tasks
func (h *Handlers) ListTasks(c *gin.Context) {
	tasks := h.taskManager.List()
	c.JSON(http.StatusOK, tasks)
}

// RunTask handles POST /api/tasks/:name/run
func (h *Handlers) RunTask(c *gin.Context) {
	name := c.Param("name")

	// Check if task exists
	task, err := h.taskManager.Get(name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Warn about dangerous tasks
	if task.Dangerous {
		confirm := c.Query("confirm")
		if confirm != "true" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   fmt.Sprintf("task '%s' is dangerous, add ?confirm=true to execute", name),
				"task":    task,
			})
			return
		}
	}

	// Run with 5 minute timeout
	result, err := h.taskManager.RunWithTimeout(name, 5*time.Minute)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// Close cleans up handlers resources
func (h *Handlers) Close() error {
	if h.dockerManager != nil {
		return h.dockerManager.Close()
	}
	return nil
}
