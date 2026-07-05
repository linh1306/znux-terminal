package specs

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const npmRegistrySearchURL = "https://registry.npmjs.org/-/v1/search"

var defaultNPMRegistryClient = &npmRegistryClient{
	baseURL:      npmRegistrySearchURL,
	httpClient:   &http.Client{Timeout: 2 * time.Second},
	debounce:     500 * time.Millisecond,
	cacheTTL:     5 * time.Minute,
	maxQuerySize: 10,
}

func init() {
	RegisterSource("npm-registry", func(source SourceSpec, ctx SourceContext, partial string) []Suggestion {
		return NPMRegistrySuggestions(partial)
	})
	RegisterSource("npm-dependencies", func(source SourceSpec, ctx SourceContext, partial string) []Suggestion {
		return NPMDependencySuggestions(ctx.CWD, ctx.CommandLine, partial)
	})
	RegisterSource("npm-scripts", func(source SourceSpec, ctx SourceContext, partial string) []Suggestion {
		return NPMScriptSuggestions(ctx.CWD, partial, ctx.IsSubcommandPosition)
	})
}

type npmPackageJSON struct {
	Scripts              map[string]string `json:"scripts"`
	Dependencies         map[string]string `json:"dependencies"`
	DevDependencies      map[string]string `json:"devDependencies"`
	OptionalDependencies map[string]string `json:"optionalDependencies"`
	PeerDependencies     map[string]string `json:"peerDependencies"`
}

// NPMScriptSuggestions returns scripts from package.json in cwd.
func NPMScriptSuggestions(cwd, partial string, insertRun bool) []Suggestion {
	pkg := readNPMPackage(cwd)
	if pkg == nil || len(pkg.Scripts) == 0 {
		return nil
	}

	names := make([]string, 0, len(pkg.Scripts))
	for name := range pkg.Scripts {
		names = append(names, name)
	}
	sort.Strings(names)

	partial = strings.ToLower(partial)
	suggestions := make([]Suggestion, 0, len(names))
	for _, name := range names {
		if partial != "" && !strings.HasPrefix(strings.ToLower(name), partial) {
			continue
		}
		suggestion := Suggestion{
			Name:        name,
			Description: pkg.Scripts[name],
			Kind:        KindSubcommand,
		}
		if insertRun {
			suggestion.InsertText = "run " + name
		}
		suggestions = append(suggestions, suggestion)
	}
	return suggestions
}

// NPMDependencySuggestions returns installed packages for npm uninstall.
func NPMDependencySuggestions(cwd, commandLine, partial string) []Suggestion {
	if npmUsesGlobalFlag(commandLine) {
		return npmGlobalDependencySuggestions(partial)
	}

	pkg := readNPMPackage(cwd)
	if pkg == nil {
		return nil
	}

	deps := map[string]string{}
	for name, version := range pkg.Dependencies {
		deps[name] = "dependencies " + version
	}
	for name, version := range pkg.DevDependencies {
		deps[name] = "devDependencies " + version
	}
	for name, version := range pkg.OptionalDependencies {
		deps[name] = "optionalDependencies " + version
	}
	for name, version := range pkg.PeerDependencies {
		deps[name] = "peerDependencies " + version
	}
	return npmDependencySuggestions(deps, partial)
}

// NPMRegistrySuggestions searches npm registry packages with debounce.
func NPMRegistrySuggestions(partial string) []Suggestion {
	return defaultNPMRegistryClient.search(partial)
}

func readNPMPackage(cwd string) *npmPackageJSON {
	if cwd == "" {
		return nil
	}

	data, err := os.ReadFile(filepath.Join(cwd, "package.json"))
	if err != nil {
		return nil
	}

	var pkg npmPackageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil
	}
	return &pkg
}

func npmUsesGlobalFlag(commandLine string) bool {
	for _, token := range strings.Fields(commandLine) {
		if token == "-g" || token == "--global" {
			return true
		}
	}
	return false
}

