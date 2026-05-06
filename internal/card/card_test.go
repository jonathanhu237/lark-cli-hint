package card

import (
	"context"
	"errors"
	"strings"
	"testing"

	"lark-cue/internal/detector"
	"lark-cue/internal/evidence"
	"lark-cue/internal/llm"
	"lark-cue/internal/retrieval"
)

func TestTemplateFallbackCard(t *testing.T) {
	scenario, _ := detector.Detect("missing required scope: docx:document:read")
	scored := []evidence.ScoredSource{{
		Source:  retrieval.Source{Type: "doc", Title: "guide", URL: "https://example.test", Content: "missing required scope docx:document:read 权限变更 重新授权", Fetched: true},
		Score:   12,
		Snippet: "missing required scope docx:document:read 权限变更 重新授权",
	}}
	card := Build(context.Background(), Input{
		Command:         []string{"node", "x.js"},
		Output:          "missing required scope: docx:document:read",
		Scenario:        scenario,
		PlannerReason:   "失败输出命中内部权限配置问题，需要检索 Feishu 知识。",
		Queries:         []string{"docx:document:read", "权限变更 重新授权", "docx:document:read"},
		Evidence:        scored,
		Confidence:      evidence.ConfidenceHigh,
		RetrievalStatus: retrieval.StatusOK,
	})
	rendered := Render(card)
	for _, want := range []string{"LLM Plan", "Reason: 失败输出命中内部权限配置问题", "- docx:document:read", "- 权限变更 重新授权", "Sources", "guide", "High"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered card missing %q:\n%s", want, rendered)
		}
	}
	if strings.Count(rendered, "- docx:document:read") != 1 {
		t.Fatalf("rendered card did not de-duplicate planner queries:\n%s", rendered)
	}
	if card.QueryCount != 2 {
		t.Fatalf("QueryCount = %d, want 2", card.QueryCount)
	}
}

func TestRenderOmitsEmptyLLMPlan(t *testing.T) {
	rendered := Render(KnowledgeCard{
		ID:          "cue_test",
		Scenario:    "FlowOps DAG import error",
		LikelyCause: "内部证据支持。",
		NextAction:  "按内部文档处理。",
		Confidence:  evidence.ConfidenceHigh,
	})
	if strings.Contains(rendered, "LLM Plan") {
		t.Fatalf("rendered empty LLM plan:\n%s", rendered)
	}
}

func TestLowConfidenceLLMDraftIsForcedToCaveat(t *testing.T) {
	scenario, _ := detector.Detect("permission denied")
	card := Build(context.Background(), Input{
		Command:  []string{"node", "x.js"},
		Output:   "permission denied",
		Scenario: scenario,
		Evidence: []evidence.ScoredSource{{
			Source: retrieval.Source{Type: "doc", Title: "weak", Content: "权限", Fetched: true},
			Score:  3,
		}},
		Confidence:      evidence.ConfidenceLow,
		RetrievalStatus: retrieval.StatusOK,
		Provider:        overclaimingProvider{},
	})
	if strings.Contains(card.LikelyCause, "definitive") {
		t.Fatalf("low-confidence card trusted overclaiming LLM draft: %+v", card)
	}
	if strings.Contains(card.NextAction, "risky") {
		t.Fatalf("low-confidence card trusted overclaiming next action: %+v", card)
	}
	if card.Caveat == "" {
		t.Fatalf("low-confidence card missing caveat: %+v", card)
	}
}

func TestHighConfidenceUnsupportedLLMDraftFallsBack(t *testing.T) {
	scenario, _ := detector.Detect("missing required scope: docx:document:read")
	card := Build(context.Background(), Input{
		Command:  []string{"node", "x.js"},
		Output:   "missing required scope: docx:document:read",
		Scenario: scenario,
		Evidence: []evidence.ScoredSource{{
			Source:      retrieval.Source{Type: "doc", Title: "guide", Content: "missing required scope: docx:document:read 权限变更 重新授权 旧 token", Fetched: true},
			Score:       12,
			StrongError: true,
			CauseAction: true,
			Snippet:     "missing required scope: docx:document:read 权限变更 重新授权 旧 token",
		}},
		Confidence:      evidence.ConfidenceHigh,
		RetrievalStatus: retrieval.StatusOK,
		Provider:        overclaimingProvider{},
	})
	if strings.Contains(card.LikelyCause, "definitive") {
		t.Fatalf("high-confidence card trusted unsupported LLM draft: %+v", card)
	}
	if strings.Contains(card.NextAction, "risky") {
		t.Fatalf("high-confidence card trusted unsupported LLM next action: %+v", card)
	}
	if !strings.Contains(card.LikelyCause, "scope") {
		t.Fatalf("expected deterministic grounded fallback cause, got %+v", card)
	}
}

