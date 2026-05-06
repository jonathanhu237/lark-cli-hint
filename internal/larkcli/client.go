package larkcli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type Client struct {
	binary  string
	profile string
	timeout time.Duration
}

func New(binary string) *Client {
	return &Client{binary: binary, timeout: 20 * time.Second}
}

func NewWithProfile(binary string, profile string) *Client {
	client := New(binary)
	client.profile = strings.TrimSpace(profile)
	return client
}

func (c *Client) RunJSON(ctx context.Context, args ...string) (map[string]any, error) {
	if c.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}
	args = c.args(args...)
	cmd := exec.CommandContext(ctx, c.binary, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	raw := stdout.String()
	if raw == "" {
		raw = stderr.String()
	}
	if err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = strings.TrimSpace(stdout.String())
		}
		return nil, fmt.Errorf("lark-cli %s failed: %w: %s", strings.Join(args, " "), err, detail)
	}
	parsed, parseErr := parseJSONObject(raw)
	if parseErr != nil && !isSemanticCLIError(parseErr) && stderr.Len() > 0 {
		parsed, parseErr = parseJSONObject(stderr.String())
	}
	if parseErr != nil {
		return nil, fmt.Errorf("parse lark-cli JSON for %s: %w", strings.Join(args, " "), parseErr)
	}
	return parsed, nil
}

func (c *Client) args(args ...string) []string {
	if strings.TrimSpace(c.profile) == "" {
		return args
	}
	out := []string{"--profile", strings.TrimSpace(c.profile)}
	out = append(out, args...)
	return out
}

func isSemanticCLIError(err error) bool {
	var semantic semanticError
	return errors.As(err, &semantic)
}

type semanticError struct {
	message string
}

func (e semanticError) Error() string {
	return e.message
}

func parseJSONObject(raw string) (map[string]any, error) {
	start := strings.IndexByte(raw, '{')
	end := strings.LastIndexByte(raw, '}')
	if start < 0 || end < start {
		return nil, errors.New("no JSON object in output")
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(raw[start:end+1]), &out); err != nil {
		return nil, err
	}
	if ok, exists := out["ok"].(bool); exists && !ok {
		return nil, semanticError{message: errorMessage(out)}
	}
	return out, nil
}

func errorMessage(out map[string]any) string {
	if msg, _ := out["message"].(string); strings.TrimSpace(msg) != "" {
		return msg
	}
	if errorObj, _ := out["error"].(map[string]any); errorObj != nil {
		if msg, _ := errorObj["message"].(string); strings.TrimSpace(msg) != "" {
			return msg
		}
		if code, ok := errorObj["code"]; ok {
			return fmt.Sprintf("lark-cli error: %v", code)
		}
	}
	encoded, err := json.Marshal(out)
	if err == nil {
		return string(encoded)
	}
	return "lark-cli returned ok=false"
}
