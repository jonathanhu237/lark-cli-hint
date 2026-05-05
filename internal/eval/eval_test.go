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

func bytesTrimSpace(value []byte) []byte {
	return []byte(strings.TrimSpace(string(value)))
}
