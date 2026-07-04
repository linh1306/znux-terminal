package suggest

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/nguyenlinh13602/goshell/internal/buffer"
	"github.com/nguyenlinh13602/goshell/internal/suggest/specs"
)

// Engine provides suggestions based on current input
type Engine struct{}

type sourceFunc func(e *Engine, source specs.SourceSpec, partial string) []specs.Suggestion

var sourceRegistry = map[string]sourceFunc{
	"path": func(e *Engine, source specs.SourceSpec, partial string) []specs.Suggestion {
		return e.getPathSuggestions(partial, source.Include)
	},
	"port": func(e *Engine, source specs.SourceSpec, partial string) []specs.Suggestion {
		return e.getPortSuggestions(partial, source)
	},
}

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
			if subs := e.getSubcommandSuggestions(spec, partial); len(subs) > 0 {
				return e.withInstallSuggestion(spec, subs)
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
	fn := sourceRegistry[source.Type]
	if fn == nil {
		return nil
	}
	return fn(e, source, partial)
}

func (e *Engine) getPathSuggestions(partial string, include []string) []specs.Suggestion {
	includeFiles, includeFolders := true, true
	priority := map[specs.SuggestionKind]int{
		specs.KindFolder: 0,
		specs.KindFile:   1,
	}
	if len(include) > 0 {
		includeFiles = false
		includeFolders = false
		priority = map[specs.SuggestionKind]int{}
		for i, item := range include {
			switch item {
			case "file":
				includeFiles = true
				priority[specs.KindFile] = i
			case "folder", "dir", "directory":
				includeFolders = true
				priority[specs.KindFolder] = i
			}
		}
	}

	suggestions := e.getFileSuggestions(partial, includeFiles, includeFolders)
	sort.SliceStable(suggestions, func(i, j int) bool {
		return priority[suggestions[i].Kind] < priority[suggestions[j].Kind]
	})
	return suggestions
}

// getFileSuggestions lists files/directories matching the partial path.
func (e *Engine) getFileSuggestions(partial string, includeFiles, includeFolders bool) []specs.Suggestion {
	// Split partial into directory and filename prefix
	var dir, prefix string
	lastSlash := strings.LastIndex(partial, "/")
	if partial == "~" {
		dir = "~/"
		prefix = ""
	} else if lastSlash < 0 {
		dir = "."
		prefix = partial
	} else {
		dir = partial[:lastSlash+1]
		prefix = partial[lastSlash+1:]
	}

	readDir := dir
	if strings.HasPrefix(dir, "~/") || dir == "~" {
		home, err := os.UserHomeDir()
		if err != nil || home == "" {
			return nil
		}
		readDir = home
		if len(dir) > 2 {
			readDir = filepath.Join(home, dir[2:])
		}
	}

	entries, err := os.ReadDir(readDir)
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
		if entry.IsDir() && !includeFolders {
			continue
		}
		if !entry.IsDir() && !includeFiles {
			continue
		}

		// Build the full suggestion path preserving the original dir prefix
		var full string
		if dir == "." {
			full = name
		} else {
			full = dir + name
		}
		if entry.IsDir() {
			full += "/"
		}

		kind := specs.KindFile
		desc := "file"
		if entry.IsDir() {
			kind = specs.KindFolder
			desc = "folder"
		}
		suggestions = append(suggestions, specs.Suggestion{Name: full, Kind: kind, Description: desc})
	}
	return suggestions
}

func (e *Engine) getPortSuggestions(partial string, source specs.SourceSpec) []specs.Suggestion {
	ports := listListeningPorts(source.Protocols)
	if len(ports) == 0 {
		return nil
	}
	sortListeningPorts(ports, source.Protocols)

	format := source.Format
	if format == "" {
		format = "port/proto"
	}

	suggestions := make([]specs.Suggestion, 0, len(ports))
	for _, port := range ports {
		name := formatPortSuggestion(port, format)
		if partial != "" && !strings.HasPrefix(name, partial) {
			continue
		}
		suggestions = append(suggestions, specs.Suggestion{
			Name:        name,
			Kind:        specs.KindValue,
			Description: port.Description(),
		})
	}
	return suggestions
}

