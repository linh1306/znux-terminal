package suggest

import (
	"os"
	"os/exec"
	"strings"

	"github.com/nguyenlinh13602/goshell/internal/buffer"
	"github.com/nguyenlinh13602/goshell/internal/suggest/specs"
)

// Engine provides suggestions based on current input
type Engine struct {
	cwd        string
	currentBuf *buffer.LineBuf
	currentCtx *buffer.Context
}

// NewEngine creates a new suggestion engine
func NewEngine() *Engine {
	return &Engine{}
}

// SetCWD sets the shell working directory used by context-aware suggestions.
func (e *Engine) SetCWD(cwd string) {
	e.cwd = cwd
}

// GetSuggestions returns suggestions for the current input
func (e *Engine) GetSuggestions(buf *buffer.LineBuf, ctx *buffer.Context) []specs.Suggestion {
	e.currentBuf = buf
	e.currentCtx = ctx

	switch ctx.Level {
	case buffer.ContextCommandPartial:
		return e.matchCommand(buf.String())

	case buffer.ContextCommand:
		// After pressing space, check for subcommand
		if spec := specs.Get(ctx.Command); spec != nil {
			return e.withInstallSuggestion(spec, e.getSubcommandSuggestions(spec, ""))
		}
		return nil

	case buffer.ContextSubcommand, buffer.ContextSubcommandPartial:
		spec := specs.Get(ctx.Command)
		if spec == nil {
			return nil
		}
		partial := ""
		if ctx.Level == buffer.ContextSubcommandPartial {
			partial = ctx.Subcommand
		}
		// If the spec has subcommands, try matching them first.
		if len(spec.Subcommands) > 0 {
			subs := e.getSubcommandSuggestions(spec, partial)
			argSuggestions := e.suggestFromArgSpecs(spec.Args, partial)
			if len(subs) > 0 || len(argSuggestions) > 0 {
				return e.withInstallSuggestion(spec, append(subs, argSuggestions...))
			}
		}
		// No matching subcommands — the partial token is an arg (e.g. "cd ./").
		return e.withInstallSuggestion(spec, e.suggestFromArgSpecs(spec.Args, partial))

	case buffer.ContextFlag, buffer.ContextFlagPartial:
		return e.getFlagSuggestions(ctx)

	case buffer.ContextArg, buffer.ContextArgPartial:
		return e.getArgSuggestions(ctx)

	default:
		return nil
	}
}

// matchCommand finds commands matching the prefix
func (e *Engine) matchCommand(prefix string) []specs.Suggestion {
	var suggestions []specs.Suggestion
	prefix = strings.ToLower(prefix)

	for _, cmd := range specs.RegisteredCommands() {
		if strings.HasPrefix(cmd, prefix) {
			if spec := specs.Get(cmd); spec != nil && e.needsInstall(spec) {
				suggestions = append(suggestions, e.installSuggestion(spec))
			}
			suggestions = append(suggestions, specs.Suggestion{
				Name: cmd,
				Kind: specs.KindSubcommand,
			})
		}
	}

	return suggestions
}

func (e *Engine) withInstallSuggestion(spec *specs.Spec, suggestions []specs.Suggestion) []specs.Suggestion {
	if !e.needsInstall(spec) {
		return suggestions
	}
	out := make([]specs.Suggestion, 0, len(suggestions)+1)
	out = append(out, e.installSuggestion(spec))
	out = append(out, suggestions...)
	return out
}

func (e *Engine) needsInstall(spec *specs.Spec) bool {
	if spec == nil || spec.Install == "" {
		return false
	}
	_, err := exec.LookPath(spec.Name)
	return err != nil
}

func (e *Engine) installSuggestion(spec *specs.Spec) specs.Suggestion {
	return specs.Suggestion{
		Name:        "Cài đặt " + spec.Name,
		Description: "Chưa cài đặt. Chọn để hiện lệnh cài đặt",
		Kind:        specs.KindInstall,
		InsertText:  spec.Install,
	}
}

// getSubcommandSuggestions returns subcommand suggestions
func (e *Engine) getSubcommandSuggestions(spec *specs.Spec, prefix string) []specs.Suggestion {
	var suggestions []specs.Suggestion
	prefix = strings.ToLower(prefix)

	for _, sub := range spec.Subcommands {
		if prefix == "" || strings.HasPrefix(strings.ToLower(sub.Name), prefix) {
			suggestions = append(suggestions, specs.Suggestion{
				Name:        sub.Name,
				Description: sub.Description,
				Kind:        specs.KindSubcommand,
			})
		}
	}

	return suggestions
}