func TestHighConfidenceChineseUnsupportedLLMDraftFallsBack(t *testing.T) {
	scenario, _ := detector.Detect("missing required scope: docx:document:read")
	card := Build(context.Background(), Input{
		Command:  []string{"node", "x.js"},
		Output:   "missing required scope: docx:document:read",
		Scenario: scenario,
		Evidence: []evidence.ScoredSource{{
			Source:      retrieval.Source{Type: "doc", Title: "guide", Content: "missing required scope: docx:document:read 权限变更 重新授权 旧 token", Fetched: true},
			Score:       12,
			StrongError: true,
			CauseAction: true,
			Snippet:     "missing required scope: docx:document:read 权限变更 重新授权 旧 token",
		}},
		Confidence:      evidence.ConfidenceHigh,
		RetrievalStatus: retrieval.StatusOK,
		Provider:        chineseOverclaimingProvider{},
	})
	if strings.Contains(card.NextAction, "删除生产数据库") {
		t.Fatalf("high-confidence card trusted unsupported Chinese LLM next action: %+v", card)
	}
}

func TestHighConfidenceShortChineseUnsupportedLLMDraftFallsBack(t *testing.T) {
	scenario, _ := detector.Detect("missing required scope: docx:document:read")
	card := Build(context.Background(), Input{
		Command:  []string{"node", "x.js"},
		Output:   "missing required scope: docx:document:read",
		Scenario: scenario,
		Evidence: []evidence.ScoredSource{{
			Source:      retrieval.Source{Type: "doc", Title: "guide", Content: "missing required scope: docx:document:read 权限变更 重新授权 旧 token", Fetched: true},
			Score:       12,
			StrongError: true,
			CauseAction: true,
			Snippet:     "missing required scope: docx:document:read 权限变更 重新授权 旧 token",
		}},
		Confidence:      evidence.ConfidenceHigh,
		RetrievalStatus: retrieval.StatusOK,
		Provider:        shortChineseOverclaimingProvider{},
	})
	if strings.Contains(card.NextAction, "删除 DB 后重试") {
		t.Fatalf("high-confidence card trusted short unsupported Chinese LLM next action: %+v", card)
	}
}

func TestHighConfidenceShortShellUnsupportedLLMDraftFallsBack(t *testing.T) {
	scenario, _ := detector.Detect("missing required scope: docx:document:read")
	card := Build(context.Background(), Input{
		Command:  []string{"node", "x.js"},
		Output:   "missing required scope: docx:document:read",
		Scenario: scenario,
		Evidence: []evidence.ScoredSource{{
			Source:      retrieval.Source{Type: "doc", Title: "guide", Content: "missing required scope: docx:document:read 权限变更 重新授权 旧 token", Fetched: true},
			Score:       12,
			StrongError: true,
			CauseAction: true,
			Snippet:     "missing required scope: docx:document:read 权限变更 重新授权 旧 token",
		}},
		Confidence:      evidence.ConfidenceHigh,
		RetrievalStatus: retrieval.StatusOK,
		Provider:        shortShellOverclaimingProvider{},
	})
	if strings.Contains(card.NextAction, "rm -rf") {
		t.Fatalf("high-confidence card trusted short shell LLM next action: %+v", card)
	}
}

