package card

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/charmbracelet/lipgloss"

	"lark-cue/internal/detector"
	"lark-cue/internal/evidence"
	"lark-cue/internal/llm"
	"lark-cue/internal/retrieval"
	"lark-cue/internal/runner"
)

type Input struct {
	Command         []string
	Output          string
	Scenario        detector.Scenario
	PlannerReason   string
	Queries         []string
	Evidence        []evidence.ScoredSource
	Confidence      evidence.Confidence
	RetrievalStatus retrieval.Status
	RetrievalError  error
	Provider        llm.CardProvider
	LLMStatus       *LLMStatus
}

type LLMStatus struct {
	Attempted bool
	Accepted  bool
	Error     string
}

type KnowledgeCard struct {
	ID              string
	Command         string
	Scenario        string
	PlannerReason   string
	Queries         []string
	LikelyCause     string
	ActionPlan      []string
	Caveat          string
	Citations       []Citation
	Confidence      evidence.Confidence
	RetrievalStatus retrieval.Status
	RetrievalError  string
	QueryCount      int
	Feedback        string
	LatencyMS       int64
	CreatedAt       time.Time
}

type Citation struct {
	Type      string `json:"type"`
	Title     string `json:"title,omitempty"`
	URL       string `json:"url,omitempty"`
	ID        string `json:"id,omitempty"`
	ChatName  string `json:"chat_name,omitempty"`
	Sender    string `json:"sender,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
	Summary   string `json:"summary,omitempty"`
}

func (k KnowledgeCard) LLMPlan() string {
	return renderPlan(k)
}

func Build(ctx context.Context, input Input) KnowledgeCard {
	card := KnowledgeCard{
		ID:              newID(),
		Command:         runner.CommandString(input.Command),
		Scenario:        scenarioLabel(input.Scenario),
		PlannerReason:   strings.TrimSpace(input.PlannerReason),
		Queries:         cleanQueries(input.Queries),
		Confidence:      input.Confidence,
		RetrievalStatus: input.RetrievalStatus,
		CreatedAt:       time.Now(),
	}
	card.QueryCount = len(card.Queries)
	if input.RetrievalError != nil {
		card.RetrievalError = input.RetrievalError.Error()
	}
	for _, item := range input.Evidence {
		card.Citations = append(card.Citations, citationFrom(item))
	}

	if input.Provider != nil && len(input.Evidence) > 0 {
		if input.LLMStatus != nil {
			input.LLMStatus.Attempted = true
		}
		if draft, err := input.Provider.GenerateCard(ctx, llm.CardInput{
			Command:    input.Command,
			Output:     input.Output,
			Scenario:   input.Scenario,
			Evidence:   input.Evidence,
			Confidence: input.Confidence,
		}); err != nil {
			if input.LLMStatus != nil {
				input.LLMStatus.Error = err.Error()
			}
		} else {
			actionPlan := cleanActionPlan(draft.ActionPlan)
			hasDraftText := strings.TrimSpace(draft.LikelyCause) != "" || strings.TrimSpace(draft.Caveat) != ""
			if hasDraftText || len(actionPlan) > 0 {
				textGrounded := !hasDraftText || draftTextGrounded(draft, input.Evidence)
				if textGrounded {
					card.LikelyCause = strings.TrimSpace(draft.LikelyCause)
					card.Caveat = strings.TrimSpace(draft.Caveat)
				} else {
					if input.LLMStatus != nil && hasDraftText {
						input.LLMStatus.Error = "draft text was not grounded in cited snippets"
					}
				}
				if textGrounded || len(actionPlan) > 0 {
					if !textGrounded {
						card.LikelyCause = fallbackCause(input.Confidence, input.RetrievalStatus, input.RetrievalError, input.Evidence)
						card.Caveat = fallbackCaveat(input.Confidence, input.RetrievalStatus, input.RetrievalError)
					}
					card.ActionPlan = actionPlan
					if input.LLMStatus != nil {
						input.LLMStatus.Accepted = true
					}
					enforceConfidence(&card, input.Confidence, input.RetrievalStatus, input.RetrievalError, input.Evidence)
					if strings.TrimSpace(card.LikelyCause) == "" {
						card.LikelyCause = fallbackCause(input.Confidence, input.RetrievalStatus, input.RetrievalError, input.Evidence)
					}
					if len(card.ActionPlan) == 0 {
						card.ActionPlan = fallbackActionPlan(input.Confidence, input.Evidence)
					}
					return card
				}
			}
			if input.LLMStatus != nil && !hasDraftText && len(actionPlan) == 0 {
				input.LLMStatus.Error = "empty card draft"
			}
		}
	}

	card.LikelyCause = fallbackCause(input.Confidence, input.RetrievalStatus, input.RetrievalError, input.Evidence)
	card.Caveat = fallbackCaveat(input.Confidence, input.RetrievalStatus, input.RetrievalError)
	card.ActionPlan = fallbackActionPlan(input.Confidence, input.Evidence)
	return card
}

func Render(k KnowledgeCard) string {
	var b strings.Builder
	fmt.Fprintf(&b, "lark-cue knowledge card [%s]\n", k.ID)
	b.WriteString("Scenario\n")
	b.WriteString(k.Scenario + "\n\n")
	if plan := renderPlan(k); plan != "" {
		b.WriteString("LLM Plan\n")
		b.WriteString(plan + "\n\n")
	}
	b.WriteString("Likely Cause\n")
	b.WriteString(valueOr(k.LikelyCause, "未找到足够证据判断具体原因。") + "\n\n")
	b.WriteString("Action Plan\n")
	b.WriteString(renderActionPlan(k.ActionPlan) + "\n\n")
	b.WriteString("Sources\n")
	if len(k.Citations) == 0 {
		b.WriteString("- 未找到可支撑结论的内部来源。\n")
	} else {
		for _, citation := range k.Citations {
			b.WriteString("- " + renderCitation(citation) + "\n")
		}
	}
	b.WriteString("\nConfidence\n")
	switch k.Confidence {
	case evidence.ConfidenceHigh:
		b.WriteString("High. 检索证据同时命中错误信号和处理动作。\n")
	case evidence.ConfidenceLow:
		b.WriteString("Low. 当前只有部分内部证据支撑，请人工核对后采用。\n")
	default:
		b.WriteString("Low. 未找到足够强的内部知识证据。\n")
	}
	if k.Caveat != "" {
		b.WriteString("Caveat\n")
		b.WriteString(k.Caveat + "\n")
	}
	if k.RetrievalError != "" {
		b.WriteString("Retrieval\n")
		b.WriteString("Real Feishu retrieval failed: " + k.RetrievalError + "\n")
	}
	return b.String()
}

func RenderStyled(k KnowledgeCard, width int) string {
	if width < 60 {
		return Render(k)
	}
	cardWidth := clamp(width-4, 72, 104)
	contentWidth := cardWidth - 6

	accent := lipgloss.Color("#5A7CFF")
	ok := lipgloss.Color("#2EAD6B")
	warn := lipgloss.Color("#E0A11B")
	muted := lipgloss.Color("#7C8798")
	text := lipgloss.Color("#E7EAF0")

	box := lipgloss.NewStyle().
		Width(cardWidth).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accent).
		Padding(1, 2)
	title := lipgloss.NewStyle().Bold(true).Foreground(accent)
	label := lipgloss.NewStyle().Bold(true).Foreground(muted)
	body := lipgloss.NewStyle().Foreground(text).Width(contentWidth)
	metaStyle := lipgloss.NewStyle().Foreground(muted)
	highPill := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#071B10")).Background(ok).Padding(0, 1)
	lowPill := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#271B04")).Background(warn).Padding(0, 1)
	confidencePill := lowPill.Render("LOW")
	if k.Confidence == evidence.ConfidenceHigh {
		confidencePill = highPill.Render("HIGH")
	}

	headerParts := []string{
		title.Render("lark-cue"),
		metaStyle.Render(k.ID),
		confidencePill,
	}
	var sections []string
	sections = append(sections, strings.Join(headerParts, "  "))
	sections = append(sections, renderStyledKV(label, body, "Scenario", k.Scenario, contentWidth))
	if plan := renderPlan(k); plan != "" {
		sections = append(sections, renderStyledKV(label, body, "LLM Plan", plan, contentWidth))
	}
	sections = append(sections, renderStyledKV(label, body, "Likely cause", valueOr(k.LikelyCause, "未找到足够证据判断具体原因。"), contentWidth))
	sections = append(sections, renderStyledKV(label, body, "Action plan", renderActionPlan(k.ActionPlan), contentWidth))

	evidenceBlock := []string{label.Render("Evidence")}
	if len(k.Citations) == 0 {
		evidenceBlock = append(evidenceBlock, body.Render("没有找到可支撑结论的内部来源。"))
	} else {
		for i, citation := range k.Citations {
			evidenceBlock = append(evidenceBlock, renderStyledCitation(i+1, citation, contentWidth, accent, muted))
		}
	}
	sections = append(sections, strings.Join(evidenceBlock, "\n"))

	confidence := "Low. 未找到足够强的内部知识证据。"
	switch k.Confidence {
	case evidence.ConfidenceHigh:
		confidence = "High. 检索证据同时命中错误信号和处理动作。"
	case evidence.ConfidenceLow:
		confidence = "Low. 当前只有部分内部证据支撑，请人工核对后采用。"
	}
	sections = append(sections, renderStyledKV(label, body, "Confidence", confidence, contentWidth))
	if k.Caveat != "" {
		sections = append(sections, renderStyledKV(label, body, "Caveat", k.Caveat, contentWidth))
	}
	if k.RetrievalError != "" {
		sections = append(sections, renderStyledKV(label, body, "Retrieval", "Real Feishu retrieval failed: "+k.RetrievalError, contentWidth))
	}

	meta := fmt.Sprintf("command: %s  |  sources: %d  |  queries: %d", k.Command, len(k.Citations), k.QueryCount)
	sections = append(sections, metaStyle.Width(contentWidth).Render(wrapText(meta, contentWidth)))
	return box.Render(strings.Join(sections, "\n\n")) + "\n"
}

func RenderPlannerStatus(scenario detector.Scenario, reason string, queries []string) string {
	var b strings.Builder
	b.WriteString("\nlark-cue LLM plan\n")
	b.WriteString("Scenario\n")
	b.WriteString(scenario.Name + "\n\n")
	if strings.TrimSpace(reason) != "" {
		b.WriteString("Reason\n")
		b.WriteString(strings.TrimSpace(reason) + "\n\n")
	}
	if len(queries) > 0 {
		b.WriteString("Queries\n")
		for _, query := range cleanQueries(queries) {
			b.WriteString("- " + query + "\n")
		}
		b.WriteString("\n")
	}
	b.WriteString("Next\n")
	b.WriteString("Searching Feishu Docs/Wiki and IM evidence...\n")
	return b.String()
}

func RenderPlannerStatusStyled(scenario detector.Scenario, reason string, queries []string, output string, width int) string {
	if width < 60 {
		return RenderPlannerStatus(scenario, reason, queries)
	}
	cardWidth := clamp(width-4, 72, 104)
	contentWidth := cardWidth - 6

	accent := lipgloss.Color("#5A7CFF")
	muted := lipgloss.Color("#7C8798")
	text := lipgloss.Color("#E7EAF0")
	ok := lipgloss.Color("#2EAD6B")

	box := lipgloss.NewStyle().
		Width(cardWidth).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accent).
		Padding(1, 2)
	title := lipgloss.NewStyle().Bold(true).Foreground(accent).Render("lark-cue LLM plan")
	badge := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#071B10")).Background(ok).Padding(0, 1).Render("PLANNED")
	label := lipgloss.NewStyle().Bold(true).Foreground(muted)
	body := lipgloss.NewStyle().Foreground(text).Width(contentWidth)

	lines := []string{
		title + "  " + badge,
		renderStyledKV(label, body, "Scenario", scenario.Name, contentWidth),
	}
	if strings.TrimSpace(reason) != "" {
		lines = append(lines, renderStyledKV(label, body, "Reason", strings.TrimSpace(reason), contentWidth))
	}
	if len(queries) > 0 {
		lines = append(lines, renderStyledKV(label, body, "Queries", renderBulletList(cleanQueries(queries), contentWidth), contentWidth))
	}
	if excerpt := failureExcerpt(output, scenario.Matched, 2); excerpt != "" {
		lines = append(lines, renderStyledKV(label, body, "Error excerpt", excerpt, contentWidth))
	}
	lines = append(lines, renderStyledKV(label, body, "Next", "Searching Feishu Docs/Wiki and IM evidence...", contentWidth))
	return box.Render(strings.Join(lines, "\n\n")) + "\n"
}

func RenderStatusStyled(scenario detector.Scenario, output string, width int) string {
	return RenderPlannerStatusStyled(scenario, "", scenario.Matched, output, width)
}

func citationFrom(item evidence.ScoredSource) Citation {
	source := item.Source
	return Citation{
		Type:      source.Type,
		Title:     source.Title,
		URL:       source.URL,
		ID:        source.ID,
		ChatName:  source.ChatName,
		Sender:    source.Sender,
		Timestamp: source.Timestamp,
		Summary:   item.Snippet,
	}
}

func scenarioLabel(scenario detector.Scenario) string {
	if strings.TrimSpace(scenario.Name) != "" {
		return strings.TrimSpace(scenario.Name)
	}
	if strings.TrimSpace(scenario.ID) != "" {
		return strings.TrimSpace(scenario.ID)
	}
	return "Internal knowledge cue"
}

func cleanQueries(queries []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, query := range queries {
		query = strings.TrimSpace(query)
		if query == "" || seen[query] {
			continue
		}
		seen[query] = true
		out = append(out, query)
	}
	return out
}

func cleanActionPlan(plan []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, step := range plan {
		step = cleanActionStep(step)
		if step == "" {
			continue
		}
		key := strings.ToLower(step)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, step)
	}
	return out
}

var actionMarkerRE = regexp.MustCompile(`^\s*(?:[/\\]\s*)?(?:(?:[-*•]\s*)|(?:\d+[\.)、]\s*))+`)

func cleanActionStep(step string) string {
	step = strings.TrimSpace(step)
	for {
		cleaned := strings.TrimSpace(actionMarkerRE.ReplaceAllString(step, ""))
		if cleaned == step {
			break
		}
		step = cleaned
	}
	return strings.TrimSpace(step)
}

func renderPlan(k KnowledgeCard) string {
	var lines []string
	if strings.TrimSpace(k.PlannerReason) != "" {
		lines = append(lines, "Reason: "+strings.TrimSpace(k.PlannerReason))
	}
	if len(k.Queries) > 0 {
		lines = append(lines, "Queries:")
		for _, query := range k.Queries {
			query = strings.TrimSpace(query)
			if query != "" {
				lines = append(lines, "- "+query)
			}
		}
	}
	return strings.Join(lines, "\n")
}

func renderStyledKV(labelStyle, bodyStyle lipgloss.Style, key, value string, width int) string {
	return labelStyle.Render(key) + "\n" + bodyStyle.Render(wrapText(value, width))
}

func renderBulletList(values []string, width int) string {
	var lines []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		wrapped := wrapText(value, width-2)
		lines = append(lines, "• "+strings.ReplaceAll(wrapped, "\n", "\n  "))
	}
	return strings.Join(lines, "\n")
}

func renderActionPlan(plan []string) string {
	plan = cleanActionPlan(plan)
	if len(plan) == 0 {
		return "1. 打开内部来源核对后再执行修复动作。"
	}
	var lines []string
	for i, step := range plan {
		lines = append(lines, fmt.Sprintf("%d. %s", i+1, step))
	}
	return strings.Join(lines, "\n")
}

func renderStyledCitation(index int, c Citation, width int, accent, muted lipgloss.Color) string {
	number := lipgloss.NewStyle().Bold(true).Foreground(accent).Render(fmt.Sprintf("%d.", index))
	source := citationSourceLabel(c)
	sourceLine := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#DDE4FF")).
		Width(width - 4).
		Render(wrapText(source, width-4))
	summary := valueOr(c.Summary, "无摘要")
	detail := lipgloss.NewStyle().Foreground(muted).Width(width - 4).Render(wrapText(summary, width-4))
	return number + " " + strings.ReplaceAll(sourceLine, "\n", "\n   ") + "\n" + indent(detail, "   ")
}

func failureExcerpt(output string, matched []string, maxLines int) string {
	if maxLines <= 0 {
		return ""
	}
	var needles []string
	for _, match := range matched {
		match = strings.TrimSpace(strings.ToLower(match))
		if match != "" {
			needles = append(needles, match)
		}
	}
	if len(needles) == 0 {
		return ""
	}
	var lines []string
	seen := map[string]bool{}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		for _, needle := range needles {
			if strings.Contains(lower, needle) {
				if !seen[line] {
					lines = append(lines, line)
					seen[line] = true
				}
				break
			}
		}
		if len(lines) >= maxLines {
			break
		}
	}
	return strings.Join(lines, "\n")
}

func citationSourceLabel(c Citation) string {
	if c.Type == "im" {
		parts := []string{"群聊"}
		if c.ChatName != "" {
			parts = append(parts, c.ChatName)
		}
		if c.Sender != "" {
			parts = append(parts, c.Sender)
		}
		if c.Timestamp != "" {
			parts = append(parts, c.Timestamp)
		}
		return strings.Join(parts, " · ")
	}
	label := valueOr(c.Title, valueOr(c.URL, c.ID))
	if c.URL != "" {
		label += " · " + c.URL
	} else if c.ID != "" && c.ID != label {
		label += " · " + c.ID
	}
	return label
}

func renderCitation(c Citation) string {
	switch c.Type {
	case "im":
		parts := []string{"群聊"}
		if c.ChatName != "" {
			parts = append(parts, c.ChatName)
		}
		if c.Sender != "" {
			parts = append(parts, c.Sender)
		}
		if c.Timestamp != "" {
			parts = append(parts, c.Timestamp)
		}
		if c.Summary != "" {
			parts = append(parts, c.Summary)
		}
		return strings.Join(parts, " | ")
	default:
		label := valueOr(c.Title, valueOr(c.URL, c.ID))
		if c.URL != "" {
			label += " | " + c.URL
		} else if c.ID != "" && c.ID != label {
			label += " | " + c.ID
		}
		if c.Summary != "" {
			label += " | " + c.Summary
		}
		return label
	}
}

func enforceConfidence(card *KnowledgeCard, conf evidence.Confidence, status retrieval.Status, retrievalErr error, evidenceItems []evidence.ScoredSource) {
	if conf != evidence.ConfidenceHigh {
		card.LikelyCause = fallbackCause(conf, status, retrievalErr, evidenceItems)
	}
	if caveat := fallbackCaveat(conf, status, retrievalErr); caveat != "" {
		card.Caveat = caveat
	}
}

func draftTextGrounded(draft llm.CardDraft, evidenceItems []evidence.ScoredSource) bool {
	evidenceText := evidenceCorpus(evidenceItems)
	fields := []string{draft.LikelyCause, draft.Caveat}
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		if !fieldGrounded(field, evidenceText) {
			return false
		}
	}
	return true
}

func evidenceCorpus(items []evidence.ScoredSource) string {
	var parts []string
	for _, item := range items {
		parts = append(parts, item.Snippet)
	}
	return normalizeWidthLower(strings.Join(parts, "\n"))
}

var latinClaimTokenRE = regexp.MustCompile(`[a-z0-9_:/.-]+`)
var cjkClaimSegmentRE = regexp.MustCompile(`[\p{Han}]+`)

func fieldGrounded(field string, evidenceText string) bool {
	fieldLower := strings.ToLower(field)
	checkedCJK := false
	for _, segment := range cjkClaimSegmentRE.FindAllString(field, -1) {
		if cjkStopSegment(segment) {
			continue
		}
		checkedCJK = true
		if !containsPositiveSignalKeyword(evidenceText, strings.ToLower(segment)) {
			return false
		}
	}
	tokens := groundedClaimTokens(fieldLower)
	if len(tokens) == 0 {
		return checkedCJK
	}
	for _, token := range tokens {
		if !containsPositiveSignalKeyword(evidenceText, token) {
			return false
		}
	}
	return true
}

func groundedClaimTokens(fieldLower string) []string {
	stop := map[string]bool{
		"a": true, "an": true, "the": true, "and": true, "or": true, "for": true,
		"with": true, "this": true, "that": true, "to": true, "of": true, "in": true,
		"on": true, "is": true, "are": true, "be": true, "by": true, "if": true,
		"as": true, "at": true, "it": true, "from": true, "after": true, "before": true,
		"then": true, "you": true, "your": true, "please": true, "do": true,
		"because": true, "check": true, "confirm": true, "ensure": true, "verify": true,
		"run": true, "retry": true, "rerun": true, "likely": true, "cause": true,
		"action": true, "next": true, "high": true, "low": true,
	}
	seen := map[string]bool{}
	var tokens []string
	for _, token := range latinClaimTokenRE.FindAllString(fieldLower, -1) {
		token = strings.Trim(token, ".:,;()[]{}'\"`")
		if token == "" || stop[token] {
			continue
		}
		if !seen[token] {
			seen[token] = true
			tokens = append(tokens, token)
		}
	}
	phrases := []string{
		"docx:document:read",
		"missing required scope",
		"tenant_access_token invalid",
		"permission denied",
		"dag import error",
		"dagbag import error",
		"variable.get",
		"billing_region",
		"billing_daily",
		"list-import-errors",
		"parse time",
		"权限变更",
		"发布权限",
		"重新授权",
		"旧 token",
		"清理旧 token",
		"刷新 token",
		"文档读取",
		"飞书开放平台",
		"开放平台",
	}
	for _, phrase := range phrases {
		if strings.Contains(fieldLower, strings.ToLower(phrase)) && !seen[strings.ToLower(phrase)] {
			seen[strings.ToLower(phrase)] = true
			tokens = append(tokens, strings.ToLower(phrase))
		}
	}
	return tokens
}

