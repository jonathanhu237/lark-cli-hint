package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"[\"docx:document:read 权限发布\"]"}}]}`))
	}))
	defer server.Close()

	provider := NewOpenAICompatible(config.LLMConfig{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Model:   "gpt-5.5",
	})

	queries, err := provider.ExpandQueries(
		context.Background(),
		[]string{"node", "examples/failing-feishu-api.js"},
		"missing required scope: docx:document:read",
		detector.Scenario{ID: detector.FeishuAPIScopeError},
		[]string{"docx:document:read"},
	)
	if err != nil {
		t.Fatalf("ExpandQueries returned error: %v", err)
	}
	if len(queries) != 1 || queries[0] != "docx:document:read 权限发布" {
		t.Fatalf("unexpected queries: %#v", queries)
	}
}
