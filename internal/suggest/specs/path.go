package specs

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func init() {
	RegisterSource("path", func(source SourceSpec, ctx SourceContext, partial string) []Suggestion {
		return pathSuggestions(partial, source.Include)
	})
}

func pathSuggestions(partial string, include []string) []Suggestion {
	includeFiles, includeFolders := true, true
	priority := map[SuggestionKind]int{
		KindFolder: 0,
		KindFile:   1,
	}
	if len(include) > 0 {
		includeFiles = false
		includeFolders = false
		priority = map[SuggestionKind]int{}
		for i, item := range include {
			switch item {
			case "file":
				includeFiles = true
				priority[KindFile] = i
			case "folder", "dir", "directory":
				includeFolders = true
				priority[KindFolder] = i
			}
		}
	}

	suggestions := fileSuggestions(partial, includeFiles, includeFolders)
	sort.SliceStable(suggestions, func(i, j int) bool {
		return priority[suggestions[i].Kind] < priority[suggestions[j].Kind]
	})
	return suggestions
}

func fileSuggestions(partial string, includeFiles, includeFolders bool) []Suggestion {
	var dir, prefix string
	lastSlash := strings.LastIndex(partial, "/")
	if partial == "~" {
		dir = "~/"
		prefix = ""
	} else if lastSlash < 0 {
		dir = "."
		prefix = partial
	} else {
		dir = partial[:lastSlash+1]
		prefix = partial[lastSlash+1:]
	}

	readDir := dir
	if strings.HasPrefix(dir, "~/") || dir == "~" {
		home, err := os.UserHomeDir()
		if err != nil || home == "" {
			return nil
		}
		readDir = home
		if len(dir) > 2 {
			readDir = filepath.Join(home, dir[2:])
		}
	}

	entries, err := os.ReadDir(readDir)
	if err != nil {
		return nil
	}

	var suggestions []Suggestion
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") && !strings.HasPrefix(prefix, ".") {
			continue
		}
		if prefix != "" && !strings.HasPrefix(name, prefix) {
			continue
		}
		if entry.IsDir() && !includeFolders {
			continue
		}
		if !entry.IsDir() && !includeFiles {
			continue
		}

		var full string
		if dir == "." {
			full = name
		} else {
			full = dir + name
		}
		if entry.IsDir() {
			full += "/"
		}

		kind := KindFile
		desc := "file"
		if entry.IsDir() {
			kind = KindFolder
			desc = "folder"
		}
		suggestions = append(suggestions, Suggestion{Name: full, Kind: kind, Description: desc})
	}
	return suggestions
}
