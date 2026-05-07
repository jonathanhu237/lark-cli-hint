package benchmark

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"lark-cue/internal/card"
	"lark-cue/internal/eval"
)

func TestLoadCasesRequiresPath(t *testing.T) {
	if _, err := LoadCases(""); err == nil || !strings.Contains(err.Error(), "--cases is required") {
		t.Fatalf("LoadCases empty err = %v, want --cases error", err)
	}
}

func TestLoadCasesRejectsMalformedJSON(t *testing.T) {
	path := writeTempFile(t, "{")
	if _, err := LoadCases(path); err == nil || !strings.Contains(err.Error(), "parse cases JSON") {
		t.Fatalf("LoadCases malformed err = %v, want parse error", err)
	}
}

func TestLoadCasesRejectsInvalidEntries(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "duplicate ids",
			body: `{"cases":[
				{"id":"a","command":["echo","x"],"expected_sources":["Guide"]},
				{"id":"a","command":["echo","y"],"expected_sources":["Guide"]}
			]}`,
			want: "duplicate id",
		},
		{
			name: "empty command",
			body: `{"cases":[{"id":"a","command":[],"expected_sources":["Guide"]}]}`,
			want: "non-empty command array",
		},
		{
			name: "empty setup command",
			body: `{"cases":[{"id":"a","command":["echo","x"],"setup":[[]],"expected_sources":["Guide"]}]}`,
			want: "setup[0]",
		},
		{
			name: "missing expected sources",
			body: `{"cases":[{"id":"a","command":["echo","x"]}]}`,
			want: "expected_sources",
		},
		{
			name: "empty expected evidence term",
			body: `{"cases":[{"id":"a","command":["echo","x"],"expected_sources":["Guide"],"expected_evidence_terms":[""]}]}`,
			want: "expected_evidence_terms",
		},
		{
			name: "min hits too high",
			body: `{"cases":[{"id":"a","command":["echo","x"],"expected_sources":["Guide"],"min_expected_hits":2}]}`,
			want: "min_expected_hits",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeTempFile(t, tt.body)
			if _, err := LoadCases(path); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("LoadCases err = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestLoadCasesValid(t *testing.T) {
	path := writeTempFile(t, `{"cases":[{
		"id":" flowops ",
		"setup":[["flowctl","init"]],
		"command":["flowctl","check","billing_daily"],
		"teardown":[["flowctl","clean"]],
		"expect_failure":true,
		"expected_sources":[" FlowOps DAG Import Error 排障 FAQ "]
	}]}`)

	cases, err := LoadCases(path)
	if err != nil {
		t.Fatalf("LoadCases error: %v", err)
	}
	if len(cases) != 1 {
		t.Fatalf("cases len = %d, want 1", len(cases))
	}
	got := cases[0]
	if got.ID != "flowops" || got.MinExpectedHits != 1 || !got.ExpectFailure {
		t.Fatalf("case not normalized: %+v", got)
	}
	if got.ExpectedSources[0] != "FlowOps DAG Import Error 排障 FAQ" {
		t.Fatalf("expected source = %q", got.ExpectedSources[0])
	}
}

func TestFlowOpsEvalCasesLoad(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "flowops-airflow", "seed", "eval-cases.json")
	cases, err := LoadCases(path)
	if err != nil {
		t.Fatalf("LoadCases FlowOps eval cases error: %v", err)
	}
	if len(cases) != 9 {
		t.Fatalf("FlowOps cases len = %d, want 9", len(cases))
	}
	got := cases[0]
	if got.Command[0] != "flowctl" || !got.ExpectFailure || got.MinExpectedHits != 1 {
		t.Fatalf("unexpected FlowOps case: %+v", got)
	}
	imCase := cases[1]
	if imCase.ID != "flowops-billing-export-source-schema-drift" || imCase.Command[2] != "billing_export_2026" {
		t.Fatalf("unexpected FlowOps IM case: %+v", imCase)
	}
	semanticCase := cases[2]
	if semanticCase.ID != "flowops-orders-reconcile-watermark-lag" || semanticCase.Command[2] != "orders_reconcile_2026" {
		t.Fatalf("unexpected FlowOps semantic case: %+v", semanticCase)
	}
	secretCase := cases[3]
	if secretCase.ID != "flowops-ad-spend-secret-rotation" || secretCase.Command[2] != "ad_spend_daily" {
		t.Fatalf("unexpected FlowOps secret case: %+v", secretCase)
	}
	capacityCase := cases[4]
	if capacityCase.ID != "flowops-inventory-snapshot-capacity" || capacityCase.Command[2] != "inventory_snapshot" {
		t.Fatalf("unexpected FlowOps capacity case: %+v", capacityCase)
	}
	featureCase := cases[5]
	if featureCase.ID != "flowops-churn-features-experiment-gate" || featureCase.Command[2] != "churn_features" {
		t.Fatalf("unexpected FlowOps feature case: %+v", featureCase)
	}
	quotaCase := cases[6]
	if quotaCase.ID != "flowops-payment-settlement-quota" || quotaCase.Command[2] != "payment_settlement" {
		t.Fatalf("unexpected FlowOps quota case: %+v", quotaCase)
	}
	networkCase := cases[7]
	if networkCase.ID != "flowops-crm-sync-egress-allowlist" || networkCase.Command[2] != "crm_sync" {
		t.Fatalf("unexpected FlowOps network case: %+v", networkCase)
	}
	governanceCase := cases[8]
	if governanceCase.ID != "flowops-customer360-pii-governance" || governanceCase.Command[2] != "customer360_pii" {
		t.Fatalf("unexpected FlowOps governance case: %+v", governanceCase)
	}
}