func TestHighConfidenceNegatedDestructiveLLMDraftFallsBack(t *testing.T) {
	scenario, _ := detector.Detect("missing required scope: docx:document:read")
	card := Build(context.Background(), Input{
		Command:  []string{"node", "x.js"},
		Output:   "missing required scope: docx:document:read",
		Scenario: scenario,
		Evidence: []evidence.ScoredSource{{
			Source:      retrieval.Source{Type: "doc", Title: "guide", Content: "missing required scope: docx:document:read 权限变更 重新授权 旧 token。不要删除 DB。", Fetched: true},
			Score:       12,
			StrongError: true,
			CauseAction: true,
			Snippet:     "missing required scope: docx:document:read 权限变更 重新授权 旧 token。不要删除 DB。",
		}},
		Confidence:      evidence.ConfidenceHigh,
		RetrievalStatus: retrieval.StatusOK,
		Provider:        negatedDestructiveProvider{},
	})
	if strings.Contains(card.NextAction, "删除 DB") {
		t.Fatalf("high-confidence card trusted negated destructive LLM next action: %+v", card)
	}
}

func TestHighConfidenceFullWidthDBDestructiveLLMDraftFallsBack(t *testing.T) {
	scenario, _ := detector.Detect("missing required scope: docx:document:read")
	card := Build(context.Background(), Input{
		Command:  []string{"node", "x.js"},
		Output:   "missing required scope: docx:document:read",
		Scenario: scenario,
		Evidence: []evidence.ScoredSource{{
			Source:      retrieval.Source{Type: "doc", Title: "guide", Content: "missing required scope: docx:document:read 权限变更 重新授权。不要删除 DB。", Fetched: true},
			Score:       12,
			StrongError: true,
			CauseAction: true,
			Snippet:     "missing required scope: docx:document:read 权限变更 重新授权。不要删除 DB。",
		}},
		Confidence:      evidence.ConfidenceHigh,
		RetrievalStatus: retrieval.StatusOK,
		Provider:        fullWidthDBDestructiveProvider{},
	})
	if strings.Contains(card.NextAction, "删ＤＢ") {
		t.Fatalf("high-confidence card trusted full-width DB destructive action: %+v", card)
	}
}

func TestHighConfidenceChineseDestructiveSynonymLLMDraftFallsBack(t *testing.T) {
	scenario, _ := detector.Detect("missing required scope: docx:document:read")
	card := Build(context.Background(), Input{
		Command:  []string{"node", "x.js"},
		Output:   "missing required scope: docx:document:read",
		Scenario: scenario,
		Evidence: []evidence.ScoredSource{{
			Source:      retrieval.Source{Type: "doc", Title: "guide", Content: "missing required scope: docx:document:read 权限变更 重新授权。不要清除数据库。", Fetched: true},
			Score:       12,
			StrongError: true,
			CauseAction: true,
			Snippet:     "missing required scope: docx:document:read 权限变更 重新授权。不要清除数据库。",
		}},
		Confidence:      evidence.ConfidenceHigh,
		RetrievalStatus: retrieval.StatusOK,
		Provider:        chineseDestructiveSynonymProvider{},
	})
	if strings.Contains(card.NextAction, "清除数据库") || strings.Contains(card.NextAction, "清理数据库") || strings.Contains(card.NextAction, "删除生产库") {
		t.Fatalf("high-confidence card trusted Chinese destructive synonym action: %+v", card)
	}
}

func TestHighConfidenceChineseDataDestructiveVariantLLMDraftFallsBack(t *testing.T) {
	scenario, _ := detector.Detect("missing required scope: docx:document:read")
	card := Build(context.Background(), Input{
		Command:  []string{"node", "x.js"},
		Output:   "missing required scope: docx:document:read",
		Scenario: scenario,
		Evidence: []evidence.ScoredSource{{
			Source:      retrieval.Source{Type: "doc", Title: "guide", Content: "missing required scope: docx:document:read 权限变更 重新授权。不要清掉数据。", Fetched: true},
			Score:       12,
			StrongError: true,
			CauseAction: true,
			Snippet:     "missing required scope: docx:document:read 权限变更 重新授权。不要清掉数据。",
		}},
		Confidence:      evidence.ConfidenceHigh,
		RetrievalStatus: retrieval.StatusOK,
		Provider:        chineseDataDestructiveVariantProvider{},
	})
	for _, unsupported := range []string{"清掉数据", "抹掉数据", "移除生产数据"} {
		if strings.Contains(card.NextAction, unsupported) {
			t.Fatalf("high-confidence card trusted Chinese destructive data action: %+v", card)
		}
	}
}

