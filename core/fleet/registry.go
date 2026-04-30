package fleet

import (
	"fmt"
	"sort"
)

// Factory creates a new Provider instance.
type Factory func() Provider

// Registry maps provider names to factories.
type Registry struct {
	factories map[string]Factory
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{factories: make(map[string]Factory)}
}

// Register adds a provider factory to the registry.
func (r *Registry) Register(name string, f Factory) error {
	if _, exists := r.factories[name]; exists {
		return fmt.Errorf("fleet: provider %q already registered", name)
	}
	r.factories[name] = f
	return nil
}

// Get creates a new Provider instance by name.
func (r *Registry) Get(name string) (Provider, error) {
	f, ok := r.factories[name]
	if !ok {
		return nil, fmt.Errorf("fleet: provider %q not registered", name)
	}
	return f(), nil
}

// Names returns all registered provider names, sorted.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
