package push

import (
	"context"
	"fmt"
	"strings"

	"lark-cue/internal/card"
)

func Prepare(k card.KnowledgeCard) string {
	var b strings.Builder
	fmt.Fprintf(&b, "**lark-cue 知识卡** `%s`\n\n", k.ID)
	fmt.Fprintf(&b, "**场景**\n%s\n\n", k.Scenario)
	fmt.Fprintf(&b, "**可能原因**\n%s\n\n", k.LikelyCause)
	fmt.Fprintf(&b, "**建议下一步**\n%s\n\n", k.NextAction)
	b.WriteString("**证据来源**\n")
	if len(k.Citations) == 0 {
		b.WriteString("- 未找到可支撑结论的内部来源。\n")
	} else {
		for _, citation := range k.Citations {
			b.WriteString("- " + renderCitation(citation) + "\n")
		}
	}
	fmt.Fprintf(&b, "\n**置信度**\n%s\n", k.Confidence)
	if k.Caveat != "" {
		fmt.Fprintf(&b, "\n**注意**\n%s\n", k.Caveat)
	}
	if k.RetrievalError != "" {
		fmt.Fprintf(&b, "\n**检索状态**\nReal Feishu retrieval failed: %s\n", k.RetrievalError)
	}
	return b.String()
}

func renderCitation(c card.Citation) string {
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
		label := firstNonEmpty(c.Title, c.URL, c.ID)
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

type Sender struct {
	client jsonRunner
}

type jsonRunner interface {
	RunJSON(ctx context.Context, args ...string) (map[string]any, error)
}

func NewSender(client jsonRunner) Sender {
	return Sender{client: client}
}

func (s Sender) Send(ctx context.Context, target string, markdown string) error {
	chatID := target
	if !strings.HasPrefix(target, "oc_") {
		resolved, err := s.resolveChat(ctx, target)
		if err != nil {
			return err
		}
		chatID = resolved
	}
	_, err := s.client.RunJSON(ctx, "im", "+messages-send", "--as", "user", "--chat-id", chatID, "--markdown", markdown, "--format", "json")
	return err
}

func (s Sender) resolveChat(ctx context.Context, name string) (string, error) {
	out, err := s.client.RunJSON(ctx, "im", "+chat-search", "--query", name, "--disable-search-by-user", "--page-size", "5", "--format", "json")
	if err != nil {
		return "", err
	}
	data, _ := out["data"].(map[string]any)
	candidates := firstArray(data, "items", "chats", "groups")
	var exactMatches []string
	for _, item := range candidates {
		obj, _ := item.(map[string]any)
		if stringAt(obj, "name") != name {
			continue
		}
		chatID := stringAt(obj, "chat_id")
		if chatID != "" {
			exactMatches = append(exactMatches, chatID)
		}
	}
	if len(exactMatches) == 1 {
		return exactMatches[0], nil
	}
	if len(exactMatches) > 1 {
		return "", fmt.Errorf("multiple Feishu chats exactly matched %q; use chat id instead", name)
	}
	return "", fmt.Errorf("no exact Feishu chat found for %q; use chat id or exact chat name", name)
}

func firstArray(obj map[string]any, keys ...string) []any {
	for _, key := range keys {
		if values, ok := obj[key].([]any); ok {
			return values
		}
	}
	return nil
}

func stringAt(obj map[string]any, key string) string {
	if obj == nil || obj[key] == nil {
		return ""
	}
	if value, ok := obj[key].(string); ok {
		return value
	}
	return fmt.Sprint(obj[key])
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