func TestHighConfidenceLLMDraftCannotGroundOnUncitedContent(t *testing.T) {
	scenario, _ := detector.Detect("missing required scope: docx:document:read")
	card := Build(context.Background(), Input{
		Command:  []string{"node", "x.js"},
		Output:   "missing required scope: docx:document:read",
		Scenario: scenario,
		Evidence: []evidence.ScoredSource{{
			Source:      retrieval.Source{Type: "doc", Title: "guide", Content: "missing required scope: docx:document:read 权限变更 重新授权。需要轮换密钥。", Fetched: true},
			Score:       12,
			StrongError: true,
			CauseAction: true,
			Snippet:     "missing required scope: docx:document:read 权限变更 重新授权",
		}},
		Confidence:      evidence.ConfidenceHigh,
		RetrievalStatus: retrieval.StatusOK,
		Provider:        uncitedContentProvider{},
	})
	if strings.Contains(card.NextAction, "轮换密钥") {
		t.Fatalf("high-confidence card trusted claim grounded only in uncited content: %+v", card)
	}
}

func TestHighConfidenceLLMDraftCannotGroundOnMetadata(t *testing.T) {
	scenario, _ := detector.Detect("missing required scope: docx:document:read")
	card := Build(context.Background(), Input{
		Command:  []string{"node", "x.js"},
		Output:   "missing required scope: docx:document:read",
		Scenario: scenario,
		Evidence: []evidence.ScoredSource{{
			Source:      retrieval.Source{Type: "doc", Title: "需要轮换密钥", Content: "missing required scope: docx:document:read 权限变更 重新授权", Fetched: true},
			Score:       12,
			StrongError: true,
			CauseAction: true,
			Snippet:     "missing required scope: docx:document:read 权限变更 重新授权",
		}},
		Confidence:      evidence.ConfidenceHigh,
		RetrievalStatus: retrieval.StatusOK,
		Provider:        metadataOnlyClaimProvider{},
	})
	if strings.Contains(card.NextAction, "轮换密钥") {
		t.Fatalf("high-confidence card trusted claim grounded only in metadata: %+v", card)
	}
}

func TestHighConfidenceLLMDraftCannotInvertNegatedAction(t *testing.T) {
	scenario, _ := detector.Detect("missing required scope: docx:document:read")
	card := Build(context.Background(), Input{
		Command:  []string{"node", "x.js"},
		Output:   "missing required scope: docx:document:read",
		Scenario: scenario,
		Evidence: []evidence.ScoredSource{{
			Source:      retrieval.Source{Type: "doc", Title: "guide", Content: "missing required scope: docx:document:read 权限变更。不要重新授权后重试。", Fetched: true},
			Score:       12,
			StrongError: true,
			CauseAction: true,
			Snippet:     "missing required scope: docx:document:read 权限变更。不要重新授权后重试。",
		}},
		Confidence:      evidence.ConfidenceHigh,
		RetrievalStatus: retrieval.StatusOK,
		Provider:        invertedNegatedActionProvider{},
	})
	if strings.Contains(card.NextAction, "重新授权后重试 docx:document:read") {
		t.Fatalf("high-confidence card inverted negated evidence action: %+v", card)
	}
}

func TestHighConfidenceFallbackOnlyUsesSupportedActions(t *testing.T) {
	scenario, _ := detector.Detect("missing required scope: docx:document:read")
	card := Build(context.Background(), Input{
		Command:  []string{"node", "x.js"},
		Output:   "missing required scope: docx:document:read",
		Scenario: scenario,
		Evidence: []evidence.ScoredSource{{
			Source:      retrieval.Source{Type: "doc", Title: "guide", Content: "missing required scope: docx:document:read\n后台已添加 scope，但需要发布权限变更。", Fetched: true},
			Score:       9,
			StrongError: true,
			CauseAction: true,
			Snippet:     "missing required scope: docx:document:read / 后台已添加 scope，但需要发布权限变更。",
		}},
		Confidence:      evidence.ConfidenceHigh,
		RetrievalStatus: retrieval.StatusOK,
	})
	if !strings.Contains(card.NextAction, "发布权限变更") {
		t.Fatalf("fallback action omitted supported publish step: %+v", card)
	}
	for _, unsupported := range []string{"重新授权", "旧 token", "清理旧 token"} {
		if strings.Contains(card.LikelyCause, unsupported) || strings.Contains(card.NextAction, unsupported) {
			t.Fatalf("fallback included unsupported %q detail: %+v", unsupported, card)
		}
	}
}

