package suggest

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/nguyenlinh13602/goshell/internal/buffer"
	"github.com/nguyenlinh13602/goshell/internal/suggest/specs"
)

// Engine provides suggestions based on current input
type Engine struct{}

// NewEngine creates a new suggestion engine
func NewEngine() *Engine {
	return &Engine{}
}

// GetSuggestions returns suggestions for the current input
func (e *Engine) GetSuggestions(buf *buffer.LineBuf, ctx *buffer.Context) []specs.Suggestion {
	switch ctx.Level {
	case buffer.ContextCommandPartial:
		return e.matchCommand(buf.String())

	case buffer.ContextCommand:
		// After pressing space, check for subcommand
		if spec := specs.Get(ctx.Command); spec != nil {
			return e.getSubcommandSuggestions(spec, "")
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
			if subs := e.getSubcommandSuggestions(spec, partial); len(subs) > 0 {
				return subs
			}
		}
		// No matching subcommands — the partial token is an arg (e.g. "cd ./").
		return e.suggestFromArgSpecs(spec.Args, partial)

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
			suggestions = append(suggestions, specs.Suggestion{
				Name: cmd,
				Kind: specs.KindSubcommand,
			})
		}
	}

	return suggestions
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

	// Prefer subcommand args if a subcommand is active
	if ctx.Subcommand != "" {
		for _, sub := range spec.Subcommands {
			if sub.Name == ctx.Subcommand {
				return e.suggestFromArgSpecs(sub.Args, "")
			}
		}
	}

	return e.suggestFromArgSpecs(spec.Args, "")
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
		switch arg.Template {
		case specs.TemplateFileSystem:
			suggestions = append(suggestions, e.getFileSuggestions(partial, false)...)
		case specs.TemplateFolder:
			suggestions = append(suggestions, e.getFileSuggestions(partial, true)...)
		}
	}
	return suggestions
}

// getFileSuggestions lists files/directories matching the partial path.
func (e *Engine) getFileSuggestions(partial string, onlyDirs bool) []specs.Suggestion {
	// Split partial into directory and filename prefix
	var dir, prefix string
	lastSlash := strings.LastIndex(partial, "/")
	if lastSlash < 0 {
		dir = "."
		prefix = partial
	} else {
		dir = partial[:lastSlash+1]
		prefix = partial[lastSlash+1:]
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var suggestions []specs.Suggestion
	for _, entry := range entries {
		name := entry.Name()
		// Hide dotfiles unless user explicitly types "."
		if strings.HasPrefix(name, ".") && !strings.HasPrefix(prefix, ".") {
			continue
		}
		if prefix != "" && !strings.HasPrefix(name, prefix) {
			continue
		}
		if onlyDirs && !entry.IsDir() {
			continue
		}

		// Build the full suggestion path preserving the original dir prefix
		var full string
		if dir == "." {
			full = name
		} else {
			full = filepath.Join(dir, name)
			// filepath.Join strips trailing slash — restore it for dirs
		}
		if entry.IsDir() {
			full += "/"
		}

		kind := specs.KindFile
		if entry.IsDir() {
			kind = specs.KindFolder
		}
		suggestions = append(suggestions, specs.Suggestion{Name: full, Kind: kind, Description: entry.Type().String()})
	}
	return suggestions
}
