package systemd

import "time"

// ServiceInfo represents a systemd service
type ServiceInfo struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	LoadState   string    `json:"load_state"`
	ActiveState string    `json:"active_state"`
	SubState    string    `json:"sub_state"`
	MainPID     uint32    `json:"main_pid"`
	ExecStart   string    `json:"exec_start"`
	User        string    `json:"user"`
	Group       string    `json:"group"`
	StartedAt   time.Time `json:"started_at,omitempty"`
	Memory      uint64    `json:"memory"`
	Tasks       uint64    `json:"tasks"`
}

// ServiceList contains a list of services
type ServiceList struct {
	Services []ServiceInfo `json:"services"`
	Total    int           `json:"total"`
}

// ServiceAction represents an action on a service
type ServiceAction struct {
	Name    string `json:"name"`
	Action  string `json:"action"` // start, stop, restart, enable, disable
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// JournalEntry represents a single log entry
type JournalEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Unit      string    `json:"unit"`
	Message   string    `json:"message"`
	Priority  int       `json:"priority"`
	PID       string    `json:"pid"`
	Hostname  string    `json:"hostname"`
}

// JournalQuery represents parameters for log queries
type JournalQuery struct {
	Unit     string `json:"unit,omitempty"`
	Priority int    `json:"priority,omitempty"` // 0-7, -1 for all
	Lines    int    `json:"lines,omitempty"`
	Since    string `json:"since,omitempty"`
	Until    string `json:"until,omitempty"`
}

// LogStream represents a stream of log entries
type LogStream struct {
	Entries []JournalEntry `json:"entries"`
	Unit    string         `json:"unit,omitempty"`
}
