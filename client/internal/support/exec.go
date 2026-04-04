package support

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func ResolveExecutable(name string) (string, error) {
	if strings.TrimSpace(name) == "" {
		return "", fmt.Errorf("executable name is empty")
	}
	if path, err := exec.LookPath(name); err == nil {
		return path, nil
	}

	for _, candidate := range expandExecutableCandidates(name) {
		if isFile(candidate) {
			return candidate, nil
		}
	}

	for _, dir := range candidateDirs() {
		for _, candidate := range expandExecutableCandidates(name) {
			if filepath.IsAbs(candidate) {
				continue
			}
			path := filepath.Join(dir, candidate)
			if isFile(path) {
				return path, nil
			}
		}
	}

	return "", fmt.Errorf("executable not found: %s", name)
}

func candidateDirs() []string {
	seen := map[string]bool{}
	var dirs []string
	appendDir := func(path string) {
		if path == "" {
			return
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return
		}
		if !seen[abs] {
			seen[abs] = true
			dirs = append(dirs, abs)
		}
	}

	if cwd, err := os.Getwd(); err == nil {
		appendDir(cwd)
		appendDir(filepath.Join(cwd, "tools"))
		appendDir(filepath.Dir(cwd))
		appendDir(filepath.Join(filepath.Dir(cwd), "tools"))
	}
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		appendDir(exeDir)
		appendDir(filepath.Join(exeDir, "tools"))
		appendDir(filepath.Dir(exeDir))
		appendDir(filepath.Join(filepath.Dir(exeDir), "tools"))
	}

	return dirs
}

func expandExecutableCandidates(name string) []string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return nil
	}
	if filepath.Ext(trimmed) != "" {
		return []string{trimmed}
	}
	if runtime.GOOS != "windows" {
		return []string{trimmed}
	}
	return []string{trimmed, trimmed + ".exe", trimmed + ".cmd", trimmed + ".bat"}
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Size() > 0
}
