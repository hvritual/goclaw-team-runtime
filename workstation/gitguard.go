package workstation

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var forbiddenLocalGitConfigPrefixes = []string{
	"core.hookspath",
	"core.fsmonitor",
	"core.sshcommand",
	"credential.helper",
	"credential.",
	"diff.",
	"filter.",
	"include.path",
	"includeif.",
	"merge.",
	"protocol.",
	"submodule.",
	"url.",
}

func auditRepositoryGitConfiguration(
	ctx context.Context,
	gitCommand, repository string,
) error {
	result, err := rawGitCommand(
		ctx,
		gitCommand,
		repository,
		"config",
		"--local",
		"--no-includes",
		"--name-only",
		"--get-regexp",
		".*",
	)
	// git config --get-regexp exits 1 when the local config is empty.
	if err != nil && result.ExitCode != 1 {
		return fmt.Errorf(
			"inspect repository Git configuration: %s",
			commandFailure(err, result),
		)
	}
	var unsafe []string
	for _, line := range strings.Split(result.Stdout, "\n") {
		name := strings.ToLower(strings.TrimSpace(line))
		if name == "" {
			continue
		}
		for _, prefix := range forbiddenLocalGitConfigPrefixes {
			if name == prefix || strings.HasPrefix(name, prefix) {
				unsafe = append(unsafe, name)
				break
			}
		}
	}
	sort.Strings(unsafe)
	if len(unsafe) > 0 {
		return fmt.Errorf(
			"repository has forbidden local Git configuration: %s",
			strings.Join(uniqueSorted(unsafe), ", "),
		)
	}
	return nil
}

func auditGitAttributesAtCommit(
	ctx context.Context,
	gitCommand, repository, commit string,
) error {
	list, err := rawGitCommand(
		ctx,
		gitCommand,
		repository,
		"ls-tree",
		"-r",
		"--name-only",
		commit,
		"--",
	)
	if err != nil {
		return fmt.Errorf(
			"list frozen tree attributes: %s",
			commandFailure(err, list),
		)
	}
	var attributeFiles []string
	for _, line := range strings.Split(list.Stdout, "\n") {
		path := filepath.ToSlash(strings.TrimSpace(line))
		if path == ".gitattributes" || strings.HasSuffix(path, "/.gitattributes") {
			attributeFiles = append(attributeFiles, path)
		}
	}
	sort.Strings(attributeFiles)
	for _, path := range attributeFiles {
		blob, err := rawGitCommand(
			ctx,
			gitCommand,
			repository,
			"show",
			commit+":"+path,
		)
		if err != nil {
			return fmt.Errorf(
				"read frozen Git attributes %s: %s",
				path,
				commandFailure(err, blob),
			)
		}
		if unsafe := unsafeGitAttribute(blob.Stdout); unsafe != "" {
			return fmt.Errorf(
				"frozen Git attributes %s enable unsupported driver %q",
				path,
				unsafe,
			)
		}
	}
	return nil
}

func unsafeGitAttribute(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		for _, field := range fields[1:] {
			lower := strings.ToLower(field)
			for _, prefix := range []string{
				"filter=",
				"diff=",
				"merge=",
				"working-tree-encoding=",
			} {
				if strings.HasPrefix(lower, prefix) {
					return field
				}
			}
		}
	}
	return ""
}

func rawGitCommand(
	ctx context.Context,
	gitCommand, repository string,
	args ...string,
) (localCommandResult, error) {
	commandCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	hardened := []string{
		"--no-pager",
		"-c", "core.hooksPath=/dev/null",
		"-c", "core.fsmonitor=false",
		"-c", "credential.helper=",
		"-c", "protocol.file.allow=never",
		"-c", "submodule.recurse=false",
		"-C", repository,
	}
	command := exec.CommandContext(
		commandCtx,
		gitCommand,
		append(hardened, args...)...,
	)
	prepareLocalCommand(command)
	command.Dir = repository
	command.Env = []string{
		"HOME=/nonexistent",
		"PATH=" + runnerSafePath,
		"LANG=C.UTF-8",
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_ATTR_NOSYSTEM=1",
		"GIT_TERMINAL_PROMPT=0",
		"GIT_ASKPASS=/bin/false",
		"SSH_ASKPASS=/bin/false",
	}
	stdout := &localLimitedBuffer{limit: maxLocalCommandOutputBytes}
	stderr := &localLimitedBuffer{limit: maxLocalCommandOutputBytes}
	command.Stdout = stdout
	command.Stderr = stderr
	started := time.Now()
	err := command.Run()
	result := localCommandResult{
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
		ExitCode:   0,
		DurationMS: time.Since(started).Milliseconds(),
		TimedOut:   errors.Is(commandCtx.Err(), context.DeadlineExceeded),
		Truncated:  stdout.truncated || stderr.truncated,
	}
	if err == nil {
		return result, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.ExitCode = exitErr.ExitCode()
	} else {
		result.ExitCode = -1
	}
	if result.TimedOut {
		return result, context.DeadlineExceeded
	}
	return result, err
}

func resolveLocalCommand(directory, name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("command is required")
	}
	if filepath.IsAbs(name) {
		return validateResolvedCommand(name)
	}
	if strings.ContainsRune(name, filepath.Separator) {
		absolute, err := filepath.Abs(filepath.Join(directory, name))
		if err != nil {
			return "", err
		}
		root, err := filepath.Abs(directory)
		if err != nil {
			return "", err
		}
		if absolute != root &&
			!strings.HasPrefix(absolute, root+string(filepath.Separator)) {
			return "", fmt.Errorf("relative command escapes worktree: %s", name)
		}
		return validateResolvedCommand(absolute)
	}
	return findRunnerCommand(name)
}
