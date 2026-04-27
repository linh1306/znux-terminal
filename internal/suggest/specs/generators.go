package specs

var generatorRegistry = map[string]func() []Suggestion{}

func RegisterGenerator(name string, fn func() []Suggestion) {
	generatorRegistry[name] = fn
}

func GetGenerator(name string) func() []Suggestion {
	return generatorRegistry[name]
}
