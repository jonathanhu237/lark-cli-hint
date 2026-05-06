package eval

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lark-cue/internal/card"
	"lark-cue/internal/retrieval"
)

func TestAppendAndAppendFeedback(t *testing.T) {
	path := filepath.Join(t.TempDir(), "evaluations.jsonl")
	err := Append(path, FromCard(card.KnowledgeCard{ID: "cue_test", Command: "node x.js", Scenario: "scenario", Feedback: "skipped"}))
	if err != nil {
		t.Fatalf("Append error: %v", err)
	}
	if err := AppendFeedback(path, "cue_test", "useful"); err != nil {
		t.Fatalf("AppendFeedback error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, `"type":"cue"`) || !strings.Contains(text, `"type":"feedback_update"`) {
		t.Fatalf("unexpected log:\n%s", text)
	}
}

func TestCueRecordIncludesRequiredReportFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "evaluations.jsonl")
	err := Append(path, FromCard(card.KnowledgeCard{
		ID:              "cue_test",
		Command:         "node x.js",
		Scenario:        "scenario",
		RetrievalStatus: retrieval.StatusFailed,
		QueryCount:      2,
		LatencyMS:       0,
		Feedback:        "skipped",
	}))
	if err != nil {
		t.Fatalf("Append error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	var record map[string]any
	if err := json.Unmarshal(bytesTrimSpace(data), &record); err != nil {
		t.Fatalf("Unmarshal error: %v\n%s", err, string(data))
	}
	if _, ok := record["latency_ms"]; !ok {
		t.Fatalf("latency_ms missing from record: %s", string(data))
	}
	if record["query_count"] != float64(2) {
		t.Fatalf("query_count = %#v, want 2", record["query_count"])
	}
	sources, ok := record["sources"].([]any)
	if !ok {
		t.Fatalf("sources = %#v, want array", record["sources"])
	}
	if len(sources) != 0 {
		t.Fatalf("sources len = %d, want empty array", len(sources))
	}
	if record["feedback"] != "skipped" {
		t.Fatalf("feedback = %#v, want skipped", record["feedback"])
	}
}

func TestReadCueRecordsAggregatesFeedbackAndMalformedLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "evaluations.jsonl")
	lines := []string{
		`{"type":"cue","card_id":"cue_1","retrieval_status":"partial","sources":[{"type":"doc","title":"Guide"}],"latency_ms":1000,"query_count":4,"feedback":"skipped","created_at":"2026-05-06T00:00:00Z"}`,
		`not-json`,
		`{"type":"feedback_update","card_id":"cue_1","feedback":"useful","created_at":"2026-05-06T00:00:01Z"}`,
		`{"type":"cue","card_id":"cue_2","retrieval_status":"ok","sources":[],"latency_ms":3000,"query_count":2,"feedback":"skipped","created_at":"2026-05-06T00:00:02Z"}`,
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	result, err := ReadCueRecords(path)
	if err != nil {
		t.Fatalf("ReadCueRecords error: %v", err)
	}
	if result.MalformedLines != 1 {
		t.Fatalf("MalformedLines = %d, want 1", result.MalformedLines)
	}
	if len(result.Records) != 2 {
		t.Fatalf("records len = %d, want 2", len(result.Records))
	}
	if result.Records[0].Feedback != "useful" {
		t.Fatalf("feedback update was not applied: %+v", result.Records[0])
	}

	summary := Summarize(result.Records, result.MalformedLines, DefaultReportLimit)
	if summary.TotalRuns != 2 {
		t.Fatalf("TotalRuns = %d, want 2", summary.TotalRuns)
	}
	if summary.StatusCounts["partial"] != 1 || summary.StatusCounts["ok"] != 1 {
		t.Fatalf("StatusCounts = %#v, want partial=1 ok=1", summary.StatusCounts)
	}
	if summary.RealRuns != 2 {
		t.Fatalf("RealRuns = %d, want 2", summary.RealRuns)
	}
	if summary.RunsWithSources != 1 || summary.TotalSources != 1 {
		t.Fatalf("source coverage = %d/%d, want 1/1", summary.RunsWithSources, summary.TotalSources)
	}
	if summary.FeedbackCounts["useful"] != 1 || summary.FeedbackCounts["skipped"] != 1 {
		t.Fatalf("FeedbackCounts = %#v, want useful=1 skipped=1", summary.FeedbackCounts)
	}
	if summary.AverageLatencyMS != 2000 {
		t.Fatalf("AverageLatencyMS = %v, want 2000", summary.AverageLatencyMS)
	}
	if summary.AverageQueryCount != 3 {
		t.Fatalf("AverageQueryCount = %v, want 3", summary.AverageQueryCount)
	}
}

