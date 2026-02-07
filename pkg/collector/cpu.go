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

// CPUCollector collects CPU metrics
type CPUCollector struct {
	interval  time.Duration
	perCore   bool
	lastStats map[string]CPUStat
}

type CPUStat struct {
	User    uint64
	Nice    uint64
	System  uint64
	Idle    uint64
	IOWait  uint64
	IRQ     uint64
	SoftIRQ uint64
	Steal   uint64
}

// NewCPUCollector creates a new CPU collector
func NewCPUCollector(interval time.Duration, perCore bool) *CPUCollector {
	return &CPUCollector{
		interval:  interval,
		perCore:   perCore,
		lastStats: make(map[string]CPUStat),
	}
}

func (c *CPUCollector) Name() string {
	return "cpu"
}

func (c *CPUCollector) Interval() time.Duration {
	return c.interval
}

func (c *CPUCollector) Collect() ([]metrics.Metric, error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return nil, fmt.Errorf("failed to open /proc/stat: %w", err)
	}

	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Failed to close /proc/stat: %v", err)
		}
	}()

	var result []metrics.Metric
	scanner := bufio.NewScanner(file)
	currentStats := make(map[string]CPUStat)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "cpu") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 8 {
			continue
		}

		cpuID := fields[0]

		// Skip per-core metrics if not enabled
		if !c.perCore && cpuID != "cpu" {
			continue
		}

		stat := CPUStat{
			User:    parseUint64(fields[1]),
			Nice:    parseUint64(fields[2]),
			System:  parseUint64(fields[3]),
			Idle:    parseUint64(fields[4]),
			IOWait:  parseUint64(fields[5]),
			IRQ:     parseUint64(fields[6]),
			SoftIRQ: parseUint64(fields[7]),
		}
		if len(fields) > 8 {
			stat.Steal = parseUint64(fields[8])
		}

		currentStats[cpuID] = stat

		// Calculate usage if we have previous stats
		if lastStat, ok := c.lastStats[cpuID]; ok {
			cpuMetrics := c.calculateCPUUsage(cpuID, lastStat, stat)
			result = append(result, cpuMetrics...)
		}
	}

	// Update last stats
	c.lastStats = currentStats

	// Also collect load average
	loadMetrics, err := c.collectLoadAverage()
	if err == nil {
		result = append(result, loadMetrics...)
	}

	return result, scanner.Err()
}

func (c *CPUCollector) calculateCPUUsage(cpuID string, last, current CPUStat) []metrics.Metric {
	labels := map[string]string{}
	if cpuID != "cpu" {
		labels["cpu"] = cpuID
	}

	lastTotal := last.User + last.Nice + last.System + last.Idle + last.IOWait + last.IRQ + last.SoftIRQ + last.Steal
	currentTotal := current.User + current.Nice + current.System + current.Idle + current.IOWait + current.IRQ + current.SoftIRQ + current.Steal

	totalDiff := float64(currentTotal - lastTotal)
	if totalDiff == 0 {
		return nil
	}

	var result []metrics.Metric

	// User CPU %
	userDiff := float64(current.User - last.User)
	result = append(result, metrics.NewMetric(
		"cpu.user",
		(userDiff/totalDiff)*100,
		"percent",
		labels,
	))

	// System CPU %
	systemDiff := float64(current.System - last.System)
	result = append(result, metrics.NewMetric(
		"cpu.system",
		(systemDiff/totalDiff)*100,
		"percent",
		labels,
	))

	// Idle CPU %
	idleDiff := float64(current.Idle - last.Idle)
	result = append(result, metrics.NewMetric(
		"cpu.idle",
		(idleDiff/totalDiff)*100,
		"percent",
		labels,
	))

	// IOWait CPU %
	iowaitDiff := float64(current.IOWait - last.IOWait)
	result = append(result, metrics.NewMetric(
		"cpu.iowait",
		(iowaitDiff/totalDiff)*100,
		"percent",
		labels,
	))

	// Total usage (100 - idle)
	usagePct := 100 - (idleDiff/totalDiff)*100
	result = append(result, metrics.NewMetric(
		"cpu.usage",
		usagePct,
		"percent",
		labels,
	))

	return result
}

func (c *CPUCollector) collectLoadAverage() ([]metrics.Metric, error) {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return nil, err
	}

	fields := strings.Fields(string(data))
	if len(fields) < 3 {
		return nil, fmt.Errorf("invalid loadavg format")
	}

	load1, _ := strconv.ParseFloat(fields[0], 64)
	load5, _ := strconv.ParseFloat(fields[1], 64)
	load15, _ := strconv.ParseFloat(fields[2], 64)

	return []metrics.Metric{
		metrics.NewMetric("cpu.load1", load1, "load", nil),
		metrics.NewMetric("cpu.load5", load5, "load", nil),
		metrics.NewMetric("cpu.load15", load15, "load", nil),
	}, nil
}

func parseUint64(s string) uint64 {
	val, _ := strconv.ParseUint(s, 10, 64)
	return val
}
