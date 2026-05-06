package push

import (
	"context"
	"errors"
	"strings"
	"testing"

	"lark-cue/internal/card"
	"lark-cue/internal/evidence"
)

func TestPrepareIncludesCaveatAndCitations(t *testing.T) {
	markdown := Prepare(card.KnowledgeCard{
		ID:            "cue_test",
		Scenario:      "FlowOps DAG import error",
		PlannerReason: "DAG import error mentions billing_region.",
		Queries:       []string{"FlowOps DAG import error", "billing_daily billing_region"},
		LikelyCause:   "billing_daily 在解析阶段读取 billing_region Variable。",
		NextAction:    "把 Variable.get 移到任务运行阶段后重新执行 flowctl check。",
		Caveat:        "证据较弱，建议打开来源核对后再执行修复动作。",
		Confidence:    evidence.ConfidenceHigh,
		Citations: []card.Citation{
			{
				Type:    "doc",
				Title:   "FlowOps DAG Import Error 排障 FAQ",
				URL:     "https://example.test/guide",
				Summary: "billing_daily billing_region Variable.get",
			},
			{
				Type:      "im",
				ChatName:  "FlowOps 排障群",
				Sender:    "李四",
				Timestamp: "2026-05-05 21:46",
				Summary:   "需要把 Variable.get 移到任务运行阶段",
			},
		},
	})
	for _, want := range []string{
		"证据较弱",
		"LLM Plan",
		"DAG import error mentions billing_region",
		"billing_daily billing_region",
		"FlowOps DAG Import Error 排障 FAQ",
		"https://example.test/guide",
		"FlowOps 排障群",
		"需要把 Variable.get 移到任务运行阶段",
	} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("push markdown missing %q:\n%s", want, markdown)
		}
	}
}

func TestResolveChatRequiresExactSingleMatch(t *testing.T) {
	sender := NewSender(fakeRunner{
		response: map[string]any{
			"ok": true,
			"data": map[string]any{
				"chats": []any{
					map[string]any{"name": "星桥开放平台排障群备份", "chat_id": "oc_wrong"},
					map[string]any{"name": "星桥开放平台排障群", "chat_id": "oc_right"},
				},
			},
		},
	})
	chatID, err := sender.resolveChat(context.Background(), "星桥开放平台排障群")
	if err != nil {
		t.Fatalf("resolveChat error: %v", err)
	}
	if chatID != "oc_right" {
		t.Fatalf("chatID = %q, want oc_right", chatID)
	}
}

func TestResolveChatRejectsPartialMatch(t *testing.T) {
	sender := NewSender(fakeRunner{
		response: map[string]any{
			"ok": true,
			"data": map[string]any{
				"chats": []any{
					map[string]any{"name": "星桥开放平台排障群备份", "chat_id": "oc_wrong"},
				},
			},
		},
	})
	_, err := sender.resolveChat(context.Background(), "星桥开放平台排障群")
	if err == nil || !strings.Contains(err.Error(), "no exact") {
		t.Fatalf("expected no exact match error, got %v", err)
	}
}

func TestResolveChatRejectsAmbiguousExactMatches(t *testing.T) {
	sender := NewSender(fakeRunner{
		response: map[string]any{
			"ok": true,
			"data": map[string]any{
				"chats": []any{
					map[string]any{"name": "星桥开放平台排障群", "chat_id": "oc_one"},
					map[string]any{"name": "星桥开放平台排障群", "chat_id": "oc_two"},
				},
			},
		},
	})
	_, err := sender.resolveChat(context.Background(), "星桥开放平台排障群")
	if err == nil || !strings.Contains(err.Error(), "multiple") {
		t.Fatalf("expected ambiguous match error, got %v", err)
	}
}

type fakeRunner struct {
	response map[string]any
	err      error
}

func (f fakeRunner) RunJSON(ctx context.Context, args ...string) (map[string]any, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.response == nil {
		return nil, errors.New("missing fake response")
	}
	return f.response, nil
}
