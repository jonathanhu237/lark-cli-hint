package push

import (
	"context"
	"errors"
	"strings"
	"testing"

	"lark-cue/internal/card"
	"lark-cue/internal/evidence"
)

func TestPrepareIncludesFixtureCaveatAndCitations(t *testing.T) {
	markdown := Prepare(card.KnowledgeCard{
		ID:          "cue_test",
		Scenario:    "检测到飞书 API 权限 / scope / token 错误。",
		LikelyCause: "应用缺少 docx:document:read。",
		NextAction:  "发布权限变更后重新授权。",
		Caveat:      "当前使用显式 demo fixture，不能当作真实 Feishu 检索结果。",
		Confidence:  evidence.ConfidenceHigh,
		Fixture:     true,
		Citations: []card.Citation{
			{
				Type:    "doc",
				Title:   "Demo fixture: 权限指南",
				URL:     "fixture://guide",
				Summary: "missing required scope docx:document:read",
				Fixture: true,
			},
			{
				Type:      "im",
				ChatName:  "Demo fixture: 排障群",
				Sender:    "李四",
				Timestamp: "2026-05-05 21:46",
				Summary:   "需要发布权限变更并重新授权",
				Fixture:   true,
			},
		},
	})
	for _, want := range []string{
		"Demo fixture / simulated Feishu content",
		"当前使用显式 demo fixture",
		"Demo fixture: 权限指南",
		"fixture://guide",
		"Demo fixture: 排障群",
		"需要发布权限变更并重新授权",
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
