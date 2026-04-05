package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"main/internal/session"
)

const (
	updateRepoOwner       = "nawf-dev"
	updateRepoName        = "AM-DL"
	updateRepoBranch      = "main"
	updateVersionFileName = "version.json"
)

type updateVersionInfo struct {
	Repo        string `json:"repo"`
	Branch      string `json:"branch"`
	Commit      string `json:"commit"`
	ShortCommit string `json:"shortCommit,omitempty"`
	Source      string `json:"source,omitempty"`
	UpdatedAt   string `json:"updatedAt,omitempty"`
	URL         string `json:"url,omitempty"`
}

type githubCommitResponse struct {
	SHA     string `json:"sha"`
	HTMLURL string `json:"html_url"`
	Commit  struct {
		Author struct {
			Date string `json:"date"`
		} `json:"author"`
	} `json:"commit"`
}

func runUpdateCheckCommand() error {
	remote, err := fetchRemoteUpdateVersion()
	if err != nil {
		return fmt.Errorf("failed to check latest update: %w", err)
	}

	local, localErr := detectLocalUpdateVersion()

	fmt.Printf("Update source: %s/%s (%s)\n", updateRepoOwner, updateRepoName, updateRepoBranch)
	if localErr != nil || strings.TrimSpace(local.Commit) == "" {
		fmt.Println("Local version: unknown")
		if localErr != nil {
			fmt.Printf("Details: %v\n", localErr)
		}
	} else {
		fmt.Printf("Local version: %s (%s)\n", local.ShortCommit, localVersionLabel(local))
	}
	fmt.Printf("Latest remote: %s (%s)\n", remote.ShortCommit, formatUpdateTimestamp(remote.UpdatedAt))

	if strings.TrimSpace(local.Commit) == "" {
		fmt.Println("Could not determine the current local build exactly. Run `amdl update` to refresh this folder.")
		return nil
	}

	if strings.EqualFold(strings.TrimSpace(local.Commit), strings.TrimSpace(remote.Commit)) {
		fmt.Println("No update available. You're already on the latest main build.")
		return nil
	}

	fmt.Printf("Update available: %s -> %s\n", local.ShortCommit, remote.ShortCommit)
	fmt.Println("Run `amdl update` to download and apply the latest files.")
	return nil
}

func runVersionCommand() error {
	local, err := detectLocalUpdateVersion()
	if err != nil || strings.TrimSpace(local.Commit) == "" {
		fmt.Println("AM-DL")
		fmt.Println("Version: unknown")
		if err != nil {
			fmt.Printf("Details: %v\n", err)
		}
		fmt.Println("Tip: run `amdl update --check` to compare with the latest remote build.")
		return nil
	}

	repo := strings.TrimSpace(local.Repo)
	if repo == "" {
		repo = updateRepoOwner + "/" + updateRepoName
	}

	fmt.Println("AM-DL")
	fmt.Printf("Repo: %s\n", repo)
	fmt.Printf("Version: %s\n", local.ShortCommit)
	if branch := strings.TrimSpace(local.Branch); branch != "" {
		fmt.Printf("Branch: %s\n", branch)
	}
	if source := strings.TrimSpace(local.Source); source != "" {
		fmt.Printf("Source: %s\n", source)
	}
	if updatedAt := strings.TrimSpace(local.UpdatedAt); updatedAt != "" {
		fmt.Printf("Updated: %s\n", formatUpdateTimestamp(updatedAt))
	}
	fmt.Println("Tip: run `amdl update --check` to compare with the latest remote build.")
	return nil
}

func detectLocalUpdateVersion() (updateVersionInfo, error) {
	root := session.Detect().WorkspaceRoot
	if gitInfo, err := detectGitUpdateVersion(root); err == nil && strings.TrimSpace(gitInfo.Commit) != "" {
		return gitInfo, nil
	}
	return readStoredUpdateVersion(root)
}

func detectGitUpdateVersion(root string) (updateVersionInfo, error) {
	if _, err := os.Stat(filepath.Join(root, ".git")); err != nil {
		return updateVersionInfo{}, err
	}
	if _, err := exec.LookPath("git"); err != nil {
		return updateVersionInfo{}, err
	}
	commit, err := gitOutput(root, "rev-parse", "HEAD")
	if err != nil {
		return updateVersionInfo{}, err
	}
	branch, err := gitOutput(root, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		branch = updateRepoBranch
	}
	return updateVersionInfo{
		Repo:        updateRepoOwner + "/" + updateRepoName,
		Branch:      strings.TrimSpace(branch),
		Commit:      strings.TrimSpace(commit),
		ShortCommit: shortCommit(commit),
		Source:      "git",
	}, nil
}

func gitOutput(root string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf(strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

func readStoredUpdateVersion(root string) (updateVersionInfo, error) {
	path := filepath.Join(root, updateVersionFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return updateVersionInfo{}, err
	}
	var info updateVersionInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return updateVersionInfo{}, err
	}
	if info.ShortCommit == "" {
		info.ShortCommit = shortCommit(info.Commit)
	}
	if info.Source == "" {
		info.Source = "version file"
	}
	return info, nil
}

func fetchRemoteUpdateVersion() (updateVersionInfo, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits/%s", updateRepoOwner, updateRepoName, updateRepoBranch)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return updateVersionInfo{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "AM-DL-Updater")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return updateVersionInfo{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return updateVersionInfo{}, fmt.Errorf("GitHub API request failed with status: %s", resp.Status)
	}

	var payload githubCommitResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return updateVersionInfo{}, err
	}

	return updateVersionInfo{
		Repo:        updateRepoOwner + "/" + updateRepoName,
		Branch:      updateRepoBranch,
		Commit:      payload.SHA,
		ShortCommit: shortCommit(payload.SHA),
		Source:      "github",
		UpdatedAt:   payload.Commit.Author.Date,
		URL:         payload.HTMLURL,
	}, nil
}

func shortCommit(commit string) string {
	trimmed := strings.TrimSpace(commit)
	if len(trimmed) <= 7 {
		return trimmed
	}
	return trimmed[:7]
}

func localVersionLabel(info updateVersionInfo) string {
	parts := []string{info.Source}
	if branch := strings.TrimSpace(info.Branch); branch != "" {
		parts = append(parts, branch)
	}
	return strings.Join(parts, ", ")
}

func formatUpdateTimestamp(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "time unknown"
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return trimmed
	}
	return parsed.UTC().Format("2006-01-02 15:04 MST")
}
