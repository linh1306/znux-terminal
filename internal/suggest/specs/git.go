package specs

import (
	"bytes"
	"strings"
	"sync"
	"time"
)

func init() {
	RegisterGenerator("git:branches", gitBranches())
	RegisterGenerator("git:local-branches", gitLocalBranches())
	RegisterGenerator("git:remotes", gitRemotes())
}

func gitBranches() func() []Suggestion {
	type cache struct {
		suggestions []Suggestion
		expiry      time.Time
		mu          sync.Mutex
	}
	c := cache{}

	return func() []Suggestion {
		c.mu.Lock()
		defer c.mu.Unlock()

		if time.Now().Before(c.expiry) && c.suggestions != nil {
			return c.suggestions
		}

		out, err := commandOutput(generatorTimeout, "git", "for-each-ref", "--format=%(refname:short)%00%(symref)", "refs/heads", "refs/remotes")
		if err != nil {
			return nil
		}

		lines := bytes.Split(bytes.TrimSpace(out), []byte("\n"))
		suggestions := make([]Suggestion, 0, len(lines))
		seen := make(map[string]struct{}, len(lines))
		for _, line := range lines {
			line = bytes.TrimSpace(line)
			if len(line) == 0 {
				continue
			}

			name, symref, ok := bytes.Cut(line, []byte{0})
			if !ok || len(symref) > 0 {
				continue
			}

			branch := strings.TrimSpace(string(name))
			if branch == "" {
				continue
			}
			if _, exists := seen[branch]; exists {
				continue
			}
			seen[branch] = struct{}{}
			suggestions = append(suggestions, Suggestion{Name: branch, Kind: KindValue})
		}

		c.suggestions = suggestions
		c.expiry = time.Now().Add(5 * time.Second)
		return suggestions
	}
}

func gitLocalBranches() func() []Suggestion {
	type cache struct {
		suggestions []Suggestion
		expiry      time.Time
		mu          sync.Mutex
	}
	c := cache{}

	return func() []Suggestion {
		c.mu.Lock()
		defer c.mu.Unlock()

		if time.Now().Before(c.expiry) && c.suggestions != nil {
			return c.suggestions
		}

		out, err := commandOutput(generatorTimeout, "git", "for-each-ref", "--format=%(refname:short)", "refs/heads")
		if err != nil {
			return nil
		}

		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		suggestions := make([]Suggestion, 0, len(lines))
		seen := make(map[string]struct{}, len(lines))
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if _, exists := seen[line]; exists {
				continue
			}
			seen[line] = struct{}{}
			suggestions = append(suggestions, Suggestion{Name: line, Kind: KindValue})
		}

		c.suggestions = suggestions
		c.expiry = time.Now().Add(5 * time.Second)
		return suggestions
	}
}

func gitRemotes() func() []Suggestion {
	type cache struct {
		suggestions []Suggestion
		expiry      time.Time
		mu          sync.Mutex
	}
	c := cache{}

	return func() []Suggestion {
		c.mu.Lock()
		defer c.mu.Unlock()

		if time.Now().Before(c.expiry) && c.suggestions != nil {
			return c.suggestions
		}

		out, err := commandOutput(generatorTimeout, "git", "remote")
		if err != nil {
			return nil
		}

		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		suggestions := make([]Suggestion, 0, len(lines))
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				suggestions = append(suggestions, Suggestion{Name: line, Kind: KindValue})
			}
		}

		c.suggestions = suggestions
		c.expiry = time.Now().Add(5 * time.Second)
		return suggestions
	}
}
