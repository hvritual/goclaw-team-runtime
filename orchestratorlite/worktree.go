package orchestratorlite

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

func runGit(ctx context.Context, repo string, args ...string) (commandResult, error) {
	return runCommand(ctx, repo, "", nil, "git", append([]string{"-C", repo}, args...)...)
}

func (s *Service) ensureWorktree(ctx context.Context, task Task) (Task, RepositorySnapshot, error) {
	if task.Compile.BaseCommit == "" {
		return Task{}, RepositorySnapshot{}, errors.New("task has no frozen base commit")
	}
	revisionKey := fmt.Sprintf("%s-r%d", task.ID, task.Compile.Revision)
	worktreePath, err := safeJoin(s.cfg.WorktreeRoot, revisionKey)
	if err != nil {
		return Task{}, RepositorySnapshot{}, err
	}
	branch := fmt.Sprintf(
		"goclaw/%s-r%d",
		strings.TrimPrefix(task.ID, "task-"),
		task.Compile.Revision,
	)
	if exists(worktreePath) {
		if _, err := runGit(ctx, worktreePath, "rev-parse", "--is-inside-work-tree"); err != nil {
			return Task{}, RepositorySnapshot{}, fmt.Errorf("existing worktree path is invalid: %w", err)
		}
	} else {
		if err := ensureDir(filepath.Dir(worktreePath)); err != nil {
			return Task{}, RepositorySnapshot{}, err
		}
		_, branchErr := runGit(ctx, task.RepoPath, "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
		var result commandResult
		if branchErr == nil {
			result, err = runGit(ctx, task.RepoPath, "worktree", "add", worktreePath, branch)
		} else {
			result, err = runGit(ctx, task.RepoPath, "worktree", "add", "-b", branch, worktreePath, task.Compile.BaseCommit)
		}
		if err != nil {
			return Task{}, RepositorySnapshot{}, fmt.Errorf("create worktree: %w: %s", err, strings.TrimSpace(result.Stderr))
		}
	}
	head, err := runGit(ctx, worktreePath, "rev-parse", "HEAD")
	if err != nil {
		return Task{}, RepositorySnapshot{}, err
	}
	if strings.TrimSpace(head.Stdout) != task.Compile.BaseCommit {
		return Task{}, RepositorySnapshot{}, fmt.Errorf(
			"worktree HEAD %s does not match frozen base %s",
			strings.TrimSpace(head.Stdout),
			task.Compile.BaseCommit,
		)
	}
	task.WorktreePath = worktreePath
	task.Branch = branch
	task.UpdatedAt = time.Now().UTC()
	snapshot, err := captureRepositorySnapshot(ctx, task)
	return task, snapshot, err
}

func captureRepositorySnapshot(ctx context.Context, task Task) (RepositorySnapshot, error) {
	head, err := runGit(ctx, task.WorktreePath, "rev-parse", "HEAD")
	if err != nil {
		return RepositorySnapshot{}, err
	}
	status, err := runGit(ctx, task.WorktreePath, "status", "--porcelain=v1")
	if err != nil {
		return RepositorySnapshot{}, err
	}
	return RepositorySnapshot{
		RepoPath:   task.RepoPath,
		Worktree:   task.WorktreePath,
		Branch:     task.Branch,
		BaseCommit: task.Compile.BaseCommit,
		HeadCommit: strings.TrimSpace(head.Stdout),
		Status:     status.Stdout,
		CapturedAt: time.Now().UTC(),
	}, nil
}

func collectChangedFiles(ctx context.Context, task Task) ([]string, error) {
	tracked, err := runGit(ctx, task.WorktreePath, "diff", "--name-only", task.Compile.BaseCommit, "--")
	if err != nil {
		return nil, err
	}
	untracked, err := runGit(ctx, task.WorktreePath, "ls-files", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}
	var result []string
	for _, content := range []string{tracked.Stdout, untracked.Stdout} {
		for _, line := range strings.Split(content, "\n") {
			line = filepath.ToSlash(strings.TrimSpace(line))
			if line != "" {
				result = append(result, line)
			}
		}
	}
	return uniqueStrings(result), nil
}

func collectDiff(ctx context.Context, task Task, changedFiles []string) (string, error) {
	var untracked []string
	trackedSet := make(map[string]bool)
	tracked, err := runGit(ctx, task.WorktreePath, "diff", "--name-only", task.Compile.BaseCommit, "--")
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(tracked.Stdout, "\n") {
		trackedSet[filepath.ToSlash(strings.TrimSpace(line))] = true
	}
	for _, path := range changedFiles {
		if !trackedSet[path] {
			untracked = append(untracked, path)
		}
	}
	if len(untracked) > 0 {
		args := append([]string{"add", "-N", "--"}, untracked...)
		if result, err := runGit(ctx, task.WorktreePath, args...); err != nil {
			return "", fmt.Errorf("mark untracked files for diff: %w: %s", err, result.Stderr)
		}
	}
	diff, err := runGit(ctx, task.WorktreePath, "diff", "--binary", "--no-ext-diff", task.Compile.BaseCommit, "--")
	if err != nil {
		return "", err
	}
	return diff.Stdout, nil
}

func evaluatePolicy(ctx context.Context, task Task, changedFiles []string) (PolicyResult, error) {
	result := PolicyResult{
		Passed:       true,
		ChangedFiles: append([]string(nil), changedFiles...),
	}
	for _, path := range changedFiles {
		clean := filepath.ToSlash(filepath.Clean(path))
		if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || filepath.IsAbs(path) {
			result.Violations = append(result.Violations, "invalid changed path: "+path)
			continue
		}
		if matchesAnyPath(clean, task.Scope.DeniedPaths) {
			result.Violations = append(result.Violations, "denied path changed: "+clean)
		}
		if len(task.Scope.AllowedPaths) > 0 && !matchesAnyPath(clean, task.Scope.AllowedPaths) {
			result.Violations = append(result.Violations, "path outside approved scope: "+clean)
		}
		if err := validateSymlinkBoundary(task.WorktreePath, clean); err != nil {
			result.Violations = append(result.Violations, err.Error())
		}
		if dependencyManifest(clean) {
			result.NewDependencies = append(result.NewDependencies, clean)
		}
	}
	if task.Scope.MaxChangedFiles > 0 && len(changedFiles) > task.Scope.MaxChangedFiles {
		result.Violations = append(result.Violations,
			fmt.Sprintf("changed files %d exceed approved limit %d", len(changedFiles), task.Scope.MaxChangedFiles))
	}
	numstat, err := runGit(ctx, task.WorktreePath, "diff", "--numstat", task.Compile.BaseCommit, "--")
	if err != nil {
		return result, err
	}
	for _, line := range strings.Split(numstat.Stdout, "\n") {
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}
		added, addErr := strconv.Atoi(parts[0])
		deleted, deleteErr := strconv.Atoi(parts[1])
		if addErr == nil {
			result.AddedLines += added
		}
		if deleteErr == nil {
			result.DeletedLines += deleted
		}
	}
	totalLines := result.AddedLines + result.DeletedLines
	if task.Scope.MaxChangedLines > 0 && totalLines > task.Scope.MaxChangedLines {
		result.Violations = append(result.Violations,
			fmt.Sprintf("changed lines %d exceed approved limit %d", totalLines, task.Scope.MaxChangedLines))
	}
	if len(result.NewDependencies) > 0 && !task.Scope.AllowNewDependency {
		result.Violations = append(result.Violations,
			"dependency manifests changed without capacity approval: "+strings.Join(result.NewDependencies, ", "))
	}
	sort.Strings(result.Violations)
	result.Passed = len(result.Violations) == 0
	return result, nil
}

