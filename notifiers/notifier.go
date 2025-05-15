// notifiers/notifier.go
package notifiers

import (
	"context"

	"server-monitor/collectors"
)

// Notifier defines the interface that all notification methods must implement
type Notifier interface {
	// Name returns the unique name of the notifier
	Name() string

	// Init initializes the notifier with its configuration
	Init(config map[string]interface{}) error

	// Notify sends an alert notification for the provided results
	Notify(ctx context.Context, results []collectors.Result) error

	// Close performs any necessary cleanup operations
	Close() error
}
