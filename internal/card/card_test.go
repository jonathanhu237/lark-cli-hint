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
	for _, want := range []string{"LLM Plan", "Reason: 失败输出命中内部权限配置问题", "- docx:document:read", "- 权限变更 重新授权", "Action Plan", "1. 按引用来源给出的处理路径执行。", "Sources", "guide", "High"} {
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
		ActionPlan:  []string{"按内部文档处理。"},
		Confidence:  evidence.ConfidenceHigh,
	})
	if strings.Contains(rendered, "LLM Plan") {
		t.Fatalf("rendered empty LLM plan:\n%s", rendered)
	}
}

func TestLLMActionPlanIsAcceptedAsOrderedSequence(t *testing.T) {
	scenario, _ := detector.Detect("missing required scope: docx:document:read")
	card := Build(context.Background(), Input{
		Command:  []string{"node", "x.js"},
		Output:   "missing required scope: docx:document:read",
		Scenario: scenario,
		Evidence: []evidence.ScoredSource{{
			Source:         retrieval.Source{Type: "doc", Title: "guide", Content: "missing required scope: docx:document:read 权限变更 重新授权 旧 token", Fetched: true},
			Score:          12,
			StrongError:    true,
			CauseAction:    true,
			ScenarioSignal: true,
			Snippet:        "missing required scope: docx:document:read 权限变更 重新授权 旧 token",
		}},
		Confidence:      evidence.ConfidenceHigh,
		RetrievalStatus: retrieval.StatusOK,
		Provider: actionPlanProvider{draft: llm.CardDraft{
			LikelyCause: "missing required scope: docx:document:read 权限变更",
			ActionPlan: []string{
				"1. 在开放平台确认 docx:document:read 权限。",
				"- 发布权限变更。",
				"/ - 发布权限变更。",
				"3) 重新授权本地身份后重试。",
			},
		}},
	})
	if len(card.ActionPlan) != 3 {
		t.Fatalf("ActionPlan = %#v, want 3 cleaned unique steps", card.ActionPlan)
	}
	for _, unexpected := range []string{"1.", "-", "/ -", "3)"} {
		if strings.Contains(strings.Join(card.ActionPlan, "\n"), unexpected) {
			t.Fatalf("ActionPlan was not cleaned: %#v", card.ActionPlan)
		}
	}
	rendered := Render(card)
	for _, want := range []string{"Action Plan", "1. 在开放平台确认", "2. 发布权限变更", "3. 重新授权本地身份后重试"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered action plan missing %q:\n%s", want, rendered)
		}
	}
}

func TestLowConfidenceLLMDraftKeepsPlanButForcesCaveat(t *testing.T) {
	scenario, _ := detector.Detect("permission denied")
	card := Build(context.Background(), Input{
		Command:  []string{"node", "x.js"},
		Output:   "permission denied",
		Scenario: scenario,
		Evidence: []evidence.ScoredSource{{
			Source:  retrieval.Source{Type: "doc", Title: "weak", Content: "权限", Fetched: true},
			Score:   3,
			Snippet: "权限",
		}},
		Confidence:      evidence.ConfidenceLow,
		RetrievalStatus: retrieval.StatusOK,
		Provider: actionPlanProvider{draft: llm.CardDraft{
			LikelyCause: "权限",
			ActionPlan:  []string{"先打开内部文档核对权限说明。", "按团队流程处理后重试。"},
		}},
	})
	if strings.Contains(card.LikelyCause, "definitive") {
		t.Fatalf("low-confidence card trusted overconfident cause: %+v", card)
	}
	if len(card.ActionPlan) != 2 || !strings.Contains(card.ActionPlan[0], "内部文档") {
		t.Fatalf("low-confidence card did not keep LLM action sequence: %+v", card)
	}
	if card.Caveat == "" {
		t.Fatalf("low-confidence card missing caveat: %+v", card)
	}
}

