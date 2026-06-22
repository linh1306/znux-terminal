package suggest

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

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

func TestParseSSListeningPorts(t *testing.T) {
	out := []byte(`tcp LISTEN 0 4096 127.0.0.1:5432 0.0.0.0:* users:(("postgres",pid=1234,fd=7))
udp UNCONN 0 0 0.0.0.0:5353 0.0.0.0:* users:(("mdns",pid=55,fd=4))
tcp LISTEN 0 4096 [::1]:8080 [::]:* users:(("node",pid=999,fd=21))
tcp LISTEN 0 4096 127.0.0.53%lo:53 0.0.0.0:* users:(("systemd-resolve",pid=777,fd=17))
tcp LISTEN 0 4096 127.0.0.1:3000 0.0.0.0:* users:(("next-server (v1",pid=484781,fd=21))
`)

	got := parseSSListeningPorts(out, []string{"tcp", "udp"})

	if len(got) != 5 {
		t.Fatalf("ports len = %d, want 5: %#v", len(got), got)
	}
	if got[0].Port != "5432" || got[0].Protocol != "tcp" {
		t.Fatalf("unexpected first port: %#v", got[0])
	}
	if got[0].Description() != "LISTEN 127.0.0.1 postgres pid=1234" {
		t.Fatalf("first description = %q", got[0].Description())
	}
	if got[1].Port != "5353" || got[1].Protocol != "udp" {
		t.Fatalf("unexpected second port: %#v", got[1])
	}
	if got[2].Port != "8080" || got[2].Protocol != "tcp" {
		t.Fatalf("unexpected third port: %#v", got[2])
	}
	if got[2].Description() != "LISTEN ::1 node pid=999" {
		t.Fatalf("third description = %q", got[2].Description())
	}
	if got[3].Description() != "LISTEN 127.0.0.53 systemd-resolve pid=777" {
		t.Fatalf("fourth description = %q", got[3].Description())
	}
	if got[4].Description() != "LISTEN 127.0.0.1 next-server pid=484781" {
		t.Fatalf("fifth description = %q", got[4].Description())
	}
}

func TestSortListeningPortsUsesProtocolOrder(t *testing.T) {
	ports := []listeningPort{
		{Protocol: "udp", Port: "53"},
		{Protocol: "tcp", Port: "3000"},
		{Protocol: "tcp", Port: "22"},
	}

	sortListeningPorts(ports, []string{"tcp", "udp"})

	if ports[0].Protocol != "tcp" || ports[0].Port != "22" {
		t.Fatalf("first port = %#v, want tcp 22", ports[0])
	}
	if ports[1].Protocol != "tcp" || ports[1].Port != "3000" {
		t.Fatalf("second port = %#v, want tcp 3000", ports[1])
	}
	if ports[2].Protocol != "udp" || ports[2].Port != "53" {
		t.Fatalf("third port = %#v, want udp 53", ports[2])
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
