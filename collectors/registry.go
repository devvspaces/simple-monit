// collector/registry.go
package collectors

import (
	"fmt"
	"sync"
)

// Registry manages the available collectors
type Registry struct {
	collectors map[string]Collector
	mu         sync.RWMutex
}

// NewRegistry creates a new collector registry
func NewRegistry() *Registry {
	return &Registry{
		collectors: make(map[string]Collector),
	}
}

// Register adds a collector to the registry
func (r *Registry) Register(collector Collector) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := collector.Name()
	if name == "" {
		return fmt.Errorf("collector has empty name")
	}

	if _, exists := r.collectors[name]; exists {
		return fmt.Errorf("collector with name '%s' already registered", name)
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
