package process

import "time"

// ProcessInfo represents a running process
type ProcessInfo struct {
	PID        int32     `json:"pid"`
	Name       string    `json:"name"`
	Username   string    `json:"username"`
	Status     string    `json:"status"`
	CPUPercent float64   `json:"cpu_percent"`
	MemPercent float32   `json:"mem_percent"`
	MemRSS     uint64    `json:"mem_rss"`
	Cmdline    string    `json:"cmdline"`
	CreateTime time.Time `json:"create_time"`
	NumThreads int32     `json:"num_threads"`
}

// ProcessList contains a list of processes
type ProcessList struct {
	Processes []ProcessInfo `json:"processes"`
	Total     int           `json:"total"`
}

// KillRequest represents a request to kill a process
type KillRequest struct {
	Signal int `json:"signal,omitempty"` // Default: 15 (SIGTERM)
}

// KillResponse represents the result of a kill operation
type KillResponse struct {
	PID     int32  `json:"pid"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}
