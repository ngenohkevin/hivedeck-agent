package docker

import "time"

// ContainerInfo represents a Docker container
type ContainerInfo struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Image      string            `json:"image"`
	ImageID    string            `json:"image_id"`
	State      string            `json:"state"`
	Status     string            `json:"status"`
	Created    time.Time         `json:"created"`
	Ports      []PortBinding     `json:"ports"`
	Labels     map[string]string `json:"labels"`
	Networks   []string          `json:"networks"`
	Mounts     []Mount           `json:"mounts"`
	SizeRw     int64             `json:"size_rw,omitempty"`
	SizeRootFs int64             `json:"size_root_fs,omitempty"`
}

// PortBinding represents a container port binding
type PortBinding struct {
	PrivatePort uint16 `json:"private_port"`
	PublicPort  uint16 `json:"public_port"`
	Type        string `json:"type"`
	IP          string `json:"ip"`
}

// Mount represents a container mount point
type Mount struct {
	Type        string `json:"type"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Mode        string `json:"mode"`
	RW          bool   `json:"rw"`
}

// ContainerList contains a list of containers
type ContainerList struct {
	Containers []ContainerInfo `json:"containers"`
	Total      int             `json:"total"`
}

// ContainerAction represents an action on a container
type ContainerAction struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Action  string `json:"action"` // start, stop, restart, remove
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// ContainerStats represents container resource statistics
type ContainerStats struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	CPUPercent   float64 `json:"cpu_percent"`
	MemoryUsage  uint64  `json:"memory_usage"`
	MemoryLimit  uint64  `json:"memory_limit"`
	MemoryPercent float64 `json:"memory_percent"`
	NetworkRx    uint64  `json:"network_rx"`
	NetworkTx    uint64  `json:"network_tx"`
	BlockRead    uint64  `json:"block_read"`
	BlockWrite   uint64  `json:"block_write"`
	PIDs         uint64  `json:"pids"`
}

// LogOptions represents options for fetching container logs
type LogOptions struct {
	Tail       string `json:"tail,omitempty"`
	Since      string `json:"since,omitempty"`
	Until      string `json:"until,omitempty"`
	Timestamps bool   `json:"timestamps,omitempty"`
	Follow     bool   `json:"follow,omitempty"`
}

// ImageInfo represents a Docker image
type ImageInfo struct {
	ID          string   `json:"id"`
	RepoTags    []string `json:"repo_tags"`
	RepoDigests []string `json:"repo_digests"`
	Size        int64    `json:"size"`
	Created     int64    `json:"created"`
}
