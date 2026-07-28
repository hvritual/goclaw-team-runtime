package orchestratorlite

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCodexExecHandParsesJSONLEvidence(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is POSIX-only")
	}
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	require.NoError(t, os.MkdirAll(repo, 0o755))
	runGitTest(t, repo, "init", "-b", "main")
	require.NoError(t, os.WriteFile(filepath.Join(repo, "README.md"), []byte("test\n"), 0o644))
	runGitTest(t, repo, "add", "README.md")
	runGitTest(t, repo, "-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "-m", "initial")

	script := filepath.Join(root, "fake-codex")
	scriptContent := `#!/bin/sh
printf '%s\n' '{"type":"thread.started","thread_id":"thread-123"}'
printf '%s\n' '{"type":"item.completed","item":{"type":"agent_message","text":"done"}}'
printf '%s\n' '{"type":"turn.completed","usage":{"input_tokens":12,"cached_input_tokens":3,"output_tokens":4,"reasoning_output_tokens":2}}'
`
	require.NoError(t, os.WriteFile(script, []byte(scriptContent), 0o755))

	hand := NewCodexExecHand(Config{CodexCommand: script, RunTimeoutSeconds: 30})
	events := filepath.Join(root, "events.jsonl")
	result, err := hand.Execute(context.Background(), HandRequest{
		Task:       Task{WorktreePath: repo},
		Prompt:     "do work",
		EventsPath: events,
	})
	require.NoError(t, err)
	require.Equal(t, "thread-123", result.ThreadID)
	require.Equal(t, "done", result.FinalText)
	require.Equal(t, 12, result.Usage.InputTokens)
	require.Equal(t, 4, result.Usage.OutputTokens)
	require.FileExists(t, events)
}

func TestCodexExecHandUsesMinimalIsolatedEnvironment(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is POSIX-only")
	}
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	require.NoError(t, os.MkdirAll(repo, 0o755))
	runGitTest(t, repo, "init", "-b", "main")

	codexHome := filepath.Join(root, "real-codex-home")
	require.NoError(t, os.MkdirAll(codexHome, 0o700))
	t.Setenv("CODEX_HOME", codexHome)
	t.Setenv("LANG", "C.UTF-8")
	t.Setenv("HTTPS_PROXY", "http://proxy.example.test:8080")
	t.Setenv("SSL_CERT_FILE", filepath.Join(root, "ca.pem"))
	for name, value := range map[string]string{
		"SSH_AUTH_SOCK":                  filepath.Join(root, "agent.sock"),
		"DOCKER_HOST":                    "unix:///var/run/docker.sock",
		"DOCKER_CONFIG":                  filepath.Join(root, "docker"),
		"KUBECONFIG":                     filepath.Join(root, "kubeconfig"),
		"AWS_ACCESS_KEY_ID":              "sensitive-access-key",
		"AWS_SECRET_ACCESS_KEY":          "sensitive-secret-key",
		"AZURE_CLIENT_SECRET":            "sensitive-client-secret",
		"GOOGLE_APPLICATION_CREDENTIALS": filepath.Join(root, "google.json"),
		"OPENAI_API_KEY":                 "sensitive-openai-token",
		"ANTHROPIC_API_KEY":              "sensitive-anthropic-token",
		"GITHUB_TOKEN":                   "sensitive-github-token",
	} {
		t.Setenv(name, value)
	}

	capturedEnvironment := filepath.Join(root, "captured-environment")
	script := filepath.Join(root, "fake-codex")
	scriptContent := strings.Join([]string{
		"#!/bin/sh",
		"env > " + shellSingleQuote(capturedEnvironment),
		`printf '%s\n' '{"type":"thread.started","thread_id":"thread-isolated"}'`,
		`printf '%s\n' '{"type":"item.completed","item":{"type":"agent_message","text":"done"}}'`,
		"",
	}, "\n")
	require.NoError(t, os.WriteFile(script, []byte(scriptContent), 0o755))

	events := filepath.Join(root, "run", "events.jsonl")
	hand := NewCodexExecHand(Config{
		CodexCommand:      script,
		RunTimeoutSeconds: 30,
	})
	_, err := hand.Execute(context.Background(), HandRequest{
		Task:       Task{WorktreePath: repo},
		RunID:      "run-isolated",
		Prompt:     "do work",
		EventsPath: events,
	})
	require.NoError(t, err)

	captured, err := os.ReadFile(capturedEnvironment)
	require.NoError(t, err)
	environment := parseEnvironment(string(captured))
	require.Equal(t, codexHome, environment["CODEX_HOME"])
	require.Equal(t, os.Getenv("PATH"), environment["PATH"])
	require.Equal(t, "C.UTF-8", environment["LANG"])
	require.Equal(t, "http://proxy.example.test:8080", environment["HTTPS_PROXY"])
	require.Equal(t, filepath.Join(root, "ca.pem"), environment["SSL_CERT_FILE"])

	isolatedHome := environment["HOME"]
	require.NotEmpty(t, isolatedHome)
	require.NotEqual(t, os.Getenv("HOME"), isolatedHome)
	runtimeRoot := filepath.Dir(isolatedHome)
	require.Equal(t, filepath.Dir(events), filepath.Dir(runtimeRoot))
	require.True(t, strings.HasPrefix(filepath.Base(runtimeRoot), ".codex-runtime-"))
	for _, name := range []string{
		"HOME",
		"XDG_CONFIG_HOME",
		"XDG_CACHE_HOME",
		"XDG_DATA_HOME",
		"XDG_STATE_HOME",
		"XDG_RUNTIME_DIR",
		"TMPDIR",
		"TMP",
		"TEMP",
	} {
		require.Truef(
			t,
			environment[name] == runtimeRoot ||
				strings.HasPrefix(environment[name], runtimeRoot+string(filepath.Separator)),
			"%s must be isolated under %s; got %s",
			name,
			runtimeRoot,
			environment[name],
		)
	}
	_, statErr := os.Stat(runtimeRoot)
	require.ErrorIs(t, statErr, os.ErrNotExist)

	for _, name := range []string{
		"SSH_AUTH_SOCK",
		"DOCKER_HOST",
		"DOCKER_CONFIG",
		"KUBECONFIG",
		"AWS_ACCESS_KEY_ID",
		"AWS_SECRET_ACCESS_KEY",
		"AZURE_CLIENT_SECRET",
		"GOOGLE_APPLICATION_CREDENTIALS",
		"OPENAI_API_KEY",
		"ANTHROPIC_API_KEY",
		"GITHUB_TOKEN",
	} {
		_, present := environment[name]
		require.Falsef(t, present, "%s must not be inherited", name)
	}
}

func TestCurrentCodexHomeFallsBackToUserHome(t *testing.T) {
	t.Setenv("CODEX_HOME", "")
	userHome, err := os.UserHomeDir()
	require.NoError(t, err)

	codexHome, err := currentCodexHome()
	require.NoError(t, err)
	require.Equal(t, filepath.Join(userHome, ".codex"), codexHome)
	require.True(t, filepath.IsAbs(codexHome))
}

func shellSingleQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

func parseEnvironment(content string) map[string]string {
	result := make(map[string]string)
	for _, line := range strings.Split(strings.TrimSpace(content), "\n") {
		name, value, ok := strings.Cut(line, "=")
		if ok {
			result[name] = value
		}
	}
	return result
}
