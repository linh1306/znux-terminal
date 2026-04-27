package specs

import (
	"os/exec"
	"strings"
	"sync"
	"time"
)

func init() {
	RegisterGenerator("docker:images", dockerImages())
	RegisterGenerator("docker:running-containers", dockerRunningContainers())
	RegisterGenerator("docker:all-containers", dockerAllContainers())
	RegisterGenerator("docker:networks", dockerNetworks())
	RegisterGenerator("docker:volumes", dockerVolumes())
}

func dockerRunningContainers() func() []Suggestion {
	return dockerContainerGenerator(false)
}

func dockerAllContainers() func() []Suggestion {
	return dockerContainerGenerator(true)
}

func dockerContainerGenerator(all bool) func() []Suggestion {
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

		args := []string{"ps", "--format", "{{.Names}}"}
		if all {
			args = []string{"ps", "-a", "--format", "{{.Names}}"}
		}
		out, err := exec.Command("docker", args...).Output()
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

func dockerImages() func() []Suggestion {
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

		out, err := exec.Command("docker", "images", "--format", "{{.Repository}}:{{.Tag}}").Output()
		if err != nil {
			return nil
		}

		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		suggestions := make([]Suggestion, 0, len(lines))
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && line != "<none>:<none>" {
				suggestions = append(suggestions, Suggestion{Name: line, Kind: KindValue})
			}
		}

		c.suggestions = suggestions
		c.expiry = time.Now().Add(10 * time.Second)
		return suggestions
	}
}

func dockerNetworks() func() []Suggestion {
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

		out, err := exec.Command("docker", "network", "ls", "--format", "{{.Name}}").Output()
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
		c.expiry = time.Now().Add(10 * time.Second)
		return suggestions
	}
}

func dockerVolumes() func() []Suggestion {
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

		out, err := exec.Command("docker", "volume", "ls", "--format", "{{.Name}}").Output()
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
		c.expiry = time.Now().Add(10 * time.Second)
		return suggestions
	}
}
