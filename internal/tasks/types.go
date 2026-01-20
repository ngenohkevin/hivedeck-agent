package tasks

import "time"

// Task represents a pre-defined safe command
type Task struct {
	Name        string `json:"name"`
	Command     string `json:"command"`
	Description string `json:"description"`
	Dangerous   bool   `json:"dangerous"`
}

// TaskList contains available tasks
type TaskList struct {
	Tasks []Task `json:"tasks"`
	Total int    `json:"total"`
}

// TaskResult represents the result of running a task
type TaskResult struct {
	Name      string        `json:"name"`
	Command   string        `json:"command"`
	Output    string        `json:"output"`
	ExitCode  int           `json:"exit_code"`
	Success   bool          `json:"success"`
	Error     string        `json:"error,omitempty"`
	StartedAt time.Time     `json:"started_at"`
	Duration  time.Duration `json:"duration"`
}
