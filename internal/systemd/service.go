package systemd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/coreos/go-systemd/v22/dbus"
)

// Manager handles systemd service operations
type Manager struct {
	allowedServices map[string]bool
}

// NewManager creates a new systemd manager
func NewManager(allowedServices []string) *Manager {
	allowed := make(map[string]bool)
	for _, s := range allowedServices {
		allowed[s] = true
	}
	return &Manager{
		allowedServices: allowed,
	}
}

// IsAllowed checks if a service is in the allowed list
func (m *Manager) IsAllowed(name string) bool {
	// Strip .service suffix for comparison
	name = strings.TrimSuffix(name, ".service")
	return m.allowedServices[name]
}

// List returns all systemd services
func (m *Manager) List(ctx context.Context) (*ServiceList, error) {
	conn, err := dbus.NewWithContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to systemd: %w", err)
	}
	defer conn.Close()

	units, err := conn.ListUnitsContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list units: %w", err)
	}

	var services []ServiceInfo
	for _, unit := range units {
		// Only include services
		if !strings.HasSuffix(unit.Name, ".service") {
			continue
		}

		// Only include allowed services if we have an allowlist
		name := strings.TrimSuffix(unit.Name, ".service")
		if len(m.allowedServices) > 0 && !m.allowedServices[name] {
			continue
		}

		info := ServiceInfo{
			Name:        name,
			Description: unit.Description,
			LoadState:   unit.LoadState,
			ActiveState: unit.ActiveState,
			SubState:    unit.SubState,
		}

		// Get additional properties
		props, err := conn.GetUnitPropertiesContext(ctx, unit.Name)
		if err == nil {
			if pid, ok := props["MainPID"].(uint32); ok {
				info.MainPID = pid
			}
			if mem, ok := props["MemoryCurrent"].(uint64); ok {
				info.Memory = mem
			}
			if tasks, ok := props["TasksCurrent"].(uint64); ok {
				info.Tasks = tasks
			}
		}

		services = append(services, info)
	}

	return &ServiceList{
		Services: services,
		Total:    len(services),
	}, nil
}

// Get returns information about a specific service
func (m *Manager) Get(ctx context.Context, name string) (*ServiceInfo, error) {
	if !m.IsAllowed(name) {
		return nil, fmt.Errorf("service '%s' is not in allowed list", name)
	}

	conn, err := dbus.NewWithContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to systemd: %w", err)
	}
	defer conn.Close()

	unitName := name
	if !strings.HasSuffix(unitName, ".service") {
		unitName = name + ".service"
	}

	props, err := conn.GetUnitPropertiesContext(ctx, unitName)
	if err != nil {
		return nil, fmt.Errorf("failed to get service properties: %w", err)
	}

	info := &ServiceInfo{
		Name: name,
	}

	if desc, ok := props["Description"].(string); ok {
		info.Description = desc
	}
	if loadState, ok := props["LoadState"].(string); ok {
		info.LoadState = loadState
	}
	if activeState, ok := props["ActiveState"].(string); ok {
		info.ActiveState = activeState
	}
	if subState, ok := props["SubState"].(string); ok {
		info.SubState = subState
	}
	if pid, ok := props["MainPID"].(uint32); ok {
		info.MainPID = pid
	}
	if mem, ok := props["MemoryCurrent"].(uint64); ok {
		info.Memory = mem
	}
	if tasks, ok := props["TasksCurrent"].(uint64); ok {
		info.Tasks = tasks
	}
	if execStart, ok := props["ExecStart"].([][]interface{}); ok && len(execStart) > 0 && len(execStart[0]) > 0 {
		if path, ok := execStart[0][0].(string); ok {
			info.ExecStart = path
		}
	}

	return info, nil
}

// Start starts a service
func (m *Manager) Start(ctx context.Context, name string) (*ServiceAction, error) {
	return m.doAction(ctx, name, "start")
}

// Stop stops a service
func (m *Manager) Stop(ctx context.Context, name string) (*ServiceAction, error) {
	return m.doAction(ctx, name, "stop")
}

// Restart restarts a service
func (m *Manager) Restart(ctx context.Context, name string) (*ServiceAction, error) {
	return m.doAction(ctx, name, "restart")
}

func (m *Manager) doAction(ctx context.Context, name, action string) (*ServiceAction, error) {
	if !m.IsAllowed(name) {
		return &ServiceAction{
			Name:    name,
			Action:  action,
			Success: false,
			Message: fmt.Sprintf("service '%s' is not in allowed list", name),
		}, nil
	}

	conn, err := dbus.NewWithContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to systemd: %w", err)
	}
	defer conn.Close()

	unitName := name
	if !strings.HasSuffix(unitName, ".service") {
		unitName = name + ".service"
	}

	resultChan := make(chan string, 1)

	switch action {
	case "start":
		_, err = conn.StartUnitContext(ctx, unitName, "replace", resultChan)
	case "stop":
		_, err = conn.StopUnitContext(ctx, unitName, "replace", resultChan)
	case "restart":
		_, err = conn.RestartUnitContext(ctx, unitName, "replace", resultChan)
	default:
		return &ServiceAction{
			Name:    name,
			Action:  action,
			Success: false,
			Message: fmt.Sprintf("unknown action: %s", action),
		}, nil
	}

	if err != nil {
		return &ServiceAction{
			Name:    name,
			Action:  action,
			Success: false,
			Message: fmt.Sprintf("failed to %s service: %v", action, err),
		}, nil
	}

	// Wait for result with timeout
	select {
	case result := <-resultChan:
		success := result == "done"
		msg := fmt.Sprintf("service %s %s: %s", name, action, result)
		return &ServiceAction{
			Name:    name,
			Action:  action,
			Success: success,
			Message: msg,
		}, nil
	case <-time.After(30 * time.Second):
		return &ServiceAction{
			Name:    name,
			Action:  action,
			Success: false,
			Message: "operation timed out",
		}, nil
	case <-ctx.Done():
		return &ServiceAction{
			Name:    name,
			Action:  action,
			Success: false,
			Message: "operation cancelled",
		}, nil
	}
}