func TestHighConfidenceUnsupportedLLMTextFallsBackButKeepsPlan(t *testing.T) {
	scenario, _ := detector.Detect("missing required scope: docx:document:read")
	card := Build(context.Background(), Input{
		Command:  []string{"node", "x.js"},
		Output:   "missing required scope: docx:document:read",
		Scenario: scenario,
		Evidence: []evidence.ScoredSource{{
			Source:         retrieval.Source{Type: "doc", Title: "guide", Content: "missing required scope: docx:document:read 权限变更 重新授权 旧 token", Fetched: true},
			Score:          12,
			StrongError:    true,
			CauseAction:    true,
			ScenarioSignal: true,
			Snippet:        "missing required scope: docx:document:read 权限变更 重新授权 旧 token",
		}},
		Confidence:      evidence.ConfidenceHigh,
		RetrievalStatus: retrieval.StatusOK,
		Provider: actionPlanProvider{draft: llm.CardDraft{
			LikelyCause: "definitive unsupported cause",
			ActionPlan:  []string{"按 LLM 给出的处理序列执行第一步。"},
		}},
	})
	if strings.Contains(card.LikelyCause, "definitive") {
		t.Fatalf("high-confidence card trusted unsupported LLM text: %+v", card)
	}
	if len(card.ActionPlan) != 1 || !strings.Contains(card.ActionPlan[0], "LLM") {
		t.Fatalf("high-confidence card did not keep LLM action sequence: %+v", card)
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
	if len(card.ActionPlan) != 1 || !strings.Contains(card.ActionPlan[0], "未找到足够内部证据") {
		t.Fatalf("no-evidence card missing transparent action plan: %+v", card)
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
		Provider: actionPlanProvider{draft: llm.CardDraft{
			LikelyCause: "FlowOps DAG import error billing_daily billing_region Variable.get。",
			ActionPlan:  []string{"把 Variable.get 移到任务运行阶段。", "执行 flowctl dags list-import-errors 验证。"},
		}},
	})
	if !strings.Contains(card.LikelyCause, "billing_region") || len(card.ActionPlan) != 2 || !strings.Contains(strings.Join(card.ActionPlan, "\n"), "list-import-errors") {
		t.Fatalf("grounded FlowOps LLM draft was not accepted: %+v", card)
	}
}

func TestFallbackDoesNotExtractConcreteActionFromEvidence(t *testing.T) {
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
	})
	if len(card.ActionPlan) != 2 {
		t.Fatalf("fallback ActionPlan = %#v, want generic two-step fallback", card.ActionPlan)
	}
	if strings.Contains(strings.Join(card.ActionPlan, "\n"), "Variable.get") || strings.Contains(strings.Join(card.ActionPlan, "\n"), "list-import-errors") {
		t.Fatalf("fallback extracted concrete repair rules from evidence: %+v", card)
	}
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
		ActionPlan:    []string{"把 Variable.get 移到任务运行阶段。", "执行 flowctl dags list-import-errors 验证。"},
		Confidence:    evidence.ConfidenceHigh,
		QueryCount:    2,
		Citations: []Citation{{
			Type:    "doc",
			Title:   "FlowOps FAQ",
			ID:      "doc_token_123",
			Summary: "FlowOps DAG import error billing_daily billing_region Variable.get",
		}},
	}, 100)
	for _, want := range []string{"lark-cue", "cue_test", "LLM Plan", "DAG import error mentions billing_region", "Action plan", "FlowOps DAG import error", "billing_daily billing_region", "Evidence", "FlowOps FAQ", "doc_token_123"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("styled card missing %q:\n%s", want, rendered)
		}
	}
}

func TestStyledStatusRenderKeepsSignalContent(t *testing.T) {
	rendered := RenderPlannerStatusStyled(detector.Scenario{
		Name:    "FlowOps DAG import error",
		Matched: []string{"FlowOps DAG import error billing_daily", "billing_daily billing_region Variable.get"},
	}, "DAG import error mentions billing_region.", []string{"FlowOps DAG import error billing_daily", "billing_daily billing_region Variable.get"}, "FlowOps DAG import error billing_daily billing_region Variable.get failed", 100)
	for _, want := range []string{"lark-cue LLM plan", "PLANNED", "FlowOps DAG import error", "Reason", "DAG import error mentions billing_region", "Queries", "• FlowOps DAG import error billing_daily", "Error excerpt", "Searching Feishu"} {
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

type actionPlanProvider struct {
	draft llm.CardDraft
}

func (p actionPlanProvider) GenerateCard(ctx context.Context, input llm.CardInput) (llm.CardDraft, error) {
	return p.draft, nil
}
