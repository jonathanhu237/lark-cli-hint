package openclaw

import (
	"bytes"
	"context"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"

	"lark-cue/internal/card"
	"lark-cue/internal/config"
	"lark-cue/internal/evidence"
	"lark-cue/internal/retrieval"
	"lark-cue/internal/runner"
)

func TestPreflightUsesAgentHelp(t *testing.T) {
	var got []string
	client := NewWithRunner(config.OpenClawConfig{Binary: "oc", TimeoutSeconds: 30}, func(ctx context.Context, command []string, streams runner.Streams) (runner.Result, error) {
		got = append([]string(nil), command...)
		return runner.Result{}, nil
	})

	if err := client.Preflight(context.Background()); err != nil {
		t.Fatalf("Preflight error: %v", err)
	}
	want := []string{"oc", "agent", "--help"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("command = %#v, want %#v", got, want)
	}
}

func TestPreflightReportsFailures(t *testing.T) {
	client := NewWithRunner(config.OpenClawConfig{}, func(ctx context.Context, command []string, streams runner.Streams) (runner.Result, error) {
		return runner.Result{}, errors.New("missing binary")
	})
	if err := client.Preflight(context.Background()); err == nil || !strings.Contains(err.Error(), "missing binary") {
		t.Fatalf("Preflight err = %v, want missing binary", err)
	}

	client = NewWithRunner(config.OpenClawConfig{}, func(ctx context.Context, command []string, streams runner.Streams) (runner.Result, error) {
		return runner.Result{ExitCode: 2}, nil
	})
	if err := client.Preflight(context.Background()); err == nil || !strings.Contains(err.Error(), "exited 2") {
		t.Fatalf("Preflight err = %v, want exit code", err)
	}
}

func TestInvokeUsesLocalMainAgentAndRoutesOutput(t *testing.T) {
	var got []string
	client := NewWithRunner(config.OpenClawConfig{Binary: "oc", TimeoutSeconds: 900}, func(ctx context.Context, command []string, streams runner.Streams) (runner.Result, error) {
		got = append([]string(nil), command...)
		_, _ = io.WriteString(streams.Stdout, "stdout\n")
		_, _ = io.WriteString(streams.Stderr, "stderr\n")
		_, _ = io.WriteString(streams.Buffer, "stdout\nstderr\n")
		return runner.Result{ExitCode: 0}, nil
	})

	var out bytes.Buffer
	result := client.Invoke(context.Background(), "fix this", &out)
	if !result.Attempted || !result.Succeeded || result.Error != "" {
		t.Fatalf("result = %+v, want success", result)
	}
	wantPrefix := []string{"oc", "agent", "--local", "--agent", "main", "--timeout", "900", "--message", "fix this"}
	if !reflect.DeepEqual(got, wantPrefix) {
		t.Fatalf("command = %#v, want %#v", got, wantPrefix)
	}
	if out.String() != "stdout\nstderr\n" {
		t.Fatalf("routed output = %q", out.String())
	}
	if result.Output != "stdout\nstderr\n" {
		t.Fatalf("captured output = %q", result.Output)
	}
}

func TestInvokeReportsFailureAndTimeout(t *testing.T) {
	client := NewWithRunner(config.OpenClawConfig{}, func(ctx context.Context, command []string, streams runner.Streams) (runner.Result, error) {
		return runner.Result{ExitCode: 7}, nil
	})
	result := client.Invoke(context.Background(), "task", io.Discard)
	if result.Succeeded || result.ExitCode != 7 || !strings.Contains(result.Error, "exited 7") {
		t.Fatalf("result = %+v, want exit failure", result)
	}

	client = NewWithRunner(config.OpenClawConfig{}, func(ctx context.Context, command []string, streams runner.Streams) (runner.Result, error) {
		<-ctx.Done()
		return runner.Result{}, ctx.Err()
	})
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	result = client.Invoke(ctx, "task", io.Discard)
	if !result.TimedOut || result.Succeeded || !strings.Contains(result.Error, "timed out") {
		t.Fatalf("result = %+v, want timeout", result)
	}
}

func TestBuildTaskIncludesContextEvidenceAndConstraints(t *testing.T) {
	task := BuildTask(TaskInput{
		WorkingDir:      "/repo",
		Command:         []string{"flowctl", "check", "billing_daily"},
		ExitCode:        1,
		Output:          "FlowOps DAG import error billing_region Variable.get",
		PlannerScenario: "FlowOps DAG import error",
		PlannerReason:   "planner matched internal FlowOps docs",
		Queries:         []string{"billing_daily billing_region"},
		Card: card.KnowledgeCard{
			Scenario:    "FlowOps DAG import error",
			LikelyCause: "billing_daily reads billing_region during DAG parse time",
			ActionPlan: []string{
				"Move Variable.get to runtime.",
				"Run flowctl dags list-import-errors.",
			},
			Confidence: evidence.ConfidenceHigh,
		},
		Evidence: []evidence.ScoredSource{{
			Source: retrieval.Source{
				Type:  "doc",
				Title: "FlowOps DAG Import Error 排障 FAQ",
				URL:   "https://example.test/flowops",
			},
			Snippet: "推荐处理：把 Variable.get 移到任务运行阶段，然后执行 flowctl dags list-import-errors 验证。",
		}},
	})

	for _, want := range []string{
		"Working directory: /repo",
		"Failed command: flowctl check billing_daily",
		"Exit code: 1",
		"FlowOps DAG import error billing_region Variable.get",
		"planner matched internal FlowOps docs",
		"billing_daily billing_region",
		"billing_daily reads billing_region",
		"Move Variable.get to runtime.",
		"FlowOps DAG Import Error 排障 FAQ",
		"https://example.test/flowops",
		"Inspect files, configs, and command outputs before editing.",
		"Rerun the failed command or an equivalent verification before finishing.",
		"Ask the user before deleting data",
	} {
		if !strings.Contains(task, want) {
			t.Fatalf("task missing %q:\n%s", want, task)
		}
	}
}

func TestRenderResultShowsStatusDetailsAndOutputExcerpt(t *testing.T) {
	rendered := RenderResult(Result{
		Succeeded: true,
		ExitCode:  0,
		LatencyMS: 1250,
		Command:   []string{"openclaw", "agent", "--local"},
		Output:    strings.Join([]string{"line1", "line2", "line3"}, "\n"),
	}, 1)

	for _, want := range []string{
		"OpenClaw result",
		"Status",
		"Succeeded",
		"agent: main",
		"OpenClaw exit code: 0",
		"duration: 1.2s",
		"wrapped command exit preserved: 1",
		"Output excerpt",
		"line3",
		"Next",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered result missing %q:\n%s", want, rendered)
		}
	}
}
