package openclaw

import (
	"fmt"
	"strings"

	"lark-cue/internal/card"
	"lark-cue/internal/evidence"
	"lark-cue/internal/runner"
)

type TaskInput struct {
	WorkingDir      string
	Command         []string
	ExitCode        int
	Output          string
	PlannerScenario string
	PlannerReason   string
	Queries         []string
	Card            card.KnowledgeCard
	Evidence        []evidence.ScoredSource
}

func BuildTask(input TaskInput) string {
	var b strings.Builder
	b.WriteString("You are OpenClaw acting inside the developer's local workspace.\n")
	b.WriteString("Use the Feishu-backed knowledge below as context, inspect local state first, then repair the failure if it is safe.\n\n")

	b.WriteString("Goal\n")
	b.WriteString("- Resolve the failed command or produce a clear explanation if local state proves the cited guidance does not apply.\n")
	b.WriteString("- Rerun the failed command or an equivalent verification before finishing.\n\n")

	b.WriteString("Execution Context\n")
	fmt.Fprintf(&b, "- Working directory: %s\n", valueOr(input.WorkingDir, "unknown"))
	fmt.Fprintf(&b, "- Failed command: %s\n", runner.CommandString(input.Command))
	fmt.Fprintf(&b, "- Exit code: %d\n", input.ExitCode)
	b.WriteString("- Captured output excerpt:\n")
	b.WriteString(indentBlock(truncateMiddle(strings.TrimSpace(input.Output), 5000)))
	b.WriteString("\n\n")

	b.WriteString("LLM Planner\n")
	fmt.Fprintf(&b, "- Scenario: %s\n", valueOr(input.PlannerScenario, input.Card.Scenario))
	if strings.TrimSpace(input.PlannerReason) != "" {
		fmt.Fprintf(&b, "- Reason: %s\n", strings.TrimSpace(input.PlannerReason))
	}
	if len(input.Queries) > 0 {
		b.WriteString("- Feishu search queries:\n")
		for _, query := range input.Queries {
			query = strings.TrimSpace(query)
			if query != "" {
				fmt.Fprintf(&b, "  - %s\n", query)
			}
		}
	}
	b.WriteString("\n")

	b.WriteString("Knowledge Card\n")
	fmt.Fprintf(&b, "- Confidence: %s\n", input.Card.Confidence)
	if strings.TrimSpace(input.Card.LikelyCause) != "" {
		fmt.Fprintf(&b, "- Likely cause: %s\n", strings.TrimSpace(input.Card.LikelyCause))
	}
	if len(input.Card.ActionPlan) > 0 {
		b.WriteString("- Action plan:\n")
		for i, step := range input.Card.ActionPlan {
			step = strings.TrimSpace(step)
			if step != "" {
				fmt.Fprintf(&b, "  %d. %s\n", i+1, step)
			}
		}
	}
	if strings.TrimSpace(input.Card.Caveat) != "" {
		fmt.Fprintf(&b, "- Caveat: %s\n", strings.TrimSpace(input.Card.Caveat))
	}
	b.WriteString("\n")

	b.WriteString("Feishu Evidence\n")
	if len(input.Evidence) == 0 {
		b.WriteString("- No strong cited Feishu evidence was available; treat the card as low confidence and inspect locally before changing anything.\n")
	} else {
		for i, item := range input.Evidence {
			fmt.Fprintf(&b, "%d. %s\n", i+1, item.Source.CitationLabel())
			if item.Source.URL != "" {
				fmt.Fprintf(&b, "   URL: %s\n", item.Source.URL)
			} else if item.Source.ID != "" {
				fmt.Fprintf(&b, "   ID: %s\n", item.Source.ID)
			}
			if snippet := strings.TrimSpace(item.Snippet); snippet != "" {
				fmt.Fprintf(&b, "   Snippet: %s\n", truncateMiddle(snippet, 900))
			}
		}
	}
	b.WriteString("\n")

	b.WriteString("Safety Constraints\n")
	b.WriteString("- Inspect files, configs, and command outputs before editing.\n")
	b.WriteString("- Adapt the action plan to actual local findings; do not blindly apply a step that does not match the workspace.\n")
	b.WriteString("- Ask the user before deleting data, changing production configuration, rotating secrets, sending messages, committing code, pushing code, or making risky external side effects.\n")
	b.WriteString("- Keep fixes minimal and explain what changed and how verification was run.\n")
	return b.String()
}

func indentBlock(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "  <empty>"
	}
	lines := strings.Split(value, "\n")
	for i, line := range lines {
		lines[i] = "  " + line
	}
	return strings.Join(lines, "\n")
}

func truncateMiddle(value string, limit int) string {
	if limit <= 0 || len(value) <= limit {
		return value
	}
	marker := "\n...[truncated]...\n"
	remaining := limit - len(marker)
	if remaining <= 0 {
		return value[:limit]
	}
	head := remaining / 2
	tail := remaining - head
	return value[:head] + marker + value[len(value)-tail:]
}

func valueOr(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}
