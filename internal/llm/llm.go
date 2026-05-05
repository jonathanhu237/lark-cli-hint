package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"lark-cue/internal/config"
	"lark-cue/internal/detector"
	"lark-cue/internal/evidence"
	"lark-cue/internal/runner"
)

type Provider interface {
	ExpandQueries(ctx context.Context, command []string, output string, scenario detector.Scenario, seeds []string) ([]string, error)
	GenerateCard(ctx context.Context, input CardInput) (CardDraft, error)
}

type CardInput struct {
	Command    []string
	Output     string
	Scenario   detector.Scenario
	Evidence   []evidence.ScoredSource
	Confidence evidence.Confidence
}

type CardDraft struct {
	LikelyCause string `json:"likely_cause"`
	NextAction  string `json:"next_action"`
	Caveat      string `json:"caveat"`
}

type OpenAICompatible struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

func NewOpenAICompatible(cfg config.LLMConfig) *OpenAICompatible {
	return &OpenAICompatible{
		baseURL: strings.TrimRight(firstNonEmpty(cfg.BaseURL, "https://api.openai.com/v1"), "/"),
		apiKey:  cfg.APIKey,
		model:   cfg.Model,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (p *OpenAICompatible) available() bool {
	return p != nil && p.apiKey != "" && p.model != ""
}

func (p *OpenAICompatible) ExpandQueries(ctx context.Context, command []string, output string, scenario detector.Scenario, seeds []string) ([]string, error) {
	if !p.available() {
		return nil, errors.New("llm provider is not configured")
	}
	prompt := fmt.Sprintf(`You generate enterprise Feishu knowledge search queries.
Return only a JSON array of 3-5 short strings.
Use the command and terminal output only.
Do not guess exact document titles, URLs, chat IDs, or final answers.
Keep original English technical terms when useful and add concise Chinese equivalents when useful.

Command: %s
Scenario: %s
Seed queries: %s
Terminal output:
%s`, runner.CommandString(command), scenario.ID, strings.Join(seeds, ", "), truncate(output, 4000))

	text, err := p.chat(ctx, "You are a precise search-query generator.", prompt)
	if err != nil {
		return nil, err
	}
	var queries []string
	if err := json.Unmarshal([]byte(extractJSONArray(text)), &queries); err != nil {
		return parseLines(text), nil
	}
	return queries, nil
}

func (p *OpenAICompatible) GenerateCard(ctx context.Context, input CardInput) (CardDraft, error) {
	if !p.available() {
		return CardDraft{}, errors.New("llm provider is not configured")
	}
	var evidenceLines []string
	for _, item := range input.Evidence {
		evidenceLines = append(evidenceLines, fmt.Sprintf("- %s: %s", item.Source.CitationLabel(), item.Snippet))
	}
	prompt := fmt.Sprintf(`Generate a compact Chinese terminal knowledge card draft.
Return only JSON with keys likely_cause, next_action, caveat.
Use only the evidence below. Do not cite or invent sources. If evidence is weak, say so.

Command: %s
Scenario: %s
Confidence gate: %s
Terminal output:
%s

Evidence:
%s`, runner.CommandString(input.Command), input.Scenario.ID, input.Confidence, truncate(input.Output, 3000), strings.Join(evidenceLines, "\n"))

	text, err := p.chat(ctx, "You compress fetched enterprise evidence into safe next steps.", prompt)
	if err != nil {
		return CardDraft{}, err
	}
	var draft CardDraft
	if err := json.Unmarshal([]byte(extractJSONObject(text)), &draft); err != nil {
		return CardDraft{}, err
	}
	if strings.TrimSpace(draft.LikelyCause) == "" && strings.TrimSpace(draft.Caveat) == "" {
		return CardDraft{}, errors.New("empty card draft")
	}
	return draft, nil
}

func (p *OpenAICompatible) chat(ctx context.Context, system, user string) (string, error) {
	body := chatRequest{
		Model: p.model,
		Messages: []chatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("llm request failed: %s", resp.Status)
	}
	var out chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", errors.New("llm returned no choices")
	}
	return out.Choices[0].Message.Content, nil
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

func extractJSONArray(text string) string {
	start := strings.IndexByte(text, '[')
	end := strings.LastIndexByte(text, ']')
	if start >= 0 && end >= start {
		return text[start : end+1]
	}
	return text
}

func extractJSONObject(text string) string {
	start := strings.IndexByte(text, '{')
	end := strings.LastIndexByte(text, '}')
	if start >= 0 && end >= start {
		return text[start : end+1]
	}
	return text
}

func parseLines(text string) []string {
	var out []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(strings.Trim(line, "-*0123456789. \t\"'`"))
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func truncate(value string, max int) string {
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max]) + "\n...[truncated]"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
