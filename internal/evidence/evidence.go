package evidence

import (
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"lark-cue/internal/retrieval"
)

type Confidence string

const (
	ConfidenceHigh Confidence = "high"
	ConfidenceLow  Confidence = "low"
	ConfidenceNone Confidence = "none"
)

type ScoredSource struct {
	Source         retrieval.Source
	Score          int
	StrongError    bool
	CauseAction    bool
	ScenarioSignal bool
	Snippet        string
}

type Context struct {
	Scenario string
	Queries  []string
	Output   string
}

var strongScopeKeywords = []string{"docx:document:read"}
var strongErrorKeywords = []string{"missing required scope", "tenant_access_token invalid", "permission denied"}
var concreteCauseKeywords = []string{
	"权限变更",
	"permission change",
	"发布权限",
	"发布应用权限",
	"publish permission",
	"重新授权",
	"re-authorize",
	"reauthorize",
	"旧 token",
	"old token",
	"清理旧 token",
	"刷新 token",
}

func Score(sources []retrieval.Source) []ScoredSource {
	return score(sources, Context{}, false)
}

func ScoreWithContext(sources []retrieval.Source, ctx Context) []ScoredSource {
	return score(sources, ctx, true)
}

func score(sources []retrieval.Source, ctx Context, useContext bool) []ScoredSource {
	var scored []ScoredSource
	contextKeywords := contextSignalKeywords(ctx)
	actionKeywords := append([]string{}, concreteCauseKeywords...)
	actionKeywords = append(actionKeywords, genericActionKeywords...)
	for _, source := range sources {
		if !source.Fetched || strings.TrimSpace(source.Content) == "" {
			continue
		}
		snippet := snippet(source.Content)
		if useContext {
			snippet = snippetForContext(source.Content, ctx, actionKeywords)
		}
		text := strings.ToLower(snippet)
		score := 0
		baseStrongScore := keywordScore(text, 4, strongScopeKeywords) +
			keywordScore(text, 3, strongErrorKeywords)
		strongScore := baseStrongScore
		contextScore := 0
		if useContext {
			contextScore = keywordScore(text, 2, contextKeywords)
			strongScore += contextScore
			if contextScore == 0 && baseStrongScore == 0 {
				continue
			}
		}
		concreteCauseScore := positiveKeywordScore(text, 2, actionKeywords)
		genericCauseScore := keywordScore(text, 1, []string{"scope", "权限", "授权"})
		scenarioScore := keywordScore(text, 1, []string{
			"docx:document:read",
			"tenant_access_token",
			"app_access_token",
			"user_access_token",
			"飞书",
			"飞书 api",
			"lark",
			"openapi",
			"open platform",
			"开放平台",
		})
		if useContext && contextScore > 0 {
			scenarioScore += contextScore
		}
		score += strongScore
		score += concreteCauseScore
		if !useContext {
			score += genericCauseScore
			score += scenarioScore
			score += keywordScore(text, 1, []string{"atlas", "文档摘要"})
		} else {
			score += scenarioScore
		}
		scored = append(scored, ScoredSource{
			Source:         source,
			Score:          score,
			StrongError:    strongScore > 0,
			CauseAction:    concreteCauseScore > 0,
			ScenarioSignal: scenarioScore > 0,
			Snippet:        snippet,
		})
	}
	sort.SliceStable(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})
	return scored
}

var genericActionKeywords = []string{
	"处理方式",
	"修复方式",
	"推荐处理",
	"建议",
	"下一步",
	"排查",
	"验证",
	"检查",
	"改用",
	"移到",
	"运行阶段",
	"执行阶段",
	"task 执行",
	"list-import-errors",
	"checklist",
	"retry",
	"rerun",
	"verify",
	"validate",
	"fix",
	"move",
	"use",
}

var contextTokenRE = regexp.MustCompile(`[\p{L}\p{N}_:./-]+`)

func contextSignalKeywords(ctx Context) []string {
	var candidates []string
	candidates = append(candidates, ctx.Scenario)
	candidates = append(candidates, ctx.Queries...)
	candidates = append(candidates, contextTokens(ctx.Scenario)...)
	for _, query := range ctx.Queries {
		candidates = append(candidates, contextTokens(query)...)
	}
	candidates = append(candidates, contextTokens(ctx.Output)...)
	return uniqueKeywords(candidates, 32)
}

func contextTokens(value string) []string {
	var tokens []string
	for _, token := range contextTokenRE.FindAllString(value, -1) {
		token = strings.Trim(token, "`'\".,;()[]{}")
		if weakContextToken(token) {
			continue
		}
		tokens = append(tokens, token)
	}
	return tokens
}

