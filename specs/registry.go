package specs

var registry = map[string]*Spec{}

// Register registers a spec for a command
func Register(name string, s *Spec) {
	registry[name] = s
}

// Get returns the spec for a command name
func Get(name string) *Spec {
	return registry[name]
}