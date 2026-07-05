package specs

import (
	"bytes"
	"context"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

func init() {
	RegisterSource("port", func(source SourceSpec, ctx SourceContext, partial string) []Suggestion {
		return portSuggestions(partial, source)
	})
}

func portSuggestions(partial string, source SourceSpec) []Suggestion {
	ports := listListeningPorts(source.Protocols)
	if len(ports) == 0 {
		return nil
	}
	sortListeningPorts(ports, source.Protocols)

	format := source.Format
	if format == "" {
		format = "port/proto"
	}

	suggestions := make([]Suggestion, 0, len(ports))
	for _, port := range ports {
		name := formatPortSuggestion(port, format)
		if partial != "" && !strings.HasPrefix(name, partial) {
			continue
		}
		suggestions = append(suggestions, Suggestion{
			Name:        name,
			Kind:        KindValue,
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

	out, err := exec.CommandContext(ctx, "ss", "-H", "-lntup").Output()
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