func TestReadCueRecordsSkipsOversizedMalformedLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "evaluations.jsonl")
	lines := []string{
		`{"type":"cue","card_id":"cue_1","retrieval_status":"ok","created_at":"2026-05-06T00:00:00Z"}`,
		strings.Repeat("{", maxReportLineBytes+1),
		`{"type":"cue","card_id":"cue_2","retrieval_status":"partial","created_at":"2026-05-06T00:00:01Z"}`,
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	result, err := ReadCueRecords(path)
	if err != nil {
		t.Fatalf("ReadCueRecords error: %v", err)
	}
	if result.MalformedLines != 1 {
		t.Fatalf("MalformedLines = %d, want 1", result.MalformedLines)
	}
	if len(result.Records) != 2 {
		t.Fatalf("records len = %d, want 2", len(result.Records))
	}
}

func TestReadCueRecordsMissingAndLimit(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.jsonl")
	result, err := ReadCueRecords(missing)
	if err != nil {
		t.Fatalf("ReadCueRecords missing error: %v", err)
	}
	if len(result.Records) != 0 {
		t.Fatalf("missing records len = %d, want 0", len(result.Records))
	}

	records := []Record{
		{CardID: "cue_1"},
		{CardID: "cue_2"},
		{CardID: "cue_3"},
	}
	limited := LimitCueRecords(records, 2)
	if len(limited) != 2 || limited[0].CardID != "cue_2" || limited[1].CardID != "cue_3" {
		t.Fatalf("limited = %+v, want cue_2 cue_3", limited)
	}
}

func TestReadCueRecordsIncludesPlannerEvents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "evaluations.jsonl")
	lines := []string{
		`{"type":"planner","command":"flowctl check billing_daily","scenario":"FlowOps DAG import error","reason":"internal platform failure","should_retrieve":true,"query_count":3,"latency_ms":800,"created_at":"2026-05-06T00:00:00Z"}`,
		`{"type":"planner","command":"python missing.py","scenario":"local file error","reason":"missing local file","should_retrieve":false,"query_count":0,"latency_ms":200,"created_at":"2026-05-06T00:00:01Z"}`,
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	result, err := ReadCueRecords(path)
	if err != nil {
		t.Fatalf("ReadCueRecords error: %v", err)
	}
	if len(result.PlannerRecords) != 2 || len(result.Events) != 2 {
		t.Fatalf("planner/events = %d/%d, want 2/2", len(result.PlannerRecords), len(result.Events))
	}
	summary := SummarizeResult(result, DefaultReportLimit)
	if summary.PlannerRuns != 2 || summary.PlannerRetrieve != 1 || summary.PlannerSkip != 1 {
		t.Fatalf("planner summary = %+v, want runs=2 retrieve=1 skip=1", summary)
	}
	report := RenderSummary(summary)
	for _, want := range []string{"Planner", "decisions: 2", "retrieve: 1", "skip: 1"} {
		if !strings.Contains(report, want) {
			t.Fatalf("report missing %q:\n%s", want, report)
		}
	}
}

func TestRenderSummaryEmptyAndPlain(t *testing.T) {
	empty := RenderSummary(Summarize(nil, 1, DefaultReportLimit))
	if !strings.Contains(empty, "No cue or planner records found") || !strings.Contains(empty, "malformed") {
		t.Fatalf("empty report missing expected content:\n%s", empty)
	}

	report := RenderSummary(Summary{
		TotalRuns:         1,
		StatusCounts:      map[string]int{"ok": 1, "unknown": 1, "stale": 1},
		RunsWithSources:   1,
		TotalSources:      2,
		AverageSources:    2,
		AverageQueryCount: 3,
		AverageLatencyMS:  1500,
		FeedbackCounts:    map[string]int{"useful": 1},
		Limit:             20,
	})
	for _, want := range []string{"lark-cue validation report", "cue runs: 1", "ok 1", "unknown 1", "stale 1", "citation coverage: 1/1", "1.5s", "useful: 1"} {
		if !strings.Contains(report, want) {
			t.Fatalf("report missing %q:\n%s", want, report)
		}
	}
}

func TestSummarizeDefaultsMissingRetrievalStatusToUnknown(t *testing.T) {
	summary := Summarize([]Record{{CardID: "cue_1"}}, 0, DefaultReportLimit)
	if summary.StatusCounts["unknown"] != 1 {
		t.Fatalf("StatusCounts = %#v, want unknown=1", summary.StatusCounts)
	}
	if !strings.Contains(RenderSummary(summary), "unknown 1") {
		t.Fatalf("plain report did not include unknown status:\n%s", RenderSummary(summary))
	}
}

func TestRenderSummaryStyledKeepsReportContent(t *testing.T) {
	rendered := RenderSummaryStyled(Summary{
		TotalRuns:         2,
		StatusCounts:      map[string]int{"ok": 1, "partial": 1, "stale": 1},
		RunsWithSources:   2,
		TotalSources:      4,
		AverageSources:    2,
		AverageQueryCount: 4,
		AverageLatencyMS:  1200,
		FeedbackCounts:    map[string]int{"useful": 1, "skipped": 1},
		Limit:             20,
	}, 100)
	for _, want := range []string{"lark-cue validation", "Runs", "Retrieval", "Evidence", "Feedback", "stale 1"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("styled report missing %q:\n%s", want, rendered)
		}
	}
}

func bytesTrimSpace(value []byte) []byte {
	return []byte(strings.TrimSpace(string(value)))
}