func npmGlobalDependencySuggestions(partial string) []Suggestion {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	out, err := exec.CommandContext(ctx, "npm", "list", "-g", "--depth=0", "--json").Output()
	if err != nil {
		return nil
	}

	var parsed struct {
		Dependencies map[string]struct {
			Version string `json:"version"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(out, &parsed); err != nil {
		return nil
	}

	deps := make(map[string]string, len(parsed.Dependencies))
	for name, dep := range parsed.Dependencies {
		desc := "global"
		if dep.Version != "" {
			desc += " " + dep.Version
		}
		deps[name] = desc
	}
	return npmDependencySuggestions(deps, partial)
}

func npmDependencySuggestions(deps map[string]string, partial string) []Suggestion {
	if len(deps) == 0 {
		return nil
	}

	names := make([]string, 0, len(deps))
	for name := range deps {
		names = append(names, name)
	}
	sort.Strings(names)

	suggestions := make([]Suggestion, 0, len(names))
	for _, name := range names {
		if partial != "" && !strings.HasPrefix(strings.ToLower(name), strings.ToLower(partial)) {
			continue
		}
		suggestions = append(suggestions, Suggestion{
			Name:        name,
			Description: deps[name],
			Kind:        KindValue,
		})
	}
	return suggestions
}

type npmRegistryClient struct {
	baseURL      string
	httpClient   *http.Client
	debounce     time.Duration
	cacheTTL     time.Duration
	maxQuerySize int

	mu        sync.Mutex
	lastQuery string
	lastSeen  time.Time
	cache     map[string]npmRegistryCacheEntry
}

type npmRegistryCacheEntry struct {
	suggestions []Suggestion
	expiry      time.Time
}

type npmRegistrySearchResponse struct {
	Objects []struct {
		Package struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Version     string `json:"version"`
		} `json:"package"`
	} `json:"objects"`
}

func (c *npmRegistryClient) search(partial string) []Suggestion {
	query := strings.TrimSpace(partial)
	if query == "" {
		return nil
	}

	now := time.Now()
	c.mu.Lock()
	if c.cache == nil {
		c.cache = map[string]npmRegistryCacheEntry{}
	}
	if entry, ok := c.cache[query]; ok && now.Before(entry.expiry) {
		out := append([]Suggestion(nil), entry.suggestions...)
		c.mu.Unlock()
		return out
	}
	if c.lastQuery != query {
		c.lastQuery = query
		c.lastSeen = now
		c.mu.Unlock()
		return nil
	}
	if now.Sub(c.lastSeen) < c.debounce {
		c.mu.Unlock()
		return nil
	}
	c.mu.Unlock()

	suggestions := c.fetch(query)

	c.mu.Lock()
	c.cache[query] = npmRegistryCacheEntry{
		suggestions: append([]Suggestion(nil), suggestions...),
		expiry:      time.Now().Add(c.cacheTTL),
	}
	c.mu.Unlock()

	return suggestions
}

func (c *npmRegistryClient) fetch(query string) []Suggestion {
	timeout := c.httpClient.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL, nil)
	if err != nil {
		return nil
	}
	q := req.URL.Query()
	q.Set("text", query)
	q.Set("size", strconv.Itoa(c.maxQuerySize))
	req.URL.RawQuery = q.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil
	}

	var parsed npmRegistrySearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil
	}

	suggestions := make([]Suggestion, 0, len(parsed.Objects))
	for _, object := range parsed.Objects {
		pkg := object.Package
		if pkg.Name == "" {
			continue
		}
		desc := pkg.Description
		if pkg.Version != "" {
			if desc != "" {
				desc += " "
			}
			desc += "v" + pkg.Version
		}
		suggestions = append(suggestions, Suggestion{
			Name:        pkg.Name,
			Description: desc,
			Kind:        KindValue,
		})
	}
	return suggestions
}

func ConfigureNPMRegistryClientForTest(baseURL string, httpClient *http.Client, debounce, cacheTTL time.Duration) func() {
	old := defaultNPMRegistryClient
	defaultNPMRegistryClient = &npmRegistryClient{
		baseURL:      baseURL,
		httpClient:   httpClient,
		debounce:     debounce,
		cacheTTL:     cacheTTL,
		maxQuerySize: 10,
	}
	return func() {
		defaultNPMRegistryClient = old
	}
}
