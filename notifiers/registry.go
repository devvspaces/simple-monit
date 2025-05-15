// notifiers/registry.go
package notifiers

import (
	"fmt"
	"sync"

	"go.uber.org/zap"
)

// Registry manages the available notifiers
type Registry struct {
	notifiers map[string]Notifier
	mu        sync.RWMutex
	logger    *zap.Logger
}

// NewRegistry creates a new notifier registry
func NewRegistry(logger *zap.Logger) *Registry {
	return &Registry{
		notifiers: make(map[string]Notifier),
		logger:    logger,
	}
}

// Register adds a notifier to the registry
func (r *Registry) Register(notifier Notifier) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := notifier.Name()
	if name == "" {
		err := fmt.Errorf("notifier has empty name")
		r.logger.Error("Failed to register notifier", zap.Error(err))
		return err
	}

	if _, exists := r.notifiers[name]; exists {
		err := fmt.Errorf("notifier with name '%s' already registered", name)
		r.logger.Error("Failed to register notifier", zap.Error(err))
		return err
	}

	r.notifiers[name] = notifier
	return nil
}

// Get returns a notifier by name
func (r *Registry) Get(name string) (Notifier, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	notifier, exists := r.notifiers[name]
	return notifier, exists
}

// GetAll returns a list of all registered notifiers
func (r *Registry) GetAll() []Notifier {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []Notifier
	for _, notifier := range r.notifiers {
		result = append(result, notifier)
	}
	return result
}

// NotifierNames returns a list of all registered notifier names
func (r *Registry) NotifierNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var names []string
	for name := range r.notifiers {
		names = append(names, name)
	}
	return names
}
