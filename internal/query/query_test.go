package query

import (
	"context"
	"testing"

	"lark-cue/internal/detector"
)

type fakeExpander struct{}

func (fakeExpander) ExpandQueries(ctx context.Context, command []string, output string, scenario detector.Scenario, seeds []string) ([]string, error) {
	return []string{"权限变更 重新授权", "旧 token scope", "docx:document:read"}, nil
}

func TestExtractSeeds(t *testing.T) {
	seeds := ExtractSeeds("missing required scope: docx:document:read\ntenant_access_token invalid or permission denied")
	want := []string{"docx:document:read", "missing required scope", "tenant_access_token invalid", "permission denied"}
	for _, expected := range want {
		if !contains(seeds, expected) {
			t.Fatalf("seeds %v missing %q", seeds, expected)
		}
	}
}

func TestBuildKeepsSeedsAndAddsLLMQueries(t *testing.T) {
	scenario, _ := detector.Detect("missing required scope: docx:document:read")
	queries := Build(context.Background(), []string{"node", "x.js"}, "missing required scope: docx:document:read", scenario, fakeExpander{})
	if !contains(queries, "docx:document:read") || !contains(queries, "权限变更 重新授权") {
		t.Fatalf("queries = %v", queries)
	}
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
