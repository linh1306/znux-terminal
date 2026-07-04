package specs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDirLoadsExternalYAMLSpecs(t *testing.T) {
	oldRegistry := registry
	registry = map[string]*Spec{}
	t.Cleanup(func() {
		registry = oldRegistry
	})

	dir := t.TempDir()
	specPath := filepath.Join(dir, "hello.yaml")
	if err := os.WriteFile(specPath, []byte(`
name: hello
subcommands:
  - name: world
    description: Say hello
`), 0o644); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	if loaded != 1 {
		t.Fatalf("loaded %d specs, want 1", loaded)
	}
	spec := Get("hello")
	if spec == nil {
		t.Fatal("missing hello spec")
	}
	if len(spec.Subcommands) != 1 || spec.Subcommands[0].Name != "world" {
		t.Fatalf("unexpected spec: %#v", spec)
	}
}

func TestLoadDirIgnoresMissingDirectory(t *testing.T) {
	loaded, err := LoadDir(filepath.Join(t.TempDir(), "missing"))
	if err != nil {
		t.Fatal(err)
	}
	if loaded != 0 {
		t.Fatalf("loaded %d specs, want 0", loaded)
	}
}

func TestLoadYAMLParsesArgumentSources(t *testing.T) {
	spec, err := LoadYAML([]byte(`
name: fuser
args:
  - name: target
    variadic: true
    sources:
      - type: path
        include: [folder, file]
      - type: port
        protocols: [tcp, udp]
        state: listening
        format: port/proto
`))
	if err != nil {
		t.Fatal(err)
	}

	if len(spec.Args) != 1 {
		t.Fatalf("args len = %d, want 1", len(spec.Args))
	}
	arg := spec.Args[0]
	if !arg.IsVariadic {
		t.Fatal("arg should be variadic")
	}
	if len(arg.Sources) != 2 {
		t.Fatalf("sources len = %d, want 2", len(arg.Sources))
	}
	if arg.Sources[0].Type != "path" || len(arg.Sources[0].Include) != 2 {
		t.Fatalf("unexpected path source: %#v", arg.Sources[0])
	}
	if arg.Sources[1].Type != "port" || arg.Sources[1].Format != "port/proto" {
		t.Fatalf("unexpected port source: %#v", arg.Sources[1])
	}
}

func TestLoadYAMLParsesInstallCommand(t *testing.T) {
	spec, err := LoadYAML([]byte(`
name: codex
install: npm install -g @openai/codex
subcommands:
  - name: login
    description: Login
`))
	if err != nil {
		t.Fatal(err)
	}

	if spec.Install != "npm install -g @openai/codex" {
		t.Fatalf("install = %q, want npm install -g @openai/codex", spec.Install)
	}
}

func TestLoadYAMLConvertsLegacyTemplateToPathSource(t *testing.T) {
	spec, err := LoadYAML([]byte(`
name: cd
args:
  - name: dir
    template: folder
`))
	if err != nil {
		t.Fatal(err)
	}

	source := spec.Args[0].Sources[0]
	if source.Type != "path" || len(source.Include) != 1 || source.Include[0] != "folder" {
		t.Fatalf("unexpected legacy source: %#v", source)
	}
}