func matchesAnyPath(path string, patterns []string) bool {
	path = filepath.ToSlash(path)
	for _, pattern := range patterns {
		pattern = filepath.ToSlash(strings.TrimSpace(pattern))
		if pattern == "" {
			continue
		}
		if matched, _ := filepath.Match(pattern, path); matched {
			return true
		}
		if strings.HasSuffix(pattern, "/**") {
			prefix := strings.TrimSuffix(pattern, "**")
			if strings.HasPrefix(path, prefix) {
				return true
			}
		}
		if !strings.ContainsAny(pattern, "*?[") {
			if path == strings.TrimSuffix(pattern, "/") || strings.HasPrefix(path, strings.TrimSuffix(pattern, "/")+"/") {
				return true
			}
		}
	}
	return false
}

func validateSymlinkBoundary(worktree, rel string) error {
	root, err := filepath.EvalSymlinks(worktree)
	if err != nil {
		return fmt.Errorf("resolve worktree boundary: %w", err)
	}
	current := worktree
	for _, part := range strings.Split(filepath.FromSlash(rel), string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			break
		}
		if err != nil {
			return fmt.Errorf("inspect changed path %s: %w", rel, err)
		}
		if info.Mode()&os.ModeSymlink == 0 {
			continue
		}
		resolved, err := filepath.EvalSymlinks(current)
		if err != nil {
			return fmt.Errorf("resolve symlink %s: %w", rel, err)
		}
		if resolved != root && !strings.HasPrefix(resolved, root+string(filepath.Separator)) {
			return fmt.Errorf("changed path crosses worktree through symlink: %s", rel)
		}
	}
	return nil
}

func dependencyManifest(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	switch base {
	case "go.mod", "go.sum", "package.json", "package-lock.json", "pnpm-lock.yaml",
		"yarn.lock", "requirements.txt", "pyproject.toml", "poetry.lock",
		"cargo.toml", "cargo.lock", "composer.json", "composer.lock",
		"gemfile", "gemfile.lock":
		return true
	default:
		return false
	}
}

func worktreeTreeHash(diff string, changed []string) string {
	return sha256Bytes([]byte(diff + "\n--FILES--\n" + strings.Join(changed, "\n")))
}