func cjkStopSegment(segment string) bool {
	stop := map[string]bool{
		"的": true, "了": true, "是": true, "和": true, "或": true, "与": true,
		"并": true, "及": true, "在": true, "对": true, "从": true, "到": true,
		"后": true, "前": true, "先": true, "再": true, "请": true, "若": true,
		"如": true, "当": true, "将": true, "可": true, "能": true, "要": true,
		"需": true, "该": true, "用": true, "为": true, "而": true,
		"可能": true, "原因": true, "建议": true, "下一步": true, "当前": true,
	}
	return stop[segment]
}

func normalizeWidthLower(value string) string {
	var b strings.Builder
	for _, r := range value {
		if r == '\u3000' {
			r = ' '
		} else if r >= '\uFF01' && r <= '\uFF5E' {
			r -= '\uFEE0'
		}
		b.WriteRune(r)
	}
	return strings.ToLower(b.String())
}

type evidenceSignals struct {
	docxScope         bool
	missingScope      bool
	tokenInvalid      bool
	accessToken       bool
	publishPermission bool
	reauthorize       bool
	oldToken          bool
	cleanupOldToken   bool
	refreshToken      bool
}

func signalsFromEvidence(items []evidence.ScoredSource) evidenceSignals {
	text := evidenceSnippetCorpus(items)
	return evidenceSignals{
		docxScope:         containsPositiveSignal(text, []string{"docx:document:read"}),
		missingScope:      containsPositiveSignal(text, []string{"missing required scope"}),
		tokenInvalid:      containsPositiveSignal(text, []string{"tenant_access_token invalid"}),
		accessToken:       containsPositiveSignal(text, []string{"tenant_access_token", "app_access_token", "user_access_token"}),
		publishPermission: containsPositiveSignal(text, []string{"权限变更", "发布权限", "发布应用权限", "permission change", "publish permission"}),
		reauthorize:       containsPositiveSignal(text, []string{"重新授权", "re-authorize", "reauthorize"}),
		oldToken:          containsPositiveSignal(text, []string{"旧 token", "old token"}),
		cleanupOldToken:   containsPositiveSignal(text, []string{"清理旧 token", "clear old token", "remove old token"}),
		refreshToken:      containsPositiveSignal(text, []string{"刷新 token", "refresh token"}),
	}
}