func TestScoreCasePassesExactCitationTitle(t *testing.T) {
	c := flowOpsCase()
	result := ScoreCase(c, Observation{
		CommandExitCode: 1,
		PlannerRecords:  []eval.Record{plannerRecord(true, 3)},
		CueRecords: []eval.Record{{
			Type:       "cue",
			Sources:    []card.Citation{{Title: "FlowOps DAG Import Error 排障 FAQ"}},
			QueryCount: 3,
			LatencyMS:  250,
			CreatedAt:  time.Now(),
		}},
	})

	if !result.Passed {
		t.Fatalf("result failed: %+v", result.Failures)
	}
	if result.ExpectedHitCount != 1 || result.ExpectedCitationHits != 1 || result.TotalCitations != 1 {
		t.Fatalf("unexpected citation scoring: %+v", result)
	}
	if result.PlannerRetrieve != "retrieve" || result.QueryCount != 3 || result.LatencyMS != 250 {
		t.Fatalf("unexpected planner/cue metadata: %+v", result)
	}
}

func TestScoreCaseFailsMissingCueRecord(t *testing.T) {
	result := ScoreCase(flowOpsCase(), Observation{
		CommandExitCode: 1,
		PlannerRecords:  []eval.Record{plannerRecord(true, 2)},
	})
	if result.Passed {
		t.Fatalf("result passed unexpectedly: %+v", result)
	}
	if !containsJoined(result.Failures, "no scored card was available") {
		t.Fatalf("missing no-card failure: %+v", result.Failures)
	}
}

func TestScoreCaseFailsExpectedFailureMismatch(t *testing.T) {
	result := ScoreCase(flowOpsCase(), Observation{
		CommandExitCode: 0,
		PlannerRecords:  []eval.Record{plannerRecord(false, 0)},
		CueRecords: []eval.Record{{
			Type:    "cue",
			Sources: []card.Citation{{Title: "FlowOps DAG Import Error 排障 FAQ"}},
		}},
	})
	if result.Passed {
		t.Fatalf("result passed unexpectedly: %+v", result)
	}
	if !containsJoined(result.Failures, "expected command failure") {
		t.Fatalf("missing expected failure mismatch: %+v", result.Failures)
	}
}

func TestScoreCaseMatchesIMChatName(t *testing.T) {
	c := Case{
		ID:              "flowops-im",
		Command:         []string{"flowctl", "check", "billing_export_2026"},
		ExpectFailure:   true,
		ExpectedSources: []string{"星桥科技 FlowOps 排障演示群"},
		MinExpectedHits: 1,
	}
	result := ScoreCase(c, Observation{
		CommandExitCode: 1,
		CueRecords: []eval.Record{{
			Type:    "cue",
			Sources: []card.Citation{{Type: "im", ChatName: "星桥科技 FlowOps 排障演示群", Summary: "source refresh"}},
		}},
	})
	if !result.Passed || !containsJoined(result.MatchedSources, "星桥科技 FlowOps 排障演示群") {
		t.Fatalf("result = %+v, want IM chat source match", result)
	}
}

