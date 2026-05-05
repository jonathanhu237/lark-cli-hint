package query

import (
	"context"
	"regexp"
	"strings"

	"lark-cue/internal/detector"
)

const (
	maxQueries = 8
	maxLength  = 80
)

var scopeTokenRE = regexp.MustCompile(`\b[a-z][a-z0-9_]*:[a-zA-Z0-9_:-]+\b`)

type Expander interface {
	ExpandQueries(ctx context.Context, command []string, output string, scenario detector.Scenario, seeds []string) ([]string, error)
}

func Build(ctx context.Context, command []string, output string, scenario detector.Scenario, expander Expander) []string {
	seeds := ExtractSeeds(output)
	if expander == nil {
		return seeds
	}
	expanded, err := expander.ExpandQueries(ctx, command, output, scenario, seeds)
	if err != nil {
		return seeds
	}
	return normalize(append(seeds, expanded...), maxQueries, maxLength)
}

func ExtractSeeds(output string) []string {
	var candidates []string
	for _, match := range scopeTokenRE.FindAllString(output, -1) {
		candidates = append(candidates, match)
	}
	lower := strings.ToLower(output)
	phrases := []string{
		"missing required scope",
		"tenant_access_token invalid",
		"permission denied",
		"scope not granted",
		"unauthorized",
	}
	for _, phrase := range phrases {
		if strings.Contains(lower, phrase) {
			candidates = append(candidates, phrase)
		}
	}
	return normalize(candidates, maxQueries, maxLength)
}

func normalize(values []string, maxCount, maxLen int) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		value = strings.Trim(value, "`'\"")
		if value == "" {
			continue
		}
		if len([]rune(value)) > maxLen {
			value = string([]rune(value)[:maxLen])
		}
		key := strings.ToLower(value)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, value)
		if len(out) >= maxCount {
			break
		}
	}
	return out
}
