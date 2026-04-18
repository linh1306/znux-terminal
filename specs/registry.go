package specs

import "sort"

var registry = map[string]*Spec{}

// Register registers a spec for a command
func Register(name string, s *Spec) {
	registry[name] = s
}

// Get returns the spec for a command name
func Get(name string) *Spec {
	return registry[name]
}

// RegisteredCommands returns all registered command names, sorted.
func RegisteredCommands() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}