package specs

import (
	"os/exec"
	"strings"
	"sync"
	"time"
)

func init() {
	Register("git", &GitSpec)
}

var GitSpec = Spec{
	Name: "git",
	Subcommands: []Subcommand{
		{
			Name:        "status",
			Description: "Show the working tree status",
			Options: []Option{
				{Names: []string{"-s", "--short"}, Description: "Give output in short format"},
				{Names: []string{"-b", "--branch"}, Description: "Show branch and tracking info"},
			},
		},
		{
			Name:        "add",
			Description: "Add file contents to the index",
			Options: []Option{
				{Names: []string{"-A", "--all"}, Description: "Add all files"},
				{Names: []string{"-p", "--patch"}, Description: "Interactively choose hunks"},
				{Names: []string{"-n", "--dry-run"}, Description: "Show what would be added"},
			},
			Args: []ArgSpec{{Name: "pathspec", Template: TemplateFileSystem}},
		},
		{
			Name:        "commit",
			Description: "Record changes to the repository",
			Options: []Option{
				{Names: []string{"-m", "--message"}, Description: "Commit message"},
				{Names: []string{"-a", "--all"}, Description: "Automatically stage modified files"},
				{Names: []string{"--amend"}, Description: "Amend the previous commit"},
				{Names: []string{"--no-edit"}, Description: "Use the selected commit message without editing"},
			},
		},
		{
			Name:        "push",
			Description: "Update remote refs",
			Options: []Option{
				{Names: []string{"-u", "--set-upstream"}, Description: "Set upstream for the branch"},
				{Names: []string{"-f", "--force"}, Description: "Force push"},
				{Names: []string{"--tags"}, Description: "Push all tags"},
			},
			Args: []ArgSpec{
				{Name: "remote", Generator: gitRemotes()},
				{Name: "branch"},
			},
		},
		{
			Name:        "pull",
			Description: "Fetch from and integrate with another repository",
			Options: []Option{
				{Names: []string{"--rebase"}, Description: "Rebase instead of merge"},
				{Names: []string{"--no-commit"}, Description: "Perform the merge but don't commit"},
			},
			Args: []ArgSpec{
				{Name: "remote", Generator: gitRemotes()},
				{Name: "branch"},
			},
		},
		{
			Name:        "checkout",
			Description: "Switch branches or restore working tree files",
			Options: []Option{
				{Names: []string{"-b"}, Description: "Create and switch to a new branch"},
				{Names: []string{"-B"}, Description: "Create/reset and switch to a branch"},
			},
			Args: []ArgSpec{
				{
					Name:       "branch",
					Generator:  gitBranches(),
					IsVariadic: true,
				},
			},
		},
		{
			Name:        "branch",
			Description: "List, create, or delete branches",
			Options: []Option{
				{Names: []string{"-a", "--all"}, Description: "List both remote and local branches"},
				{Names: []string{"-d", "--delete"}, Description: "Delete a branch"},
				{Names: []string{"-D"}, Description: "Force delete a branch"},
				{Names: []string{"-m", "--move"}, Description: "Rename a branch"},
			},
			Args: []ArgSpec{{Name: "branch", Generator: gitBranches()}},
		},
		{
			Name:        "merge",
			Description: "Join two or more development histories",
			Options: []Option{
				{Names: []string{"--no-ff"}, Description: "Create a merge commit even if fast-forward"},
				{Names: []string{"--squash"}, Description: "Squash all commits into one"},
				{Names: []string{"--abort"}, Description: "Abort the current merge"},
			},
			Args: []ArgSpec{{Name: "branch", Generator: gitBranches()}},
		},
		{
			Name:        "rebase",
			Description: "Reapply commits on top of another base",
			Options: []Option{
				{Names: []string{"-i", "--interactive"}, Description: "Interactive rebase"},
				{Names: []string{"--continue"}, Description: "Continue after resolving conflicts"},
				{Names: []string{"--abort"}, Description: "Abort the rebase"},
				{Names: []string{"--skip"}, Description: "Skip the current patch"},
			},
			Args: []ArgSpec{{Name: "branch", Generator: gitBranches()}},
		},
		{
			Name:        "log",
			Description: "Show commit logs",
			Options: []Option{
				{Names: []string{"--oneline"}, Description: "One line per commit"},
				{Names: []string{"--graph"}, Description: "Show ASCII graph"},
				{Names: []string{"-n", "--max-count"}, Description: "Limit the number of commits"},
				{Names: []string{"--author"}, Description: "Filter by author"},
			},
			Args: []ArgSpec{{Name: "revision"}},
		},
		{
			Name:        "diff",
			Description: "Show changes between commits, commit and working tree, etc",
			Options: []Option{
				{Names: []string{"--staged"}, Description: "Show staged changes"},
				{Names: []string{"--name-only"}, Description: "Show only names of changed files"},
				{Names: []string{"--stat"}, Description: "Show diffstat"},
			},
			Args: []ArgSpec{
				{Name: "commit", Generator: gitBranches()},
				{Name: "commit", Generator: gitBranches()},
			},
		},
		{
			Name:        "stash",
			Description: "Stash changes in a dirty working directory",
			Options: []Option{
				{Names: []string{"push", "-m"}, Description: "Stash with message"},
				{Names: []string{"pop"}, Description: "Apply and remove stash"},
				{Names: []string{"list"}, Description: "List all stashes"},
				{Names: []string{"drop"}, Description: "Remove a stash"},
				{Names: []string{"apply"}, Description: "Apply a stash without removing"},
			},
		},
		{
			Name:        "reset",
			Description: "Reset current HEAD to the specified state",
			Options: []Option{
				{Names: []string{"--soft"}, Description: "Keep changes staged"},
				{Names: []string{"--mixed"}, Description: "Keep changes unstaged (default)"},
				{Names: []string{"--hard"}, Description: "Discard all changes"},
			},
			Args: []ArgSpec{{Name: "commit", Generator: gitBranches()}},
		},
		{
			Name:        "restore",
			Description: "Restore working tree files",
			Options: []Option{
				{Names: []string{"-s", "--source"}, Description: "Restore from source"},
				{Names: []string{"-S", "--staged"}, Description: "Restore staged content"},
			},
			Args: []ArgSpec{{Name: "pathspec", Template: TemplateFileSystem}},
		},
		{
			Name:        "switch",
			Description: "Switch branches",
			Options: []Option{
				{Names: []string{"-c", "--create"}, Description: "Create and switch to a new branch"},
				{Names: []string{"-C"}, Description: "Create/reset and switch to a branch"},
			},
			Args: []ArgSpec{{Name: "branch", Generator: gitBranches()}},
		},
		{
			Name:        "fetch",
			Description: "Download objects and refs from another repository",
			Options: []Option{
				{Names: []string{"--all"}, Description: "Fetch all remotes"},
				{Names: []string{"--prune"}, Description: "Remove deleted remote branches"},
			},
			Args: []ArgSpec{{Name: "remote", Generator: gitRemotes()}},
		},
		{
			Name:        "remote",
			Description: "Manage set of tracked repositories",
			Options: []Option{
				{Names: []string{"-v", "--verbose"}, Description: "Show remote URL"},
				{Names: []string{"add"}, Description: "Add a remote"},
				{Names: []string{"remove"}, Description: "Remove a remote"},
			},
		},
		{
			Name:        "tag",
			Description: "Create, list, delete, or verify a tag",
			Options: []Option{
				{Names: []string{"-a", "--annotate"}, Description: "Create an annotated tag"},
				{Names: []string{"-d", "--delete"}, Description: "Delete a tag"},
				{Names: []string{"-m", "--message"}, Description: "Tag message"},
			},
			Args: []ArgSpec{{Name: "tagname"}},
		},
		{
			Name:        "show",
			Description: "Show various types of objects",
			Options: []Option{
				{Names: []string{"--stat"}, Description: "Show diffstat"},
			},
			Args: []ArgSpec{{Name: "object", Generator: gitBranches()}},
		},
		{
			Name:        "blame",
			Description: "Show what revision and author last modified each line",
			Options: []Option{
				{Names: []string{"-L"}, Description: "Annotate only given line range"},
				{Names: []string{"-w"}, Description: "Ignore whitespace changes"},
			},
			Args: []ArgSpec{{Name: "file", Template: TemplateFileSystem}},
		},
		{
			Name:        "clean",
			Description: "Remove untracked files from the working tree",
			Options: []Option{
				{Names: []string{"-n", "--dry-run"}, Description: "Show what would be removed"},
				{Names: []string{"-d"}, Description: "Remove directories too"},
				{Names: []string{"-f", "--force"}, Description: "Force clean"},
				{Names: []string{"-x"}, Description: "Also remove ignored files"},
			},
			Args: []ArgSpec{{Name: "pathspec", Template: TemplateFileSystem}},
		},
		{
			Name:        "config",
			Description: "Get and set repository or global options",
			Options: []Option{
				{Names: []string{"--global"}, Description: "Use global config"},
				{Names: []string{"--local"}, Description: "Use local repo config"},
				{Names: []string{"--list"}, Description: "List all config"},
			},
			Args: []ArgSpec{
				{Name: "name"},
				{Name: "value"},
			},
		},
		{
			Name:        "clone",
			Description: "Clone a repository into a new directory",
			Options: []Option{
				{Names: []string{"--depth"}, Description: "Shallow clone depth"},
				{Names: []string{"-b", "--branch"}, Description: "Branch to clone"},
				{Names: []string{"--single-branch"}, Description: "Clone only one branch"},
			},
			Args: []ArgSpec{{Name: "repository"}},
		},
		{
			Name:        "init",
			Description: "Create an empty Git repository",
			Options: []Option{
				{Names: []string{"-b"}, Description: "Initial branch name"},
				{Names: []string{"--bare"}, Description: "Create a bare repository"},
			},
			Args: []ArgSpec{{Name: "directory"}},
		},
		{
			Name:        " rm",
			Description: "Remove files from the working tree and index",
			Options: []Option{
				{Names: []string{"--cached"}, Description: "Remove only from index"},
				{Names: []string{"-f", "--force"}, Description: "Force remove"},
				{Names: []string{"-r"}, Description: "Recursive remove"},
			},
			Args: []ArgSpec{{Name: "pathspec", Template: TemplateFileSystem}},
		},
		{
			Name:        "mv",
			Description: "Move or rename a file, directory, or symlink",
			Options: []Option{
				{Names: []string{"-f", "--force"}, Description: "Force rename"},
			},
			Args: []ArgSpec{
				{Name: "source", Template: TemplateFileSystem},
				{Name: "destination"},
			},
		},
		{
			Name:        "rev-parse",
			Description: "Pick out and massage parameters",
			Options: []Option{
				{Names: []string{"--is-inside-work-tree"}, Description: "Check if inside work tree"},
				{Names: []string{"--show-toplevel"}, Description: "Show top-level directory"},
			},
			Args: []ArgSpec{{Name: "name"}},
		},
		{
			Name:        "ls-files",
			Description: "Show information about files in the index and working tree",
			Options: []Option{
				{Names: []string{"--stage"}, Description: "Show staged contents"},
				{Names: []string{"--others"}, Description: "Show other (untracked) files"},
				{Names: []string{"-c", "--cached"}, Description: "Show cached files"},
			},
			Args: []ArgSpec{{Name: "pathspec", Template: TemplateFileSystem}},
		},
		{
			Name:        "ls-tree",
			Description: "List the contents of a tree object",
			Options: []Option{
				{Names: []string{"-r", "--recursive"}, Description: "Recurse into subtrees"},
				{Names: []string{"--name-only"}, Description: "Show only file names"},
			},
			Args: []ArgSpec{
				{Name: "tree", Generator: gitBranches()},
				{Name: "path", Template: TemplateFileSystem},
			},
		},
		{
			Name:        "describe",
			Description: "Give an object a human-readable name",
			Options: []Option{
				{Names: []string{"--always"}, Description: "Show unique commit"},
				{Names: []string{"--tags"}, Description: "Use tags"},
			},
			Args: []ArgSpec{{Name: "commit", Generator: gitBranches()}},
		},
		{
			Name:        "reflog",
			Description: "Manage reflog information",
			Options: []Option{
				{Names: []string{"--date"}, Description: "Date format"},
			},
			Args: []ArgSpec{{Name: "ref", Generator: gitBranches()}},
		},
		{
			Name:        "worktree",
			Description: "Manage multiple working trees",
			Options: []Option{
				{Names: []string{"add"}, Description: "Create a worktree"},
				{Names: []string{"list"}, Description: "List worktrees"},
				{Names: []string{"remove"}, Description: "Remove a worktree"},
				{Names: []string{"prune"}, Description: "Prune worktree info"},
			},
		},
	},
	Options: []Option{
		{Names: []string{"-C"}, Description: "Run as if started in path"},
		{Names: []string{"--version"}, Description: "Print git version"},
		{Names: []string{"--help"}, Description: "Print help"},
	},
}

// CachedBranches returns a generator that caches branch list for 5 seconds
func gitBranches() func() []Suggestion {
	cache := struct {
		suggestions []Suggestion
		expiry      time.Time
		mu          sync.Mutex
	}{}

	return func() []Suggestion {
		cache.mu.Lock()
		defer cache.mu.Unlock()

		if time.Now().Before(cache.expiry) && cache.suggestions != nil {
			return cache.suggestions
		}

		cmd := exec.Command("git", "branch", "-a", "--no-color", "--format=%(refname:short)")
		out, err := cmd.Output()
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

		cache.suggestions = suggestions
		cache.expiry = time.Now().Add(5 * time.Second)
		return suggestions
	}
}

// gitRemotes returns a generator for remote names
func gitRemotes() func() []Suggestion {
	return func() []Suggestion {
		cmd := exec.Command("git", "remote")
		out, err := cmd.Output()
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
		return suggestions
	}
}
