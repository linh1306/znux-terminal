package specs

import (
	"embed"
	"fmt"

	"gopkg.in/yaml.v3"
)

//go:embed data/*.yaml
var dataFS embed.FS

func init() {
	entries, err := dataFS.ReadDir("data")
	if err != nil {
		panic(err)
	}
	for _, entry := range entries {
		data, err := dataFS.ReadFile("data/" + entry.Name())
		if err != nil {
			panic(err)
		}
		spec := MustLoadYAML(data)
		Register(spec.Name, spec)
	}
}

// YAML intermediate types — mirrors Spec but uses strings for generator/template

type yamlArgSpec struct {
	Name      string `yaml:"name"`
	Generator string `yaml:"generator,omitempty"`
	Variadic  bool   `yaml:"variadic,omitempty"`
	Template  string `yaml:"template,omitempty"`
}

type yamlOption struct {
	Names       []string      `yaml:"names"`
	Description string        `yaml:"description"`
	Args        []yamlArgSpec `yaml:"args,omitempty"`
}

type yamlSubcommand struct {
	Name        string           `yaml:"name"`
	Description string           `yaml:"description"`
	Subcommands []yamlSubcommand `yaml:"subcommands,omitempty"`
	Options     []yamlOption     `yaml:"options,omitempty"`
	Args        []yamlArgSpec    `yaml:"args,omitempty"`
}

type yamlSpec struct {
	Name        string           `yaml:"name"`
	Subcommands []yamlSubcommand `yaml:"subcommands,omitempty"`
	Options     []yamlOption     `yaml:"options,omitempty"`
	Args        []yamlArgSpec    `yaml:"args,omitempty"`
}

func templateFromString(s string) Template {
	switch s {
	case "filesystem":
		return TemplateFileSystem
	case "folder":
		return TemplateFolder
	default:
		return TemplateNone
	}
}

func convertArgSpec(y yamlArgSpec) ArgSpec {
	return ArgSpec{
		Name:       y.Name,
		Generator:  y.Generator,
		IsVariadic: y.Variadic,
		Template:   templateFromString(y.Template),
	}
}

func convertArgSpecs(ys []yamlArgSpec) []ArgSpec {
	if len(ys) == 0 {
		return nil
	}
	out := make([]ArgSpec, len(ys))
	for i, y := range ys {
		out[i] = convertArgSpec(y)
	}
	return out
}

func convertOption(y yamlOption) Option {
	return Option{
		Names:       y.Names,
		Description: y.Description,
		Args:        convertArgSpecs(y.Args),
	}
}

func convertOptions(ys []yamlOption) []Option {
	if len(ys) == 0 {
		return nil
	}
	out := make([]Option, len(ys))
	for i, y := range ys {
		out[i] = convertOption(y)
	}
	return out
}

func convertSubcommand(y yamlSubcommand) Subcommand {
	return Subcommand{
		Name:        y.Name,
		Description: y.Description,
		Subcommands: convertSubcommands(y.Subcommands),
		Options:     convertOptions(y.Options),
		Args:        convertArgSpecs(y.Args),
	}
}

func convertSubcommands(ys []yamlSubcommand) []Subcommand {
	if len(ys) == 0 {
		return nil
	}
	out := make([]Subcommand, len(ys))
	for i, y := range ys {
		out[i] = convertSubcommand(y)
	}
	return out
}

// LoadYAML parses YAML bytes into a Spec.
func LoadYAML(data []byte) (*Spec, error) {
	var ys yamlSpec
	if err := yaml.Unmarshal(data, &ys); err != nil {
		return nil, fmt.Errorf("yaml unmarshal: %w", err)
	}
	return &Spec{
		Name:        ys.Name,
		Subcommands: convertSubcommands(ys.Subcommands),
		Options:     convertOptions(ys.Options),
		Args:        convertArgSpecs(ys.Args),
	}, nil
}

// MustLoadYAML panics if the YAML cannot be parsed.
func MustLoadYAML(data []byte) *Spec {
	spec, err := LoadYAML(data)
	if err != nil {
		panic(err)
	}
	return spec
}
