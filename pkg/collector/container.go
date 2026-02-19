package collector

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/abhishekkarki/instvisor/pkg/metrics"
)

// ContainerCollector collects metrics from Docker containers via cgroup v2
// Reads directly from cgroups without Docker SDK dependency
type ContainerCollector struct {
	interval  time.Duration
	lastStats map[string]ContainerStat
}

// ContainerStat stores previous container statistics for rate calculation
type ContainerStat struct {
	CPUUsageUsec uint64
	ReadBytes    uint64
	WriteBytes   uint64
	ReadOps      uint64
	WriteOps     uint64
}

// ContainerInfo holds container metadata
type ContainerInfo struct {
	ID    string
	Name  string
	Image string
}

// NewContainerCollector creates a new container collector
func NewContainerCollector(interval time.Duration) *ContainerCollector {
	return &ContainerCollector{
		interval:  interval,
		lastStats: make(map[string]ContainerStat),
	}
}

func (c *ContainerCollector) Name() string {
	return "container"
}

func (c *ContainerCollector) Interval() time.Duration {
	return c.interval
}

func (c *ContainerCollector) Collect() ([]metrics.Metric, error) {
	// Find all Docker container cgroups
	pattern := "/sys/fs/cgroup/system.slice/docker-*.scope"
	cgroupPaths, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to list container cgroups: %w", err)
	}

	// If no containers found, return empty (not an error)
	if len(cgroupPaths) == 0 {
		return []metrics.Metric{}, nil
	}

	var result []metrics.Metric
	currentStats := make(map[string]ContainerStat)

	for _, cgroupPath := range cgroupPaths {
		// Extract container ID from path
		// /sys/fs/cgroup/system.slice/docker-<FULL_ID>.scope
		basename := filepath.Base(cgroupPath)
		containerID := strings.TrimPrefix(basename, "docker-")
		containerID = strings.TrimSuffix(containerID, ".scope")

		// Get container metadata (name, image)
		info := c.getContainerInfo(containerID)

		// Create container labels
		labels := map[string]string{
			"container_id":   containerID[:12], // Short ID
			"container_name": info.Name,
			"image":          info.Image,
		}

		// Collect CPU metrics
		cpuMetrics, cpuStat, err := c.collectCPU(cgroupPath, containerID, labels)
		if err == nil {
			result = append(result, cpuMetrics...)
			if cpuStat != nil {
				currentStats[containerID] = *cpuStat
			}
		}

		// Collect Memory metrics
		memMetrics, err := c.collectMemory(cgroupPath, labels)
		if err == nil {
			result = append(result, memMetrics...)
		}

		// Collect I/O metrics
		ioMetrics, ioStats, err := c.collectIO(cgroupPath, containerID, labels)
		if err == nil {
			result = append(result, ioMetrics...)
			if ioStats != nil {
				// Merge I/O stats
				stat := currentStats[containerID]
				stat.ReadBytes = ioStats.ReadBytes
				stat.WriteBytes = ioStats.WriteBytes
				stat.ReadOps = ioStats.ReadOps
				stat.WriteOps = ioStats.WriteOps
				currentStats[containerID] = stat
			}
		}
	}

	// Update last stats for next collection
	c.lastStats = currentStats

	return result, nil
}

// getContainerInfo reads container metadata from Docker's config file
func (c *ContainerCollector) getContainerInfo(containerID string) ContainerInfo {
	info := ContainerInfo{
		ID:    containerID,
		Name:  containerID[:12], // Default to short ID
		Image: "unknown",
	}

	// Try to read Docker container config
	configPath := fmt.Sprintf("/var/lib/docker/containers/%s/config.v2.json", containerID)
	data, err := os.ReadFile(configPath)
	if err != nil {
		// Can't read config (maybe permissions), use defaults
		return info
	}

	// Parse Docker config JSON
	var config struct {
		Name   string `json:"Name"`
		Config struct {
			Image string `json:"Image"`
		} `json:"Config"`
	}

	if err := json.Unmarshal(data, &config); err == nil {
		// Remove leading "/" from container name
		info.Name = strings.TrimPrefix(config.Name, "/")
		info.Image = config.Config.Image
	}

	return info
}

// collectCPU reads CPU metrics from cgroup
func (c *ContainerCollector) collectCPU(cgroupPath, containerID string, labels map[string]string) ([]metrics.Metric, *ContainerStat, error) {
	data, err := os.ReadFile(filepath.Join(cgroupPath, "cpu.stat"))
	if err != nil {
		return nil, nil, err
	}

	var usageUsec, userUsec, systemUsec uint64

	// Parse cpu.stat file
	// Format:
	// usage_usec 192042081
	// user_usec 94901971
	// system_usec 97140110
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		value, _ := strconv.ParseUint(fields[1], 10, 64)
		switch fields[0] {
		case "usage_usec":
			usageUsec = value
		case "user_usec":
			userUsec = value
		case "system_usec":
			systemUsec = value
		}
	}

	var result []metrics.Metric
	stat := &ContainerStat{CPUUsageUsec: usageUsec}

	// Calculate CPU usage percentage if we have previous data
	if lastStat, ok := c.lastStats[containerID]; ok {
		usageDiff := float64(usageUsec - lastStat.CPUUsageUsec)
		timeDiff := float64(c.interval.Microseconds())

		if timeDiff > 0 && usageDiff >= 0 {
			// CPU usage % = (microseconds used / microseconds elapsed) * 100
			cpuPercent := (usageDiff / timeDiff) * 100

			result = append(result, metrics.NewMetric(
				"container.cpu.usage_percent",
				cpuPercent,
				"percent",
				labels,
			))
		}
	}

	// Store cumulative usage for reference
	result = append(result, metrics.NewMetric(
		"container.cpu.usage_total_usec",
		float64(usageUsec),
		"microseconds",
		labels,
	))

	result = append(result, metrics.NewMetric(
		"container.cpu.user_usec",
		float64(userUsec),
		"microseconds",
		labels,
	))

	result = append(result, metrics.NewMetric(
		"container.cpu.system_usec",
		float64(systemUsec),
		"microseconds",
		labels,
	))

	return result, stat, nil
}

