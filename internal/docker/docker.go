package docker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

// Manager handles Docker operations
type Manager struct {
	client *client.Client
}

// NewManager creates a new Docker manager
func NewManager() (*Manager, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	return &Manager{
		client: cli,
	}, nil
}

// IsAvailable checks if Docker is available
func (m *Manager) IsAvailable(ctx context.Context) bool {
	_, err := m.client.Ping(ctx)
	return err == nil
}

// Close closes the Docker client
func (m *Manager) Close() error {
	return m.client.Close()
}

// ListContainers returns all containers
func (m *Manager) ListContainers(ctx context.Context, all bool) (*ContainerList, error) {
	containers, err := m.client.ContainerList(ctx, container.ListOptions{
		All:  all,
		Size: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	var result []ContainerInfo
	for _, c := range containers {
		info := ContainerInfo{
			ID:         c.ID[:12],
			Name:       strings.TrimPrefix(c.Names[0], "/"),
			Image:      c.Image,
			ImageID:    c.ImageID,
			State:      c.State,
			Status:     c.Status,
			Labels:     c.Labels,
			SizeRw:     c.SizeRw,
			SizeRootFs: c.SizeRootFs,
		}

		// Convert ports
		for _, p := range c.Ports {
			info.Ports = append(info.Ports, PortBinding{
				PrivatePort: p.PrivatePort,
				PublicPort:  p.PublicPort,
				Type:        p.Type,
				IP:          p.IP,
			})
		}

		// Get network names
		for name := range c.NetworkSettings.Networks {
			info.Networks = append(info.Networks, name)
		}

		// Convert mounts
		for _, mount := range c.Mounts {
			info.Mounts = append(info.Mounts, Mount{
				Type:        string(mount.Type),
				Source:      mount.Source,
				Destination: mount.Destination,
				Mode:        mount.Mode,
				RW:          mount.RW,
			})
		}

		result = append(result, info)
	}

	return &ContainerList{
		Containers: result,
		Total:      len(result),
	}, nil
}

// GetContainer returns information about a specific container
func (m *Manager) GetContainer(ctx context.Context, id string) (*ContainerInfo, error) {
	inspect, err := m.client.ContainerInspect(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	info := &ContainerInfo{
		ID:      inspect.ID[:12],
		Name:    strings.TrimPrefix(inspect.Name, "/"),
		Image:   inspect.Config.Image,
		ImageID: inspect.Image,
		State:   inspect.State.Status,
		Status:  inspect.State.Status,
		Labels:  inspect.Config.Labels,
	}

	// Get network names
	for name := range inspect.NetworkSettings.Networks {
		info.Networks = append(info.Networks, name)
	}

	// Convert mounts
	for _, mount := range inspect.Mounts {
		info.Mounts = append(info.Mounts, Mount{
			Type:        string(mount.Type),
			Source:      mount.Source,
			Destination: mount.Destination,
			Mode:        mount.Mode,
			RW:          mount.RW,
		})
	}

	return info, nil
}

// StartContainer starts a container
func (m *Manager) StartContainer(ctx context.Context, id string) (*ContainerAction, error) {
	if err := m.client.ContainerStart(ctx, id, container.StartOptions{}); err != nil {
		return &ContainerAction{
			ID:      id,
			Action:  "start",
			Success: false,
			Message: fmt.Sprintf("failed to start container: %v", err),
		}, nil
	}

	return &ContainerAction{
		ID:      id,
		Action:  "start",
		Success: true,
		Message: "container started",
	}, nil
}

// StopContainer stops a container
func (m *Manager) StopContainer(ctx context.Context, id string) (*ContainerAction, error) {
	timeout := 30
	if err := m.client.ContainerStop(ctx, id, container.StopOptions{Timeout: &timeout}); err != nil {
		return &ContainerAction{
			ID:      id,
			Action:  "stop",
			Success: false,
			Message: fmt.Sprintf("failed to stop container: %v", err),
		}, nil
	}

	return &ContainerAction{
		ID:      id,
		Action:  "stop",
		Success: true,
		Message: "container stopped",
	}, nil
}

// RestartContainer restarts a container
func (m *Manager) RestartContainer(ctx context.Context, id string) (*ContainerAction, error) {
	timeout := 30
	if err := m.client.ContainerRestart(ctx, id, container.StopOptions{Timeout: &timeout}); err != nil {
		return &ContainerAction{
			ID:      id,
			Action:  "restart",
			Success: false,
			Message: fmt.Sprintf("failed to restart container: %v", err),
		}, nil
	}

	return &ContainerAction{
		ID:      id,
		Action:  "restart",
		Success: true,
		Message: "container restarted",
	}, nil
}

// GetContainerLogs returns container logs
func (m *Manager) GetContainerLogs(ctx context.Context, id string, opts LogOptions) ([]string, error) {
	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: opts.Timestamps,
		Tail:       opts.Tail,
		Since:      opts.Since,
		Until:      opts.Until,
	}

	if options.Tail == "" {
		options.Tail = "100"
	}

	reader, err := m.client.ContainerLogs(ctx, id, options)
	if err != nil {
		return nil, fmt.Errorf("failed to get container logs: %w", err)
	}
	defer reader.Close()

	var logs []string
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		// Docker logs have an 8-byte header for each line
		if len(line) > 8 {
			line = line[8:]
		}
		logs = append(logs, line)
	}

	return logs, nil
}

// StreamContainerLogs streams container logs in real-time
func (m *Manager) StreamContainerLogs(ctx context.Context, id string, logChan chan<- string) error {
	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Tail:       "50",
	}

	reader, err := m.client.ContainerLogs(ctx, id, options)
	if err != nil {
		return fmt.Errorf("failed to stream container logs: %w", err)
	}

	go func() {
		defer reader.Close()
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			line := scanner.Text()
			if len(line) > 8 {
				line = line[8:]
			}
			select {
			case logChan <- line:
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

// GetContainerStats returns container resource statistics
func (m *Manager) GetContainerStats(ctx context.Context, id string) (*ContainerStats, error) {
	stats, err := m.client.ContainerStats(ctx, id, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get container stats: %w", err)
	}
	defer stats.Body.Close()

	var v types.StatsJSON
	if err := decodeStats(stats.Body, &v); err != nil {
		return nil, fmt.Errorf("failed to decode stats: %w", err)
	}

	// Calculate CPU percentage
	cpuDelta := float64(v.CPUStats.CPUUsage.TotalUsage - v.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(v.CPUStats.SystemUsage - v.PreCPUStats.SystemUsage)
	cpuPercent := 0.0
	if systemDelta > 0.0 && cpuDelta > 0.0 {
		cpuPercent = (cpuDelta / systemDelta) * float64(len(v.CPUStats.CPUUsage.PercpuUsage)) * 100.0
	}

	// Calculate memory percentage
	memPercent := 0.0
	if v.MemoryStats.Limit > 0 {
		memPercent = float64(v.MemoryStats.Usage) / float64(v.MemoryStats.Limit) * 100.0
	}

	// Calculate network I/O
	var netRx, netTx uint64
	for _, net := range v.Networks {
		netRx += net.RxBytes
		netTx += net.TxBytes
	}

	// Calculate block I/O
	var blockRead, blockWrite uint64
	for _, bio := range v.BlkioStats.IoServiceBytesRecursive {
		switch bio.Op {
		case "Read":
			blockRead += bio.Value
		case "Write":
			blockWrite += bio.Value
		}
	}

	return &ContainerStats{
		ID:            id,
		CPUPercent:    cpuPercent,
		MemoryUsage:   v.MemoryStats.Usage,
		MemoryLimit:   v.MemoryStats.Limit,
		MemoryPercent: memPercent,
		NetworkRx:     netRx,
		NetworkTx:     netTx,
		BlockRead:     blockRead,
		BlockWrite:    blockWrite,
		PIDs:          v.PidsStats.Current,
	}, nil
}

// ListImages returns all images
func (m *Manager) ListImages(ctx context.Context) ([]ImageInfo, error) {
	images, err := m.client.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}

	var result []ImageInfo
	for _, img := range images {
		result = append(result, ImageInfo{
			ID:          img.ID,
			RepoTags:    img.RepoTags,
			RepoDigests: img.RepoDigests,
			Size:        img.Size,
			Created:     img.Created,
		})
	}

	return result, nil
}

func decodeStats(reader io.Reader, v *types.StatsJSON) error {
	dec := bufio.NewReader(reader)
	data, err := io.ReadAll(dec)
	if err != nil {
		return err
	}
	return unmarshalJSON(data, v)
}

func unmarshalJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
