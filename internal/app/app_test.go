package app

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lark-cue/internal/card"
	"lark-cue/internal/config"
	"lark-cue/internal/eval"
	"lark-cue/internal/llm"
	"lark-cue/internal/openclaw"
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

func TestRunHelpWritesStdoutAndSucceeds(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Main([]string{"run", "--help"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if !strings.Contains(stdout.String(), "Usage:") || !strings.Contains(stdout.String(), "--no-openclaw") || strings.Contains(stdout.String(), "--no-feedback-prompt") {
		t.Fatalf("unexpected run help:\n%s", stdout.String())
	}
}

func TestRunRequiresLLMBeforeExecutingCommand(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	marker := filepath.Join(t.TempDir(), "ran")

	var stdout, stderr bytes.Buffer
	code := Main([]string{"run", "--", "sh", "-c", "touch " + marker}, strings.NewReader(""), &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "LLM configuration is required") {
		t.Fatalf("stderr missing LLM config error:\n%s", stderr.String())
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("wrapped command appears to have executed; stat err=%v", err)
	}
}

func TestSendPushRequiresExplicitFlag(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("LARK_CUE_SEND_PUSH_DEFAULT", "true")
	t.Setenv("LARK_CUE_PUSH_CHAT", "oc_should_not_send")
	t.Setenv("LARK_CUE_EVAL_LOG", filepath.Join(t.TempDir(), "eval.jsonl"))
	installRunFakes(t, fakeCueProvider{
		decision: flowOpsDecision(),
		cardErr:  errors.New("use fallback"),
	}, fakeRetriever{sources: flowOpsSources(), status: retrieval.StatusOK})

	var stdout, stderr bytes.Buffer
	code := Main([]string{
		"run",
		"--",
		"sh", "-c", "echo 'FlowOps DAG import error billing_daily billing_region Variable.get' >&2; exit 1",
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

func TestRunPreflightsOpenClawBeforeExecutingCommand(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	installRunFakes(t, fakeCueProvider{}, fakeRetriever{})
	preflightErr := errors.New("openclaw missing")
	newOpenClawClient = func(cfg config.OpenClawConfig) openClawClient {
		return &fakeOpenClawClient{preflightErr: preflightErr}
	}
	marker := filepath.Join(t.TempDir(), "ran")

	var stdout, stderr bytes.Buffer
	code := Main([]string{
		"run",
		"--",
		"sh", "-c", "touch " + marker,
	}, strings.NewReader(""), &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("wrapped command appears to have executed; stat err=%v", err)
	}
	for _, want := range []string{"OpenClaw is required by default", "--no-openclaw"} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("stderr missing %q:\n%s", want, stderr.String())
		}
	}
}

func TestRunNoOpenClawSkipsPreflightAndInvocation(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("LARK_CUE_EVAL_LOG", filepath.Join(t.TempDir(), "eval.jsonl"))
	fakeOpenClaw := installRunFakes(t, fakeCueProvider{
		decision: flowOpsDecision(),
		cardErr:  errors.New("use fallback"),
	}, fakeRetriever{sources: flowOpsSources(), status: retrieval.StatusOK})
	fakeOpenClaw.preflightErr = errors.New("should not be called")

	var stdout, stderr bytes.Buffer
	code := Main([]string{
		"run",
		"--no-openclaw",
		"--",
		"sh", "-c", "echo 'FlowOps DAG import error billing_daily billing_region Variable.get' >&2; exit 1",
	}, strings.NewReader(""), &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want wrapped command exit 1", code)
	}
	if fakeOpenClaw.preflightCalled || fakeOpenClaw.invokeCalled {
		t.Fatalf("OpenClaw should be skipped, preflight=%v invoke=%v", fakeOpenClaw.preflightCalled, fakeOpenClaw.invokeCalled)
	}
	if !strings.Contains(stderr.String(), "lark-cue knowledge card") {
		t.Fatalf("card should still render in card-only mode:\n%s", stderr.String())
	}
}

func TestRunDoesNotPromptForFeedback(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("LARK_CUE_EVAL_LOG", filepath.Join(t.TempDir(), "eval.jsonl"))
	installRunFakes(t, fakeCueProvider{
		decision: flowOpsDecision(),
		cardErr:  errors.New("use fallback"),
	}, fakeRetriever{sources: flowOpsSources(), status: retrieval.StatusOK})

	stdin, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("Open devnull: %v", err)
	}
	defer stdin.Close()

	var stdout, stderr bytes.Buffer
	code := Main([]string{
		"run",
		"--",
		"sh", "-c", "echo 'FlowOps DAG import error billing_daily billing_region Variable.get' >&2; exit 1",
	}, stdin, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want wrapped command exit 1", code)
	}
	if strings.Contains(stdout.String(), "Was this cue useful?") || strings.Contains(stderr.String(), "Was this cue useful?") {
		t.Fatalf("non-TTY stdin printed feedback prompt:\nstdout=%s\nstderr=%s", stdout.String(), stderr.String())
	}
}

func TestRunPlannerSkipDoesNotRetrieveOrRenderCard(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	logPath := filepath.Join(t.TempDir(), "eval.jsonl")
	t.Setenv("LARK_CUE_EVAL_LOG", logPath)
	retriever := &recordingRetriever{}
	installRunFakes(t, fakeCueProvider{
		decision: llm.PlanDecision{
			ShouldRetrieve: false,
			Scenario:       "local file path error",
			Reason:         "missing local file",
		},
	}, retriever)

	var stdout, stderr bytes.Buffer
	code := Main([]string{
		"run",
		"--",
		"sh", "-c", "echo \"python: can't open file missing.py\" >&2; exit 2",
	}, strings.NewReader(""), &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want wrapped command exit 2", code)
	}
	if retriever.called {
		t.Fatal("retriever was called for should_retrieve=false")
	}
	if strings.Contains(stderr.String(), "lark-cue knowledge card") {
		t.Fatalf("skip decision rendered card:\n%s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "no internal knowledge lookup recommended") {
		t.Fatalf("stderr missing skip message:\n%s", stderr.String())
	}
	result, err := eval.ReadCueRecords(logPath)
	if err != nil {
		t.Fatalf("ReadCueRecords error: %v", err)
	}
	if len(result.PlannerRecords) != 1 || result.PlannerRecords[0].ShouldRetrieve == nil || *result.PlannerRecords[0].ShouldRetrieve {
		t.Fatalf("planner record not written correctly: %+v", result.PlannerRecords)
	}
}

func TestRunPlannerTrueWithNoQueriesLogsOriginalDecision(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	logPath := filepath.Join(t.TempDir(), "eval.jsonl")
	t.Setenv("LARK_CUE_EVAL_LOG", logPath)
	retriever := &recordingRetriever{}
	installRunFakes(t, fakeCueProvider{
		decision: llm.PlanDecision{
			ShouldRetrieve: true,
			Scenario:       "FlowOps DAG import error",
			Reason:         "planner returned empty queries",
			Queries:        []string{" ", ""},
		},
	}, retriever)

	var stdout, stderr bytes.Buffer
	code := Main([]string{
		"run",
		"--",
		"sh", "-c", "echo 'FlowOps DAG import error' >&2; exit 1",
	}, strings.NewReader(""), &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want wrapped command exit 1", code)
	}
	if retriever.called {
		t.Fatal("retriever was called with no normalized queries")
	}
	result, err := eval.ReadCueRecords(logPath)
	if err != nil {
		t.Fatalf("ReadCueRecords error: %v", err)
	}
	if len(result.PlannerRecords) != 1 || result.PlannerRecords[0].ShouldRetrieve == nil || !*result.PlannerRecords[0].ShouldRetrieve {
		t.Fatalf("planner record did not preserve original should_retrieve=true: %+v", result.PlannerRecords)
	}
	if result.PlannerRecords[0].QueryCount != 0 {
		t.Fatalf("query count = %d, want 0", result.PlannerRecords[0].QueryCount)
	}
}

func TestRunKeepsKnowledgeCardOffStdout(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("LARK_CUE_EVAL_LOG", filepath.Join(t.TempDir(), "eval.jsonl"))
	installRunFakes(t, fakeCueProvider{
		decision: flowOpsDecision(),
		cardErr:  errors.New("use fallback"),
	}, fakeRetriever{sources: flowOpsSources(), status: retrieval.StatusOK})

	var stdout, stderr bytes.Buffer
	code := Main([]string{
		"run",
		"--",
		"sh", "-c", "echo command-output; echo 'FlowOps DAG import error billing_daily billing_region Variable.get' >&2; exit 1",
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
	planAt := strings.Index(stderr.String(), "lark-cue LLM plan")
	cardAt := strings.Index(stderr.String(), "lark-cue knowledge card")
	if planAt < 0 || cardAt < 0 || planAt > cardAt {
		t.Fatalf("LLM plan should render before final card:\n%s", stderr.String())
	}
	for _, want := range []string{"Reason", "DAG import failure mentions billing_daily", "Queries", "- billing_daily billing_region"} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("stderr missing early LLM plan field %q:\n%s", want, stderr.String())
		}
	}
}

func TestRunInvokesOpenClawAfterCardAndPreservesExitCode(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	logPath := filepath.Join(t.TempDir(), "eval.jsonl")
	t.Setenv("LARK_CUE_EVAL_LOG", logPath)
	fakeOpenClaw := installRunFakes(t, fakeCueProvider{
		decision: flowOpsDecision(),
		cardErr:  errors.New("use fallback"),
	}, fakeRetriever{sources: flowOpsSources(), status: retrieval.StatusOK})
	fakeOpenClaw.output = "openclaw stdout\nopenclaw stderr\n"
	fakeOpenClaw.result = openclaw.Result{Attempted: true, Succeeded: false, ExitCode: 7, Error: "agent failed", LatencyMS: 25}

	var stdout, stderr bytes.Buffer
	code := Main([]string{
		"run",
		"--",
		"sh", "-c", "echo command-output; echo 'FlowOps DAG import error billing_daily billing_region Variable.get' >&2; exit 1",
	}, strings.NewReader(""), &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want wrapped command exit 1", code)
	}
	if strings.TrimSpace(stdout.String()) != "command-output" {
		t.Fatalf("stdout was polluted: %q", stdout.String())
	}
	if !fakeOpenClaw.preflightCalled || !fakeOpenClaw.invokeCalled {
		t.Fatalf("OpenClaw calls = preflight %v invoke %v", fakeOpenClaw.preflightCalled, fakeOpenClaw.invokeCalled)
	}
	cardAt := strings.Index(stderr.String(), "lark-cue knowledge card")
	openClawAt := strings.Index(stderr.String(), "openclaw stdout")
	if cardAt < 0 || openClawAt < 0 || cardAt > openClawAt {
		t.Fatalf("OpenClaw output should appear after card:\n%s", stderr.String())
	}
	for _, want := range []string{
		"Working directory:",
		"Failed command: sh -c",
		"FlowOps DAG Import Error 排障 FAQ",
		"Ask the user before deleting data",
	} {
		if !strings.Contains(fakeOpenClaw.task, want) {
			t.Fatalf("OpenClaw task missing %q:\n%s", want, fakeOpenClaw.task)
		}
	}
	if !strings.Contains(stderr.String(), "OpenClaw handoff failed: agent failed") {
		t.Fatalf("stderr missing OpenClaw failure diagnostic:\n%s", stderr.String())
	}
	result, err := eval.ReadCueRecords(logPath)
	if err != nil {
		t.Fatalf("ReadCueRecords error: %v", err)
	}
	if len(result.Records) != 1 {
		t.Fatalf("cue records len = %d, want 1", len(result.Records))
	}
	record := result.Records[0]
	if record.OpenClawAttempted == nil || !*record.OpenClawAttempted || record.OpenClawSucceeded == nil || *record.OpenClawSucceeded || record.OpenClawError != "agent failed" {
		t.Fatalf("OpenClaw eval fields = %+v", record)
	}
}

func TestRunPlannerSkipDoesNotInvokeOpenClaw(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	fakeOpenClaw := installRunFakes(t, fakeCueProvider{
		decision: llm.PlanDecision{
			ShouldRetrieve: false,
			Scenario:       "local file path error",
			Reason:         "missing local file",
		},
	}, &recordingRetriever{})

	var stdout, stderr bytes.Buffer
	code := Main([]string{
		"run",
		"--",
		"sh", "-c", "echo \"python: can't open file missing.py\" >&2; exit 2",
	}, strings.NewReader(""), &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want wrapped command exit 2", code)
	}
	if !fakeOpenClaw.preflightCalled {
		t.Fatal("default mode should preflight OpenClaw before executing command")
	}
	if fakeOpenClaw.invokeCalled {
		t.Fatal("OpenClaw was invoked for planner skip")
	}
}

func TestRunCapsPlannerQueriesForFeishuSearch(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("LARK_CUE_EVAL_LOG", filepath.Join(t.TempDir(), "eval.jsonl"))
	retriever := &capturingRetriever{sources: flowOpsSources(), status: retrieval.StatusOK}
	installRunFakes(t, fakeCueProvider{
		decision: llm.PlanDecision{
			ShouldRetrieve: true,
			Scenario:       "FlowOps DAG import error",
			Reason:         "planner returned long keyword query",
			Queries: []string{
				"FlowOps DAG import error billing_daily Variable.get billing_region parse time",
				"billing_daily",
			},
		},
		cardErr: errors.New("use fallback"),
	}, retriever)

	var stdout, stderr bytes.Buffer
	code := Main([]string{
		"run",
		"--",
		"sh", "-c", "echo 'FlowOps DAG import error billing_daily billing_region Variable.get' >&2; exit 1",
	}, strings.NewReader(""), &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want wrapped command exit 1", code)
	}
	if len(retriever.queries) != 2 {
		t.Fatalf("retriever queries = %#v, want 2", retriever.queries)
	}
	for _, query := range retriever.queries {
		if len([]rune(query)) > 30 {
			t.Fatalf("query exceeded Feishu search limit: %q", query)
		}
	}
}

func TestRunPlannerTrueWithNoEvidenceStillRendersLowConfidenceCard(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("LARK_CUE_EVAL_LOG", filepath.Join(t.TempDir(), "eval.jsonl"))
	installRunFakes(t, fakeCueProvider{
		decision: flowOpsDecision(),
		cardErr:  errors.New("should not draft without evidence"),
	}, fakeRetriever{status: retrieval.StatusOK})

	var stdout, stderr bytes.Buffer
	code := Main([]string{
		"run",
		"--",
		"sh", "-c", "echo 'FlowOps DAG import error billing_daily billing_region Variable.get' >&2; exit 1",
	}, strings.NewReader(""), &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want wrapped command exit 1", code)
	}
	for _, want := range []string{"lark-cue knowledge card", "Low", "未找到可支撑结论的内部来源"} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("stderr missing %q:\n%s", want, stderr.String())
		}
	}
	if strings.Contains(stderr.String(), "billing_region Variable.get。推荐处理") {
		t.Fatalf("no-evidence card appears to invent retrieved support:\n%s", stderr.String())
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
	for _, want := range []string{"lark-cue validation report", "cue runs: 1", "ok 1", "citation coverage: 1/1", "avg queries/run: 3.0"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}
	if strings.Contains(stdout.String(), "Feedback") {
		t.Fatalf("stdout should not include feedback section:\n%s", stdout.String())
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
	if !strings.Contains(stdout.String(), "No cue or planner records found") {
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

func TestBenchmarkRunMissingCasesExitsTwo(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	var stdout, stderr bytes.Buffer
	code := Main([]string{"benchmark", "run"}, strings.NewReader(""), &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "--cases is required") {
		t.Fatalf("stderr missing --cases error:\n%s", stderr.String())
	}
}

func TestBenchmarkRunPassesAndKeepsNormalEvalLogClean(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	casesPath := writeBenchmarkCases(t, `{"cases":[{
		"id":"flowops-dag-import",
		"command":["sh","-c","echo 'FlowOps DAG import error billing_daily billing_region Variable.get' >&2; exit 1"],
		"expect_failure":true,
		"expected_sources":["FlowOps DAG Import Error 排障 FAQ"],
		"min_expected_hits":1
	}]}`)
	installRunFakes(t, fakeCueProvider{
		decision: flowOpsDecision(),
		cardErr:  errors.New("use fallback"),
	}, fakeRetriever{sources: flowOpsSources(), status: retrieval.StatusOK})

	var stdout, stderr bytes.Buffer
	code := Main([]string{"benchmark", "run", "--cases", casesPath}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0; stderr=%s\nstdout=%s", code, stderr.String(), stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	for _, want := range []string{
		"lark-cue benchmark report",
		"cases: 1/1 passed",
		"expected-source hit rate: 1/1",
		"PASS flowops-dag-import",
		"FlowOps DAG Import Error 排障 FAQ",
		"planner: retrieve",
		"queries: 3",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}
	normalLog := filepath.Join(home, ".lark-cue", "evaluations.jsonl")
	if _, err := os.Stat(normalLog); !os.IsNotExist(err) {
		t.Fatalf("normal eval log was written; stat err=%v", err)
	}
}

func TestBenchmarkRunNoOpenClawSkipsPreflight(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	casesPath := writeBenchmarkCases(t, `{"cases":[{
		"id":"flowops-dag-import",
		"command":["sh","-c","echo 'FlowOps DAG import error billing_daily billing_region Variable.get' >&2; exit 1"],
		"expect_failure":true,
		"expected_sources":["FlowOps DAG Import Error 排障 FAQ"]
	}]}`)
	fakeOpenClaw := installRunFakes(t, fakeCueProvider{
		decision: flowOpsDecision(),
		cardErr:  errors.New("use fallback"),
	}, fakeRetriever{sources: flowOpsSources(), status: retrieval.StatusOK})
	fakeOpenClaw.preflightErr = errors.New("should not be called")

	var stdout, stderr bytes.Buffer
	code := Main([]string{"benchmark", "run", "--no-openclaw", "--cases", casesPath}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0; stderr=%s\nstdout=%s", code, stderr.String(), stdout.String())
	}
	if fakeOpenClaw.preflightCalled || fakeOpenClaw.invokeCalled {
		t.Fatalf("OpenClaw should be skipped, preflight=%v invoke=%v", fakeOpenClaw.preflightCalled, fakeOpenClaw.invokeCalled)
	}
	if !strings.Contains(stdout.String(), "PASS flowops-dag-import") {
		t.Fatalf("stdout missing pass:\n%s", stdout.String())
	}
}

func TestBenchmarkRunExecutesSetupAndTeardown(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := t.TempDir()
	setupMarker := filepath.Join(dir, "setup")
	teardownMarker := filepath.Join(dir, "teardown")
	casesPath := writeBenchmarkCases(t, `{"cases":[{
		"id":"setup-teardown",
		"setup":[["sh","-c","touch '`+setupMarker+`'"]],
		"command":["sh","-c","test -f '`+setupMarker+`'; echo 'FlowOps DAG import error billing_daily billing_region Variable.get' >&2; exit 1"],
		"teardown":[["sh","-c","touch '`+teardownMarker+`'"]],
		"expect_failure":true,
		"expected_sources":["FlowOps DAG Import Error 排障 FAQ"]
	}]}`)
	installRunFakes(t, fakeCueProvider{
		decision: flowOpsDecision(),
		cardErr:  errors.New("use fallback"),
	}, fakeRetriever{sources: flowOpsSources(), status: retrieval.StatusOK})

	var stdout, stderr bytes.Buffer
	code := Main([]string{"benchmark", "run", "--cases", casesPath}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0; stderr=%s\nstdout=%s", code, stderr.String(), stdout.String())
	}
	if _, err := os.Stat(teardownMarker); err != nil {
		t.Fatalf("teardown marker missing: %v", err)
	}
}

func TestBenchmarkRunSetsTemporaryEvalLogForAuxCommands(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	normalLog := filepath.Join(t.TempDir(), "normal-eval.jsonl")
	t.Setenv("LARK_CUE_EVAL_LOG", normalLog)
	envMarker := filepath.Join(t.TempDir(), "eval-env")
	casesPath := writeBenchmarkCases(t, `{"cases":[{
		"id":"aux-env",
		"setup":[["sh","-c","printf %s \"$LARK_CUE_EVAL_LOG\" > \"$1\"","sh","`+envMarker+`"]],
		"command":["sh","-c","echo 'FlowOps DAG import error billing_daily billing_region Variable.get' >&2; exit 1"],
		"expect_failure":true,
		"expected_sources":["FlowOps DAG Import Error 排障 FAQ"]
	}]}`)
	installRunFakes(t, fakeCueProvider{
		decision: flowOpsDecision(),
		cardErr:  errors.New("use fallback"),
	}, fakeRetriever{sources: flowOpsSources(), status: retrieval.StatusOK})

	var stdout, stderr bytes.Buffer
	code := Main([]string{"benchmark", "run", "--cases", casesPath}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0; stderr=%s\nstdout=%s", code, stderr.String(), stdout.String())
	}
	data, err := os.ReadFile(envMarker)
	if err != nil {
		t.Fatalf("read env marker: %v", err)
	}
	seenLog := string(data)
	if seenLog == normalLog || !strings.Contains(seenLog, "lark-cue-benchmark-") {
		t.Fatalf("setup saw eval log %q, want benchmark temp log not normal log %q", seenLog, normalLog)
	}
	if got := os.Getenv("LARK_CUE_EVAL_LOG"); got != normalLog {
		t.Fatalf("LARK_CUE_EVAL_LOG after benchmark = %q, want restored %q", got, normalLog)
	}
	if _, err := os.Stat(normalLog); !os.IsNotExist(err) {
		t.Fatalf("normal eval log was written; stat err=%v", err)
	}
}

func TestBenchmarkRunEvalLogReadErrorExitsTwo(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	casesPath := writeBenchmarkCases(t, `{"cases":[{
		"id":"read-error",
		"command":["sh","-c","echo 'FlowOps DAG import error billing_daily billing_region Variable.get' >&2; exit 1"],
		"expect_failure":true,
		"expected_sources":["FlowOps DAG Import Error 排障 FAQ"]
	}]}`)
	installRunFakes(t, fakeCueProvider{
		decision: flowOpsDecision(),
		cardErr:  errors.New("use fallback"),
	}, fakeRetriever{sources: flowOpsSources(), status: retrieval.StatusOK})
	oldReadCueRecords := readCueRecords
	readCueRecords = func(path string) (eval.ReadResult, error) {
		return eval.ReadResult{}, errors.New("read failed")
	}
	t.Cleanup(func() {
		readCueRecords = oldReadCueRecords
	})

	var stdout, stderr bytes.Buffer
	code := Main([]string{"benchmark", "run", "--cases", casesPath}, strings.NewReader(""), &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2; stderr=%s\nstdout=%s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stderr.String(), "failed to read benchmark evaluation log") {
		t.Fatalf("stderr missing eval read error:\n%s", stderr.String())
	}
	if strings.Contains(stdout.String(), "lark-cue benchmark report") {
		t.Fatalf("runner error rendered case report:\n%s", stdout.String())
	}
}

func TestBenchmarkRunRunsAllCasesAndReturnsOneOnCaseFailure(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	marker := filepath.Join(t.TempDir(), "second")
	casesPath := writeBenchmarkCases(t, `{"cases":[
		{
			"id":"pass",
			"command":["sh","-c","echo 'FlowOps DAG import error billing_daily billing_region Variable.get' >&2; exit 1"],
			"expect_failure":true,
			"expected_sources":["FlowOps DAG Import Error 排障 FAQ"]
		},
		{
			"id":"fail",
			"command":["sh","-c","touch '`+marker+`'; echo 'FlowOps DAG import error billing_daily billing_region Variable.get' >&2; exit 1"],
			"expect_failure":true,
			"expected_sources":["Different FAQ"]
		}
	]}`)
	installRunFakes(t, fakeCueProvider{
		decision: flowOpsDecision(),
		cardErr:  errors.New("use fallback"),
	}, fakeRetriever{sources: flowOpsSources(), status: retrieval.StatusOK})

	var stdout, stderr bytes.Buffer
	code := Main([]string{"benchmark", "run", "--cases", casesPath}, strings.NewReader(""), &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1; stderr=%s\nstdout=%s", code, stderr.String(), stdout.String())
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("second case did not run: %v", err)
	}
	for _, want := range []string{"PASS pass", "FAIL fail", "expected source hits 0 below minimum 1"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestBenchmarkRunFailsMissingCueAndExpectedFailureMismatch(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	casesPath := writeBenchmarkCases(t, `{"cases":[
		{
			"id":"missing-cue",
			"command":["sh","-c","echo 'local failure' >&2; exit 1"],
			"expect_failure":true,
			"expected_sources":["FlowOps DAG Import Error 排障 FAQ"]
		},
		{
			"id":"unexpected-success",
			"command":["sh","-c","exit 0"],
			"expect_failure":true,
			"expected_sources":["FlowOps DAG Import Error 排障 FAQ"]
		}
	]}`)
	installRunFakes(t, fakeCueProvider{
		decision: llm.PlanDecision{
			ShouldRetrieve: false,
			Scenario:       "local failure",
			Reason:         "not an internal knowledge issue",
		},
	}, fakeRetriever{sources: flowOpsSources(), status: retrieval.StatusOK})

	var stdout, stderr bytes.Buffer
	code := Main([]string{"benchmark", "run", "--cases", casesPath}, strings.NewReader(""), &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1; stderr=%s\nstdout=%s", code, stderr.String(), stdout.String())
	}
	for _, want := range []string{"FAIL missing-cue", "no scored card was available", "FAIL unexpected-success", "expected command failure but command exited 0"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}
}

func installRunFakes(t *testing.T, provider fakeCueProvider, retriever retrieval.Retriever) *fakeOpenClawClient {
	t.Helper()
	oldProvider := newPlannerProvider
	oldRetriever := newRetriever
	oldOpenClaw := newOpenClawClient
	fakeOpenClaw := &fakeOpenClawClient{
		result: openclaw.Result{Attempted: true, Succeeded: true, ExitCode: 0},
	}
	newPlannerProvider = func(cfg config.LLMConfig) (cueProvider, error) {
		return provider, nil
	}
	newRetriever = func(cfg config.FeishuConfig) retrieval.Retriever {
		return retriever
	}
	newOpenClawClient = func(cfg config.OpenClawConfig) openClawClient {
		fakeOpenClaw.cfg = cfg
		return fakeOpenClaw
	}
	t.Cleanup(func() {
		newPlannerProvider = oldProvider
		newRetriever = oldRetriever
		newOpenClawClient = oldOpenClaw
	})
	return fakeOpenClaw
}

func writeBenchmarkCases(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "cases.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("WriteFile cases: %v", err)
	}
	return path
}

type fakeCueProvider struct {
	decision llm.PlanDecision
	planErr  error
	draft    llm.CardDraft
	cardErr  error
}

func (f fakeCueProvider) PlanRetrieval(ctx context.Context, input llm.PlanInput) (llm.PlanDecision, error) {
	if f.planErr != nil {
		return llm.PlanDecision{}, f.planErr
	}
	return f.decision, nil
}

func (f fakeCueProvider) GenerateCard(ctx context.Context, input llm.CardInput) (llm.CardDraft, error) {
	if f.cardErr != nil {
		return llm.CardDraft{}, f.cardErr
	}
	if strings.TrimSpace(f.draft.LikelyCause) != "" || strings.TrimSpace(f.draft.Caveat) != "" || len(f.draft.ActionPlan) > 0 {
		return f.draft, nil
	}
	return llm.CardDraft{}, errors.New("empty draft")
}

type fakeRetriever struct {
	sources []retrieval.Source
	status  retrieval.Status
	err     error
}

func (f fakeRetriever) Retrieve(ctx context.Context, queries []string) ([]retrieval.Source, retrieval.Status, error) {
	return f.sources, f.status, f.err
}

type recordingRetriever struct {
	called bool
}

func (r *recordingRetriever) Retrieve(ctx context.Context, queries []string) ([]retrieval.Source, retrieval.Status, error) {
	r.called = true
	return nil, retrieval.StatusFailed, errors.New("should not be called")
}

type capturingRetriever struct {
	queries []string
	sources []retrieval.Source
	status  retrieval.Status
	err     error
}

type fakeOpenClawClient struct {
	cfg             config.OpenClawConfig
	preflightErr    error
	preflightCalled bool
	invokeCalled    bool
	task            string
	output          string
	result          openclaw.Result
}

func (f *fakeOpenClawClient) Preflight(ctx context.Context) error {
	f.preflightCalled = true
	return f.preflightErr
}

func (f *fakeOpenClawClient) Invoke(ctx context.Context, task string, stderr io.Writer) openclaw.Result {
	f.invokeCalled = true
	f.task = task
	if f.output != "" {
		_, _ = io.WriteString(stderr, f.output)
	}
	result := f.result
	if !result.Attempted {
		result.Attempted = true
	}
	return result
}

func (r *capturingRetriever) Retrieve(ctx context.Context, queries []string) ([]retrieval.Source, retrieval.Status, error) {
	r.queries = append([]string(nil), queries...)
	return r.sources, r.status, r.err
}

func flowOpsDecision() llm.PlanDecision {
	return llm.PlanDecision{
		ShouldRetrieve: true,
		Scenario:       "FlowOps DAG import error",
		Reason:         "DAG import failure mentions billing_daily and billing_region Variable lookup.",
		Queries: []string{
			"FlowOps DAG import error billing_daily",
			"billing_daily billing_region Variable.get",
			"flowctl dags list-import-errors",
		},
	}
}

func flowOpsSources() []retrieval.Source {
	return []retrieval.Source{{
		Type:    "doc",
		Title:   "FlowOps DAG Import Error 排障 FAQ",
		URL:     "https://example.test/flowops",
		Content: "FlowOps DAG import error billing_daily billing_region Variable.get。推荐处理：把 Variable.get 移到任务运行阶段，然后执行 flowctl dags list-import-errors 验证。",
		Fetched: true,
	}}
}
