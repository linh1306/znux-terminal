package specs

import (
	"strings"
	"sync"
	"time"
)

func init() {
	RegisterGenerator("git:branches", gitBranches())
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

		out, err := commandOutput(generatorTimeout, "git", "branch", "-a", "--no-color", "--format=%(refname:short)")
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
