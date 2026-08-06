package process

import (
	"fmt"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/v4/process"
)

// Manager handles process operations
type Manager struct {
	// AllowedProcessNames contains process names that can be killed
	AllowedProcessNames map[string]bool
}

// NewManager creates a new process manager
func NewManager() *Manager {
	return &Manager{
		AllowedProcessNames: map[string]bool{
			"claude":       true, // Claude Code CLI
			"node":         true, // Claude Code runs as node processes
			"bun":          true, // Telegram plugin runtime
			"npx":          true, // MCP servers
			"npm exec":     true, // MCP servers (npm exec form)
			"npm":          true, // npm processes
			"python":       true, // Python MCP servers (routeros)
			"tmux: server": true, // tmux server process
			"tmux":         true, // tmux processes
			"uvx":          true, // Python tool runner (jellyfin MCP)
		},
	}
}

// List returns all running processes
func (m *Manager) List() (*ProcessList, error) {
	procs, err := process.Processes()
	if err != nil {
		return nil, fmt.Errorf("failed to get processes: %w", err)
	}

	var processes []ProcessInfo
	for _, p := range procs {
		info, err := m.getProcessInfo(p)
		if err != nil {
			continue
		}
		processes = append(processes, *info)
	}

	// Sort by CPU usage descending
	sort.Slice(processes, func(i, j int) bool {
		return processes[i].CPUPercent > processes[j].CPUPercent
	})

	return &ProcessList{
		Processes: processes,
		Total:     len(processes),
	}, nil
}

// ListTop returns the top N processes by CPU usage
func (m *Manager) ListTop(n int) (*ProcessList, error) {
	list, err := m.List()
	if err != nil {
		return nil, err
	}

	if n > len(list.Processes) {
		n = len(list.Processes)
	}

	return &ProcessList{
		Processes: list.Processes[:n],
		Total:     list.Total,
	}, nil
}

// Search returns processes matching a name or cmdline pattern
func (m *Manager) Search(pattern string) (*ProcessList, error) {
	list, err := m.List()
	if err != nil {
		return nil, err
	}

	pattern = strings.ToLower(pattern)
	var matched []ProcessInfo
	for _, p := range list.Processes {
		if strings.Contains(strings.ToLower(p.Name), pattern) ||
			strings.Contains(strings.ToLower(p.Cmdline), pattern) {
			matched = append(matched, p)
		}
	}

	return &ProcessList{
		Processes: matched,
		Total:     len(matched),
	}, nil
}

// Get returns information about a specific process
func (m *Manager) Get(pid int32) (*ProcessInfo, error) {
	p, err := process.NewProcess(pid)
	if err != nil {
		return nil, fmt.Errorf("process not found: %w", err)
	}

	return m.getProcessInfo(p)
}

// Kill terminates a process with the given signal
func (m *Manager) Kill(pid int32, signal int) (*KillResponse, error) {
	p, err := process.NewProcess(pid)
	if err != nil {
		return &KillResponse{
			PID:     pid,
			Success: false,
			Message: fmt.Sprintf("process not found: %v", err),
		}, nil
	}

	// Check if process is in allowed list
	name, _ := p.Name()
	if !m.IsAllowed(name) {
		return &KillResponse{
			PID:     pid,
			Success: false,
			Message: fmt.Sprintf("killing process '%s' is not allowed", name),
		}, nil
	}

	// Default to SIGTERM
	if signal == 0 {
		signal = int(syscall.SIGTERM)
	}

	if err := p.SendSignal(syscall.Signal(signal)); err != nil {
		return &KillResponse{
			PID:     pid,
			Success: false,
			Message: fmt.Sprintf("failed to kill process: %v", err),
		}, nil
	}

	return &KillResponse{
		PID:     pid,
		Success: true,
		Message: fmt.Sprintf("signal %d sent to process %d", signal, pid),
	}, nil
}

// IsAllowed checks if a process name is in the allowed list
func (m *Manager) IsAllowed(name string) bool {
	if m.AllowedProcessNames[name] {
		return true
	}
	// Check prefix match for truncated names (e.g. "npm exec @upsta...")
	for allowed := range m.AllowedProcessNames {
		if strings.HasPrefix(name, allowed) {
			return true
		}
	}
	return false
}

// AllowProcess adds a process name to the allowed list
func (m *Manager) AllowProcess(name string) {
	m.AllowedProcessNames[name] = true
}

func (m *Manager) getProcessInfo(p *process.Process) (*ProcessInfo, error) {
	name, err := p.Name()
	if err != nil {
		return nil, err
	}

	username, _ := p.Username()
	status, _ := p.Status()
	cpuPercent, _ := p.CPUPercent()
	memPercent, _ := p.MemoryPercent()
	memInfo, _ := p.MemoryInfo()
	cmdline, _ := p.Cmdline()
	createTime, _ := p.CreateTime()
	numThreads, _ := p.NumThreads()

	var memRSS uint64
	if memInfo != nil {
		memRSS = memInfo.RSS
	}

	var statusStr string
	if len(status) > 0 {
		statusStr = status[0]
	}

	return &ProcessInfo{
		PID:        p.Pid,
		Name:       name,
		Username:   username,
		Status:     statusStr,
		CPUPercent: cpuPercent,
		MemPercent: memPercent,
		MemRSS:     memRSS,
		Cmdline:    cmdline,
		CreateTime: time.UnixMilli(createTime),
		NumThreads: numThreads,
	}, nil
}
