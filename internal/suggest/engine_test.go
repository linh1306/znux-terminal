package suggest

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nguyenlinh13602/goshell/internal/buffer"
	"github.com/nguyenlinh13602/goshell/internal/suggest/specs"
)

func TestGetSuggestionsExpandsHomeDirectoryForFolders(t *testing.T) {
	loadTestSpecs(t)

	home := t.TempDir()
	t.Setenv("HOME", home)

	mustMkdir(t, filepath.Join(home, "projects"))
	mustWriteFile(t, filepath.Join(home, "plain.txt"))

	buf := buffer.NewLineBuf()
	buf.SetString("cd ~/")
	ctx := buffer.NewParser().GetCurrentContext(buf)

	got := NewEngine().GetSuggestions(buf, &ctx)

	assertHasSuggestion(t, got, "~/projects/", specs.KindFolder)
	assertNoSuggestion(t, got, "~/plain.txt")
}

func TestGetSuggestionsExpandsBareHomeDirectoryForFolders(t *testing.T) {
	loadTestSpecs(t)

	home := t.TempDir()
	t.Setenv("HOME", home)

	mustMkdir(t, filepath.Join(home, "projects"))

	buf := buffer.NewLineBuf()
	buf.SetString("cd ~")
	ctx := buffer.NewParser().GetCurrentContext(buf)

	got := NewEngine().GetSuggestions(buf, &ctx)

	assertHasSuggestion(t, got, "~/projects/", specs.KindFolder)
}

func TestGetSuggestionsFiltersPathSourcesByInclude(t *testing.T) {
	specs.Register("paths-only", &specs.Spec{
		Name: "paths-only",
		Args: []specs.ArgSpec{{
			Name:    "target",
			Sources: []specs.SourceSpec{{Type: "path", Include: []string{"file"}}},
		}},
	})

	dir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Fatal(err)
		}
	})

	mustMkdir(t, filepath.Join(dir, "folder"))
	mustWriteFile(t, filepath.Join(dir, "file.txt"))

	buf := buffer.NewLineBuf()
	buf.SetString("paths-only ")
	ctx := buffer.NewParser().GetCurrentContext(buf)

	got := NewEngine().GetSuggestions(buf, &ctx)

	assertHasSuggestion(t, got, "file.txt", specs.KindFile)
	assertNoSuggestion(t, got, "folder/")
	for _, s := range got {
		if s.Name == "file.txt" && s.Description != "file" {
			t.Fatalf("file description = %q, want file", s.Description)
		}
	}
}

func TestGetSuggestionsOrdersPathSourcesByInclude(t *testing.T) {
	specs.Register("ordered-path", &specs.Spec{
		Name: "ordered-path",
		Args: []specs.ArgSpec{{
			Name:    "target",
			Sources: []specs.SourceSpec{{Type: "path", Include: []string{"folder", "file"}}},
		}},
	})

	dir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Fatal(err)
		}
	})

	mustWriteFile(t, filepath.Join(dir, "aaa.txt"))
	mustMkdir(t, filepath.Join(dir, "zzz"))

	buf := buffer.NewLineBuf()
	buf.SetString("ordered-path ")
	ctx := buffer.NewParser().GetCurrentContext(buf)

	got := NewEngine().GetSuggestions(buf, &ctx)

	if len(got) < 2 {
		t.Fatalf("suggestions len = %d, want at least 2: %#v", len(got), got)
	}
	if got[0].Kind != specs.KindFolder || got[0].Name != "zzz/" {
		t.Fatalf("first suggestion = %#v, want folder zzz/", got[0])
	}
}

func TestGetSuggestionsFiltersArgPartial(t *testing.T) {
	specs.Register("partial-path", &specs.Spec{
		Name: "partial-path",
		Args: []specs.ArgSpec{{
			Name:    "target",
			Sources: []specs.SourceSpec{{Type: "path", Include: []string{"file", "folder"}}},
		}},
	})

	dir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Fatal(err)
		}
	})

	mustWriteFile(t, filepath.Join(dir, "alpha.txt"))
	mustWriteFile(t, filepath.Join(dir, "beta.txt"))

	buf := buffer.NewLineBuf()
	buf.SetString("partial-path al")
	ctx := buffer.NewParser().GetCurrentContext(buf)

	got := NewEngine().GetSuggestions(buf, &ctx)

	assertHasSuggestion(t, got, "alpha.txt", specs.KindFile)
	assertNoSuggestion(t, got, "beta.txt")
}

