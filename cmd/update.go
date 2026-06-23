//go:build !windows

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/nguyenlinh13602/goshell/internal/prompt"
)

const (
	defaultReleaseURL     = "https://github.com/linh1306/znux-terminal/releases/latest/download/znux"
	defaultSuggestAPIURL  = "https://api.github.com/repos/linh1306/znux-terminal/contents/suggest?ref=main"
	releaseURLEnv         = "ZNUX_RELEASE_URL"
	suggestAPIURLEnv      = "ZNUX_SUGGEST_API_URL"
	downloadTimeout       = 60 * time.Second
	defaultBinaryFileMode = 0o755
	defaultSpecFileMode   = 0o644
)

type remoteSuggestFile struct {
	Name        string `json:"name"`
	DownloadURL string `json:"download_url"`
	Type        string `json:"type"`
}

func handleCLI(args []string) (bool, int) {
	if len(args) == 0 {
		return false, 0
	}

	switch args[0] {
	case "update":
		if err := runUpdate(); err != nil {
			fmt.Fprintf(os.Stderr, "znux update: %v\n", err)
			return true, 1
		}
		return true, 0
	case "suggest":
		if err := runSuggest(); err != nil {
			fmt.Fprintf(os.Stderr, "znux suggest: %v\n", err)
			return true, 1
		}
		return true, 0
	case "-h", "--help", "help":
		printUsage()
		return true, 0
	default:
		return false, 0
	}
}

func printUsage() {
	fmt.Fprintln(os.Stdout, "Usage:")
	fmt.Fprintln(os.Stdout, "  znux          start terminal")
	fmt.Fprintln(os.Stdout, "  znux update   download latest znux to ~/.local/bin")
	fmt.Fprintln(os.Stdout, "  znux suggest  manage command suggestion files")
}

func runUpdate() error {
	target, err := userBinaryPath()
	if err != nil {
		return err
	}

	url := envOrDefault(releaseURLEnv, defaultReleaseURL)
	fmt.Fprintf(os.Stdout, "Downloading %s\n", url)
	if err := downloadFile(url, target, defaultBinaryFileMode); err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "Installed znux to %s\n", target)
	return nil
}

func runSuggest() error {
	dir, err := userSuggestDir()
	if err != nil {
		return err
	}

	local, err := listLocalSuggestFiles(dir)
	if err != nil {
		return err
	}

	remote, err := fetchRemoteSuggestFiles(envOrDefault(suggestAPIURLEnv, defaultSuggestAPIURL))
	if err != nil {
		return err
	}
	if len(remote) == 0 {
		return errors.New("no remote suggest files found")
	}

	choices := suggestChoices(remote)
	defaultIndexes := installedDefaultIndexes(choices, local)
	selected, err := prompt.Options(os.Stdin, os.Stdout, "Suggest Commands", choices, defaultIndexes)
	if err != nil {
		return err
	}
	selectedSet := nameSet(selected)

	toDelete := subtractNames(local, selectedSet)
	if len(toDelete) > 0 {
		fmt.Fprintln(os.Stdout, "The following local suggest files will be deleted:")
		for _, name := range toDelete {
			fmt.Fprintf(os.Stdout, "  - %s\n", name)
		}
		confirm, err := prompt.Select(os.Stdin, os.Stdout, "Delete unselected suggest files?", []string{"No", "Yes"}, 0)
		if err != nil {
			return err
		}
		if confirm != "Yes" {
			return errors.New("cancelled before deleting suggest files")
		}
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create suggest dir %q: %w", dir, err)
	}

	remoteByCommand := make(map[string]remoteSuggestFile, len(remote))
	for _, file := range remote {
		remoteByCommand[commandName(file.Name)] = file
	}

	for _, name := range selected {
		file, ok := remoteByCommand[name]
		if !ok {
			continue
		}
		target := filepath.Join(dir, file.Name)
		if err := downloadFile(file.DownloadURL, target, defaultSpecFileMode); err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "Installed suggest %s\n", file.Name)
	}

	for _, name := range toDelete {
		if err := os.Remove(filepath.Join(dir, name)); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("delete suggest %q: %w", name, err)
		}
		fmt.Fprintf(os.Stdout, "Deleted suggest %s\n", name)
	}

	fmt.Fprintf(os.Stdout, "Suggest directory: %s\n", dir)
	return nil
}

func userBinaryPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "bin", "znux"), nil
}

func userSuggestDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".znux", "suggest"), nil
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func downloadFile(url, target string, mode os.FileMode) error {
	if strings.TrimSpace(url) == "" {
		return errors.New("download URL is empty")
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("create target dir: %w", err)
	}

	client := &http.Client{Timeout: downloadTimeout}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("download failed with HTTP %d", resp.StatusCode)
	}

	tmp, err := os.CreateTemp(filepath.Dir(target), "."+filepath.Base(target)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, target); err != nil {
		return err
	}
	return nil
}

func fetchRemoteSuggestFiles(url string) ([]remoteSuggestFile, error) {
	client := &http.Client{Timeout: downloadTimeout}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch suggest list failed with HTTP %d", resp.StatusCode)
	}

	var files []remoteSuggestFile
	if err := json.NewDecoder(resp.Body).Decode(&files); err != nil {
		return nil, err
	}

	filtered := files[:0]
	for _, file := range files {
		if file.Type == "file" && strings.HasSuffix(file.Name, ".yaml") && file.DownloadURL != "" {
			filtered = append(filtered, file)
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Name < filtered[j].Name
	})
	return filtered, nil
}

func listLocalSuggestFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read suggest dir %q: %w", dir, err)
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		files = append(files, entry.Name())
	}
	sort.Strings(files)
	return files, nil
}

func suggestChoices(files []remoteSuggestFile) []string {
	choices := make([]string, len(files))
	for i, file := range files {
		choices[i] = commandName(file.Name)
	}
	return choices
}

func installedDefaultIndexes(choices []string, installed []string) []int {
	installedCommands := make([]string, 0, len(installed))
	for _, file := range installed {
		installedCommands = append(installedCommands, commandName(file))
	}
	installedSet := nameSet(installedCommands)
	indexes := make([]int, 0, len(installed))
	for i, choice := range choices {
		if installedSet[choice] {
			indexes = append(indexes, i)
		}
	}
	return indexes
}

func nameSet(names []string) map[string]bool {
	set := make(map[string]bool, len(names))
	for _, name := range names {
		set[name] = true
	}
	return set
}

func subtractNames(names []string, selected map[string]bool) []string {
	out := make([]string, 0)
	for _, name := range names {
		if !selected[commandName(name)] {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

func commandName(file string) string {
	return strings.TrimSuffix(file, ".yaml")
}
