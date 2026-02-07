package collector

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/abhishekkarki/instvisor/pkg/metrics"
)

// DiskCollector collects disk I/O metrics
type DiskCollector struct {
	interval  time.Duration
	devices   []string
	lastStats map[string]DiskStat
}

type DiskStat struct {
	ReadsCompleted  uint64
	ReadBytes       uint64
	WritesCompleted uint64
	WriteBytes      uint64
	IOTime          uint64
}

// NewDiskCollector creates a new disk collector
func NewDiskCollector(interval time.Duration, devices []string) *DiskCollector {
	return &DiskCollector{
		interval:  interval,
		devices:   devices,
		lastStats: make(map[string]DiskStat),
	}
}

func (d *DiskCollector) Name() string {
	return "disk"
}

func (d *DiskCollector) Interval() time.Duration {
	return d.interval
}

func (d *DiskCollector) Collect() ([]metrics.Metric, error) {
	file, err := os.Open("/proc/diskstats")
	if err != nil {
		return nil, fmt.Errorf("failed to open /proc/diskstats: %w", err)
	}

	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Failed to close /proc/diskstats: %v", err)
		}
	}()

	var result []metrics.Metric
	scanner := bufio.NewScanner(file)
	currentStats := make(map[string]DiskStat)

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 14 {
			continue
		}

		device := fields[2]

		// Filter by specific devices if configured
		if len(d.devices) > 0 && !contains(d.devices, device) {
			continue
		}

		// Skip partition entries (e.g., sda1, nvme0n1p1) if we're monitoring the whole disk
		if isPartition(device) {
			continue
		}

		stat := DiskStat{
			ReadsCompleted:  parseUint64(fields[3]),
			ReadBytes:       parseUint64(fields[5]) * 512, // sectors to bytes
			WritesCompleted: parseUint64(fields[7]),
			WriteBytes:      parseUint64(fields[9]) * 512, // sectors to bytes
			IOTime:          parseUint64(fields[12]),
		}

		currentStats[device] = stat

		// Calculate rates if we have previous stats
		if lastStat, ok := d.lastStats[device]; ok {
			diskMetrics := d.calculateDiskMetrics(device, lastStat, stat)
			result = append(result, diskMetrics...)
		}
	}

	// Update last stats
	d.lastStats = currentStats

	return result, scanner.Err()
}

func (d *DiskCollector) calculateDiskMetrics(device string, last, current DiskStat) []metrics.Metric {
	labels := map[string]string{"device": device}
	timeDiff := d.interval.Seconds()

	var result []metrics.Metric

	// Read throughput (bytes/sec)
	readBytesDiff := float64(current.ReadBytes - last.ReadBytes)
	result = append(result, metrics.NewMetric(
		"disk.read_bytes_per_sec",
		readBytesDiff/timeDiff,
		"bytes/sec",
		labels,
	))

	// Write throughput (bytes/sec)
	writeBytesDiff := float64(current.WriteBytes - last.WriteBytes)
	result = append(result, metrics.NewMetric(
		"disk.write_bytes_per_sec",
		writeBytesDiff/timeDiff,
		"bytes/sec",
		labels,
	))

	// Read IOPS
	readOpsDiff := float64(current.ReadsCompleted - last.ReadsCompleted)
	result = append(result, metrics.NewMetric(
		"disk.read_ops_per_sec",
		readOpsDiff/timeDiff,
		"ops/sec",
		labels,
	))

	// Write IOPS
	writeOpsDiff := float64(current.WritesCompleted - last.WritesCompleted)
	result = append(result, metrics.NewMetric(
		"disk.write_ops_per_sec",
		writeOpsDiff/timeDiff,
		"ops/sec",
		labels,
	))

	// Utilization percentage
	ioTimeDiff := float64(current.IOTime - last.IOTime)
	utilization := (ioTimeDiff / (timeDiff * 1000)) * 100 // IOTime is in ms
	if utilization > 100 {
		utilization = 100
	}
	result = append(result, metrics.NewMetric(
		"disk.utilization_percent",
		utilization,
		"percent",
		labels,
	))

	return result
}

func isPartition(device string) bool {
	// Simple heuristic: partitions usually end with a number
	if len(device) == 0 {
		return false
	}
	lastChar := device[len(device)-1]
	return lastChar >= '0' && lastChar <= '9'
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
