package specs

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

const specsDirEnv = "ZNUX_SUGGEST_SPECS_DIR"

func init() {
	if err := LoadDefaultSpecs(); err != nil {
		panic(err)
	}
}

// LoadDefaultSpecs loads command specs from an external directory.
//
// Search order:
//   - $ZNUX_SUGGEST_SPECS_DIR
//   - ./suggest next to the executable
//   - ./suggest from the current working directory
func LoadDefaultSpecs() error {
	for _, dir := range defaultSpecDirs() {
		loaded, err := LoadDir(dir)
		if err != nil {
			return err
		}
		if loaded > 0 {
			return nil
		}
	}
	return nil
}

func defaultSpecDirs() []string {
	dirs := make([]string, 0, 8)
	if envDir := os.Getenv(specsDirEnv); envDir != "" {
		dirs = append(dirs, envDir)
	}

	if exe, err := os.Executable(); err == nil && exe != "" {
		exeDir := filepath.Dir(exe)
		dirs = append(dirs, filepath.Join(exeDir, "suggest"))
	}

	dirs = append(dirs, "suggest")

	return dirs
}

// LoadDir loads all YAML spec files in dir and returns how many specs were loaded.
func LoadDir(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("read specs dir %q: %w", dir, err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	loaded := 0
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return loaded, fmt.Errorf("read spec %q: %w", path, err)
		}
		spec, err := LoadYAML(data)
		if err != nil {
			return loaded, fmt.Errorf("load spec %q: %w", path, err)
		}
		Register(spec.Name, spec)
		loaded++
	}
	return loaded, nil
}

// YAML intermediate types — mirrors Spec but uses strings for generator/template

type yamlArgSpec struct {
	Name      string           `yaml:"name"`
	Generator string           `yaml:"generator,omitempty"`
	Variadic  bool             `yaml:"variadic,omitempty"`
	Template  string           `yaml:"template,omitempty"`
	Sources   []yamlSourceSpec `yaml:"sources,omitempty"`
}

type yamlSourceSpec struct {
	Type      string   `yaml:"type"`
	Include   []string `yaml:"include,omitempty"`
	Protocols []string `yaml:"protocols,omitempty"`
	State     string   `yaml:"state,omitempty"`
	Format    string   `yaml:"format,omitempty"`
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
	template := templateFromString(y.Template)
	return ArgSpec{
		Name:       y.Name,
		Generator:  y.Generator,
		IsVariadic: y.Variadic,
		Template:   template,
		Sources:    convertSourceSpecs(y.Sources, template),
	}
}

func convertSourceSpecs(ys []yamlSourceSpec, template Template) []SourceSpec {
	out := make([]SourceSpec, 0, len(ys)+1)
	for _, y := range ys {
		out = append(out, SourceSpec{
			Type:      y.Type,
			Include:   append([]string(nil), y.Include...),
			Protocols: append([]string(nil), y.Protocols...),
			State:     y.State,
			Format:    y.Format,
		})
	}

	switch template {
	case TemplateFileSystem:
		out = append(out, SourceSpec{Type: "path", Include: []string{"folder", "file"}})
	case TemplateFolder:
		out = append(out, SourceSpec{Type: "path", Include: []string{"folder"}})
	}

	if len(out) == 0 {
		return nil
	}
	return out
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
