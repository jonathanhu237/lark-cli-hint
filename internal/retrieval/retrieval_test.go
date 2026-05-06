package retrieval

import (
	"context"
	"errors"
	"slices"
	"testing"
)

func TestPartialRetrievalFailureIsReported(t *testing.T) {
	retriever := NewLarkRetriever(fakeJSONRunner{
		failPrefixes: [][]string{{"docs", "+search"}},
	})
	sources, status, err := retriever.Retrieve(context.Background(), []string{"missing required scope"})
	if err == nil {
		t.Fatal("expected partial retrieval error")
	}
	if status != StatusPartial {
		t.Fatalf("status = %s, want partial", status)
	}
	if len(sources) != 1 || sources[0].Type != "im" {
		t.Fatalf("sources = %+v, want one IM source", sources)
	}
}

func TestRetrieveSearchesDocsAndIMForEachQuery(t *testing.T) {
	runner := &recordingJSONRunner{}
	retriever := NewLarkRetriever(runner)
	_, status, err := retriever.Retrieve(context.Background(), []string{"alpha", "beta"})
	if err != nil {
		t.Fatalf("Retrieve error: %v", err)
	}
	if status != StatusOK {
		t.Fatalf("status = %s, want ok", status)
	}
	want := []string{
		"docs:alpha",
		"im:alpha",
		"docs:beta",
		"im:beta",
	}
	if !slices.Equal(runner.calls, want) {
		t.Fatalf("calls = %#v, want %#v", runner.calls, want)
	}
}

type fakeJSONRunner struct {
	failPrefixes [][]string
}

func (f fakeJSONRunner) RunJSON(ctx context.Context, args ...string) (map[string]any, error) {
	for _, prefix := range f.failPrefixes {
		if len(args) >= len(prefix) && slices.Equal(args[:len(prefix)], prefix) {
			return nil, errors.New("forced failure")
		}
	}
	if len(args) >= 2 && args[0] == "docs" && args[1] == "+search" {
		return map[string]any{
			"ok": true,
			"data": map[string]any{
				"results": []any{},
			},
		}, nil
	}
	if len(args) >= 2 && args[0] == "im" && args[1] == "+messages-search" {
		return map[string]any{
			"ok": true,
			"data": map[string]any{
				"messages": []any{
					map[string]any{
						"message_id":  "om_test",
						"content":     "missing required scope: docx:document:read 权限变更 重新授权",
						"chat_name":   "排障群",
						"create_time": "2026-05-06 09:00",
						"sender": map[string]any{
							"name": "李四",
						},
					},
				},
			},
		}, nil
	}
	return map[string]any{"ok": true, "data": map[string]any{}}, nil
}

type recordingJSONRunner struct {
	calls []string
}

func (r *recordingJSONRunner) RunJSON(ctx context.Context, args ...string) (map[string]any, error) {
	if len(args) >= 4 && args[0] == "docs" && args[1] == "+search" {
		r.calls = append(r.calls, "docs:"+args[3])
		return map[string]any{"ok": true, "data": map[string]any{"results": []any{}}}, nil
	}
	if len(args) >= 4 && args[0] == "im" && args[1] == "+messages-search" {
		r.calls = append(r.calls, "im:"+args[3])
		return map[string]any{"ok": true, "data": map[string]any{"messages": []any{}}}, nil
	}
	return map[string]any{"ok": true, "data": map[string]any{}}, nil
}
