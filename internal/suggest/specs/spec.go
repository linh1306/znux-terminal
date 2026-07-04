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
	KindInstall
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
	InsertText  string
}

// Completion returns the text inserted into the input when this suggestion is accepted.
func (s Suggestion) Completion() string {
	if s.InsertText != "" {
		return s.InsertText
	}
	return s.Name
}

// SourceSpec describes one dynamic suggestion source for an argument.
type SourceSpec struct {
	Type      string
	Include   []string
	Protocols []string
	State     string
	Format    string
}

// ArgSpec describes an argument specification
type ArgSpec struct {
	Name       string
	Generator  string // name registered in generator registry; empty = no dynamic suggestion
	IsVariadic bool
	Template   Template
	Sources    []SourceSpec
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
	Install     string
	Subcommands []Subcommand
	Options     []Option
	Args        []ArgSpec
}
