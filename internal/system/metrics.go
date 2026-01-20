package system

import (
	"fmt"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
)

// Collector handles system metrics collection
type Collector struct{}

// NewCollector creates a new metrics collector
func NewCollector() *Collector {
	return &Collector{}
}

// GetCPUInfo retrieves CPU usage and information
func (c *Collector) GetCPUInfo() (*CPUInfo, error) {
	// Get CPU info
	cpuInfo, err := cpu.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to get cpu info: %w", err)
	}

	// Get CPU usage (total)
	percentTotal, err := cpu.Percent(200*time.Millisecond, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get cpu percent: %w", err)
	}

	// Get per-CPU usage
	percentPerCPU, err := cpu.Percent(200*time.Millisecond, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get per-cpu percent: %w", err)
	}

	// Get load average
	loadAvg, err := load.Avg()
	if err != nil {
		// Load average might not be available on all systems
		loadAvg = &load.AvgStat{}
	}

	var modelName string
	var mhz float64
	if len(cpuInfo) > 0 {
		modelName = cpuInfo[0].ModelName
		mhz = cpuInfo[0].Mhz
	}

	var usageTotal float64
	if len(percentTotal) > 0 {
		usageTotal = percentTotal[0]
	}

	return &CPUInfo{
		Cores:       len(cpuInfo),
		ModelName:   modelName,
		Mhz:         mhz,
		UsageTotal:  usageTotal,
		UsagePerCPU: percentPerCPU,
		LoadAvg1:    loadAvg.Load1,
		LoadAvg5:    loadAvg.Load5,
		LoadAvg15:   loadAvg.Load15,
	}, nil
}

// GetMemoryInfo retrieves memory usage information
func (c *Collector) GetMemoryInfo() (*MemoryInfo, error) {
	vmem, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to get virtual memory: %w", err)
	}

	swap, err := mem.SwapMemory()
	if err != nil {
		// Swap might not be available
		swap = &mem.SwapMemoryStat{}
	}

	return &MemoryInfo{
		Total:        vmem.Total,
		Available:    vmem.Available,
		Used:         vmem.Used,
		UsedPercent:  vmem.UsedPercent,
		Free:         vmem.Free,
		Buffers:      vmem.Buffers,
		Cached:       vmem.Cached,
		SwapTotal:    swap.Total,
		SwapUsed:     swap.Used,
		SwapFree:     swap.Free,
		SwapPercent:  swap.UsedPercent,
	}, nil
}

// GetDiskInfo retrieves disk partition information
func (c *Collector) GetDiskInfo() (*DiskInfo, error) {
	partitions, err := disk.Partitions(false)
	if err != nil {
		return nil, fmt.Errorf("failed to get disk partitions: %w", err)
	}

	var diskPartitions []DiskPartition
	for _, p := range partitions {
		// Skip pseudo filesystems
		if p.Fstype == "squashfs" || p.Fstype == "tmpfs" || p.Fstype == "devtmpfs" {
			continue
		}

		usage, err := disk.Usage(p.Mountpoint)
		if err != nil {
			continue
		}

		diskPartitions = append(diskPartitions, DiskPartition{
			Device:      p.Device,
			Mountpoint:  p.Mountpoint,
			Fstype:      p.Fstype,
			Total:       usage.Total,
			Used:        usage.Used,
			Free:        usage.Free,
			UsedPercent: usage.UsedPercent,
		})
	}

	return &DiskInfo{
		Partitions: diskPartitions,
	}, nil
}

// GetNetworkInfo retrieves network interface information
func (c *Collector) GetNetworkInfo() (*NetworkInfo, error) {
	counters, err := net.IOCounters(true)
	if err != nil {
		return nil, fmt.Errorf("failed to get network io counters: %w", err)
	}

	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get network interfaces: %w", err)
	}

	// Build address map
	addrMap := make(map[string][]string)
	for _, iface := range interfaces {
		var addrs []string
		for _, addr := range iface.Addrs {
			addrs = append(addrs, addr.Addr)
		}
		addrMap[iface.Name] = addrs
	}

	var netInterfaces []NetworkInterface
	for _, counter := range counters {
		// Skip loopback
		if counter.Name == "lo" {
			continue
		}

		netInterfaces = append(netInterfaces, NetworkInterface{
			Name:        counter.Name,
			BytesSent:   counter.BytesSent,
			BytesRecv:   counter.BytesRecv,
			PacketsSent: counter.PacketsSent,
			PacketsRecv: counter.PacketsRecv,
			Errin:       counter.Errin,
			Errout:      counter.Errout,
			Dropin:      counter.Dropin,
			Dropout:     counter.Dropout,
			Addrs:       addrMap[counter.Name],
		})
	}

	return &NetworkInfo{
		Interfaces: netInterfaces,
	}, nil
}

// GetAllMetrics retrieves all system metrics
func (c *Collector) GetAllMetrics() (*AllMetrics, error) {
	host, err := GetHostInfo()
	if err != nil {
		return nil, err
	}

	cpuInfo, err := c.GetCPUInfo()
	if err != nil {
		return nil, err
	}

	memory, err := c.GetMemoryInfo()
	if err != nil {
		return nil, err
	}

	diskInfo, err := c.GetDiskInfo()
	if err != nil {
		return nil, err
	}

	network, err := c.GetNetworkInfo()
	if err != nil {
		return nil, err
	}

	return &AllMetrics{
		Timestamp: time.Now(),
		Host:      *host,
		CPU:       *cpuInfo,
		Memory:    *memory,
		Disk:      *diskInfo,
		Network:   *network,
	}, nil
}
