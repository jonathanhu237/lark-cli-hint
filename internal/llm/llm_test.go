package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"lark-cue/internal/config"
	"lark-cue/internal/detector"
)

func TestOpenAICompatibleOmitsTemperature(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if _, ok := payload["temperature"]; ok {
			t.Fatalf("request included unsupported temperature field: %#v", payload["temperature"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"[\"FlowOps DAG import error billing_daily\"]"}}]}`))
	}))
	defer server.Close()

	provider := NewOpenAICompatible(config.LLMConfig{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Model:   "gpt-5.5",
	})

	queries, err := provider.ExpandQueries(
		context.Background(),
		[]string{"flowctl", "check", "billing_daily"},
		"Broken DAG billing_daily Variable billing_region does not exist",
		detector.Scenario{ID: "flowops_dag_import_error"},
		[]string{"billing_daily", "billing_region"},
	)
	if err != nil {
		t.Fatalf("ExpandQueries returned error: %v", err)
	}
	if len(queries) != 1 || queries[0] != "FlowOps DAG import error billing_daily" {
		t.Fatalf("unexpected queries: %#v", queries)
	}
}

func TestPlanRetrievalParsesDecisionAndNormalizesQueries(t *testing.T) {
	var gotUserPrompt string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		messages := payload["messages"].([]any)
		user := messages[1].(map[string]any)
		gotUserPrompt = user["content"].(string)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"should_retrieve\":true,\"scenario\":\"FlowOps DAG import error\",\"reason\":\"DAG import error mentions billing_region.\",\"queries\":[\" FlowOps DAG import error billing_daily \",\"billing_daily billing_region Variable.get\",\"FlowOps DAG import error billing_daily\",\"\"]}"}}]}`))
	}))
	defer server.Close()

	provider := NewOpenAICompatible(config.LLMConfig{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Model:   "gpt-5.5",
	})

	decision, err := provider.PlanRetrieval(context.Background(), PlanInput{
		Command:  []string{"flowctl", "dags", "list-import-errors"},
		ExitCode: 1,
		Output:   "Broken DAG billing_daily Variable billing_region does not exist",
	})
	if err != nil {
		t.Fatalf("PlanRetrieval returned error: %v", err)
	}
	if !decision.ShouldRetrieve || decision.Scenario != "FlowOps DAG import error" {
		t.Fatalf("unexpected decision: %+v", decision)
	}
	if len(decision.Queries) != 2 || decision.Queries[0] != "FlowOps DAG import error billing_daily" || decision.Queries[1] != "billing_daily billing_region Variable.get" {
		t.Fatalf("queries not normalized: %#v", decision.Queries)
	}
	if !strings.Contains(gotUserPrompt, "keyword-style Feishu search phrases") {
		t.Fatalf("planner prompt missing keyword guidance:\n%s", gotUserPrompt)
	}
}

func TestPlanRetrievalFalseDropsQueries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"should_retrieve\":false,\"scenario\":\"local file error\",\"reason\":\"missing local file\",\"queries\":[\"should be dropped\"]}"}}]}`))
	}))
	defer server.Close()

	provider := NewOpenAICompatible(config.LLMConfig{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Model:   "gpt-5.5",
	})
	decision, err := provider.PlanRetrieval(context.Background(), PlanInput{
		Command:  []string{"python", "missing.py"},
		ExitCode: 2,
		Output:   "python: can't open file missing.py",
	})
	if err != nil {
		t.Fatalf("PlanRetrieval returned error: %v", err)
	}
	if decision.ShouldRetrieve || len(decision.Queries) != 0 {
		t.Fatalf("false decision retained queries: %+v", decision)
	}
}

func TestPlanRetrievalMalformedOutputReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"not json"}}]}`))
	}))
	defer server.Close()

	provider := NewOpenAICompatible(config.LLMConfig{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Model:   "gpt-5.5",
	})
	if _, err := provider.PlanRetrieval(context.Background(), PlanInput{Command: []string{"flowctl"}, ExitCode: 1}); err == nil {
		t.Fatal("expected malformed planner output error")
	}
}

func TestNormalizeQueriesCapsAndDeduplicates(t *testing.T) {
	queries := NormalizeQueries([]string{
		" FlowOps DAG import error ",
		"flowops dag import error",
		"",
		"billing_daily billing_region",
		strings.Repeat("x", 130),
	}, 2, 12)
	if len(queries) != 2 {
		t.Fatalf("len = %d, want 2: %#v", len(queries), queries)
	}
	if queries[0] != "FlowOps DAG " || queries[1] != "billing_dail" {
		t.Fatalf("unexpected normalized queries: %#v", queries)
	}
}
