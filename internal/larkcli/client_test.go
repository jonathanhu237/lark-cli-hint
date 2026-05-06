package larkcli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseJSONObjectRejectsTopLevelMessageError(t *testing.T) {
	_, err := parseJSONObject(`{"ok":false,"message":"auth failed"}`)
	if err == nil || !strings.Contains(err.Error(), "auth failed") {
		t.Fatalf("expected top-level ok=false error, got %v", err)
	}
}

func TestParseJSONObjectRejectsNestedErrorMessage(t *testing.T) {
	_, err := parseJSONObject(`{"ok":false,"error":{"code":999,"message":"token expired"}}`)
	if err == nil || !strings.Contains(err.Error(), "token expired") {
		t.Fatalf("expected nested ok=false error, got %v", err)
	}
}

func TestParseJSONObjectAcceptsMissingOK(t *testing.T) {
	out, err := parseJSONObject(`{"data":{"items":[]}}`)
	if err != nil {
		t.Fatalf("expected missing ok to be accepted, got %v", err)
	}
	if _, ok := out["data"].(map[string]any); !ok {
		t.Fatalf("parsed data missing: %+v", out)
	}
}

func TestRunJSONDoesNotOverwriteOKFalseWithStderrJSON(t *testing.T) {
	client := &Client{
		binary:  "sh",
		timeout: 0,
	}
	_, err := client.RunJSON(context.Background(), "-c", "printf '%s' '{\"ok\":false,\"error\":{\"message\":\"token expired\"}}'; printf '%s' '{\"ok\":true}' >&2")
	if err == nil || !strings.Contains(err.Error(), "token expired") {
		t.Fatalf("expected stdout ok=false to be preserved, got %v", err)
	}
}

func TestRunJSONPassesProfileBeforeCommand(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "lark-cli")
	if err := os.WriteFile(bin, []byte(`#!/bin/sh
if [ "$*" != "--profile demo docs +search --query FlowOps --format json" ]; then
  printf '{"ok":false,"message":"args=%s"}' "$*" >&2
  exit 1
fi
printf '{"ok":true,"data":{"results":[]}}'
`), 0o755); err != nil {
		t.Fatalf("write helper: %v", err)
	}
	client := NewWithProfile(bin, "demo")
	if _, err := client.RunJSON(context.Background(), "docs", "+search", "--query", "FlowOps", "--format", "json"); err != nil {
		t.Fatalf("RunJSON error: %v", err)
	}
}
