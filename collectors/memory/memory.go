// collectors/memory/memory.go
package memory

import (
	"context"
	"fmt"
	"time"

	"server-monitor/collectors"

	"github.com/shirou/gopsutil/v3/mem"
	"go.uber.org/zap"
)

// MemoryCollector implements the Collector interface for memory monitoring
type MemoryCollector struct {
	thresholdPercent float64
	collectorName    string
	logger           *zap.Logger
}

// NewMemoryCollector creates a new memory collector
func NewMemoryCollector(logger *zap.Logger) *MemoryCollector {
	return &MemoryCollector{
		collectorName: "memory",
		logger:        logger,
	}
}

// Name returns the name of the collector
func (c *MemoryCollector) Name() string {
	return c.collectorName
}

// Init initializes the memory collector with configuration
func (c *MemoryCollector) Init(settings map[string]interface{}) error {
	// Set default threshold
	c.thresholdPercent = 90.0

	// Override with config if provided
	if val, ok := settings["threshold_percent"].(float64); ok {
		c.thresholdPercent = val
	}

	return nil
}

// Collect gathers memory metrics
func (c *MemoryCollector) Collect(ctx context.Context) ([]collectors.Result, error) {
	// Check if context is cancelled
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		// Continue processing
	}

	// Get memory stats
	memStats, err := mem.VirtualMemory()
	if err != nil {
		c.logger.Error("Failed to get memory stats", zap.Error(err))
		return nil, err
	}

	// Calculate memory usage
	usedPercent := memStats.UsedPercent

	// Create metrics map
	metrics := map[string]float64{
		"total_gb":     float64(memStats.Total) / (1024 * 1024 * 1024),
		"used_gb":      float64(memStats.Used) / (1024 * 1024 * 1024),
		"free_gb":      float64(memStats.Free) / (1024 * 1024 * 1024),
		"used_percent": usedPercent,
	}

	// Check threshold
	isHealthy := true
	var message string

	if usedPercent > c.thresholdPercent {
		isHealthy = false
		message = fmt.Sprintf("High memory usage: %.2f%% used (threshold: %.2f%%)",
			usedPercent, c.thresholdPercent)
	}

	// Add thresholds that were evaluated
	thresholds := []collectors.Threshold{
		{
			Type:     "percentage",
			Metric:   "used_percent",
			Operator: "greater_than",
			Value:    c.thresholdPercent,
			Severity: "warning",
		},
	}

	// Create result
	result := collectors.Result{
		IsHealthy:  isHealthy,
		Collector:  c.Name(),
		Timestamp:  time.Now(),
		Metrics:    metrics,
		Thresholds: thresholds,
	}

	// Add message if unhealthy
	if !isHealthy {
		result.Message = message
	}

	c.logger.Info("Memory metrics collected", zap.Any("result", result))
	return []collectors.Result{result}, nil
}

// Cleanup performs any necessary cleanup
func (c *MemoryCollector) Cleanup() error {
	// No cleanup needed for memory collector
	return nil
}
