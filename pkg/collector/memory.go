package collector

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/abhishekkarki/instvisor/pkg/metrics"
)

// MemoryCollector collects memory metrics
type MemoryCollector struct {
	interval    time.Duration
	includeSwap bool
}

// NewMemoryCollector creates a new memory collector
func NewMemoryCollector(interval time.Duration, includeSwap bool) *MemoryCollector {
	return &MemoryCollector{
		interval:    interval,
		includeSwap: includeSwap,
	}
}

func (m *MemoryCollector) Name() string {
	return "memory"
}

func (m *MemoryCollector) Interval() time.Duration {
	return m.interval
}

func (m *MemoryCollector) Collect() ([]metrics.Metric, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return nil, fmt.Errorf("failed to open /proc/meminfo: %w", err)
	}

	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Failed to close file: %v", err)
		}
	}()

	memInfo := make(map[string]uint64)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		key := strings.TrimSuffix(fields[0], ":")
		value, _ := strconv.ParseUint(fields[1], 10, 64)

		// Convert from KB to bytes
		memInfo[key] = value * 1024
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	var result []metrics.Metric

	// Total memory
	memTotal := memInfo["MemTotal"]
	result = append(result, metrics.NewMetric("memory.total", float64(memTotal), "bytes", nil))

	// Free memory
	memFree := memInfo["MemFree"]
	result = append(result, metrics.NewMetric("memory.free", float64(memFree), "bytes", nil))

	// Available memory
	memAvailable := memInfo["MemAvailable"]
	result = append(result, metrics.NewMetric("memory.available", float64(memAvailable), "bytes", nil))

	// Used memory (Total - Free - Buffers - Cached)
	buffers := memInfo["Buffers"]
	cached := memInfo["Cached"]
	memUsed := memTotal - memFree - buffers - cached
	result = append(result, metrics.NewMetric("memory.used", float64(memUsed), "bytes", nil))

	// Memory usage percentage
	if memTotal > 0 {
		usagePercent := (float64(memUsed) / float64(memTotal)) * 100
		result = append(result, metrics.NewMetric("memory.usage_percent", usagePercent, "percent", nil))
	}

	// Buffers
	result = append(result, metrics.NewMetric("memory.buffers", float64(buffers), "bytes", nil))

	// Cached
	result = append(result, metrics.NewMetric("memory.cached", float64(cached), "bytes", nil))

	// Swap metrics
	if m.includeSwap {
		swapTotal := memInfo["SwapTotal"]
		swapFree := memInfo["SwapFree"]
		swapUsed := swapTotal - swapFree

		result = append(result, metrics.NewMetric("memory.swap_total", float64(swapTotal), "bytes", nil))
		result = append(result, metrics.NewMetric("memory.swap_free", float64(swapFree), "bytes", nil))
		result = append(result, metrics.NewMetric("memory.swap_used", float64(swapUsed), "bytes", nil))

		if swapTotal > 0 {
			swapUsagePercent := (float64(swapUsed) / float64(swapTotal)) * 100
			result = append(result, metrics.NewMetric("memory.swap_usage_percent", swapUsagePercent, "percent", nil))
		}
	}

	return result, nil
}