func TestScoreCaseRequiresExpectedEvidenceTerms(t *testing.T) {
	c := Case{
		ID:                    "flowops-im",
		Command:               []string{"flowctl", "check", "churn_features"},
		ExpectFailure:         true,
		ExpectedSources:       []string{"星桥科技 FlowOps 排障演示群"},
		ExpectedEvidenceTerms: []string{"EXP-883", "cohort_v3"},
		MinExpectedHits:       1,
	}
	result := ScoreCase(c, Observation{
		CommandExitCode: 1,
		CueRecords: []eval.Record{{
			Type: "cue",
			Sources: []card.Citation{{
				Type:     "im",
				ChatName: "星桥科技 FlowOps 排障演示群",
				Summary:  "churn_features 要走 EXP-883 override 到 cohort_v3",
			}},
		}},
	})
	if !result.Passed || len(result.MatchedEvidenceTerms) != 2 {
		t.Fatalf("result = %+v, want evidence term match", result)
	}

	missing := ScoreCase(c, Observation{
		CommandExitCode: 1,
		CueRecords: []eval.Record{{
			Type:    "cue",
			Sources: []card.Citation{{Type: "im", ChatName: "星桥科技 FlowOps 排障演示群", Summary: "unrelated message"}},
		}},
	})
	if missing.Passed || !containsJoined(missing.Failures, "cohort_v3") {
		t.Fatalf("missing result = %+v, want failed evidence term", missing)
	}
}

func TestSummarizeAndRenderReport(t *testing.T) {
	pass := ScoreCase(flowOpsCase(), Observation{
		CommandExitCode: 1,
		PlannerRecords:  []eval.Record{plannerRecord(true, 3)},
		CueRecords: []eval.Record{{
			Type:       "cue",
			Sources:    []card.Citation{{Title: "FlowOps DAG Import Error 排障 FAQ"}, {Title: "Other"}},
			QueryCount: 3,
			LatencyMS:  100,
		}},
	})
	fail := ScoreCase(Case{
		ID:              "missing",
		Command:         []string{"flowctl", "check", "orders"},
		ExpectFailure:   true,
		ExpectedSources: []string{"Orders FAQ"},
		MinExpectedHits: 1,
	}, Observation{CommandExitCode: 1, PlannerRecords: []eval.Record{plannerRecord(true, 1)}})

	summary := Summarize([]CaseResult{pass, fail}, false)
	if summary.PassedCases != 1 || summary.ExpectedSourceHitCases != 1 || summary.DistinctMatchedSources != 1 || summary.DistinctExpectedSources != 2 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	if summary.ExpectedCitationHits != 1 || summary.TotalCitations != 2 || summary.AverageLatencyMS != 100 {
		t.Fatalf("unexpected precision/latency: %+v", summary)
	}
	out := RenderSummary(summary)
	for _, want := range []string{
		"lark-cue benchmark report",
		"cases: 1/2 passed",
		"expected-source hit rate: 1/2",
		"source coverage: 1/2",
		"citation precision: 1/2",
		"PASS flowops-dag-import",
		"FAIL missing",
		"no scored card was available",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("report missing %q:\n%s", want, out)
		}
	}
}

func writeTempFile(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "cases.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}
	return path
}

func plannerRecord(should bool, queryCount int) eval.Record {
	return eval.Record{
		Type:           "planner",
		ShouldRetrieve: &should,
		QueryCount:     queryCount,
	}
}

func flowOpsCase() Case {
	return Case{
		ID:              "flowops-dag-import",
		Command:         []string{"flowctl", "check", "billing_daily"},
		ExpectFailure:   true,
		ExpectedSources: []string{"FlowOps DAG Import Error 排障 FAQ"},
		MinExpectedHits: 1,
	}
}

func containsJoined(values []string, want string) bool {
	return strings.Contains(strings.Join(values, "\n"), want)
}
