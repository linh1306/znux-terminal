package buffer

import "testing"

func TestParserUsesLastPipelineCommand(t *testing.T) {
	ctx := contextFor("echo hi | git che")

	if ctx.Level != ContextSubcommandPartial {
		t.Fatalf("level = %v, want %v", ctx.Level, ContextSubcommandPartial)
	}
	if ctx.Command != "git" {
		t.Fatalf("command = %q, want git", ctx.Command)
	}
	if ctx.Subcommand != "che" {
		t.Fatalf("subcommand = %q, want che", ctx.Subcommand)
	}
}

func TestParserIgnoresLeadingAssignments(t *testing.T) {
	ctx := contextFor("GIT_DIR=.git git checkout ")

	if ctx.Level != ContextArg {
		t.Fatalf("level = %v, want %v", ctx.Level, ContextArg)
	}
	if ctx.Command != "git" {
		t.Fatalf("command = %q, want git", ctx.Command)
	}
	if ctx.Subcommand != "checkout" {
		t.Fatalf("subcommand = %q, want checkout", ctx.Subcommand)
	}
}

func TestParserFallsBackForIncompleteQuote(t *testing.T) {
	ctx := contextFor(`git commit -m "hello`)

	if ctx.Level != ContextArgPartial {
		t.Fatalf("level = %v, want %v", ctx.Level, ContextArgPartial)
	}
	if ctx.Command != "git" {
		t.Fatalf("command = %q, want git", ctx.Command)
	}
	if ctx.Subcommand != "commit" {
		t.Fatalf("subcommand = %q, want commit", ctx.Subcommand)
	}
}

func contextFor(input string) Context {
	buf := NewLineBuf()
	buf.SetString(input)
	return NewParser().GetCurrentContext(buf)
}
