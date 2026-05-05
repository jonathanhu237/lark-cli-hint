package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunMissingCommandIsClear(t *testing.T) {
	var stderr bytes.Buffer
	code := Main([]string{"run"}, strings.NewReader(""), &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "missing wrapped command") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunRequiresSeparator(t *testing.T) {
	var stderr bytes.Buffer
	code := Main([]string{"run", "echo", "ok"}, strings.NewReader(""), &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "missing --") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestSendPushRequiresExplicitFlag(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("LARK_CUE_SEND_PUSH_DEFAULT", "true")
	t.Setenv("LARK_CUE_PUSH_CHAT", "oc_should_not_send")
	t.Setenv("LARK_CUE_EVAL_LOG", filepath.Join(t.TempDir(), "eval.jsonl"))

	var stdout, stderr bytes.Buffer
	code := Main([]string{
		"run",
		"--demo-fixture",
		"--no-feedback-prompt",
		"--",
		"sh", "-c", "echo 'missing required scope: docx:document:read' >&2; exit 1",
	}, strings.NewReader(""), &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want wrapped command exit 1", code)
	}
	if !strings.Contains(stderr.String(), "Push Preview") {
		t.Fatalf("expected config default to prepare preview, stderr=%q", stderr.String())
	}
	if strings.Contains(stderr.String(), "Feishu push sent") || strings.Contains(stderr.String(), "failed to send Feishu push") || strings.Contains(stderr.String(), "push send requested") {
		t.Fatalf("config default attempted send, stderr=%q", stderr.String())
	}
}

func TestRunWithDevNullDoesNotPromptForFeedback(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("LARK_CUE_EVAL_LOG", filepath.Join(t.TempDir(), "eval.jsonl"))

	stdin, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("Open devnull: %v", err)
	}
	defer stdin.Close()

	var stdout, stderr bytes.Buffer
	code := Main([]string{
		"run",
		"--demo-fixture",
		"--",
		"sh", "-c", "echo 'missing required scope: docx:document:read' >&2; exit 1",
	}, stdin, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want wrapped command exit 1", code)
	}
	if strings.Contains(stdout.String(), "Was this cue useful?") || strings.Contains(stderr.String(), "Was this cue useful?") {
		t.Fatalf("non-TTY stdin printed feedback prompt:\nstdout=%s\nstderr=%s", stdout.String(), stderr.String())
	}
}

func TestRunDetectsFeishuSignalInTruncatedMiddleOutput(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("LARK_CUE_EVAL_LOG", filepath.Join(t.TempDir(), "eval.jsonl"))

	var stdout, stderr bytes.Buffer
	code := Main([]string{
		"run",
		"--demo-fixture",
		"--no-feedback-prompt",
		"--",
		"sh", "-c", "printf '%140000s' '' | tr ' ' x; echo 'LarkApiError: missing required scope: docx:document:read'; printf '%140000s' '' | tr ' ' y; exit 1",
	}, strings.NewReader(""), &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want wrapped command exit 1", code)
	}
	if !strings.Contains(stderr.String(), "detected Feishu API auth/scope/token error") {
		t.Fatalf("middle Feishu signal was not detected, stderr=%q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "lark-cue knowledge card") {
		t.Fatalf("middle Feishu signal did not produce card")
	}
}

func TestRunKeepsKnowledgeCardOffStdout(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("LARK_CUE_EVAL_LOG", filepath.Join(t.TempDir(), "eval.jsonl"))

	var stdout, stderr bytes.Buffer
	code := Main([]string{
		"run",
		"--demo-fixture",
		"--no-feedback-prompt",
		"--",
		"sh", "-c", "echo command-output; echo 'missing required scope: docx:document:read' >&2; exit 1",
	}, strings.NewReader(""), &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want wrapped command exit 1", code)
	}
	if strings.TrimSpace(stdout.String()) != "command-output" {
		t.Fatalf("stdout was polluted by cue output: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "lark-cue knowledge card") {
		t.Fatalf("expected cue on stderr, got %q", stderr.String())
	}
}