func TestGetSuggestionsUsesCurrentArgOnly(t *testing.T) {
	specs.RegisterGenerator("test:first-arg", func() []specs.Suggestion {
		return []specs.Suggestion{{Name: "origin", Kind: specs.KindValue}}
	})
	specs.RegisterGenerator("test:second-arg", func() []specs.Suggestion {
		return []specs.Suggestion{
			{Name: "main", Kind: specs.KindValue},
			{Name: "origin/main", Kind: specs.KindValue},
		}
	})
	specs.Register("split-command", &specs.Spec{
		Name: "split-command",
		Subcommands: []specs.Subcommand{{
			Name: "push",
			Args: []specs.ArgSpec{
				{Name: "remote", Generator: "test:first-arg"},
				{Name: "branch", Generator: "test:second-arg"},
			},
		}},
	})

	buf := buffer.NewLineBuf()
	buf.SetString("split-command push ")
	ctx := buffer.NewParser().GetCurrentContext(buf)

	got := NewEngine().GetSuggestions(buf, &ctx)

	assertHasSuggestion(t, got, "origin", specs.KindValue)
	assertNoSuggestion(t, got, "main")
	assertNoSuggestion(t, got, "origin/main")

	buf.SetString("split-command push origin ")
	ctx = buffer.NewParser().GetCurrentContext(buf)

	got = NewEngine().GetSuggestions(buf, &ctx)

	assertHasSuggestion(t, got, "main", specs.KindValue)
	assertHasSuggestion(t, got, "origin/main", specs.KindValue)
	assertNoSuggestion(t, got, "origin")
}

func TestGetSuggestionsUsesRegisteredSource(t *testing.T) {
	specs.RegisterSource("test:dynamic-source", func(source specs.SourceSpec, ctx specs.SourceContext, partial string) []specs.Suggestion {
		if ctx.CWD == "" {
			t.Fatal("source context should include cwd")
		}
		return []specs.Suggestion{{Name: "dynamic-value", Kind: specs.KindValue}}
	})
	specs.Register("source-command", &specs.Spec{
		Name: "source-command",
		Args: []specs.ArgSpec{{
			Name:    "value",
			Sources: []specs.SourceSpec{{Type: "test:dynamic-source"}},
		}},
	})

	buf := buffer.NewLineBuf()
	buf.SetString("source-command dyn")
	ctx := buffer.NewParser().GetCurrentContext(buf)

	got := NewEngine().GetSuggestions(buf, &ctx)

	assertHasSuggestion(t, got, "dynamic-value", specs.KindValue)
}

func TestGetSuggestionsPrependsInstallForMissingCommand(t *testing.T) {
	specs.Register("znux-missing-test-command", &specs.Spec{
		Name:    "znux-missing-test-command",
		Install: "sudo apt install znux-missing-test-command",
		Subcommands: []specs.Subcommand{{
			Name:        "run",
			Description: "Run test command",
		}},
	})

	buf := buffer.NewLineBuf()
	buf.SetString("znux-missing")
	ctx := buffer.NewParser().GetCurrentContext(buf)

	got := NewEngine().GetSuggestions(buf, &ctx)

	if len(got) < 2 {
		t.Fatalf("suggestions len = %d, want at least 2: %#v", len(got), got)
	}
	if got[0].Kind != specs.KindInstall {
		t.Fatalf("first suggestion = %#v, want install suggestion", got[0])
	}
	if got[0].Completion() != "sudo apt install znux-missing-test-command" {
		t.Fatalf("install completion = %q", got[0].Completion())
	}
}

