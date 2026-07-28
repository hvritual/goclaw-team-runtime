package orchestratorlite

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// safeRuntimeEnvironmentNames is deliberately narrow. In particular,
// credentials, agent sockets, Docker/Kubernetes configuration, and cloud/API
// tokens are not inherited by Codex or sandboxed verification subprocesses.
var safeRuntimeEnvironmentNames = []string{
	"PATH",
	"LANG",
	"LANGUAGE",
	"LC_ALL",
	"LC_CTYPE",
	"TERM",
	"COLORTERM",
	"TZ",
	"HTTP_PROXY",
	"HTTPS_PROXY",
	"ALL_PROXY",
	"NO_PROXY",
	"http_proxy",
	"https_proxy",
	"all_proxy",
	"no_proxy",
	"SSL_CERT_FILE",
	"SSL_CERT_DIR",
	"CURL_CA_BUNDLE",
	"REQUESTS_CA_BUNDLE",
	"NODE_EXTRA_CA_CERTS",
	"NIX_SSL_CERT_FILE",
	"GIT_SSL_CAINFO",
	"GIT_SSL_CAPATH",
	"GRPC_DEFAULT_SSL_ROOTS_FILE_PATH",
	"GOROOT",
	"GOPATH",
	"GOMODCACHE",
	"GOCACHE",
	"GOENV",
	"GOPROXY",
	"GONOPROXY",
	"GONOSUMDB",
	"GOPRIVATE",
	"SYSTEMROOT",
	"WINDIR",
	"COMSPEC",
	"PATHEXT",
}

type Hand interface {
	Execute(ctx context.Context, request HandRequest) (HandResult, error)
	Review(ctx context.Context, request ReviewRequest) (IndependentReview, HandResult, error)
}

type HandRequest struct {
	Task             Task
	RunID            string
	Prompt           string
	ResumeThreadID   string
	EventsPath       string
	ReadOnly         bool
	OutputSchemaPath string
}

type ReviewRequest struct {
	Task       Task
	RunID      string
	Diff       string
	Evidence   []VerificationResult
	EventsPath string
}

type CodexExecHand struct {
	command string
	model   string
	timeout time.Duration
}

func NewCodexExecHand(cfg Config) *CodexExecHand {
	return &CodexExecHand{
		command: cfg.CodexCommand,
		model:   cfg.CodexModel,
		timeout: time.Duration(cfg.RunTimeoutSeconds) * time.Second,
	}
}

func (h *CodexExecHand) Execute(ctx context.Context, request HandRequest) (HandResult, error) {
	return h.run(ctx, request)
}

func (h *CodexExecHand) Review(ctx context.Context, request ReviewRequest) (IndependentReview, HandResult, error) {
	schemaPath := request.EventsPath + ".review-schema.json"
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"passed":         map[string]any{"type": "boolean"},
			"summary":        map[string]any{"type": "string"},
			"findings":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"required_fixes": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		},
		"required":             []string{"passed", "summary", "findings", "required_fixes"},
		"additionalProperties": false,
	}
	if err := writeJSONAtomic(schemaPath, schema, 0o600); err != nil {
		return IndependentReview{}, HandResult{}, err
	}
	defer os.Remove(schemaPath)
	taskJSON, _ := json.MarshalIndent(request.Task, "", "  ")
	verificationJSON, _ := json.MarshalIndent(request.Evidence, "", "  ")
	prompt := strings.Join([]string{
		"You are the independent checker. Do not modify files.",
		"Review the frozen task contract, actual diff, and deterministic verification evidence.",
		"Reject scope drift, unverifiable completion claims, missing tests, security regressions, or changes that do not satisfy the goal.",
		"Return only the requested structured review.",
		"\nFROZEN_TASK:\n" + string(taskJSON),
		"\nDIFF:\n" + request.Diff,
		"\nVERIFICATION:\n" + string(verificationJSON),
	}, "\n")
	handResult, err := h.run(ctx, HandRequest{
		Task:             request.Task,
		RunID:            request.RunID + "-review",
		Prompt:           prompt,
		EventsPath:       request.EventsPath,
		ReadOnly:         true,
		OutputSchemaPath: schemaPath,
	})
	if err != nil {
		return IndependentReview{}, handResult, err
	}
	var review IndependentReview
	if err := json.Unmarshal([]byte(strings.TrimSpace(handResult.FinalText)), &review); err != nil {
		return IndependentReview{}, handResult, fmt.Errorf("decode independent review: %w", err)
	}
	review.ThreadID = handResult.ThreadID
	return review, handResult, nil
}

