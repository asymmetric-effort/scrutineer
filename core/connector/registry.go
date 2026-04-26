package connector

import (
	"errors"
	"sort"
)

// Registry maps connector names to their factories.
type Registry struct {
	factories map[string]Factory
}

// NewRegistry creates a new, empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		factories: make(map[string]Factory),
	}
}

// Register adds a connector factory under the given name.
// Returns an error if name is already registered.
func (r *Registry) Register(name string, f Factory) error {
	if _, exists := r.factories[name]; exists {
		return errors.New("connector: factory already registered for " + name)
	}
	r.factories[name] = f
	return nil
}

// Get creates a new connector instance by name.
// Returns an error if name is not registered.
func (r *Registry) Get(name string) (Connector, error) {
	f, exists := r.factories[name]
	if !exists {
		return nil, errors.New("connector: no factory registered for " + name)
	}
	return f(), nil
}

// Names returns all registered connector names in sorted order.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
