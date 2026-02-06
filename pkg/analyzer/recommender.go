package analyzer

import (
	"fmt"
	"math"
)

// Recommender generates instance sizing recommendations
type Recommender struct {
	analyzer *Analyzer
}

// NewRecommender creates a new recommender
func NewRecommender(analyzer *Analyzer) *Recommender {
	return &Recommender{
		analyzer: analyzer,
	}
}

// RecommendationConfig configures the recommendation engine
type RecommendationConfig struct {
	CPUHeadroomPercent    float64 // Default: 20%
	MemoryHeadroomPercent float64 // Default: 15%
	UseP95                bool    // Use P95 instead of P99
	CurrentCPUCores       int     // Current instance cores
	CurrentMemoryGB       float64 // Current instance memory
}

// DefaultRecommendationConfig returns sensible defaults
func DefaultRecommendationConfig() *RecommendationConfig {
	return &RecommendationConfig{
		CPUHeadroomPercent:    20.0,
		MemoryHeadroomPercent: 15.0,
		UseP95:                true,
		CurrentCPUCores:       0, // Unknown
		CurrentMemoryGB:       0, // Unknown
	}
}

// GenerateRecommendation creates a sizing recommendation
func (r *Recommender) GenerateRecommendation(analysis *ResourceAnalysis, config *RecommendationConfig) (*InstanceRecommendation, error) {
	if config == nil {
		config = DefaultRecommendationConfig()
	}

	rec := &InstanceRecommendation{
		CPUHeadroom:              config.CPUHeadroomPercent,
		MemoryHeadroom:           config.MemoryHeadroomPercent,
		CloudProviderSuggestions: make(map[string][]string),
		Rationale:                make([]string, 0),
		Warnings:                 make([]string, 0),
	}

	// Determine which percentile to use
	cpuPercentile := analysis.CPU.P95
	memPercentile := analysis.Memory.P95

	if !config.UseP95 {
		cpuPercentile = analysis.CPU.P99
		memPercentile = analysis.Memory.P99
	}

	rec.CurrentCPUUsage = cpuPercentile
	rec.CurrentMemoryUsage = memPercentile

	// Calculate required resources with headroom
	requiredCPUPercent := cpuPercentile * (1 + config.CPUHeadroomPercent/100)
	requiredMemPercent := memPercentile * (1 + config.MemoryHeadroomPercent/100)

	// Estimate required cores
	// If current usage is X% on N cores, and we want headroom, calculate needed cores
	if config.CurrentCPUCores > 0 {
		// Current cores * (required % / 100)
		neededCores := float64(config.CurrentCPUCores) * (requiredCPUPercent / 100.0)
		rec.RecommendedCPU = int(math.Ceil(neededCores))

		// Don't recommend less than 1 core
		if rec.RecommendedCPU < 1 {
			rec.RecommendedCPU = 1
		}

		// Add rationale
		if rec.RecommendedCPU < config.CurrentCPUCores {
			savings := float64(config.CurrentCPUCores-rec.RecommendedCPU) / float64(config.CurrentCPUCores) * 100
			rec.Rationale = append(rec.Rationale,
				fmt.Sprintf("Can reduce from %d to %d vCPUs (%.0f%% reduction)",
					config.CurrentCPUCores, rec.RecommendedCPU, savings))
			rec.EstimatedSavings = savings
		} else if rec.RecommendedCPU > config.CurrentCPUCores {
			rec.Rationale = append(rec.Rationale,
				fmt.Sprintf("Recommend increasing from %d to %d vCPUs for headroom",
					config.CurrentCPUCores, rec.RecommendedCPU))
			rec.Warnings = append(rec.Warnings, "Current CPU allocation may be insufficient")
		} else {
			rec.Rationale = append(rec.Rationale, "Current CPU allocation is appropriate")
		}
	} else {
		// No current info - make best guess based on usage
		// Assume we want < 80% utilization target
		targetUtilization := 80.0
		rec.RecommendedCPU = int(math.Ceil(cpuPercentile / targetUtilization * 2))
		if rec.RecommendedCPU < 1 {
			rec.RecommendedCPU = 1
		}
		rec.Rationale = append(rec.Rationale,
			fmt.Sprintf("Based on %.1f%% CPU usage, recommend %d vCPUs",
				cpuPercentile, rec.RecommendedCPU))
	}

	// Calculate required memory
	if config.CurrentMemoryGB > 0 {
		neededMemGB := config.CurrentMemoryGB * (requiredMemPercent / 100.0)
		rec.RecommendedMemory = math.Ceil(neededMemGB)

		if rec.RecommendedMemory < config.CurrentMemoryGB {
			savings := (config.CurrentMemoryGB - rec.RecommendedMemory) / config.CurrentMemoryGB * 100
			rec.Rationale = append(rec.Rationale,
				fmt.Sprintf("Can reduce from %.1fGB to %.1fGB RAM (%.0f%% reduction)",
					config.CurrentMemoryGB, rec.RecommendedMemory, savings))
		} else if rec.RecommendedMemory > config.CurrentMemoryGB {
			rec.Rationale = append(rec.Rationale,
				fmt.Sprintf("Recommend increasing from %.1fGB to %.1fGB RAM",
					config.CurrentMemoryGB, rec.RecommendedMemory))
			rec.Warnings = append(rec.Warnings, "Current memory allocation may be insufficient")
		}
	} else {
		// Assume 2GB per vCPU as baseline, adjust based on memory pressure
		rec.RecommendedMemory = float64(rec.RecommendedCPU) * 2
		if memPercentile > 50 {
			// Memory-intensive workload
			rec.RecommendedMemory = float64(rec.RecommendedCPU) * 4
		}
		rec.Rationale = append(rec.Rationale,
			fmt.Sprintf("Based on %.1f%% memory usage, recommend %.1fGB RAM",
				memPercentile, rec.RecommendedMemory))
	}

	// Add pattern-specific recommendations
	switch analysis.Pattern {
	case PatternBursty:
		rec.Rationale = append(rec.Rationale,
			"Bursty workload: Consider burstable instance types (T-series on AWS, B-series on Azure)")
		rec.CloudProviderSuggestions["AWS"] = r.suggestAWSInstances(rec.RecommendedCPU, rec.RecommendedMemory, true)
	case PatternSteadyState:
		rec.Rationale = append(rec.Rationale,
			"Steady workload: Standard instances with reserved capacity recommended")
		rec.CloudProviderSuggestions["AWS"] = r.suggestAWSInstances(rec.RecommendedCPU, rec.RecommendedMemory, false)
	case PatternScheduled:
		rec.Rationale = append(rec.Rationale,
			"Scheduled workload: Consider auto-scaling or scheduled scaling policies")
	}

	// OTC suggestions
	rec.CloudProviderSuggestions["OTC"] = r.suggestOTCInstances(rec.RecommendedCPU, rec.RecommendedMemory)

	return rec, nil
}