func (h *CodexExecHand) run(ctx context.Context, request HandRequest) (HandResult, error) {
	if request.Task.WorktreePath == "" {
		return HandResult{}, errors.New("worktree path is required")
	}
	runCtx := ctx
	cancel := func() {}
	if h.timeout > 0 {
		runCtx, cancel = context.WithTimeout(ctx, h.timeout)
	}
	defer cancel()

	args := []string{"--ask-for-approval", "never", "exec", "--json"}
	if request.ReadOnly {
		args = append(args, "--sandbox", "read-only")
	} else {
		args = append(args, "--sandbox", "workspace-write")
	}
	if h.model != "" && h.model != "default" {
		args = append(args, "--model", h.model)
	}
	if request.OutputSchemaPath != "" {
		args = append(args, "--output-schema", request.OutputSchemaPath)
	}
	if request.ResumeThreadID != "" {
		args = append(args, "resume", request.ResumeThreadID, "-")
	} else {
		args = append(args, "-")
	}

	environment, cleanupEnvironment, err := isolatedCodexEnvironment(request.EventsPath)
	if err != nil {
		return HandResult{}, fmt.Errorf("prepare isolated Codex environment: %w", err)
	}
	defer cleanupEnvironment()

	command := exec.CommandContext(runCtx, h.command, args...)
	command.Dir = request.Task.WorktreePath
	command.Env = environment
	command.Stdin = strings.NewReader(request.Prompt)
	stdout, err := command.StdoutPipe()
	if err != nil {
		return HandResult{}, err
	}
	stderr := &limitedBuffer{limit: maxCapturedCommandBytes}
	command.Stderr = stderr
	if err := command.Start(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return HandResult{}, fmt.Errorf("codex executable %q not found; install Codex CLI and run `codex login`: %w", h.command, err)
		}
		return HandResult{}, err
	}
	eventFile, err := os.OpenFile(request.EventsPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		_ = command.Process.Kill()
		_ = command.Wait()
		return HandResult{}, err
	}
	result := HandResult{}
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	var raw strings.Builder
	for scanner.Scan() {
		line := append([]byte(nil), scanner.Bytes()...)
		_, _ = eventFile.Write(append(line, '\n'))
		if raw.Len()+len(line) < maxCapturedCommandBytes {
			raw.Write(line)
			raw.WriteByte('\n')
		}
		var event map[string]any
		if json.Unmarshal(line, &event) != nil {
			continue
		}
		eventType, _ := event["type"].(string)
		switch eventType {
		case "thread.started":
			result.ThreadID, _ = event["thread_id"].(string)
		case "item.completed":
			item, _ := event["item"].(map[string]any)
			if itemType, _ := item["type"].(string); itemType == "agent_message" {
				if text, _ := item["text"].(string); text != "" {
					result.FinalText = text
				}
			}
		case "turn.completed":
			usage, _ := event["usage"].(map[string]any)
			result.Usage = CodexUsage{
				InputTokens:           intValue(usage["input_tokens"]),
				CachedInputTokens:     intValue(usage["cached_input_tokens"]),
				OutputTokens:          intValue(usage["output_tokens"]),
				ReasoningOutputTokens: intValue(usage["reasoning_output_tokens"]),
			}
		}
	}
	closeErr := eventFile.Close()
	waitErr := command.Wait()
	result.Stdout = raw.String()
	result.Stderr = stderr.String()
	if closeErr != nil {
		return result, closeErr
	}
	if scanner.Err() != nil {
		return result, scanner.Err()
	}
	if waitErr != nil {
		var exitErr *exec.ExitError
		if errors.As(waitErr, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
		}
		if runCtx.Err() != nil {
			return result, runCtx.Err()
		}
		return result, fmt.Errorf("codex exec failed with code %d: %s", result.ExitCode, strings.TrimSpace(result.Stderr))
	}
	return result, nil
}

