package detector

import (
	"strings"
	"sync"
)

const FeishuAPIScopeError = "feishu_api_scope_error"

type Scenario struct {
	ID      string
	Name    string
	Matched []string
}

func Detect(output string) (Scenario, bool) {
	lower := strings.ToLower(output)
	var matched []string
	if containsScopeToken(lower, "docx:document:read") {
		matched = append(matched, "docx:document:read")
	}
	if strings.Contains(lower, "tenant_access_token invalid") {
		matched = append(matched, "tenant_access_token invalid")
	}
	if strings.Contains(lower, "missing required scope") && (hasFeishuContext(lower) || hasFeishuScopeToken(lower)) {
		matched = append(matched, "missing required scope")
	}
	if strings.Contains(lower, "scope not granted") && (hasFeishuContext(lower) || hasFeishuScopeToken(lower)) {
		matched = append(matched, "scope not granted")
	}
	if strings.Contains(lower, "permission denied") && hasFeishuContext(lower) {
		matched = append(matched, "permission denied")
	}
	if len(matched) == 0 {
		return Scenario{}, false
	}

	return Scenario{
		ID:      FeishuAPIScopeError,
		Name:    "Feishu API auth/scope/token error",
		Matched: matched,
	}, true
}

func hasFeishuContext(lower string) bool {
	if hasFeishuScopeToken(lower) {
		return true
	}
	contextPatterns := []string{
		"飞书",
		"开放平台",
		"open platform",
	}
	for _, pattern := range contextPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	wordPatterns := []string{
		"feishu",
		"larkapierror",
		"openapi",
		"tenant_access_token",
		"app_access_token",
		"user_access_token",
	}
	for _, pattern := range wordPatterns {
		if containsWord(lower, pattern) {
			return true
		}
	}
	return false
}

func hasFeishuScopeToken(lower string) bool {
	scopeTokens := []string{
		"docx:document:read",
	}
	for _, pattern := range scopeTokens {
		if containsScopeToken(lower, pattern) {
			return true
		}
	}
	return false
}

func containsScopeToken(text string, token string) bool {
	for start := strings.Index(text, token); start >= 0; {
		end := start + len(token)
		beforeOK := start == 0 || !isScopeTokenChar(text[start-1])
		afterOK := end == len(text) || !isScopeTokenChar(text[end])
		if beforeOK && afterOK {
			return true
		}
		next := end
		if next >= len(text) {
			break
		}
		if idx := strings.Index(text[next:], token); idx >= 0 {
			start = next + idx
		} else {
			break
		}
	}
	return false
}

func isScopeTokenChar(ch byte) bool {
	return isWordChar(ch) || ch == ':' || ch == '-'
}

func containsWord(text string, word string) bool {
	for start := strings.Index(text, word); start >= 0; {
		end := start + len(word)
		beforeOK := start == 0 || !isWordChar(text[start-1])
		afterOK := end == len(text) || !isWordChar(text[end])
		if beforeOK && afterOK {
			return true
		}
		next := end
		if next >= len(text) {
			break
		}
		if idx := strings.Index(text[next:], word); idx >= 0 {
			start = next + idx
		} else {
			break
		}
	}
	return false
}

func isWordChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '_'
}

type SignalBuffer struct {
	mu       sync.Mutex
	limit    int
	window   string
	hits     []string
	hitBytes int
}

func NewSignalBuffer(limit int) *SignalBuffer {
	if limit <= 0 {
		limit = 4096
	}
	return &SignalBuffer{limit: limit}
}

func (b *SignalBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	combined := b.window + string(p)
	if _, ok := Detect(combined); ok {
		excerpt := signalExcerpt(combined, b.limit)
		if excerpt != "" && !containsString(b.hits, excerpt) {
			b.appendHit(excerpt)
		}
	}
	if len(combined) > b.limit {
		b.window = combined[len(combined)-b.limit:]
	} else {
		b.window = combined
	}
	return len(p), nil
}

func (b *SignalBuffer) appendHit(excerpt string) {
	if len(excerpt) > b.limit {
		excerpt = excerpt[:b.limit]
	}
	added := len(excerpt)
	if len(b.hits) > 0 {
		added++
	}
	if b.hitBytes+added > b.limit {
		return
	}
	b.hits = append(b.hits, excerpt)
	b.hitBytes += added
}

func (b *SignalBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return strings.Join(b.hits, "\n")
}

func signalExcerpt(text string, limit int) string {
	if limit <= 0 || len(text) <= limit {
		return text
	}
	lower := strings.ToLower(text)
	index := firstSignalIndex(lower)
	if index < 0 {
		return text[len(text)-limit:]
	}
	start := index - limit/2
	if start < 0 {
		start = 0
	}
	end := start + limit
	if end > len(text) {
		end = len(text)
		start = end - limit
		if start < 0 {
			start = 0
		}
	}
	return text[start:end]
}

func firstSignalIndex(lower string) int {
	signals := []string{
		"missing required scope",
		"docx:document:read",
		"tenant_access_token invalid",
		"scope not granted",
		"permission denied",
	}
	first := -1
	for _, signal := range signals {
		if idx := strings.Index(lower, signal); idx >= 0 && (first < 0 || idx < first) {
			first = idx
		}
	}
	return first
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
