// collector/registry.go
package collectors

import (
	"fmt"
	"sync"

	"go.uber.org/zap"
)

// Registry manages the available collectors
type Registry struct {
	collectors map[string]Collector
	mu         sync.RWMutex
	logger     *zap.Logger
}

// NewRegistry creates a new collector registry
func NewRegistry(logger *zap.Logger) *Registry {
	return &Registry{
		collectors: make(map[string]Collector),
		logger:     logger,
	}
}

// Register adds a collector to the registry
func (r *Registry) Register(collector Collector) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := collector.Name()
	if name == "" {
		err := fmt.Errorf("collector has empty name")
		r.logger.Error("Failed to register collector", zap.Error(err))
		return err
	}

	if _, exists := r.collectors[name]; exists {
		err := fmt.Errorf("collector with name '%s' already registered", name)
		r.logger.Error("Failed to register collector", zap.Error(err))
		return err
	}

	r.collectors[name] = collector
	return nil
}

// Get returns a collector by name
func (r *Registry) Get(name string) (Collector, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	collector, exists := r.collectors[name]
	return collector, exists
}

// GetAll returns a list of all registered collectors
func (r *Registry) GetAll() []Collector {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []Collector
	for _, collector := range r.collectors {
		result = append(result, collector)
	}
	return result
}

// CollectorNames returns a list of all registered collector names
func (r *Registry) CollectorNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var names []string
	for name := range r.collectors {
		names = append(names, name)
	}
	return names
}
