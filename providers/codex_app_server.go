package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/smallnest/goclaw/config"
)

const (
	defaultCodexCommand = "codex"
	defaultCodexTimeout = 120 * time.Second
)

// CodexAppServerProvider adapts the local Codex app-server protocol to
// GoClaw's LLM provider interface. Codex owns authentication, so this provider
// works with `codex login` and ChatGPT subscription entitlements without
// copying OAuth tokens into GoClaw.
//
// A fresh, ephemeral, read-only Codex thread is used for every provider call.
// GoClaw remains the tool orchestrator: Codex is instructed to return a
// structured decision envelope rather than executing tools itself.
type CodexAppServerProvider struct {
	command    string
	args       []string
	workingDir string
	model      string
	maxTokens  int
	timeout    time.Duration
}

type codexRPCMessage struct {
	ID     json.RawMessage `json:"id,omitempty"`
	Method string          `json:"method,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *codexRPCError  `json:"error,omitempty"`
}

type codexRPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

type codexDecision struct {
	Content      string     `json:"content"`
	ToolCalls    []ToolCall `json:"tool_calls"`
	FinishReason string     `json:"finish_reason"`
}

func NewCodexAppServerProvider(model string, maxTokens int, runtime *config.ModelRuntimeConfig) (*CodexAppServerProvider, error) {
	command := defaultCodexCommand
	args := []string{"app-server", "--listen", "stdio://"}
	workingDir := ""
	timeout := defaultCodexTimeout
	if runtime != nil {
		if strings.TrimSpace(runtime.Command) != "" {
			command = runtime.Command
		}
		if len(runtime.Args) > 0 {
			args = append([]string(nil), runtime.Args...)
		}
		workingDir = runtime.WorkingDir
		if runtime.TimeoutSeconds > 0 {
			timeout = time.Duration(runtime.TimeoutSeconds) * time.Second
		}
	}
	if strings.TrimSpace(model) == "" {
		model = "default"
	}
	if workingDir != "" {
		absolute, err := filepath.Abs(workingDir)
		if err != nil {
			return nil, fmt.Errorf("resolve codex working directory: %w", err)
		}
		workingDir = absolute
	}
	return &CodexAppServerProvider{
		command:    command,
		args:       args,
		workingDir: workingDir,
		model:      model,
		maxTokens:  maxTokens,
		timeout:    timeout,
	}, nil
}

func (p *CodexAppServerProvider) Chat(ctx context.Context, messages []Message, tools []ToolDefinition, options ...ChatOption) (*Response, error) {
	opts := &ChatOptions{Model: p.model, MaxTokens: p.maxTokens}
	for _, option := range options {
		option(opts)
	}
	if strings.TrimSpace(opts.Model) == "" {
		opts.Model = p.model
	}

	callCtx := ctx
	cancel := func() {}
	if p.timeout > 0 {
		callCtx, cancel = context.WithTimeout(ctx, p.timeout)
	}
	defer cancel()

	prompt, err := buildCodexDecisionPrompt(messages, tools, opts)
	if err != nil {
		return nil, err
	}
	decision, usage, err := p.run(callCtx, prompt, opts.Model)
	if err != nil {
		if errors.Is(callCtx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("codex app-server timed out after %s: %w", p.timeout, err)
		}
		return nil, err
	}
	if decision.FinishReason == "" {
		if len(decision.ToolCalls) > 0 {
			decision.FinishReason = "tool_calls"
		} else {
			decision.FinishReason = "stop"
		}
	}
	for i := range decision.ToolCalls {
		if decision.ToolCalls[i].ID == "" {
			decision.ToolCalls[i].ID = "call_" + uuid.NewString()
		}
	}
	return &Response{
		Content:      decision.Content,
		ToolCalls:    decision.ToolCalls,
		FinishReason: decision.FinishReason,
		Usage:        usage,
	}, nil
}

func (p *CodexAppServerProvider) ChatWithTools(ctx context.Context, messages []Message, tools []ToolDefinition, options ...ChatOption) (*Response, error) {
	return p.Chat(ctx, messages, tools, options...)
}

func (p *CodexAppServerProvider) Close() error { return nil }

func buildCodexDecisionPrompt(messages []Message, tools []ToolDefinition, opts *ChatOptions) (string, error) {
	messageJSON, err := json.Marshal(messages)
	if err != nil {
		return "", fmt.Errorf("encode messages for codex: %w", err)
	}
	toolJSON, err := json.Marshal(tools)
	if err != nil {
		return "", fmt.Errorf("encode tools for codex: %w", err)
	}
	var b strings.Builder
	b.WriteString("You are the language-model decision engine embedded in GoClaw. ")
	b.WriteString("Do not run commands, edit files, browse, or invoke your own tools. ")
	b.WriteString("Read the conversation and the available GoClaw tool definitions below. ")
	b.WriteString("Return one JSON object matching the requested schema. ")
	b.WriteString("If a GoClaw tool is needed, put it in tool_calls and do not pretend it already ran. ")
	b.WriteString("When the conversation contains tool results, use them to produce the next decision. ")
	if opts.MaxTokens > 0 {
		fmt.Fprintf(&b, "Keep the answer within approximately %d output tokens. ", opts.MaxTokens)
	}
	b.WriteString("\n\nCONVERSATION_JSON:\n")
	b.Write(messageJSON)
	b.WriteString("\n\nGOCLAW_TOOLS_JSON:\n")
	b.Write(toolJSON)
	return b.String(), nil
}

func codexOutputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"content": map[string]interface{}{"type": "string"},
			"tool_calls": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"id":     map[string]interface{}{"type": "string"},
						"name":   map[string]interface{}{"type": "string"},
						"params": map[string]interface{}{"type": "object", "additionalProperties": true},
					},
					"required":             []string{"id", "name", "params"},
					"additionalProperties": false,
				},
			},
			"finish_reason": map[string]interface{}{"type": "string"},
		},
		"required":             []string{"content", "tool_calls", "finish_reason"},
		"additionalProperties": false,
	}
}

func (p *CodexAppServerProvider) run(ctx context.Context, prompt, model string) (*codexDecision, Usage, error) {
	decisionDir, err := os.MkdirTemp("", "goclaw-codex-decision-*")
	if err != nil {
		return nil, Usage{}, fmt.Errorf("create isolated codex decision directory: %w", err)
	}
	defer os.RemoveAll(decisionDir)

	cmd := exec.CommandContext(ctx, p.command, p.args...)
	if p.workingDir != "" {
		cmd.Dir = p.workingDir
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, Usage{}, fmt.Errorf("open codex stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, Usage{}, fmt.Errorf("open codex stdout: %w", err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return nil, Usage{}, fmt.Errorf("codex executable %q was not found; install Codex CLI and run `codex login`: %w", p.command, err)
		}
		return nil, Usage{}, fmt.Errorf("start codex app-server: %w", err)
	}

	var closeOnce sync.Once
	closeInput := func() { closeOnce.Do(func() { _ = stdin.Close() }) }
	defer func() {
		closeInput()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	}()

	send := func(value interface{}) error {
		data, marshalErr := json.Marshal(value)
		if marshalErr != nil {
			return marshalErr
		}
		data = append(data, '\n')
		if _, writeErr := stdin.Write(data); writeErr != nil {
			return fmt.Errorf("write codex request: %w", writeErr)
		}
		return nil
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	if err := send(map[string]interface{}{
		"method": "initialize",
		"id":     1,
		"params": map[string]interface{}{
			"clientInfo": map[string]string{
				"name":    "goclaw",
				"title":   "GoClaw Codex Runtime Adapter",
				"version": "0.1.0",
			},
		},
	}); err != nil {
		return nil, Usage{}, err
	}
	if _, err := readCodexResponse(scanner, send, 1, nil); err != nil {
		return nil, Usage{}, withCodexStderr("initialize codex app-server", err, stderr.String())
	}
	if err := send(map[string]interface{}{"method": "initialized", "params": map[string]interface{}{}}); err != nil {
		return nil, Usage{}, err
	}
	threadParams := map[string]interface{}{
		"ephemeral": true,
		"sandbox":   "read-only",
		"cwd":       decisionDir,
	}
	if model != "" && model != "default" {
		threadParams["model"] = model
	}
	if err := send(map[string]interface{}{
		"method": "thread/start",
		"id":     2,
		"params": threadParams,
	}); err != nil {
		return nil, Usage{}, err
	}
	threadResponse, err := readCodexResponse(scanner, send, 2, nil)
	if err != nil {
		return nil, Usage{}, withCodexStderr("start codex thread", err, stderr.String())
	}
	var threadResult struct {
		Thread struct {
			ID string `json:"id"`
		} `json:"thread"`
	}
	if err := json.Unmarshal(threadResponse, &threadResult); err != nil || threadResult.Thread.ID == "" {
		return nil, Usage{}, withCodexStderr("decode codex thread response", err, stderr.String())
	}

	turnParams := map[string]interface{}{
		"threadId": threadResult.Thread.ID,
		"input": []map[string]string{
			{"type": "text", "text": prompt},
		},
		"sandboxPolicy": map[string]interface{}{"type": "readOnly"},
		"outputSchema":  codexOutputSchema(),
	}
	if model != "" && model != "default" {
		turnParams["model"] = model
	}
	turnParams["cwd"] = decisionDir
	if err := send(map[string]interface{}{"method": "turn/start", "id": 3, "params": turnParams}); err != nil {
		return nil, Usage{}, err
	}

	var finalText strings.Builder
	completed := false
	var usage Usage
	consumeNotification := func(message codexRPCMessage) error {
		if message.Method == "" {
			return nil
		}
		switch message.Method {
		case "item/agentMessage/delta":
			var params struct {
				Delta string `json:"delta"`
			}
			if json.Unmarshal(message.Params, &params) == nil {
				finalText.WriteString(params.Delta)
			}
		case "item/completed":
			if text := completedAgentMessage(message.Params); text != "" {
				finalText.Reset()
				finalText.WriteString(text)
			}
		case "turn/completed":
			var params map[string]interface{}
			if err := json.Unmarshal(message.Params, &params); err != nil {
				return fmt.Errorf("decode codex turn completion: %w", err)
			}
			turn, _ := params["turn"].(map[string]interface{})
			if errValue, ok := turn["error"].(map[string]interface{}); ok {
				return fmt.Errorf("codex turn failed: %v", errValue["message"])
			}
			status, _ := turn["status"].(string)
			switch status {
			case "failed", "interrupted", "cancelled":
				return fmt.Errorf("codex turn ended with status %s", status)
			}
			usage = parseCodexUsage(turn["usage"])
			if usage.TotalTokens == 0 {
				usage = parseCodexUsage(params["usage"])
			}
			completed = true
		}
		return nil
	}
	_, err = readCodexResponse(scanner, send, 3, consumeNotification)
	if err != nil {
		return nil, Usage{}, withCodexStderr("run codex turn", err, stderr.String())
	}
	for !completed && scanner.Scan() {
		var message codexRPCMessage
		if err := json.Unmarshal(scanner.Bytes(), &message); err != nil {
			continue
		}
		if err := consumeNotification(message); err != nil {
			return nil, Usage{}, withCodexStderr("run codex turn", err, stderr.String())
		}
	}
	if !completed {
		if ctx.Err() != nil {
			return nil, Usage{}, ctx.Err()
		}
		return nil, Usage{}, withCodexStderr("wait for codex turn", scanner.Err(), stderr.String())
	}

	var decision codexDecision
	raw := strings.TrimSpace(finalText.String())
	if err := json.Unmarshal([]byte(raw), &decision); err != nil {
		return nil, Usage{}, fmt.Errorf("decode structured codex response: %w (response: %s)", err, truncate(raw, 500))
	}
	return &decision, usage, nil
}

func parseCodexUsage(value interface{}) Usage {
	raw, _ := value.(map[string]interface{})
	input := intNumber(raw, "inputTokens", "input_tokens", "promptTokens", "prompt_tokens")
	output := intNumber(raw, "outputTokens", "output_tokens", "completionTokens", "completion_tokens")
	total := intNumber(raw, "totalTokens", "total_tokens")
	if total == 0 {
		total = input + output
	}
	return Usage{PromptTokens: input, CompletionTokens: output, TotalTokens: total}
}

func intNumber(values map[string]interface{}, keys ...string) int {
	for _, key := range keys {
		switch value := values[key].(type) {
		case float64:
			return int(value)
		case int:
			return value
		case json.Number:
			parsed, _ := value.Int64()
			return int(parsed)
		}
	}
	return 0
}

// readCodexResponse reads until the requested response id. Notifications are
// passed to onNotification, while server-initiated requests are rejected so a
// read-only model decision can never stall on an unhandled approval.
func readCodexResponse(scanner *bufio.Scanner, send func(interface{}) error, wantedID int, onNotification func(codexRPCMessage) error) (json.RawMessage, error) {
	for scanner.Scan() {
		var message codexRPCMessage
		if err := json.Unmarshal(scanner.Bytes(), &message); err != nil {
			continue
		}
		if message.Method != "" && len(message.ID) > 0 {
			var id interface{}
			_ = json.Unmarshal(message.ID, &id)
			if err := send(map[string]interface{}{
				"id": id,
				"error": map[string]interface{}{
					"code":    -32000,
					"message": "GoClaw Codex adapter does not permit server-initiated actions",
				},
			}); err != nil {
				return nil, err
			}
			continue
		}
		if message.Method != "" {
			if onNotification != nil {
				if err := onNotification(message); err != nil {
					return nil, err
				}
			}
			continue
		}
		var id int
		if len(message.ID) > 0 && json.Unmarshal(message.ID, &id) == nil && id == wantedID {
			if message.Error != nil {
				return nil, fmt.Errorf("codex RPC error %d: %s", message.Error.Code, message.Error.Message)
			}
			return message.Result, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return nil, io.ErrUnexpectedEOF
}

func completedAgentMessage(raw json.RawMessage) string {
	var params struct {
		Item struct {
			Type    string `json:"type"`
			Text    string `json:"text"`
			Content string `json:"content"`
		} `json:"item"`
	}
	if json.Unmarshal(raw, &params) != nil || params.Item.Type != "agentMessage" {
		return ""
	}
	if params.Item.Text != "" {
		return params.Item.Text
	}
	return params.Item.Content
}

func withCodexStderr(operation string, err error, stderr string) error {
	if err == nil {
		err = errors.New("unknown error")
	}
	stderr = strings.TrimSpace(stderr)
	if stderr == "" {
		return fmt.Errorf("%s: %w", operation, err)
	}
	return fmt.Errorf("%s: %w (stderr: %s)", operation, err, truncate(stderr, 800))
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max] + "…"
}
