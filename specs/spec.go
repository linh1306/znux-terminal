package specs

// SuggestionKind represents the type of a suggestion
type SuggestionKind int

const (
	KindSubcommand SuggestionKind = iota
	KindOption
	KindArg
	KindFile
	KindFolder
	KindValue
)

// Template represents the type of argument template
type Template int

const (
	TemplateNone Template = iota
	TemplateFileSystem
	TemplateFolder
)

// Suggestion represents a single autocomplete suggestion
type Suggestion struct {
	Name        string
	Description string
	Kind        SuggestionKind
}

// ArgSpec describes an argument specification
type ArgSpec struct {
	Name       string
	Generator  string // name registered in generator registry; empty = no dynamic suggestion
	IsVariadic bool
	Template   Template
}

// Option describes a command-line option
type Option struct {
	Names       []string
	Description string
	Args        []ArgSpec
}

// Subcommand describes a nested subcommand
type Subcommand struct {
	Name        string
	Description string
	Subcommands []Subcommand
	Options     []Option
	Args        []ArgSpec
}

// Spec describes the complete specification for a command
type Spec struct {
	Name        string
	Subcommands []Subcommand
	Options     []Option
	Args        []ArgSpec
}