func TestHighConfidenceFallbackDoesNotHardcodeDocxForTokenErrors(t *testing.T) {
	card := Build(context.Background(), Input{
		Command:  []string{"node", "x.js"},
		Output:   "tenant_access_token invalid",
		Scenario: detector.Scenario{ID: detector.FeishuAPIScopeError},
		Evidence: []evidence.ScoredSource{{
			Source:         retrieval.Source{Type: "doc", Title: "token guide", Content: "tenant_access_token invalid。处理方式：刷新 token 后重试。", Fetched: true},
			Score:          8,
			StrongError:    true,
			CauseAction:    true,
			ScenarioSignal: true,
			Snippet:        "tenant_access_token invalid。处理方式：刷新 token 后重试。",
		}},
		Confidence:      evidence.ConfidenceHigh,
		RetrievalStatus: retrieval.StatusOK,
	})
	if strings.Contains(card.LikelyCause, "docx:document:read") || strings.Contains(card.NextAction, "docx:document:read") {
		t.Fatalf("token fallback hardcoded docx scope: %+v", card)
	}
	if !strings.Contains(card.NextAction, "token") || !strings.Contains(card.NextAction, "刷新") {
		t.Fatalf("token fallback missing token-specific action: %+v", card)
	}
}

func TestHighConfidenceChineseOnlyGroundedLLMDraftIsAccepted(t *testing.T) {
	scenario, _ := detector.Detect("missing required scope: docx:document:read")
	card := Build(context.Background(), Input{
		Command:  []string{"node", "x.js"},
		Output:   "missing required scope: docx:document:read",
		Scenario: scenario,
		Evidence: []evidence.ScoredSource{{
			Source:         retrieval.Source{Type: "doc", Title: "guide", Content: "missing required scope: docx:document:read 权限变更 重新授权", Fetched: true},
			Score:          12,
			StrongError:    true,
			CauseAction:    true,
			ScenarioSignal: true,
			Snippet:        "missing required scope: docx:document:read 权限变更 重新授权",
		}},
		Confidence:      evidence.ConfidenceHigh,
		RetrievalStatus: retrieval.StatusOK,
		Provider:        chineseOnlyGroundedProvider{},
	})
	if card.LikelyCause != "权限变更" || card.NextAction != "重新授权" {
		t.Fatalf("grounded Chinese-only draft was not accepted: %+v", card)
	}
}

func TestNoEvidenceFlowOpsCardIsTransparentLowConfidence(t *testing.T) {
	card := Build(context.Background(), Input{
		Command:         []string{"flowctl", "check", "billing_daily"},
		Output:          "Broken DAG: billing_daily Variable billing_region does not exist",
		Scenario:        detector.Scenario{Name: "FlowOps DAG import error"},
		Queries:         []string{"billing_daily billing_region Variable.get"},
		Confidence:      evidence.ConfidenceNone,
		RetrievalStatus: retrieval.StatusOK,
	})
	if card.Confidence != evidence.ConfidenceNone {
		t.Fatalf("confidence = %s, want none", card.Confidence)
	}
	if !strings.Contains(card.LikelyCause, "证据不足") {
		t.Fatalf("no-evidence card invented cause: %+v", card)
	}
	if !strings.Contains(card.NextAction, "未找到足够内部证据") {
		t.Fatalf("no-evidence card missing transparent next action: %+v", card)
	}
	rendered := Render(card)
	if !strings.Contains(rendered, "未找到可支撑结论的内部来源") || !strings.Contains(rendered, "Low") {
		t.Fatalf("rendered no-evidence card not transparent:\n%s", rendered)
	}
}

