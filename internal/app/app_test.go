package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lark-cue/internal/card"
	"lark-cue/internal/eval"
	"lark-cue/internal/retrieval"
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

func TestEvalReportReadsLogAndWritesStdoutOnly(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	logPath := filepath.Join(t.TempDir(), "eval.jsonl")
	t.Setenv("LARK_CUE_EVAL_LOG", logPath)
	if err := eval.Append(logPath, eval.FromCard(card.KnowledgeCard{
		ID:              "cue_test",
		Command:         "node x.js",
		Scenario:        "scenario",
		RetrievalStatus: retrieval.StatusOK,
		Citations: []card.Citation{{
			Type:  "doc",
			Title: "Guide",
		}},
		LatencyMS:  1200,
		QueryCount: 3,
		Feedback:   "skipped",
	})); err != nil {
		t.Fatalf("Append error: %v", err)
	}
	if err := eval.AppendFeedback(logPath, "cue_test", "useful"); err != nil {
		t.Fatalf("AppendFeedback error: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := Main([]string{"eval", "report", "--limit", "10"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0; stderr=%s", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	for _, want := range []string{"lark-cue validation report", "cue runs: 1", "ok 1", "citation coverage: 1/1", "useful: 1"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}
	if strings.Contains(stdout.String(), "\x1b[") {
		t.Fatalf("non-TTY report output included ANSI: %q", stdout.String())
	}
}

func TestEvalReportEmptyLogAndInvalidLimit(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("LARK_CUE_EVAL_LOG", filepath.Join(t.TempDir(), "missing.jsonl"))

	var stdout, stderr bytes.Buffer
	code := Main([]string{"eval", "report"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0; stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "No cue records found") {
		t.Fatalf("empty report missing message:\n%s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"eval", "report", "--limit", "0"}, strings.NewReader(""), &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "--limit requires a positive integer") {
		t.Fatalf("stderr missing limit error:\n%s", stderr.String())
	}
}