func weakContextToken(token string) bool {
	lower := strings.ToLower(strings.TrimSpace(token))
	if lower == "" {
		return true
	}
	stop := map[string]bool{
		"the": true, "and": true, "or": true, "for": true, "with": true, "from": true,
		"this": true, "that": true, "error": true, "failed": true, "failure": true,
		"issue": true, "problem": true, "help": true, "fix": true,
		"的": true, "了": true, "和": true, "或": true, "与": true,
	}
	if stop[lower] {
		return true
	}
	if utf8.RuneCountInString(lower) < 4 && !strings.ContainsAny(lower, ":_/-") {
		return true
	}
	return false
}

func uniqueKeywords(values []string, limit int) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(strings.ToLower(value))
		if value == "" || weakContextToken(value) {
			continue
		}
		if seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

func snippetForContext(content string, ctx Context, actionKeywords []string) string {
	keywords := contextSignalKeywords(ctx)
	lines := strings.Split(content, "\n")
	var contextLine string
	var actionLine string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		if contextLine == "" && containsAnyKeyword(lower, keywords) {
			contextLine = line
		}
		if actionLine == "" && containsAnyPositiveKeyword(lower, actionKeywords) {
			actionLine = line
		}
		if contextLine != "" && actionLine != "" {
			break
		}
	}
	switch {
	case contextLine != "" && actionLine != "" && contextLine == actionLine:
		return truncate(contextLine, 260)
	case contextLine != "" && actionLine != "":
		return truncate(contextLine+" / "+actionLine, 260)
	case contextLine != "":
		return truncate(contextLine, 220)
	case actionLine != "":
		return truncate(actionLine, 220)
	}
	return truncate(strings.TrimSpace(content), 220)
}

func Select(scored []ScoredSource) ([]ScoredSource, Confidence) {
	var selected []ScoredSource
	confidence := ConfidenceNone
	for _, item := range scored {
		if item.Score < 3 {
			continue
		}
		selected = append(selected, item)
		if item.Score >= 6 && item.StrongError && item.CauseAction && item.ScenarioSignal {
			confidence = ConfidenceHigh
		} else if confidence == ConfidenceNone {
			confidence = ConfidenceLow
		}
		if len(selected) >= 5 {
			break
		}
	}
	return selected, confidence
}

func keywordScore(text string, weight int, keywords []string) int {
	score := 0
	for _, keyword := range keywords {
		if strings.Contains(text, strings.ToLower(keyword)) {
			score += weight
		}
	}
	return score
}

func positiveKeywordScore(text string, weight int, keywords []string) int {
	score := 0
	for _, keyword := range keywords {
		if containsPositiveKeyword(text, keyword) {
			score += weight
		}
	}
	return score
}

func containsPositiveKeyword(text string, keyword string) bool {
	lowerText := strings.ToLower(text)
	lowerKeyword := strings.ToLower(keyword)
	for start := strings.Index(lowerText, lowerKeyword); start >= 0; {
		end := start + len(lowerKeyword)
		if !isNegatedAround(lowerText, start, end) {
			return true
		}
		next := end
		if next >= len(lowerText) {
			break
		}
		if idx := strings.Index(lowerText[next:], lowerKeyword); idx >= 0 {
			start = next + idx
		} else {
			break
		}
	}
	return false
}

func isNegatedAround(text string, start int, end int) bool {
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

func snippet(content string) string {
	lines := strings.Split(content, "\n")
	var strongLine string
	var causeLine string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		lower := strings.ToLower(line)
		if line == "" {
			continue
		}
		if strongLine == "" && (containsAnyKeyword(lower, strongScopeKeywords) || containsAnyKeyword(lower, strongErrorKeywords)) {
			strongLine = line
		}
		if causeLine == "" && containsAnyPositiveKeyword(lower, concreteCauseKeywords) {
			causeLine = line
		}
		if strongLine != "" && causeLine != "" {
			break
		}
	}
	switch {
	case strongLine != "" && causeLine != "" && strongLine == causeLine:
		return truncate(strongLine, 260)
	case strongLine != "" && causeLine != "":
		return truncate(strongLine+" / "+causeLine, 260)
	case strongLine != "":
		return truncate(strongLine, 220)
	case causeLine != "":
		return truncate(causeLine, 220)
	}
	return truncate(strings.TrimSpace(content), 220)
}

func containsAnyKeyword(text string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(text, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

func containsAnyPositiveKeyword(text string, keywords []string) bool {
	for _, keyword := range keywords {
		if containsPositiveKeyword(text, keyword) {
			return true
		}
	}
	return false
}

func truncate(value string, max int) string {
	if utf8.RuneCountInString(value) <= max {
		return value
	}
	runes := []rune(value)
	return string(runes[:max]) + "..."
}
