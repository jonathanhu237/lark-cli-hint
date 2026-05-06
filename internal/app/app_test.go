package app

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lark-cue/internal/card"
	"lark-cue/internal/config"
	"lark-cue/internal/eval"
	"lark-cue/internal/llm"
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
		"--no-feedback-prompt",
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

func TestRunWithDevNullDoesNotPromptForFeedback(t *testing.T) {
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
		"--no-feedback-prompt",
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
		"--no-feedback-prompt",
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

func installRunFakes(t *testing.T, provider fakeCueProvider, retriever retrieval.Retriever) {
	t.Helper()
	oldProvider := newPlannerProvider
	oldRetriever := newRetriever
	newPlannerProvider = func(cfg config.LLMConfig) (cueProvider, error) {
		return provider, nil
	}
	newRetriever = func(cfg config.FeishuConfig) retrieval.Retriever {
		return retriever
	}
	t.Cleanup(func() {
		newPlannerProvider = oldProvider
		newRetriever = oldRetriever
	})
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
	if strings.TrimSpace(f.draft.LikelyCause) != "" || strings.TrimSpace(f.draft.Caveat) != "" {
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
