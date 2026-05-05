package evidence

import (
	"strings"
	"testing"

	"lark-cue/internal/retrieval"
)

func TestScoreAndSelectStrongEvidence(t *testing.T) {
	sources := []retrieval.Source{
		{
			Type:    "doc",
			Title:   "guide",
			Content: "missing required scope: docx:document:read 权限变更 重新授权 旧 token Atlas 飞书 API",
			Fetched: true,
		},
		{
			Type:    "doc",
			Title:   "theme",
			Content: "前端 主题色 深色模式 卡片样式",
			Fetched: true,
		},
	}
	selected, confidence := Select(Score(sources))
	if confidence != ConfidenceHigh {
		t.Fatalf("confidence = %s, want high", confidence)
	}
	if len(selected) != 1 || selected[0].Source.Title != "guide" {
		t.Fatalf("selected = %+v", selected)
	}
}

func TestWeakOrMissingEvidence(t *testing.T) {
	selected, confidence := Select(Score([]retrieval.Source{{Type: "doc", Content: "unrelated", Fetched: true}}))
	if confidence != ConfidenceNone || len(selected) != 0 {
		t.Fatalf("selected/confidence = %+v/%s", selected, confidence)
	}
}

func TestSearchMetadataDoesNotDriveEvidence(t *testing.T) {
	sources := []retrieval.Source{
		{
			Type:    "doc",
			Title:   "missing required scope: docx:document:read",
			Summary: "权限变更 重新授权 旧 token",
			Content: "前端主题色调整记录，与权限错误无关。",
			Fetched: true,
		},
	}
	selected, confidence := Select(Score(sources))
	if confidence != ConfidenceNone || len(selected) != 0 {
		t.Fatalf("metadata drove evidence selected=%+v confidence=%s", selected, confidence)
	}
}

func TestHighConfidenceRequiresStrongErrorSignal(t *testing.T) {
	sources := []retrieval.Source{
		{
			Type:    "doc",
			Title:   "generic permissions",
			Content: "scope 权限 授权 权限变更 重新授权 旧 token 开放平台 Atlas",
			Fetched: true,
		},
	}
	selected, confidence := Select(Score(sources))
	if len(selected) == 0 {
		t.Fatal("expected weak generic evidence to be selected")
	}
	if confidence != ConfidenceLow {
		t.Fatalf("confidence = %s, want low", confidence)
	}
}

func TestHighConfidenceRequiresConcreteCauseAction(t *testing.T) {
	sources := []retrieval.Source{
		{
			Type:    "doc",
			Title:   "generic permissions",
			Content: "missing required scope: docx:document:read scope 权限 授权 Atlas",
			Fetched: true,
		},
	}
	selected, confidence := Select(Score(sources))
	if len(selected) == 0 {
		t.Fatal("expected generic evidence to be selected")
	}
	if confidence != ConfidenceLow {
		t.Fatalf("confidence = %s, want low", confidence)
	}
}

func TestNegatedCauseActionDoesNotProduceHighConfidence(t *testing.T) {
	sources := []retrieval.Source{
		{
			Type:    "doc",
			Title:   "negated repair",
			Content: "missing required scope: docx:document:read。不是权限变更，也不要重新授权；无需清理旧 token。",
			Fetched: true,
		},
	}
	selected, confidence := Select(Score(sources))
	if len(selected) == 0 {
		t.Fatal("expected error evidence to be selected as weak evidence")
	}
	if confidence == ConfidenceHigh {
		t.Fatalf("negated repair terms produced high confidence: %+v", selected)
	}
	if selected[0].CauseAction {
		t.Fatalf("negated repair terms counted as cause/action support: %+v", selected[0])
	}
}

func TestSuffixNegatedCauseActionDoesNotProduceHighConfidence(t *testing.T) {
	sources := []retrieval.Source{
		{
			Type:    "doc",
			Title:   "suffix negated repair",
			Content: "missing required scope: docx:document:read。权限变更不是原因；重新授权不需要；old token cleanup not required。",
			Fetched: true,
		},
	}
	selected, confidence := Select(Score(sources))
	if len(selected) == 0 {
		t.Fatal("expected error evidence to be selected as weak evidence")
	}
	if confidence == ConfidenceHigh || selected[0].CauseAction {
		t.Fatalf("suffix-negated repair terms counted as support: selected=%+v confidence=%s", selected, confidence)
	}
}

func TestChineseBuYongNegationDoesNotProduceHighConfidence(t *testing.T) {
	sources := []retrieval.Source{
		{
			Type:    "doc",
			Title:   "bu yong negated repair",
			Content: "missing required scope: docx:document:read。权限变更不用处理；不用重新授权；旧 token 不用清理。",
			Fetched: true,
		},
	}
	selected, confidence := Select(Score(sources))
	if len(selected) == 0 {
		t.Fatal("expected error evidence to be selected as weak evidence")
	}
	if confidence == ConfidenceHigh || selected[0].CauseAction {
		t.Fatalf("不用-negated repair terms counted as support: selected=%+v confidence=%s", selected, confidence)
	}
}

func TestHighConfidenceRequiresFeishuScenarioSignal(t *testing.T) {
	sources := []retrieval.Source{
		{
			Type:    "doc",
			Title:   "generic oauth permissions",
			Content: "missing required scope: repo。permission change requires re-authorize.",
			Fetched: true,
		},
	}
	selected, confidence := Select(Score(sources))
	if len(selected) == 0 {
		t.Fatal("expected generic OAuth evidence to be selected as weak evidence")
	}
	if confidence == ConfidenceHigh || selected[0].ScenarioSignal {
		t.Fatalf("generic OAuth evidence became high-confidence Feishu evidence: selected=%+v confidence=%s", selected, confidence)
	}
}

func TestHighConfidenceRequiresCitedCauseAction(t *testing.T) {
	sources := []retrieval.Source{
		{
			Type:  "doc",
			Title: "uncited action",
			Content: strings.Join([]string{
				"missing required scope: docx:document:read",
				strings.Repeat("x", 320) + " 重新授权",
			}, "\n"),
			Fetched: true,
		},
	}
	selected, confidence := Select(Score(sources))
	if len(selected) == 0 {
		t.Fatal("expected error evidence to be selected as weak evidence")
	}
	if strings.Contains(selected[0].Snippet, "重新授权") {
		t.Fatalf("test setup expected snippet to omit action support: %q", selected[0].Snippet)
	}
	if confidence == ConfidenceHigh || selected[0].CauseAction {
		t.Fatalf("uncited action support produced high confidence: selected=%+v confidence=%s", selected, confidence)
	}
}

func TestSnippetIncludesStrongErrorAndCauseActionLines(t *testing.T) {
	sources := []retrieval.Source{
		{
			Type:  "doc",
			Title: "split support",
			Content: strings.Join([]string{
				"missing required scope: docx:document:read",
				"处理方式：新增权限后发布权限变更，再重新授权本地身份。",
			}, "\n"),
			Fetched: true,
		},
	}
	selected, confidence := Select(Score(sources))
	if confidence != ConfidenceHigh || len(selected) != 1 {
		t.Fatalf("selected/confidence = %+v/%s", selected, confidence)
	}
	if !strings.Contains(selected[0].Snippet, "docx:document:read") || !strings.Contains(selected[0].Snippet, "重新授权") {
		t.Fatalf("snippet did not include both error and action support: %q", selected[0].Snippet)
	}
}