// collectMemory reads memory metrics from cgroup
func (c *ContainerCollector) collectMemory(cgroupPath string, labels map[string]string) ([]metrics.Metric, error) {
	// Read current memory usage
	currentData, err := os.ReadFile(filepath.Join(cgroupPath, "memory.current"))
	if err != nil {
		return nil, err
	}
	current, _ := strconv.ParseUint(strings.TrimSpace(string(currentData)), 10, 64)

	// Read memory limit
	maxData, err := os.ReadFile(filepath.Join(cgroupPath, "memory.max"))
	if err != nil {
		return nil, err
	}
	maxStr := strings.TrimSpace(string(maxData))

	var result []metrics.Metric

	// Current memory usage in bytes
	result = append(result, metrics.NewMetric(
		"container.memory.usage_bytes",
		float64(current),
		"bytes",
		labels,
	))

	// Memory limit and usage percentage (if limit is set)
	if maxStr != "max" {
		max, _ := strconv.ParseUint(maxStr, 10, 64)

		result = append(result, metrics.NewMetric(
			"container.memory.limit_bytes",
			float64(max),
			"bytes",
			labels,
		))

		if max > 0 {
			usagePercent := (float64(current) / float64(max)) * 100
			result = append(result, metrics.NewMetric(
				"container.memory.usage_percent",
				usagePercent,
				"percent",
				labels,
			))
		}
	}

	return result, nil
}

// collectIO reads I/O metrics from cgroup
func (c *ContainerCollector) collectIO(cgroupPath, containerID string, labels map[string]string) ([]metrics.Metric, *ContainerStat, error) {
	data, err := os.ReadFile(filepath.Join(cgroupPath, "io.stat"))
	if err != nil {
		return nil, nil, err
	}

	var totalReadBytes, totalWriteBytes, totalReadOps, totalWriteOps uint64

	// Parse io.stat file
	// Format:
	// 252:0 rbytes=282308608 wbytes=13915918336 rios=64750 wios=1099429
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		// Parse key=value pairs
		for _, field := range fields[1:] {
			parts := strings.Split(field, "=")
			if len(parts) != 2 {
				continue
			}

			value, _ := strconv.ParseUint(parts[1], 10, 64)
			switch parts[0] {
			case "rbytes":
				totalReadBytes += value
			case "wbytes":
				totalWriteBytes += value
			case "rios":
				totalReadOps += value
			case "wios":
				totalWriteOps += value
			}
		}
	}

	var result []metrics.Metric
	ioStats := &ContainerStat{
		ReadBytes:  totalReadBytes,
		WriteBytes: totalWriteBytes,
		ReadOps:    totalReadOps,
		WriteOps:   totalWriteOps,
	}

	// Calculate I/O rates if we have previous data
	if lastStat, ok := c.lastStats[containerID]; ok {
		timeDiff := c.interval.Seconds()

		readBytesDiff := totalReadBytes - lastStat.ReadBytes
		writeBytesDiff := totalWriteBytes - lastStat.WriteBytes
		readOpsDiff := totalReadOps - lastStat.ReadOps
		writeOpsDiff := totalWriteOps - lastStat.WriteOps

		result = append(result, metrics.NewMetric(
			"container.io.read_bytes_per_sec",
			float64(readBytesDiff)/timeDiff,
			"bytes/sec",
			labels,
		))

		result = append(result, metrics.NewMetric(
			"container.io.write_bytes_per_sec",
			float64(writeBytesDiff)/timeDiff,
			"bytes/sec",
			labels,
		))

		result = append(result, metrics.NewMetric(
			"container.io.read_ops_per_sec",
			float64(readOpsDiff)/timeDiff,
			"ops/sec",
			labels,
		))

		result = append(result, metrics.NewMetric(
			"container.io.write_ops_per_sec",
			float64(writeOpsDiff)/timeDiff,
			"ops/sec",
			labels,
		))
	}

	// Store cumulative totals
	result = append(result, metrics.NewMetric(
		"container.io.read_bytes_total",
		float64(totalReadBytes),
		"bytes",
		labels,
	))

	result = append(result, metrics.NewMetric(
		"container.io.write_bytes_total",
		float64(totalWriteBytes),
		"bytes",
		labels,
	))

	return result, ioStats, nil
}