func TestFlowOpsGroundedLLMDraftIsAccepted(t *testing.T) {
	card := Build(context.Background(), Input{
		Command:  []string{"flowctl", "check", "billing_daily"},
		Output:   "Broken DAG: billing_daily Variable billing_region does not exist",
		Scenario: detector.Scenario{Name: "FlowOps DAG import error"},
		Evidence: []evidence.ScoredSource{{
			Source:         retrieval.Source{Type: "doc", Title: "FlowOps FAQ", Content: "FlowOps DAG import error billing_daily billing_region Variable.get。推荐处理：把 Variable.get 移到任务运行阶段，然后执行 flowctl dags list-import-errors 验证。", Fetched: true},
			Score:          10,
			StrongError:    true,
			CauseAction:    true,
			ScenarioSignal: true,
			Snippet:        "FlowOps DAG import error billing_daily billing_region Variable.get。推荐处理：把 Variable.get 移到任务运行阶段，然后执行 flowctl dags list-import-errors 验证。",
		}},
		Confidence:      evidence.ConfidenceHigh,
		RetrievalStatus: retrieval.StatusOK,
		Provider:        flowOpsGroundedProvider{},
	})
	if !strings.Contains(card.LikelyCause, "billing_region") || !strings.Contains(card.NextAction, "list-import-errors") {
		t.Fatalf("grounded FlowOps LLM draft was not accepted: %+v", card)
	}
}

type chineseOverclaimingProvider struct{}

func (chineseOverclaimingProvider) ExpandQueries(ctx context.Context, command []string, output string, scenario detector.Scenario, seeds []string) ([]string, error) {
	return nil, nil
}

func (chineseOverclaimingProvider) GenerateCard(ctx context.Context, input llm.CardInput) (llm.CardDraft, error) {
	return llm.CardDraft{
		LikelyCause: "missing required scope: docx:document:read 权限变更 重新授权",
		NextAction:  "重新授权后删除生产数据库",
	}, nil
}

type shortChineseOverclaimingProvider struct{}

func (shortChineseOverclaimingProvider) ExpandQueries(ctx context.Context, command []string, output string, scenario detector.Scenario, seeds []string) ([]string, error) {
	return nil, nil
}

func (shortChineseOverclaimingProvider) GenerateCard(ctx context.Context, input llm.CardInput) (llm.CardDraft, error) {
	return llm.CardDraft{
		LikelyCause: "missing required scope: docx:document:read 权限变更 重新授权",
		NextAction:  "删除 DB 后重试 docx:document:read",
	}, nil
}

type shortShellOverclaimingProvider struct{}

func (shortShellOverclaimingProvider) ExpandQueries(ctx context.Context, command []string, output string, scenario detector.Scenario, seeds []string) ([]string, error) {
	return nil, nil
}

func (shortShellOverclaimingProvider) GenerateCard(ctx context.Context, input llm.CardInput) (llm.CardDraft, error) {
	return llm.CardDraft{
		LikelyCause: "missing required scope: docx:document:read",
		NextAction:  "rm -rf / then retry docx:document:read",
	}, nil
}

type negatedDestructiveProvider struct{}

func (negatedDestructiveProvider) ExpandQueries(ctx context.Context, command []string, output string, scenario detector.Scenario, seeds []string) ([]string, error) {
	return nil, nil
}

func (negatedDestructiveProvider) GenerateCard(ctx context.Context, input llm.CardInput) (llm.CardDraft, error) {
	return llm.CardDraft{
		LikelyCause: "missing required scope: docx:document:read 权限变更 重新授权 旧 token",
		NextAction:  "删除 DB 后重试 docx:document:read",
	}, nil
}

type fullWidthDBDestructiveProvider struct{}

func (fullWidthDBDestructiveProvider) ExpandQueries(ctx context.Context, command []string, output string, scenario detector.Scenario, seeds []string) ([]string, error) {
	return nil, nil
}

