// collectors/disk/disk.go
package disk

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"server-monitor/collectors"

	"go.uber.org/zap"
	"golang.org/x/sys/unix"
)

// DiskCollector implements the Collector interface for disk space monitoring
type DiskCollector struct {
	paths         []PathConfig
	collectorName string
	logger        *zap.Logger
}

// PathConfig represents the configuration for a single disk path to monitor
type PathConfig struct {
	Path             string  `json:"path"`
	ThresholdGB      float64 `json:"threshold_gb"`
	ThresholdPercent float64 `json:"threshold_percent"`
}

// NewDiskCollector creates a new disk space collector
func NewDiskCollector(logger *zap.Logger) *DiskCollector {
	return &DiskCollector{
		collectorName: "disk_space",
		logger:        logger,
	}
}

// Name returns the name of the collector
func (c *DiskCollector) Name() string {
	return c.collectorName
}

// Init initializes the disk collector with configuration
func (c *DiskCollector) Init(settings map[string]interface{}) error {
	// Get paths array from settings
	pathsRaw, ok := settings["paths"]
	if !ok {
		err := fmt.Errorf("missing 'paths' configuration for disk collector")
		c.logger.Error("Init error", zap.Error(err))
		return err
	}

	// Convert paths to the correct type
	pathsArray, ok := pathsRaw.([]interface{})
	if !ok {
		err := fmt.Errorf("'paths' should be an array")
		c.logger.Error("Init error", zap.Error(err))
		return err
	}

	// Process each path configuration
	for _, pathRaw := range pathsArray {
		pathMap, ok := pathRaw.(map[string]interface{})
		if !ok {
			err := fmt.Errorf("each path should be an object")
			c.logger.Error("Init error", zap.Error(err))
			return err
		}

		path, ok := pathMap["path"].(string)
		if !ok {
			err := fmt.Errorf("path must be a string")
			c.logger.Error("Init error", zap.Error(err))
			return err
		}

		// Resolve path to absolute path
		absPath, err := filepath.Abs(path)
		if err != nil {
			err := fmt.Errorf("could not resolve path %s: %w", path, err)
			c.logger.Error("Init error", zap.Error(err))
		}

		// Default thresholds if not provided
		thresholdGB := 5.0
		if val, ok := pathMap["threshold_gb"].(float64); ok {
			thresholdGB = val
		}

		thresholdPercent := 90.0
		if val, ok := pathMap["threshold_percent"].(float64); ok {
			thresholdPercent = val
		}

		c.paths = append(c.paths, PathConfig{
			Path:             absPath,
			ThresholdGB:      thresholdGB,
			ThresholdPercent: thresholdPercent,
		})
	}

	if len(c.paths) == 0 {
		err := fmt.Errorf("no valid paths configured for disk collector")
		c.logger.Error("Init error", zap.Error(err))
		return err
	}

	return nil
}

// Collect gathers disk space metrics
func (c *DiskCollector) Collect(ctx context.Context) ([]collectors.Result, error) {
	var results []collectors.Result

	for _, path := range c.paths {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
			// Continue processing
		}

		// Get disk usage stats
		var stat unix.Statfs_t
		if err := unix.Statfs(path.Path, &stat); err != nil {
			c.logger.Error("Failed to get disk stats", zap.String("path", path.Path), zap.Error(err))
			return results, err
		}

		// Calculate disk usage metrics
		totalBytes := float64(stat.Blocks) * float64(stat.Bsize)
		freeBytes := float64(stat.Bfree) * float64(stat.Bsize)
		usedBytes := totalBytes - freeBytes

		// Convert to GB
		totalGB := totalBytes / (1024 * 1024 * 1024)
		freeGB := freeBytes / (1024 * 1024 * 1024)
		usedGB := usedBytes / (1024 * 1024 * 1024)

		// Calculate percentages
		usedPercent := (usedBytes / totalBytes) * 100

		// Create metrics map
		metrics := map[string]float64{
			"total_gb":     totalGB,
			"free_gb":      freeGB,
			"used_gb":      usedGB,
			"used_percent": usedPercent,
		}

		// Check thresholds
		isHealthy := true
		var message string

		if freeGB < path.ThresholdGB {
			isHealthy = false
			message = fmt.Sprintf("Low disk space on %s: %.2fGB free (threshold: %.2fGB)",
				path.Path, freeGB, path.ThresholdGB)
		} else if usedPercent > path.ThresholdPercent {
			isHealthy = false
			message = fmt.Sprintf("High disk usage on %s: %.2f%% used (threshold: %.2f%%)",
				path.Path, usedPercent, path.ThresholdPercent)
		}

		// Add thresholds that were evaluated
		thresholds := []collectors.Threshold{
			{
				Type:     "absolute",
				Metric:   "free_gb",
				Operator: "less_than",
				Value:    path.ThresholdGB,
				Severity: "critical",
			},
			{
				Type:     "percentage",
				Metric:   "used_percent",
				Operator: "greater_than",
				Value:    path.ThresholdPercent,
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
			Metadata: map[string]interface{}{
				"path": path.Path,
			},
		}

		// Add message if unhealthy
		if !isHealthy {
			result.Message = message
		}

		results = append(results, result)
	}

	c.logger.Info("Collected disk metrics", zap.Any("results", results))
	return results, nil
}

// Cleanup performs any necessary cleanup
func (c *DiskCollector) Cleanup() error {
	// No cleanup needed for disk collector
	return nil
}
