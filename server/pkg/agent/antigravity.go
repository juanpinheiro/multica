package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"
)

// antigravityBackend implements Backend by spawning the Antigravity CLI
// with --output-format stream-json and parsing its NDJSON event stream.
// The protocol is similar to Gemini's stream-json format.
type antigravityBackend struct {
	cfg Config
	// resultTeardownGrace overrides defaultResultTeardownGrace. Zero means use
	// the default; tests set a small value to keep the hung-after-result path
	// fast.
	resultTeardownGrace time.Duration
}

func (b *antigravityBackend) Execute(ctx context.Context, prompt string, opts ExecOptions) (*Session, error) {
	execPath := b.cfg.ExecutablePath
	if execPath == "" {
		execPath = "antigravity"
	}
	if _, err := exec.LookPath(execPath); err != nil {
		return nil, fmt.Errorf("antigravity executable not found at %q: %w", execPath, err)
	}

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 20 * time.Minute
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)

	args := buildAntigravityArgs(prompt, opts, b.cfg.Logger)

	cmd := exec.CommandContext(runCtx, execPath, args...)
	hideAgentWindow(cmd)
	b.cfg.Logger.Info("agent command", "exec", execPath, "args", args)
	cmd.WaitDelay = 10 * time.Second
	if opts.Cwd != "" {
		cmd.Dir = opts.Cwd
	}
	cmd.Env = buildEnv(b.cfg.Env)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("antigravity stdout pipe: %w", err)
	}
	cmd.Stderr = newLogWriter(b.cfg.Logger, "[antigravity:stderr] ")

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("start antigravity: %w", err)
	}

	b.cfg.Logger.Info("antigravity started", "pid", cmd.Process.Pid, "cwd", opts.Cwd, "model", opts.Model)

	msgCh := make(chan Message, 256)
	resCh := make(chan Result, 1)

	go func() {
		<-runCtx.Done()
		_ = stdout.Close()
	}()

	go func() {
		defer cancel()
		defer close(msgCh)
		defer close(resCh)

		startTime := time.Now()
		var output strings.Builder
		var sessionID string
		finalStatus := "completed"
		var finalError string
		resultSeen := false
		usage := make(map[string]TokenUsage)

		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}

			var evt antigravityStreamEvent
			if err := json.Unmarshal([]byte(line), &evt); err != nil {
				continue
			}

			switch evt.Type {
			case "init":
				if evt.SessionID != "" {
					sessionID = evt.SessionID
				}
				trySend(msgCh, Message{Type: MessageStatus, Status: "running"})

			case "message":
				if evt.Role == "assistant" && evt.Content != "" {
					output.WriteString(evt.Content)
					trySend(msgCh, Message{Type: MessageText, Content: evt.Content})
				}

			case "tool_use":
				var params map[string]any
				if evt.Parameters != nil {
					_ = json.Unmarshal(evt.Parameters, &params)
				}
				trySend(msgCh, Message{
					Type:   MessageToolUse,
					Tool:   evt.ToolName,
					CallID: evt.ToolID,
					Input:  params,
				})

			case "tool_result":
				trySend(msgCh, Message{
					Type:   MessageToolResult,
					CallID: evt.ToolID,
					Output: evt.Output,
				})

			case "error":
				errText := evt.Message
				trySend(msgCh, Message{Type: MessageError, Content: errText})
				if finalStatus == "completed" {
					finalStatus = "failed"
					finalError = errText
				}

			case "result":
				if evt.Status == "error" && evt.Error != nil {
					finalStatus = "failed"
					finalError = evt.Error.Message
				}
				if evt.Stats != nil {
					b.accumulateUsage(usage, evt.Stats)
				}
				// The result is the authoritative completion signal. Tear the
				// process down proactively so a CLI that emits its result and
				// then hangs resolves now instead of blocking the reader until
				// the idle watchdog.
				if !resultSeen {
					resultSeen = true
					scheduleResultTeardown(runCtx, cancel, resolveResultTeardownGrace(b.resultTeardownGrace))
				}
			}
		}

		waitErr := cmd.Wait()
		duration := time.Since(startTime)

		// A run that emitted a result has already recorded its real disposition.
		// Don't let the teardown cancel — or a non-zero exit from the force-kill
		// that follows it — clobber that into aborted/failed.
		if !resultSeen {
			if runCtx.Err() == context.DeadlineExceeded {
				finalStatus = "timeout"
				finalError = fmt.Sprintf("antigravity timed out after %s", timeout)
			} else if runCtx.Err() == context.Canceled {
				finalStatus = "aborted"
				finalError = "execution cancelled"
			} else if waitErr != nil && finalStatus == "completed" {
				finalStatus = "failed"
				finalError = fmt.Sprintf("antigravity exited with error: %v", waitErr)
			}
		}

		b.cfg.Logger.Info("antigravity finished", "pid", cmd.Process.Pid, "status", finalStatus, "duration", duration.Round(time.Millisecond).String())

		resCh <- Result{
			Status:     finalStatus,
			Output:     output.String(),
			Error:      finalError,
			DurationMs: duration.Milliseconds(),
			SessionID:  sessionID,
			Usage:      usage,
		}
	}()

	return &Session{Messages: msgCh, Result: resCh}, nil
}

