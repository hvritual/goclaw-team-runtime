package ouroboros

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	maxRepositoryEntries  = 200
	maxManifestBytes      = 8 * 1024
	maxManifestTotalBytes = 32 * 1024
)

var repositoryManifestNames = map[string]struct{}{
	"Cargo.toml":          {},
	"Makefile":            {},
	"build.gradle":        {},
	"build.gradle.kts":    {},
	"go.mod":              {},
	"go.work":             {},
	"package.json":        {},
	"pom.xml":             {},
	"pnpm-workspace.yaml": {},
	"pyproject.toml":      {},
	"settings.gradle":     {},
}

func inspectRepository(path string) (string, RepositoryContext, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", RepositoryContext{}, err
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", RepositoryContext{}, fmt.Errorf("resolve repository path: %w", err)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", RepositoryContext{}, err
	}
	if !info.IsDir() {
		return "", RepositoryContext{}, errors.New("repository path is not a directory")
	}
	entries, err := os.ReadDir(resolved)
	if err != nil {
		return "", RepositoryContext{}, err
	}
	context := RepositoryContext{
		Manifests:  make(map[string]string),
		CapturedAt: time.Now().UTC(),
	}
	if _, err := os.Stat(filepath.Join(resolved, ".git")); err == nil {
		context.GitPresent = true
	}
	totalManifestBytes := 0
	for _, entry := range entries {
		name := entry.Name()
		if isSensitiveRepositoryName(name) {
			continue
		}
		if len(context.TopLevel) < maxRepositoryEntries {
			context.TopLevel = append(context.TopLevel, name)
		} else {
			context.Truncated = true
		}
		if _, ok := repositoryManifestNames[name]; !ok || entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		entryInfo, infoErr := entry.Info()
		if infoErr != nil || !entryInfo.Mode().IsRegular() {
			continue
		}
		remaining := maxManifestTotalBytes - totalManifestBytes
		if remaining <= 0 {
			context.Truncated = true
			continue
		}
		limit := maxManifestBytes
		if remaining < limit {
			limit = remaining
		}
		content, truncated, readErr := readBoundedFile(filepath.Join(resolved, name), limit)
		if readErr != nil {
			return "", RepositoryContext{}, fmt.Errorf("read repository manifest %s: %w", name, readErr)
		}
		context.Manifests[name] = content
		totalManifestBytes += len(content)
		context.Truncated = context.Truncated || truncated
	}
	sort.Strings(context.TopLevel)
	return resolved, context, nil
}

func readBoundedFile(path string, maximum int) (string, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", false, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, int64(maximum)+1))
	if err != nil {
		return "", false, err
	}
	if len(data) <= maximum {
		return string(data), false, nil
	}
	return string(data[:maximum]), true, nil
}

func isSensitiveRepositoryName(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	if strings.HasPrefix(lower, ".") {
		return true
	}
	for _, fragment := range []string{"credential", "secret", "token", "private-key", "private_key"} {
		if strings.Contains(lower, fragment) {
			return true
		}
	}
	return false
}
