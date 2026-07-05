package specs

// SourceContext carries shell/editor state needed by dynamic sources.
type SourceContext struct {
	CWD                  string
	CommandLine          string
	Command              string
	Subcommand           string
	ArgIndex             int
	IsSubcommandPosition bool
}

type SourceFunc func(source SourceSpec, ctx SourceContext, partial string) []Suggestion

var sourceRegistry = map[string]SourceFunc{}

func RegisterSource(name string, fn SourceFunc) {
	sourceRegistry[name] = fn
}

func GetSource(name string) SourceFunc {
	return sourceRegistry[name]
}
