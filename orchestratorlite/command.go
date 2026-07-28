package orchestratorlite

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

const maxCapturedCommandBytes = 16 * 1024 * 1024

type commandResult struct {
	Stdout     string
	Stderr     string
	ExitCode   int
	DurationMS int64
	TimedOut   bool
}

type limitedBuffer struct {
	buffer bytes.Buffer
	limit  int
}

func (b *limitedBuffer) Write(data []byte) (int, error) {
	original := len(data)
	remaining := b.limit - b.buffer.Len()
	if remaining > 0 {
		if len(data) > remaining {
			data = data[:remaining]
		}
		_, _ = b.buffer.Write(data)
	}
	return original, nil
}

func (b *limitedBuffer) String() string {
	return b.buffer.String()
}

func runCommand(ctx context.Context, dir, stdin string, env []string, name string, args ...string) (commandResult, error) {
	var environment []string
	if env != nil {
		environment = environmentOverrides(os.Environ(), env)
	}
	return runCommandWithEnvironment(
		ctx,
		dir,
		stdin,
		environment,
		name,
		args...,
	)
}

func runCommandWithEnvironment(
	ctx context.Context,
	dir, stdin string,
	environment []string,
	name string,
	args ...string,
) (commandResult, error) {
	if name == "" {
		return commandResult{}, errors.New("command name is required")
	}
	start := time.Now()
	command := exec.CommandContext(ctx, name, args...)
	command.Dir = dir
	if stdin != "" {
		command.Stdin = bytes.NewBufferString(stdin)
	}
	if environment != nil {
		command.Env = environment
	}
	stdout := &limitedBuffer{limit: maxCapturedCommandBytes}
	stderr := &limitedBuffer{limit: maxCapturedCommandBytes}
	command.Stdout = stdout
	command.Stderr = stderr
	err := command.Run()
	result := commandResult{
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
		ExitCode:   0,
		DurationMS: time.Since(start).Milliseconds(),
		TimedOut:   errors.Is(ctx.Err(), context.DeadlineExceeded),
	}
	if err == nil {
		return result, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.ExitCode = exitErr.ExitCode()
		return result, fmt.Errorf("%s exited with code %d", name, result.ExitCode)
	}
	result.ExitCode = -1
	if errors.Is(err, exec.ErrNotFound) {
		return result, fmt.Errorf("executable %q was not found: %w", name, err)
	}
	if ctx.Err() != nil {
		return result, ctx.Err()
	}
	return result, err
}

func environmentOverrides(environment, overrides []string) []string {
	names := make(map[string]struct{}, len(overrides))
	for _, entry := range overrides {
		name, _, found := strings.Cut(entry, "=")
		if found {
			names[strings.ToUpper(strings.TrimSpace(name))] = struct{}{}
		}
	}
	result := make([]string, 0, len(environment)+len(overrides))
	for _, entry := range environment {
		name, _, found := strings.Cut(entry, "=")
		if !found {
			continue
		}
		if _, overridden := names[strings.ToUpper(strings.TrimSpace(name))]; overridden {
			continue
		}
		result = append(result, entry)
	}
	return append(result, overrides...)
}

func copyLimited(dst io.Writer, src io.Reader, limit int64) error {
	_, err := io.Copy(dst, io.LimitReader(src, limit))
	return err
}
