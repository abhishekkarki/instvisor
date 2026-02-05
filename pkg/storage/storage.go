package storage

import (
	"time"

	"github.com/abhishekkarki/instvisor/pkg/metrics"
)

// Storage defines the interface for metric storage
type Storage interface {
	// WriteMetrics writes a batch of metrics
	WriteMetrics(metrics []metrics.Metric) error

	// QueryMetrics retrieves metrics based on criteria
	QueryMetrics(name string, start, end time.Time, labels map[string]string) ([]metrics.Metric, error)

	// DeleteOldMetrics removes metrics older than the specified time
	DeleteOldMetrics(before time.Time) error

	// Close closes the storage
	Close() error
}