// suggestAWSInstances suggests AWS EC2 instance types
func (r *Recommender) suggestAWSInstances(cpu int, memGB float64, burstable bool) []string {
	var suggestions []string

	if burstable {
		// T-series (burstable)
		switch {
		case cpu <= 2 && memGB <= 8:
			suggestions = append(suggestions, "t3.small", "t3.medium")
		case cpu <= 4 && memGB <= 16:
			suggestions = append(suggestions, "t3.large", "t3.xlarge")
		case cpu <= 8 && memGB <= 32:
			suggestions = append(suggestions, "t3.2xlarge")
		}
	} else {
		// M-series (general purpose) or C-series (compute optimized)
		memPerCore := memGB / float64(cpu)

		if memPerCore < 3 {
			// Compute optimized (C-series)
			switch {
			case cpu <= 2:
				suggestions = append(suggestions, "c5.large")
			case cpu <= 4:
				suggestions = append(suggestions, "c5.xlarge")
			case cpu <= 8:
				suggestions = append(suggestions, "c5.2xlarge")
			case cpu <= 16:
				suggestions = append(suggestions, "c5.4xlarge")
			}
		} else {
			// General purpose (M-series)
			switch {
			case cpu <= 2 && memGB <= 8:
				suggestions = append(suggestions, "m5.large")
			case cpu <= 4 && memGB <= 16:
				suggestions = append(suggestions, "m5.xlarge")
			case cpu <= 8 && memGB <= 32:
				suggestions = append(suggestions, "m5.2xlarge")
			case cpu <= 16 && memGB <= 64:
				suggestions = append(suggestions, "m5.4xlarge")
			}
		}
	}

	return suggestions
}

// suggestOTCInstances suggests Open Telekom Cloud instance types
func (r *Recommender) suggestOTCInstances(cpu int, memGB float64) []string {
	var suggestions []string

	// OTC ECS flavors: s3 (general), c3 (compute), m3 (memory)
	memPerCore := memGB / float64(cpu)

	if memPerCore < 3 {
		// Compute optimized
		switch {
		case cpu <= 2:
			suggestions = append(suggestions, "c3.large.2")
		case cpu <= 4:
			suggestions = append(suggestions, "c3.xlarge.2")
		case cpu <= 8:
			suggestions = append(suggestions, "c3.2xlarge.2")
		}
	} else if memPerCore > 6 {
		// Memory optimized
		switch {
		case cpu <= 2:
			suggestions = append(suggestions, "m3.large.8")
		case cpu <= 4:
			suggestions = append(suggestions, "m3.xlarge.8")
		case cpu <= 8:
			suggestions = append(suggestions, "m3.2xlarge.8")
		}
	} else {
		// General purpose
		switch {
		case cpu <= 1:
			suggestions = append(suggestions, "s3.small.1")
		case cpu <= 2:
			suggestions = append(suggestions, "s3.medium.2", "s3.large.2")
		case cpu <= 4:
			suggestions = append(suggestions, "s3.xlarge.2", "s3.xlarge.4")
		case cpu <= 8:
			suggestions = append(suggestions, "s3.2xlarge.2", "s3.2xlarge.4")
		}
	}

	return suggestions
}
