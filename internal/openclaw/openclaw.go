package openclaw

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"lark-cue/internal/config"
	"lark-cue/internal/runner"
)

const DefaultAgent = "main"

type CommandRunner func(ctx context.Context, command []string, streams runner.Streams) (runner.Result, error)

type Client struct {
	cfg config.OpenClawConfig
	run CommandRunner
}

type Result struct {
	Attempted bool
	Succeeded bool
	TimedOut  bool
	ExitCode  int
	Error     string
	LatencyMS int64
	Command   []string
}

func New(cfg config.OpenClawConfig) *Client {
	return NewWithRunner(cfg, runner.Run)
}

func NewWithRunner(cfg config.OpenClawConfig, run CommandRunner) *Client {
	if run == nil {
		run = runner.Run
	}
	return &Client{cfg: normalizeConfig(cfg), run: run}
}

func (c *Client) Preflight(ctx context.Context) error {
	command := []string{c.cfg.Binary, "agent", "--help"}
	result, err := c.run(ctx, command, runner.Streams{
		Stdout: io.Discard,
		Stderr: io.Discard,
		Buffer: runner.NewBoundedBuffer(8 * 1024),
	})
	if err != nil {
		return fmt.Errorf("OpenClaw preflight failed: %w", err)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("OpenClaw preflight exited %d", result.ExitCode)
	}
	return nil
}

func (c *Client) Invoke(ctx context.Context, task string, stderr io.Writer) Result {
	if stderr == nil {
		stderr = io.Discard
	}
	command := c.invokeCommand(task)
	started := time.Now()
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(c.cfg.TimeoutSeconds)*time.Second)
	defer cancel()

	result := Result{
		Attempted: true,
		ExitCode:  -1,
		Command:   append([]string(nil), command...),
	}
	runResult, err := c.run(timeoutCtx, command, runner.Streams{
		Stdout: stderr,
		Stderr: stderr,
		Buffer: runner.NewBoundedBuffer(8 * 1024),
	})
	result.LatencyMS = time.Since(started).Milliseconds()
	result.ExitCode = runResult.ExitCode
	if errors.Is(timeoutCtx.Err(), context.DeadlineExceeded) {
		result.TimedOut = true
		result.Error = "OpenClaw invocation timed out"
		return result
	}
	if err != nil {
		result.Error = err.Error()
		return result
	}
	if runResult.ExitCode != 0 {
		result.Error = fmt.Sprintf("OpenClaw exited %d", runResult.ExitCode)
		return result
	}
	result.Succeeded = true
	return result
}

func (c *Client) invokeCommand(task string) []string {
	command := []string{c.cfg.Binary, "agent"}
	command = append(command, "--local")
	command = append(command,
		"--agent", DefaultAgent,
		"--timeout", fmt.Sprintf("%d", c.cfg.TimeoutSeconds),
		"--message", task,
	)
	return command
}

func normalizeConfig(cfg config.OpenClawConfig) config.OpenClawConfig {
	cfg.Binary = strings.TrimSpace(cfg.Binary)
	if cfg.Binary == "" {
		cfg.Binary = "openclaw"
	}
	if cfg.TimeoutSeconds <= 0 {
		cfg.TimeoutSeconds = 900
	}
	return cfg
}
