package metrics

import "time"

// Metric represents a single metric data point
type Metric struct {
	Timestamp time.Time         `json:"timestamp"`
	Name      string            `json:"name"`
	Value     float64           `json:"value"`
	Labels    map[string]string `json:"labels,omitempty"`
	Unit      string            `json:"unit"`
}

// MetricType defines the type of metric
type MetricType string

const (
	MetricTypeCPU     MetricType = "cpu"
	MetricTypeMemory  MetricType = "memory"
	MetricTypeDisk    MetricType = "disk"
	MetricTypeNetwork MetricType = "network"
	MetricTypeProcess MetricType = "process"
)

// NewMetric creates a new metric with current timestamp
func NewMetric(name string, value float64, unit string, labels map[string]string) Metric {
	if labels == nil {
		labels = make(map[string]string)
	}
	return Metric{
		Timestamp: time.Now(),
		Name:      name,
		Value:     value,
		Unit:      unit,
		Labels:    labels,
	}
}
