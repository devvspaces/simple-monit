// collectors/collector.go
package collectors

import (
	"context"
	"time"
)

// Threshold represents a monitoring threshold
type Threshold struct {
	Type     string  // "absolute" or "percentage"
	Metric   string  // Name of the metric
	Operator string  // "less_than", "greater_than", "equals"
	Value    float64 // Threshold value
	Severity string  // "warning", "critical", etc.
}

// Result represents the result of a collection operation
type Result struct {
	IsHealthy  bool                   `json:"is_healthy"`
	Collector  string                 `json:"collector"`
	Timestamp  time.Time              `json:"timestamp"`
	Message    string                 `json:"message"`
	Metrics    map[string]float64     `json:"metrics"`
	Thresholds []Threshold            `json:"thresholds,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// Collector defines the interface that all collectors must implement
type Collector interface {
	// Name returns the unique name of the collector
	Name() string

	// Init initializes the collector with its configuration
	Init(settings map[string]interface{}) error

	// Collect performs the collection operation and returns results
	Collect(ctx context.Context) ([]Result, error)

	// Cleanup performs any necessary cleanup operations
	Cleanup() error
}