func isolatedCodexEnvironment(eventsPath string) ([]string, func(), error) {
	if strings.TrimSpace(eventsPath) == "" {
		return nil, nil, errors.New("events path is required")
	}
	eventsDir, err := filepath.Abs(filepath.Dir(eventsPath))
	if err != nil {
		return nil, nil, fmt.Errorf("resolve events directory: %w", err)
	}
	if err := os.MkdirAll(eventsDir, 0o755); err != nil {
		return nil, nil, fmt.Errorf("create events directory: %w", err)
	}
	runtimeRoot, err := os.MkdirTemp(eventsDir, ".codex-runtime-")
	if err != nil {
		return nil, nil, fmt.Errorf("create runtime directory: %w", err)
	}
	cleanup := func() {
		_ = os.RemoveAll(runtimeRoot)
	}

	directories := map[string]string{
		"HOME":            filepath.Join(runtimeRoot, "home"),
		"XDG_CONFIG_HOME": filepath.Join(runtimeRoot, "xdg-config"),
		"XDG_CACHE_HOME":  filepath.Join(runtimeRoot, "xdg-cache"),
		"XDG_DATA_HOME":   filepath.Join(runtimeRoot, "xdg-data"),
		"XDG_STATE_HOME":  filepath.Join(runtimeRoot, "xdg-state"),
		"XDG_RUNTIME_DIR": filepath.Join(runtimeRoot, "xdg-runtime"),
		"TMPDIR":          filepath.Join(runtimeRoot, "tmp"),
		"TMP":             filepath.Join(runtimeRoot, "tmp"),
		"TEMP":            filepath.Join(runtimeRoot, "tmp"),
	}
	created := make(map[string]struct{})
	for _, directory := range directories {
		if _, ok := created[directory]; ok {
			continue
		}
		if err := os.Mkdir(directory, 0o700); err != nil {
			cleanup()
			return nil, nil, fmt.Errorf("create isolated directory: %w", err)
		}
		created[directory] = struct{}{}
	}

	codexHome, err := currentCodexHome()
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	environment := inheritedSafeRuntimeEnvironment()
	if strings.TrimSpace(os.Getenv("PATH")) == "" {
		cleanup()
		return nil, nil, errors.New("PATH is required")
	}
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
		environment = append(environment, name+"="+directories[name])
	}
	environment = append(environment, "CODEX_HOME="+codexHome)
	return environment, cleanup, nil
}

func inheritedSafeRuntimeEnvironment() []string {
	environment := make([]string, 0, len(safeRuntimeEnvironmentNames))
	for _, name := range safeRuntimeEnvironmentNames {
		if value, ok := os.LookupEnv(name); ok {
			environment = append(environment, name+"="+value)
		}
	}
	return environment
}

func currentCodexHome() (string, error) {
	path := strings.TrimSpace(os.Getenv("CODEX_HOME"))
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve user home for Codex OAuth: %w", err)
		}
		path = filepath.Join(home, ".codex")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve Codex OAuth directory: %w", err)
	}
	return filepath.Clean(absolute), nil
}

func intValue(value any) int {
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case int:
		return typed
	case json.Number:
		number, _ := typed.Int64()
		return int(number)
	default:
		return 0
	}
}
