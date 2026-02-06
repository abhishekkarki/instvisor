package analyzer

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/abhishekkarki/instvisor/pkg/storage"
)

// Analyzer analyzes metrics and generates insights
type Analyzer struct {
	storage storage.Storage
}

// NewAnalyzer creates a new analyzer
func NewAnalyzer(store storage.Storage) *Analyzer {
	return &Analyzer{
		storage: store,
	}
}

// AnalyzePeriod analyzes metrics for a given time period
func (a *Analyzer) AnalyzePeriod(period time.Duration) (*ResourceAnalysis, error) {
	end := time.Now()
	start := end.Add(-period)

	analysis := &ResourceAnalysis{
		Period:       period,
		Observations: make([]string, 0),
	}

	// Analyze CPU
	cpuStats, err := a.analyzeMetric("cpu.usage", start, end, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze CPU: %w", err)
	}
	analysis.CPU = cpuStats

	// Analyze Memory
	memStats, err := a.analyzeMetric("memory.usage_percent", start, end, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze memory: %w", err)
	}
	analysis.Memory = memStats

	// Analyze Disk (average across all devices)
	diskStats, err := a.analyzeMetric("disk.utilization_percent", start, end, nil)
	if err == nil {
		analysis.Disk = diskStats
	}

	// Detect workload pattern
	analysis.Pattern, analysis.Confidence = a.detectPattern(cpuStats)

	// Generate observations
	analysis.Observations = a.generateObservations(analysis)

	return analysis, nil
}

// analyzeMetric computes statistics for a specific metric
func (a *Analyzer) analyzeMetric(metricName string, start, end time.Time, labels map[string]string) (*ResourceStats, error) {
	// Query metrics from storage
	metricsData, err := a.storage.QueryMetrics(metricName, start, end, labels)
	if err != nil {
		return nil, err
	}

	if len(metricsData) == 0 {
		return nil, fmt.Errorf("no data found for metric %s", metricName)
	}

	// Extract values
	values := make([]float64, len(metricsData))
	for i, m := range metricsData {
		values[i] = m.Value
	}

	// Sort for percentile calculations
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	stats := &ResourceStats{
		MetricName: metricName,
		Samples:    len(values),
		StartTime:  start,
		EndTime:    end,
		Min:        sorted[0],
		Max:        sorted[len(sorted)-1],
		Mean:       mean(values),
		Median:     percentile(sorted, 50),
		P50:        percentile(sorted, 50),
		P90:        percentile(sorted, 90),
		P95:        percentile(sorted, 95),
		P99:        percentile(sorted, 99),
		StdDev:     stdDev(values),
	}

	return stats, nil
}

// detectPattern identifies the workload pattern
func (a *Analyzer) detectPattern(cpuStats *ResourceStats) (WorkloadPattern, float64) {
	if cpuStats == nil {
		return PatternSteadyState, 0.0
	}

	// Calculate coefficient of variation (CV = stddev / mean)
	cv := cpuStats.StdDev / cpuStats.Mean

	// Calculate range ratio (max / mean)
	rangeRatio := cpuStats.Max / cpuStats.Mean

	// Pattern detection logic
	switch {
	case cv < 0.2 && rangeRatio < 1.5:
		// Low variation, consistent load
		return PatternSteadyState, 0.9

	case cv > 0.5 || rangeRatio > 3.0:
		// High variation, unpredictable spikes
		return PatternBursty, 0.8

	case cv >= 0.2 && cv <= 0.5:
		// Moderate variation - could be scheduled or growing
		// TODO: Implement time-series analysis to distinguish
		return PatternScheduled, 0.6

	default:
		return PatternSteadyState, 0.5
	}
}

// generateObservations creates human-readable insights
func (a *Analyzer) generateObservations(analysis *ResourceAnalysis) []string {
	var obs []string

	if analysis.CPU != nil {
		if analysis.CPU.P95 < 30 {
			obs = append(obs, fmt.Sprintf("CPU is underutilized (P95: %.1f%%)", analysis.CPU.P95))
		} else if analysis.CPU.P95 > 80 {
			obs = append(obs, fmt.Sprintf("CPU is heavily utilized (P95: %.1f%%)", analysis.CPU.P95))
		}

		if analysis.CPU.Max > 95 {
			obs = append(obs, "CPU saturation detected - consider more cores")
		}
	}

	if analysis.Memory != nil {
		if analysis.Memory.P95 < 40 {
			obs = append(obs, fmt.Sprintf("Memory is underutilized (P95: %.1f%%)", analysis.Memory.P95))
		} else if analysis.Memory.P95 > 85 {
			obs = append(obs, fmt.Sprintf("Memory pressure detected (P95: %.1f%%)", analysis.Memory.P95))
		}
	}

	switch analysis.Pattern {
	case PatternBursty:
		obs = append(obs, "Bursty workload detected - consider burstable instances or auto-scaling")
	case PatternSteadyState:
		obs = append(obs, "Steady-state workload - predictable resource usage")
	case PatternScheduled:
		obs = append(obs, "Periodic pattern detected - may benefit from scheduled scaling")
	}

	return obs
}

// Statistical helper functions
func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}

	index := (p / 100.0) * float64(len(sorted)-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))

	if lower == upper {
		return sorted[lower]
	}

	// Linear interpolation
	weight := index - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}

func stdDev(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	m := mean(values)
	variance := 0.0

	for _, v := range values {
		diff := v - m
		variance += diff * diff
	}

	variance /= float64(len(values))
	return math.Sqrt(variance)
}
