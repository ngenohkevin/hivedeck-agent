package tasks

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/ngenohkevin/hivedeck-agent/config"
)

// Manager handles task execution
type Manager struct {
	tasks map[string]config.Task
}

// NewManager creates a new task manager
func NewManager(tasks map[string]config.Task) *Manager {
	return &Manager{
		tasks: tasks,
	}
}

// List returns all available tasks
func (m *Manager) List() *TaskList {
	var taskList []Task
	for _, t := range m.tasks {
		taskList = append(taskList, Task{
			Name:        t.Name,
			Command:     t.Command,
			Description: t.Description,
			Dangerous:   t.Dangerous,
		})
	}

	return &TaskList{
		Tasks: taskList,
		Total: len(taskList),
	}
}

// Get returns a specific task by name
func (m *Manager) Get(name string) (*Task, error) {
	t, ok := m.tasks[name]
	if !ok {
		return nil, fmt.Errorf("task '%s' not found", name)
	}

	return &Task{
		Name:        t.Name,
		Command:     t.Command,
		Description: t.Description,
		Dangerous:   t.Dangerous,
	}, nil
}

// Run executes a task by name
func (m *Manager) Run(ctx context.Context, name string) (*TaskResult, error) {
	t, ok := m.tasks[name]
	if !ok {
		return nil, fmt.Errorf("task '%s' not found", name)
	}

	startTime := time.Now()

	// Create command with context
	cmd := exec.CommandContext(ctx, "bash", "-c", t.Command)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	duration := time.Since(startTime)

	result := &TaskResult{
		Name:      t.Name,
		Command:   t.Command,
		StartedAt: startTime,
		Duration:  duration,
	}

	// Combine stdout and stderr
	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += stderr.String()
	}
	result.Output = output

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
		} else {
			result.ExitCode = -1
		}
		result.Success = false
		result.Error = err.Error()
	} else {
		result.ExitCode = 0
		result.Success = true
	}

	return result, nil
}

// RunWithTimeout executes a task with a specific timeout
func (m *Manager) RunWithTimeout(name string, timeout time.Duration) (*TaskResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return m.Run(ctx, name)
}

// Exists checks if a task exists
func (m *Manager) Exists(name string) bool {
	_, ok := m.tasks[name]
	return ok
}

// IsDangerous checks if a task is marked as dangerous
func (m *Manager) IsDangerous(name string) bool {
	t, ok := m.tasks[name]
	if !ok {
		return true // Unknown tasks are considered dangerous
	}
	return t.Dangerous
}