func TestGetSuggestionsAddsNPMScriptsFromPackageJSON(t *testing.T) {
	loadTestSpecs(t)

	dir := t.TempDir()
	mustWriteFileContent(t, filepath.Join(dir, "package.json"), `{
		"scripts": {
			"dev": "vite --host",
			"build": "vite build"
		}
	}`)

	engine := NewEngine()
	engine.SetCWD(dir)

	buf := buffer.NewLineBuf()
	buf.SetString("npm ")
	ctx := buffer.NewParser().GetCurrentContext(buf)

	got := engine.GetSuggestions(buf, &ctx)

	assertHasSuggestion(t, got, "dev", specs.KindSubcommand)
	assertHasSuggestion(t, got, "build", specs.KindSubcommand)
	assertSuggestionCompletion(t, got, "dev", "run dev")

	buf.SetString("npm run d")
	ctx = buffer.NewParser().GetCurrentContext(buf)

	got = engine.GetSuggestions(buf, &ctx)

	assertHasSuggestion(t, got, "dev", specs.KindSubcommand)
	assertNoSuggestion(t, got, "build")
	assertSuggestionCompletion(t, got, "dev", "dev")
}

func TestGetSuggestionsSuggestsNPMDependenciesForUninstall(t *testing.T) {
	loadTestSpecs(t)

	dir := t.TempDir()
	mustWriteFileContent(t, filepath.Join(dir, "package.json"), `{
		"dependencies": {
			"react": "^19.0.0"
		},
		"devDependencies": {
			"vite": "^7.0.0"
		}
	}`)

	engine := NewEngine()
	engine.SetCWD(dir)

	buf := buffer.NewLineBuf()
	buf.SetString("npm uninstall v")
	ctx := buffer.NewParser().GetCurrentContext(buf)

	got := engine.GetSuggestions(buf, &ctx)

	assertHasSuggestion(t, got, "vite", specs.KindValue)
	assertNoSuggestion(t, got, "react")
}

func TestNPMRegistryClientDebouncesSearch(t *testing.T) {
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		if got := r.URL.Query().Get("text"); got != "react" {
			t.Fatalf("query text = %q, want react", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"objects":[{"package":{"name":"react","description":"React","version":"19.1.0"}}]}`))
	}))
	defer server.Close()

	restore := specs.ConfigureNPMRegistryClientForTest(server.URL, server.Client(), 50*time.Millisecond, time.Minute)
	defer restore()

	if got := specs.NPMRegistrySuggestions("react"); len(got) != 0 {
		t.Fatalf("first debounced search = %#v, want none", got)
	}
	if requests != 0 {
		t.Fatalf("requests = %d, want 0 before debounce", requests)
	}

	time.Sleep(60 * time.Millisecond)
	got := specs.NPMRegistrySuggestions("react")

	assertHasSuggestion(t, got, "react", specs.KindValue)
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
}

func loadTestSpecs(t *testing.T) {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to locate test file")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	loaded, err := specs.LoadDir(filepath.Join(repoRoot, "suggest"))
	if err != nil {
		t.Fatal(err)
	}
	if loaded == 0 {
		t.Fatal("no suggestion specs loaded")
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWriteFile(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustWriteFileContent(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertHasSuggestion(t *testing.T, suggestions []specs.Suggestion, name string, kind specs.SuggestionKind) {
	t.Helper()
	for _, s := range suggestions {
		if s.Name == name && s.Kind == kind {
			return
		}
	}
	t.Fatalf("missing suggestion %q (%v) in %#v", name, kind, suggestions)
}

func assertNoSuggestion(t *testing.T, suggestions []specs.Suggestion, name string) {
	t.Helper()
	for _, s := range suggestions {
		if s.Name == name {
			t.Fatalf("unexpected suggestion %q in %#v", name, suggestions)
		}
	}
}

func assertSuggestionCompletion(t *testing.T, suggestions []specs.Suggestion, name, completion string) {
	t.Helper()
	for _, s := range suggestions {
		if s.Name == name {
			if got := s.Completion(); got != completion {
				t.Fatalf("completion for %q = %q, want %q", name, got, completion)
			}
			return
		}
	}
	t.Fatalf("missing suggestion %q in %#v", name, suggestions)
}
