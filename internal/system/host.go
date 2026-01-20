package system

import (
	"fmt"
	"time"

	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/sensors"
)

// GetHostInfo retrieves system host information
func GetHostInfo() (*HostInfo, error) {
	info, err := host.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to get host info: %w", err)
	}

	// Get temperature sensors
	var temps []Temperature
	sensorStats, err := sensors.SensorsTemperatures()
	if err == nil {
		for _, sensor := range sensorStats {
			if sensor.Temperature > 0 {
				temps = append(temps, Temperature{
					SensorKey:   sensor.SensorKey,
					Temperature: sensor.Temperature,
				})
			}
		}
	}

	return &HostInfo{
		Hostname:        info.Hostname,
		OS:              info.OS,
		Platform:        info.Platform,
		PlatformVersion: info.PlatformVersion,
		KernelVersion:   info.KernelVersion,
		KernelArch:      info.KernelArch,
		Uptime:          info.Uptime,
		UptimeHuman:     formatUptime(info.Uptime),
		BootTime:        info.BootTime,
		Procs:           info.Procs,
		Temperatures:    temps,
	}, nil
}

// formatUptime converts uptime seconds to human readable format
func formatUptime(seconds uint64) string {
	duration := time.Duration(seconds) * time.Second

	days := int(duration.Hours() / 24)
	hours := int(duration.Hours()) % 24
	minutes := int(duration.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}
