package suggest

import (
	"strings"

	"github.com/nguyenlinh13602/goshell/internal/buffer"
	"github.com/nguyenlinh13602/goshell/specs"
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
		if spec := specs.Get(ctx.Command); spec != nil {
			partial := ""
			if ctx.Level == buffer.ContextSubcommandPartial {
				partial = ctx.Subcommand
			}
			return e.getSubcommandSuggestions(spec, partial)
		}
		return nil

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

	// Commands that have specs
	registered := []string{"git", "docker", "kubectl", "npm", "go"}
	for _, cmd := range registered {
		if strings.HasPrefix(cmd, prefix) {
			suggestions = append(suggestions, specs.Suggestion{
				Name: cmd,
				Kind: specs.KindSubcommand,
			})
		}
	}

	// If no match, could add more commands here
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

	var suggestions []specs.Suggestion

	// Find the subcommand being typed
	var currentSubcommand *specs.Subcommand
	if ctx.Subcommand != "" {
		for _, sub := range spec.Subcommands {
			if sub.Name == ctx.Subcommand {
				currentSubcommand = &sub
				break
			}
		}
	}

	// If we have a subcommand with args that have generators
	if currentSubcommand != nil {
		for _, arg := range currentSubcommand.Args {
			if arg.Generator != nil {
				dynamic := arg.Generator()
				for _, s := range dynamic {
					suggestions = append(suggestions, s)
				}
			}
			if arg.Template == specs.TemplateFileSystem || arg.Template == specs.TemplateFolder {
				// File system suggestions would be handled by shell
				// For now, return nil to let shell handle it
			}
		}
	}

	// Also check top-level args
	for _, arg := range spec.Args {
		if arg.Generator != nil {
			dynamic := arg.Generator()
			for _, s := range dynamic {
				suggestions = append(suggestions, s)
			}
		}
	}

	return suggestions
}
