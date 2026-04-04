package session

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const wrapperContainerName = "apple-music-wrapper"

type State struct {
	WorkspaceRoot string
	DataDir       string
	CacheDir      string
	Cached        bool
}

func Detect() State {
	root := resolveWorkspaceRoot()
	dataDir := filepath.Join(root, "wrapper-docker", "rootfs", "data")
	cacheDir := filepath.Join(dataDir, "data", "com.apple.android.music")
	return State{
		WorkspaceRoot: root,
		DataDir:       dataDir,
		CacheDir:      cacheDir,
		Cached:        dirHasEntries(cacheDir),
	}
}

func ScriptPath(name string) (string, error) {
	root := resolveWorkspaceRoot()
	path := filepath.Join(root, name)
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("required helper script not found: %s", path)
	}
	return path, nil
}

func ClearLocalSession() error {
	state := Detect()
	if err := os.RemoveAll(state.DataDir); err != nil {
		return err
	}
	return os.MkdirAll(state.DataDir, os.ModePerm)
}

func StopWrapperContainer() error {
	if _, err := exec.LookPath("docker"); err != nil {
		return nil
	}
	stop := exec.Command("docker", "stop", wrapperContainerName)
	_ = stop.Run()
	rm := exec.Command("docker", "rm", wrapperContainerName)
	_ = rm.Run()
	return nil
}

func StatusLine() string {
	state := Detect()
	if state.Cached {
		return fmt.Sprintf("cached (%s)", state.CacheDir)
	}
	return "missing"
}

func resolveWorkspaceRoot() string {
	seen := map[string]bool{}
	var candidates []string
	appendCandidate := func(path string) {
		if path == "" {
			return
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return
		}
		if !seen[abs] {
			seen[abs] = true
			candidates = append(candidates, abs)
		}
	}

	if cwd, err := os.Getwd(); err == nil {
		appendCandidate(cwd)
		appendCandidate(filepath.Dir(cwd))
	}
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		appendCandidate(exeDir)
		appendCandidate(filepath.Dir(exeDir))
	}

	for _, candidate := range candidates {
		if looksLikeWorkspaceRoot(candidate) {
			return candidate
		}
	}
	if len(candidates) > 0 {
		return candidates[0]
	}
	return "."
}

func looksLikeWorkspaceRoot(path string) bool {
	required := []string{"wrapper-docker", "README.md"}
	for _, name := range required {
		if _, err := os.Stat(filepath.Join(path, name)); err != nil {
			return false
		}
	}
	return true
}

func dirHasEntries(path string) bool {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false
	}
	if len(entries) == 0 {
		return false
	}
	// Ignore obvious placeholders only if they ever appear.
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	joined := strings.Join(names, ",")
	return joined != ""
}
