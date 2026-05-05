package evidence

import (
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
	var scored []ScoredSource
	for _, source := range sources {
		if !source.Fetched || strings.TrimSpace(source.Content) == "" {
			continue
		}
		snippet := snippet(source.Content)
		text := strings.ToLower(snippet)
		score := 0
		strongScore := keywordScore(text, 4, strongScopeKeywords) +
			keywordScore(text, 3, strongErrorKeywords)
		concreteCauseScore := positiveKeywordScore(text, 2, concreteCauseKeywords)
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
		score += strongScore
		score += concreteCauseScore
		score += genericCauseScore
		score += scenarioScore
		score += keywordScore(text, 1, []string{"atlas", "文档摘要"})
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