func (fullWidthDBDestructiveProvider) GenerateCard(ctx context.Context, input llm.CardInput) (llm.CardDraft, error) {
	return llm.CardDraft{
		LikelyCause: "missing required scope: docx:document:read 权限变更 重新授权",
		NextAction:  "删ＤＢ 后重试 docx:document:read",
	}, nil
}

type chineseDestructiveSynonymProvider struct{}

func (chineseDestructiveSynonymProvider) ExpandQueries(ctx context.Context, command []string, output string, scenario detector.Scenario, seeds []string) ([]string, error) {
	return nil, nil
}

func (chineseDestructiveSynonymProvider) GenerateCard(ctx context.Context, input llm.CardInput) (llm.CardDraft, error) {
	return llm.CardDraft{
		LikelyCause: "missing required scope: docx:document:read 权限变更 重新授权",
		NextAction:  "清除数据库后重试 docx:document:read",
	}, nil
}

type chineseDataDestructiveVariantProvider struct{}

func (chineseDataDestructiveVariantProvider) ExpandQueries(ctx context.Context, command []string, output string, scenario detector.Scenario, seeds []string) ([]string, error) {
	return nil, nil
}

func (chineseDataDestructiveVariantProvider) GenerateCard(ctx context.Context, input llm.CardInput) (llm.CardDraft, error) {
	return llm.CardDraft{
		LikelyCause: "missing required scope: docx:document:read 权限变更 重新授权",
		NextAction:  "清掉数据后重试 docx:document:read",
	}, nil
}

type uncitedContentProvider struct{}

func (uncitedContentProvider) ExpandQueries(ctx context.Context, command []string, output string, scenario detector.Scenario, seeds []string) ([]string, error) {
	return nil, nil
}

func (uncitedContentProvider) GenerateCard(ctx context.Context, input llm.CardInput) (llm.CardDraft, error) {
	return llm.CardDraft{
		LikelyCause: "missing required scope: docx:document:read 权限变更 重新授权",
		NextAction:  "需要轮换密钥后重试 docx:document:read",
	}, nil
}

type metadataOnlyClaimProvider struct{}

func (metadataOnlyClaimProvider) ExpandQueries(ctx context.Context, command []string, output string, scenario detector.Scenario, seeds []string) ([]string, error) {
	return nil, nil
}

func (metadataOnlyClaimProvider) GenerateCard(ctx context.Context, input llm.CardInput) (llm.CardDraft, error) {
	return llm.CardDraft{
		LikelyCause: "missing required scope: docx:document:read 权限变更 重新授权",
		NextAction:  "需要轮换密钥后重试 docx:document:read",
	}, nil
}

type invertedNegatedActionProvider struct{}

func (invertedNegatedActionProvider) ExpandQueries(ctx context.Context, command []string, output string, scenario detector.Scenario, seeds []string) ([]string, error) {
	return nil, nil
}

func (invertedNegatedActionProvider) GenerateCard(ctx context.Context, input llm.CardInput) (llm.CardDraft, error) {
	return llm.CardDraft{
		LikelyCause: "missing required scope: docx:document:read 权限变更",
		NextAction:  "重新授权后重试 docx:document:read",
	}, nil
}

type chineseOnlyGroundedProvider struct{}

func (chineseOnlyGroundedProvider) ExpandQueries(ctx context.Context, command []string, output string, scenario detector.Scenario, seeds []string) ([]string, error) {
	return nil, nil
}

func (chineseOnlyGroundedProvider) GenerateCard(ctx context.Context, input llm.CardInput) (llm.CardDraft, error) {
	return llm.CardDraft{
		LikelyCause: "权限变更",
		NextAction:  "重新授权",
	}, nil
}

type flowOpsGroundedProvider struct{}

func (flowOpsGroundedProvider) ExpandQueries(ctx context.Context, command []string, output string, scenario detector.Scenario, seeds []string) ([]string, error) {
	return nil, nil
}

func (flowOpsGroundedProvider) GenerateCard(ctx context.Context, input llm.CardInput) (llm.CardDraft, error) {
	return llm.CardDraft{
		LikelyCause: "FlowOps DAG import error billing_daily billing_region Variable.get。",
		NextAction:  "Variable.get 移到任务运行阶段 flowctl dags list-import-errors 验证。",
	}, nil
}