func (b *antigravityBackend) accumulateUsage(usage map[string]TokenUsage, stats *antigravityStreamStats) {
	for model, m := range stats.Models {
		u := usage[model]
		u.InputTokens += int64(m.InputTokens)
		u.OutputTokens += int64(m.OutputTokens)
		u.CacheReadTokens += int64(m.Cached)
		usage[model] = u
	}
}

// ── Antigravity stream-json event types ──

type antigravityStreamEvent struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id,omitempty"`
	Model     string `json:"model,omitempty"`

	// message fields
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`

	// tool_use fields
	ToolName   string          `json:"tool_name,omitempty"`
	ToolID     string          `json:"tool_id,omitempty"`
	Parameters json.RawMessage `json:"parameters,omitempty"`

	// tool_result fields
	Output string `json:"output,omitempty"`

	// error fields
	Status  string `json:"status,omitempty"`
	Message string `json:"message,omitempty"`

	// result fields
	Error *antigravityStreamError `json:"error,omitempty"`
	Stats *antigravityStreamStats `json:"stats,omitempty"`
}

type antigravityStreamError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type antigravityStreamStats struct {
	TotalTokens  int                             `json:"total_tokens"`
	InputTokens  int                             `json:"input_tokens"`
	OutputTokens int                             `json:"output_tokens"`
	Models       map[string]antigravityModelStats `json:"models,omitempty"`
}

type antigravityModelStats struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	Cached       int `json:"cached"`
}

// ── Arg builder ──

// antigravityBlockedArgs are flags hardcoded by the daemon that must not be
// overridden by user-configured custom_args. Overriding these would break
// the daemon↔antigravity communication protocol.
var antigravityBlockedArgs = map[string]blockedArgMode{
	"-p":              blockedStandalone, // non-interactive prompt flag
	"--output-format": blockedWithValue,  // stream-json protocol
	"--yolo":          blockedStandalone, // auto-approve tool use
}

// buildAntigravityArgs assembles the argv for a one-shot antigravity invocation.
//
// Flags:
//
//	-p <prompt>                non-interactive prompt
//	--output-format stream-json streaming NDJSON output
//	--yolo                     auto-approve all tool executions
//	--model <id>               optional model override
//	--system-prompt <s>        extra system instructions
//	--resume <id>              resume a previous session
func buildAntigravityArgs(prompt string, opts ExecOptions, logger *slog.Logger) []string {
	args := []string{
		"-p", prompt,
		"--output-format", "stream-json",
		"--yolo",
	}
	if opts.Model != "" {
		args = append(args, "--model", opts.Model)
	}
	if opts.SystemPrompt != "" {
		args = append(args, "--system-prompt", opts.SystemPrompt)
	}
	if opts.ResumeSessionID != "" {
		args = append(args, "--resume", opts.ResumeSessionID)
	}
	args = append(args, filterCustomArgs(opts.CustomArgs, antigravityBlockedArgs, logger)...)
	return args
}
