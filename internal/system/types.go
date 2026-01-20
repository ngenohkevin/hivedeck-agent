package system

import "time"

// HostInfo contains system identification information
type HostInfo struct {
	Hostname        string `json:"hostname"`
	OS              string `json:"os"`
	Platform        string `json:"platform"`
	PlatformVersion string `json:"platform_version"`
	KernelVersion   string `json:"kernel_version"`
	KernelArch      string `json:"kernel_arch"`
	Uptime          uint64 `json:"uptime"`
	UptimeHuman     string `json:"uptime_human"`
	BootTime        uint64 `json:"boot_time"`
	Procs           uint64 `json:"procs"`
}

// CPUInfo contains CPU usage information
type CPUInfo struct {
	Cores       int       `json:"cores"`
	ModelName   string    `json:"model_name"`
	Mhz         float64   `json:"mhz"`
	UsageTotal  float64   `json:"usage_total"`
	UsagePerCPU []float64 `json:"usage_per_cpu"`
	LoadAvg1    float64   `json:"load_avg_1"`
	LoadAvg5    float64   `json:"load_avg_5"`
	LoadAvg15   float64   `json:"load_avg_15"`
}

// MemoryInfo contains memory usage information
type MemoryInfo struct {
	Total        uint64  `json:"total"`
	Available    uint64  `json:"available"`
	Used         uint64  `json:"used"`
	UsedPercent  float64 `json:"used_percent"`
	Free         uint64  `json:"free"`
	Buffers      uint64  `json:"buffers"`
	Cached       uint64  `json:"cached"`
	SwapTotal    uint64  `json:"swap_total"`
	SwapUsed     uint64  `json:"swap_used"`
	SwapFree     uint64  `json:"swap_free"`
	SwapPercent  float64 `json:"swap_percent"`
}

// DiskInfo contains disk partition information
type DiskInfo struct {
	Partitions []DiskPartition `json:"partitions"`
}

// DiskPartition represents a single disk partition
type DiskPartition struct {
	Device      string  `json:"device"`
	Mountpoint  string  `json:"mountpoint"`
	Fstype      string  `json:"fstype"`
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsedPercent float64 `json:"used_percent"`
}

// NetworkInfo contains network I/O information
type NetworkInfo struct {
	Interfaces []NetworkInterface `json:"interfaces"`
}

// NetworkInterface represents a single network interface
type NetworkInterface struct {
	Name        string   `json:"name"`
	BytesSent   uint64   `json:"bytes_sent"`
	BytesRecv   uint64   `json:"bytes_recv"`
	PacketsSent uint64   `json:"packets_sent"`
	PacketsRecv uint64   `json:"packets_recv"`
	Errin       uint64   `json:"errin"`
	Errout      uint64   `json:"errout"`
	Dropin      uint64   `json:"dropin"`
	Dropout     uint64   `json:"dropout"`
	Addrs       []string `json:"addrs"`
}

// AllMetrics contains all system metrics combined
type AllMetrics struct {
	Timestamp time.Time   `json:"timestamp"`
	Host      HostInfo    `json:"host"`
	CPU       CPUInfo     `json:"cpu"`
	Memory    MemoryInfo  `json:"memory"`
	Disk      DiskInfo    `json:"disk"`
	Network   NetworkInfo `json:"network"`
}

// Temperature represents CPU/GPU temperature
type Temperature struct {
	SensorKey   string  `json:"sensor_key"`
	Temperature float64 `json:"temperature"`
}