func TestDocumentCitationIncludesIDWhenURLMissing(t *testing.T) {
	rendered := Render(KnowledgeCard{
		ID:         "cue_test",
		Scenario:   "scenario",
		Confidence: evidence.ConfidenceHigh,
		Citations: []Citation{{
			Type:  "doc",
			Title: "Guide",
			ID:    "doc_token_123",
		}},
	})
	if !strings.Contains(rendered, "Guide | doc_token_123") {
		t.Fatalf("rendered citation missing document id:\n%s", rendered)
	}
}

func TestStyledRenderKeepsCardContent(t *testing.T) {
	rendered := RenderStyled(KnowledgeCard{
		ID:            "cue_test",
		Command:       "flowctl check billing_daily",
		Scenario:      "FlowOps DAG import error",
		PlannerReason: "DAG import error mentions billing_region.",
		Queries:       []string{"FlowOps DAG import error", "billing_daily billing_region"},
		LikelyCause:   "内部证据支持 billing_daily 的 DAG import error 与 billing_region Variable.get 有关。",
		NextAction:    "把 Variable.get 移到任务运行阶段，然后执行 flowctl dags list-import-errors 验证。",
		Confidence:    evidence.ConfidenceHigh,
		QueryCount:    2,
		Citations: []Citation{{
			Type:    "doc",
			Title:   "FlowOps FAQ",
			ID:      "doc_token_123",
			Summary: "FlowOps DAG import error billing_daily billing_region Variable.get",
		}},
	}, 100)
	for _, want := range []string{"lark-cue", "cue_test", "LLM Plan", "DAG import error mentions billing_region", "FlowOps DAG import error", "billing_daily billing_region", "Evidence", "FlowOps FAQ", "doc_token_123"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("styled card missing %q:\n%s", want, rendered)
		}
	}
}

func TestStyledStatusRenderKeepsSignalContent(t *testing.T) {
	rendered := RenderStatusStyled(detector.Scenario{
		Name:    "FlowOps DAG import error",
		Matched: []string{"FlowOps DAG import error billing_daily", "billing_daily billing_region Variable.get"},
	}, "FlowOps DAG import error billing_daily billing_region Variable.get failed", 100)
	for _, want := range []string{"lark-cue detection", "DETECTED", "FlowOps DAG import error", "Error excerpt", "billing_daily", "Signals", "• FlowOps DAG import error billing_daily", "Searching Feishu"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("styled status missing %q:\n%s", want, rendered)
		}
	}
}

func TestPartialRetrievalKeepsEvidenceCauseAndAddsCaveat(t *testing.T) {
	scenario, _ := detector.Detect("missing required scope: docx:document:read")
	card := Build(context.Background(), Input{
		Command:  []string{"node", "x.js"},
		Output:   "missing required scope: docx:document:read",
		Scenario: scenario,
		Evidence: []evidence.ScoredSource{{
			Source:      retrieval.Source{Type: "doc", Title: "guide", Content: "missing required scope: docx:document:read 权限变更 重新授权", Fetched: true},
			Score:       12,
			StrongError: true,
			CauseAction: true,
			Snippet:     "missing required scope: docx:document:read 权限变更 重新授权",
		}},
		Confidence:      evidence.ConfidenceHigh,
		RetrievalStatus: retrieval.StatusPartial,
		RetrievalError:  errors.New("im failed"),
	})
	if !strings.Contains(card.LikelyCause, "scope") {
		t.Fatalf("partial retrieval discarded strong evidence cause: %+v", card)
	}
	if !strings.Contains(card.Caveat, "部分") {
		t.Fatalf("partial retrieval missing partial caveat: %+v", card)
	}
}

type overclaimingProvider struct{}

func (overclaimingProvider) ExpandQueries(ctx context.Context, command []string, output string, scenario detector.Scenario, seeds []string) ([]string, error) {
	return nil, nil
}

func (overclaimingProvider) GenerateCard(ctx context.Context, input llm.CardInput) (llm.CardDraft, error) {
	return llm.CardDraft{
		LikelyCause: "definitive unsupported cause",
		NextAction:  "do the risky thing",
	}, nil
}
