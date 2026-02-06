package analyzer

import "time"

// ResourceStats contains statistical analysis of resource usage
type ResourceStats struct {
	MetricName string

	// Statistical measures
	Min    float64
	Max    float64
	Mean   float64
	Median float64
	P50    float64 // 50th percentile
	P90    float64 // 90th percentile
	P95    float64 // 95th percentile
	P99    float64 // 99th percentile

	// Pattern information
	StdDev  float64
	Samples int

	// Time range
	StartTime time.Time
	EndTime   time.Time
}

// WorkloadPattern describe the nature of the workload
type WorkloadPattern string

const (
	PatternSteadyState WorkloadPattern = "steady_state" // Consistent load
	PatternBursty      WorkloadPattern = "bursty"       // Irregular spikes
	PatternScheduled   WorkloadPattern = "scheduled"    // Regular patterns (e.g., cron jobs)
	PatternGrowing     WorkloadPattern = "growing"      // Increasing trend
)

// ResourceAnalysis contains complete analysis of a time period
type ResourceAnalysis struct {
	Period time.Duration

	// Resource statistics
	CPU     *ResourceStats
	Memory  *ResourceStats
	Disk    *ResourceStats
	Network *ResourceStats

	// Pattern detection
	Pattern    WorkloadPattern
	Confidence float64 // 0-1, how confident we are in the pattern

	// Observations
	Observations []string
}

// InstanceRecommendation suggests optimal sizing
type InstanceRecommendation struct {
	// Current usage summary
	CurrentCPUUsage    float64 // P95 CPU usage
	CurrentMemoryUsage float64 // P95 memory usage in GB

	// Recommended specs
	RecommendedCPU    int     // Number of vCPUs
	RecommendedMemory float64 // RAM in GB
	RecommendedDisk   float64 // Disk throughput in MB/s

	// Safety margins
	CPUHeadroom    float64 // % headroom for bursts
	MemoryHeadroom float64 // % headroom for bursts

	// Instance type suggestions (cloud-specific)
	CloudProviderSuggestions map[string][]string

	// Cost analysis
	EstimatedSavings float64 // % saving vs current

	// Reasoning
	Rationale []string
	Warnings  []string
}