// getFlagSuggestions returns flag suggestions for current command
func (e *Engine) getFlagSuggestions(ctx *buffer.Context) []specs.Suggestion {
	var suggestions []specs.Suggestion

	spec := specs.Get(ctx.Command)
	if spec == nil {
		return nil
	}

	prefix := strings.ToLower(ctx.Flag)

	// Global options for the command
	for _, opt := range spec.Options {
		for _, name := range opt.Names {
			if prefix == "" || strings.HasPrefix(name, prefix) {
				suggestions = append(suggestions, specs.Suggestion{
					Name:        name,
					Description: opt.Description,
					Kind:        specs.KindOption,
				})
			}
		}
	}

	// Subcommand options if we have a subcommand
	if ctx.Subcommand != "" {
		for _, sub := range spec.Subcommands {
			if sub.Name == ctx.Subcommand {
				for _, opt := range sub.Options {
					for _, name := range opt.Names {
						if prefix == "" || strings.HasPrefix(name, prefix) {
							suggestions = append(suggestions, specs.Suggestion{
								Name:        name,
								Description: opt.Description,
								Kind:        specs.KindOption,
							})
						}
					}
				}
			}
		}
	}

	return suggestions
}

// getArgSuggestions returns argument suggestions
func (e *Engine) getArgSuggestions(ctx *buffer.Context) []specs.Suggestion {
	spec := specs.Get(ctx.Command)
	if spec == nil {
		return nil
	}
	partial := ""
	if ctx.Level == buffer.ContextArgPartial {
		partial = ctx.PartialWord
	}

	// Prefer subcommand args if a subcommand is active
	if ctx.Subcommand != "" {
		for _, sub := range spec.Subcommands {
			if sub.Name == ctx.Subcommand {
				return e.suggestFromArgSpecs(selectArgSpecs(sub.Args, ctx.ArgIndex), partial)
			}
		}
	}

	return e.suggestFromArgSpecs(selectArgSpecs(spec.Args, ctx.ArgIndex), partial)
}

func selectArgSpecs(args []specs.ArgSpec, index int) []specs.ArgSpec {
	if len(args) == 0 {
		return nil
	}
	if index < len(args) {
		return []specs.ArgSpec{args[index]}
	}

	last := args[len(args)-1]
	if last.IsVariadic {
		return []specs.ArgSpec{last}
	}
	return nil
}

// suggestFromArgSpecs produces suggestions from a list of ArgSpec, filtered by partial.
func (e *Engine) suggestFromArgSpecs(args []specs.ArgSpec, partial string) []specs.Suggestion {
	var suggestions []specs.Suggestion
	for _, arg := range args {
		if arg.Generator != "" {
			if fn := specs.GetGenerator(arg.Generator); fn != nil {
				for _, s := range fn() {
					if partial == "" || strings.HasPrefix(s.Name, partial) {
						suggestions = append(suggestions, s)
					}
				}
			}
		}
		for _, source := range arg.Sources {
			suggestions = append(suggestions, e.suggestFromSource(source, partial)...)
		}
	}
	return suggestions
}

func (e *Engine) suggestFromSource(source specs.SourceSpec, partial string) []specs.Suggestion {
	fn := specs.GetSource(source.Type)
	if fn == nil {
		return nil
	}
	return fn(source, e.sourceContext(), partial)
}

func (e *Engine) effectiveCWD() string {
	if e.cwd != "" {
		return e.cwd
	}
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return cwd
}

func (e *Engine) currentLine() string {
	if e.currentBuf == nil {
		return ""
	}
	return e.currentBuf.String()
}

func (e *Engine) sourceContext() specs.SourceContext {
	ctx := specs.SourceContext{
		CWD:         e.effectiveCWD(),
		CommandLine: e.currentLine(),
	}
	if e.currentCtx == nil {
		return ctx
	}
	ctx.Command = e.currentCtx.Command
	ctx.Subcommand = e.currentCtx.Subcommand
	ctx.ArgIndex = e.currentCtx.ArgIndex
	ctx.IsSubcommandPosition = e.currentCtx.Level == buffer.ContextSubcommand || e.currentCtx.Level == buffer.ContextSubcommandPartial
	return ctx
}
