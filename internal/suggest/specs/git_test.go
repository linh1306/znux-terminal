package specs

import "testing"

func TestGitBranchesExcludeRemoteHeadSymrefs(t *testing.T) {
	branches := gitBranches()()

	for _, s := range branches {
		if s.Name == "origin" {
			t.Fatalf("unexpected branch suggestion %q in %#v", s.Name, branches)
		}
	}

	if !hasSuggestion(branches, "main") {
		t.Fatalf("missing main in %#v", branches)
	}
	if !hasSuggestion(branches, "origin/main") {
		t.Fatalf("missing origin/main in %#v", branches)
	}
}

func TestGitLocalBranchesExcludeRemoteTrackingRefs(t *testing.T) {
	branches := gitLocalBranches()()

	for _, s := range branches {
		if s.Name == "origin" || s.Name == "origin/main" {
			t.Fatalf("unexpected local branch suggestion %q in %#v", s.Name, branches)
		}
	}

	if !hasSuggestion(branches, "main") {
		t.Fatalf("missing main in %#v", branches)
	}
}

func hasSuggestion(suggestions []Suggestion, name string) bool {
	for _, s := range suggestions {
		if s.Name == name {
			return true
		}
	}
	return false
}