func evidenceSnippetCorpus(items []evidence.ScoredSource) string {
	var parts []string
	for _, item := range items {
		parts = append(parts, item.Snippet)
	}
	return normalizeWidthLower(strings.Join(parts, "\n"))
}

func containsPositiveSignal(text string, keywords []string) bool {
	for _, keyword := range keywords {
		if containsPositiveSignalKeyword(text, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

func containsPositiveSignalKeyword(text string, keyword string) bool {
	for start := strings.Index(text, keyword); start >= 0; {
		end := start + len(keyword)
		if !signalNegatedAround(text, start, end) {
			return true
		}
		next := end
		if next >= len(text) {
			break
		}
		if idx := strings.Index(text[next:], keyword); idx >= 0 {
			start = next + idx
		} else {
			break
		}
	}
	return false
}

func signalNegatedAround(text string, start int, end int) bool {
	windowStart := start - 32
	if windowStart < 0 {
		windowStart = 0
	}
	prefix := text[windowStart:start]
	if sep := strings.LastIndexAny(prefix, "。；;，,\n.!?？"); sep >= 0 {
		prefix = prefix[sep+1:]
	}
	prefixNegators := []string{
		"不是",
		"并非",
		"无需",
		"不需要",
		"不要",
		"不用",
		"不必",
		"不能",
		"禁止",
		"别",
		"not ",
		"no ",
		"without ",
		"do not",
		"don't",
		"dont",
		"never",
		"not required",
		"no need",
	}
	for _, negator := range prefixNegators {
		if strings.Contains(prefix, negator) {
			return true
		}
	}

	windowEnd := end + 32
	if windowEnd > len(text) {
		windowEnd = len(text)
	}
	suffix := text[end:windowEnd]
	if sep := strings.IndexAny(suffix, "。；;，,\n.!?？"); sep >= 0 {
		suffix = suffix[:sep]
	}
	suffixNegators := []string{
		"不是",
		"并非",
		"无需",
		"不需要",
		"不要",
		"不用",
		"不必",
		"无关",
		"not required",
		"not needed",
		"is not",
		"are not",
		"was not",
		"were not",
		"no need",
	}
	for _, negator := range suffixNegators {
		if strings.Contains(suffix, negator) {
			return true
		}
	}
	return false
}

func fallbackCause(conf evidence.Confidence, status retrieval.Status, retrievalErr error, evidenceItems []evidence.ScoredSource) string {
	if retrievalErr != nil && conf == evidence.ConfidenceNone {
		return "真实 Feishu 检索不可用，暂不判断具体原因。"
	}
	if conf == evidence.ConfidenceHigh {
		if !looksLikeFeishuEvidence(evidenceItems) {
			return "内部证据支持该失败场景；关键线索：" + firstEvidenceSnippet(evidenceItems) + "。"
		}
		signals := signalsFromEvidence(evidenceItems)
		parts := []string{"证据显示当前错误与飞书 API 权限、scope 或 token 状态有关"}
		if signals.docxScope {
			parts[0] = "证据显示当前错误与飞书文档读取 scope 配置有关"
		} else if signals.missingScope {
			parts[0] = "证据显示当前错误与飞书 API 所需 scope 配置有关"
		} else if signals.tokenInvalid || signals.accessToken {
			parts[0] = "证据显示当前错误与飞书访问 token 状态有关"
		}
		if signals.publishPermission {
			parts = append(parts, "来源提到新增权限后需要发布权限变更")
		}
		if signals.reauthorize {
			parts = append(parts, "来源提到需要重新授权本地开发身份")
		}
		if signals.oldToken {
			parts = append(parts, "来源提到旧 token 可能不包含新增 scope")
		}
		return strings.Join(parts, "，") + "。"
	}
	return "检索到的内部证据不足以支撑确定结论。"
}

func fallbackActionPlan(conf evidence.Confidence, evidenceItems []evidence.ScoredSource) []string {
	if conf == evidence.ConfidenceHigh && len(evidenceItems) > 0 {
		return []string{
			"按引用来源给出的处理路径执行。",
			"修复后重新执行失败命令验证结果。",
		}
	}
	if len(evidenceItems) > 0 {
		return []string{"打开引用来源核对具体处理路径；证据不足时先不要执行高风险变更。"}
	}
	return []string{"未找到足够内部证据；补充知识库或扩大关键词后人工搜索。"}
}

func fallbackCaveat(conf evidence.Confidence, status retrieval.Status, retrievalErr error) string {
	if retrievalErr != nil {
		if status == retrieval.StatusPartial {
			return "部分 Feishu 检索失败，当前结论只基于已成功读取的来源。"
		}
		return "没有真实检索证据时，lark-cue 不会给出高置信度结论。"
	}
	if conf != evidence.ConfidenceHigh {
		if status == retrieval.StatusOK && conf == evidence.ConfidenceNone {
			return "已检索内部知识库，但没有找到足够强的证据支撑具体原因。"
		}
		return "证据较弱，建议打开来源核对后再执行修复动作。"
	}
	return ""
}

func looksLikeFeishuEvidence(evidenceItems []evidence.ScoredSource) bool {
	text := evidenceSnippetCorpus(evidenceItems)
	for _, keyword := range []string{
		"docx:document:read",
		"tenant_access_token",
		"app_access_token",
		"user_access_token",
		"missing required scope",
		"飞书",
		"开放平台",
	} {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func firstEvidenceSnippet(evidenceItems []evidence.ScoredSource) string {
	for _, item := range evidenceItems {
		if strings.TrimSpace(item.Snippet) != "" {
			return strings.TrimSpace(item.Snippet)
		}
	}
	return "请查看引用来源"
}

func newID() string {
	var bytes [4]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return fmt.Sprintf("cue_%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("cue_%d_%s", time.Now().Unix(), hex.EncodeToString(bytes[:]))
}

func valueOr(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func wrapText(value string, width int) string {
	if width <= 0 {
		return value
	}
	var lines []string
	for _, paragraph := range strings.Split(value, "\n") {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			lines = append(lines, "")
			continue
		}
		var line strings.Builder
		lineWidth := 0
		for _, token := range wrapTokens(paragraph) {
			tokenWidth := lipgloss.Width(token)
			if strings.TrimSpace(token) == "" && lineWidth == 0 {
				continue
			}
			if lineWidth > 0 && lineWidth+tokenWidth > width {
				lines = append(lines, strings.TrimSpace(line.String()))
				line.Reset()
				lineWidth = 0
			}
			if tokenWidth > width {
				for _, part := range splitByWidth(token, width) {
					partWidth := lipgloss.Width(part)
					if lineWidth > 0 && lineWidth+partWidth > width {
						lines = append(lines, strings.TrimSpace(line.String()))
						line.Reset()
						lineWidth = 0
					}
					line.WriteString(part)
					lineWidth += partWidth
				}
				continue
			}
			if strings.TrimSpace(token) == "" && lineWidth == 0 {
				continue
			}
			line.WriteString(token)
			lineWidth += tokenWidth
		}
		if line.Len() > 0 {
			lines = append(lines, strings.TrimSpace(line.String()))
		}
	}
	return strings.Join(lines, "\n")
}

func wrapTokens(value string) []string {
	var tokens []string
	var ascii strings.Builder
	flushASCII := func() {
		if ascii.Len() == 0 {
			return
		}
		tokens = append(tokens, ascii.String())
		ascii.Reset()
	}
	for _, r := range value {
		switch {
		case unicode.IsSpace(r):
			flushASCII()
			tokens = append(tokens, string(r))
		case r <= 127:
			ascii.WriteRune(r)
		default:
			flushASCII()
			tokens = append(tokens, string(r))
		}
	}
	flushASCII()
	return tokens
}

func splitByWidth(value string, width int) []string {
	var parts []string
	var part strings.Builder
	partWidth := 0
	for _, r := range value {
		cellWidth := lipgloss.Width(string(r))
		if partWidth > 0 && partWidth+cellWidth > width {
			parts = append(parts, part.String())
			part.Reset()
			partWidth = 0
		}
		part.WriteRune(r)
		partWidth += cellWidth
	}
	if part.Len() > 0 {
		parts = append(parts, part.String())
	}
	return parts
}

func indent(value, prefix string) string {
	lines := strings.Split(value, "\n")
	for i, line := range lines {
		if line == "" {
			continue
		}
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}

func clamp(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}