func sortListeningPorts(ports []listeningPort, protocols []string) {
	priority := map[string]int{}
	for i, protocol := range protocols {
		priority[strings.ToLower(protocol)] = i
	}

	sort.SliceStable(ports, func(i, j int) bool {
		left := protocolPriority(ports[i].Protocol, priority)
		right := protocolPriority(ports[j].Protocol, priority)
		if left != right {
			return left < right
		}
		leftPort, leftErr := strconv.Atoi(ports[i].Port)
		rightPort, rightErr := strconv.Atoi(ports[j].Port)
		if leftErr == nil && rightErr == nil && leftPort != rightPort {
			return leftPort < rightPort
		}
		return ports[i].Port < ports[j].Port
	})
}

func protocolPriority(protocol string, priority map[string]int) int {
	if value, ok := priority[strings.ToLower(protocol)]; ok {
		return value
	}
	return len(priority)
}

type listeningPort struct {
	Protocol string
	State    string
	Port     string
	Address  string
	Process  string
}

func (p listeningPort) Description() string {
	state := p.State
	if state == "" {
		state = "LISTEN"
	}

	parts := []string{state}
	if p.Address != "" {
		parts = append(parts, p.Address)
	}
	if p.Process != "" {
		parts = append(parts, p.Process)
	}
	return strings.Join(parts, " ")
}

func formatPortSuggestion(port listeningPort, format string) string {
	switch format {
	case "port":
		return port.Port
	case "proto/port":
		return port.Protocol + "/" + port.Port
	default:
		return port.Port + "/" + port.Protocol
	}
}

func listListeningPorts(protocols []string) []listeningPort {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	args := []string{"-H", "-lntup"}
	out, err := exec.CommandContext(ctx, "ss", args...).Output()
	if err != nil {
		return nil
	}
	return parseSSListeningPorts(out, protocols)
}

func parseSSListeningPorts(out []byte, protocols []string) []listeningPort {
	allowed := map[string]bool{}
	for _, proto := range protocols {
		allowed[strings.ToLower(proto)] = true
	}

	seen := map[string]bool{}
	var ports []listeningPort
	for _, line := range bytes.Split(out, []byte{'\n'}) {
		fields := strings.Fields(string(line))
		if len(fields) < 5 {
			continue
		}
		proto := strings.ToLower(fields[0])
		if len(allowed) > 0 && !allowed[proto] {
			continue
		}
		state := fields[1]
		localAddress := fields[4]
		port := extractPort(localAddress)
		if port == "" {
			continue
		}
		address := extractHost(localAddress)
		process := extractSSProcess(fields[5:])
		key := proto + "/" + port + "/" + address + "/" + process
		if seen[key] {
			continue
		}
		seen[key] = true
		ports = append(ports, listeningPort{
			Protocol: proto,
			State:    state,
			Port:     port,
			Address:  address,
			Process:  process,
		})
	}
	return ports
}

func extractPort(address string) string {
	if i := strings.LastIndex(address, ":"); i >= 0 && i < len(address)-1 {
		port := strings.Trim(address[i+1:], "[]")
		if _, err := strconv.Atoi(port); err == nil {
			return port
		}
	}
	return ""
}

func extractHost(address string) string {
	if i := strings.LastIndex(address, ":"); i >= 0 {
		host := strings.Trim(address[:i], "[]")
		if zone := strings.LastIndex(host, "%"); zone >= 0 {
			host = host[:zone]
		}
		if host == "" || host == "*" {
			return "*"
		}
		return host
	}
	host := strings.Trim(address, "[]")
	if zone := strings.LastIndex(host, "%"); zone >= 0 {
		host = host[:zone]
	}
	return host
}

func extractSSProcess(fields []string) string {
	raw := strings.Join(fields, " ")
	start := strings.Index(raw, `users:((`)
	if start < 0 {
		return ""
	}

	raw = raw[start+len(`users:((`):]
	end := strings.Index(raw, "))")
	if end >= 0 {
		raw = raw[:end]
	}

	name := extractQuotedSSProcessName(raw)
	if name == "" {
		return ""
	}

	pid := ""
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "pid=") {
			pid = strings.TrimPrefix(part, "pid=")
			break
		}
	}
	if pid == "" {
		return name
	}
	return name + " pid=" + pid
}

func extractQuotedSSProcessName(raw string) string {
	start := strings.Index(raw, `"`)
	if start < 0 {
		return ""
	}
	rest := raw[start+1:]
	end := strings.Index(rest, `"`)
	if end < 0 {
		return cleanSSProcessName(rest)
	}
	return cleanSSProcessName(rest[:end])
}

func cleanSSProcessName(name string) string {
	name = strings.TrimSpace(name)
	if i := strings.Index(name, " ("); i >= 0 {
		name = name[:i]
	}
	return strings.TrimSpace(name)
}
