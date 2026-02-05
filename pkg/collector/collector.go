package collector

import (
	"time"

	"github.com/abhishekkarki/instvisor/pkg/metrics"
)

// Collector defines the interface for metric collectors
type Collector interface {
	// Name returns the collector name
	Name() string

	// Collect gathers metrics and returns them
	Collect() ([]metrics.Metric, error)

	// Interval returns how often this collector should run
	Interval() time.Duration
}
