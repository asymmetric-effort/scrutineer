package expression

import "fmt"

// Func is the signature for built-in expression functions.
type Func func(args []any) (any, error)

// Registry maps function names to implementations.
type Registry struct {
	funcs map[string]Func
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{funcs: make(map[string]Func)}
}

// Register adds a function to the registry.
func (r *Registry) Register(name string, fn Func) error {
	if _, exists := r.funcs[name]; exists {
		return fmt.Errorf("expression: function %q already registered", name)
	}
	r.funcs[name] = fn
	return nil
}

// Get returns the function with the given name.
func (r *Registry) Get(name string) (Func, bool) {
	fn, ok := r.funcs[name]
	return fn, ok
}

// Names returns all registered function names.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.funcs))
	for name := range r.funcs {
		names = append(names, name)
	}
	return names
}

// DefaultRegistry returns a registry pre-loaded with all built-in functions.
func DefaultRegistry() *Registry {
	r := NewRegistry()
	registerBuiltins(r)
	return r
}
