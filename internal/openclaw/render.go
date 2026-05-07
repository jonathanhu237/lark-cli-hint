package openclaw

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func RenderResult(result Result, wrappedExitCode int) string {
	var b strings.Builder
	b.WriteString("OpenClaw result\n")
	b.WriteString("Status\n")
	b.WriteString(resultStatus(result) + "\n\n")

	for _, section := range outputSections(result) {
		b.WriteString(section.title + "\n")
		b.WriteString(section.body + "\n\n")
	}

	b.WriteString("Details\n")
	fmt.Fprintf(&b, "- agent: %s\n", DefaultAgent)
	fmt.Fprintf(&b, "- OpenClaw exit code: %d\n", result.ExitCode)
	fmt.Fprintf(&b, "- duration: %s\n", formatMillis(result.LatencyMS))
	fmt.Fprintf(&b, "- wrapped command exit preserved: %d\n", wrappedExitCode)
	if strings.TrimSpace(result.Error) != "" {
		fmt.Fprintf(&b, "- error: %s\n", strings.TrimSpace(result.Error))
	}

	b.WriteString("\nNext\n")
	b.WriteString(nextGuidance(result))
	b.WriteByte('\n')
	return b.String()
}

func RenderResultStyled(result Result, wrappedExitCode int, width int) string {
	if width < 60 {
		return RenderResult(result, wrappedExitCode)
	}
	cardWidth := clamp(width-4, 72, 104)
	contentWidth := cardWidth - 6

	accent := statusColor(result)
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

	details := []string{
		"agent: " + DefaultAgent,
		fmt.Sprintf("OpenClaw exit code: %d", result.ExitCode),
		"duration: " + formatMillis(result.LatencyMS),
		fmt.Sprintf("wrapped command exit preserved: %d", wrappedExitCode),
	}
	if strings.TrimSpace(result.Error) != "" {
		details = append(details, "error: "+strings.TrimSpace(result.Error))
	}

	sections := []string{
		title.Render("OpenClaw result"),
		renderKV(label, body, "Status", resultStatus(result)),
	}
	for _, section := range outputSections(result) {
		sections = append(sections, renderKV(label, body, section.title, section.body))
	}
	sections = append(sections, renderKV(label, body, "Details", strings.Join(details, "\n")))
	sections = append(sections, renderKV(label, body, "Next", nextGuidance(result)))
	return box.Render(strings.Join(sections, "\n\n")) + "\n"
}

func resultStatus(result Result) string {
	switch {
	case result.Succeeded:
		return "Succeeded"
	case result.TimedOut:
		return "Timed out"
	default:
		return "Failed"
	}
}

func nextGuidance(result Result) string {
	switch {
	case result.Succeeded:
		return "Review the OpenClaw summary above. If verification was not completed, rerun the failed command or continue in the OpenClaw session."
	case result.TimedOut:
		return "OpenClaw did not finish within the timeout. Continue in the OpenClaw session or rerun with a narrower failure context."
	default:
		return "Review the OpenClaw error and short output excerpt above. The original command exit code is preserved so the shell still reflects the failed workflow."
	}
}

type resultSection struct {
	title string
	body  string
}

func outputSections(result Result) []resultSection {
	named := extractNamedOutputSections(result.Output)
	if len(named) > 0 {
		return named
	}
	excerptTitle := "Summary"
	if !result.Succeeded {
		excerptTitle = "Output excerpt"
	}
	if excerpt := outputExcerpt(result.Output, 6); excerpt != "" {
		return []resultSection{{title: excerptTitle, body: excerpt}}
	}
	return nil
}

func extractNamedOutputSections(output string) []resultSection {
	lines := cleanOutputLines(output)
	if len(lines) == 0 {
		return nil
	}
	labels := map[string]string{
		"what i found":   "Findings",
		"what i changed": "Changes",
		"verification":   "Verification",
		"summary":        "Summary",
		"result":         "Result",
	}
	order := []string{"Findings", "Changes", "Verification", "Summary", "Result"}
	values := make(map[string][]string)
	var current string
	for _, line := range lines {
		key := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(line), ":"))
		if label, ok := labels[key]; ok {
			current = label
			continue
		}
		if current == "" {
			continue
		}
		values[current] = appendLimited(values[current], line, 5)
	}

	var sections []resultSection
	for _, label := range order {
		lines := values[label]
		if len(lines) == 0 {
			continue
		}
		sections = append(sections, resultSection{
			title: label,
			body:  strings.Join(lines, "\n"),
		})
	}
	return sections
}

func outputExcerpt(output string, maxLines int) string {
	lines := cleanOutputLines(output)
	if len(lines) == 0 || maxLines <= 0 {
		return ""
	}
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	return strings.Join(lines, "\n")
}

func cleanOutputLines(output string) []string {
	output = strings.TrimSpace(output)
	if output == "" {
		return nil
	}
	rawLines := strings.Split(output, "\n")
	lines := make([]string, 0, len(rawLines))
	for _, line := range rawLines {
		line = strings.TrimRight(line, " \t")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "[plugins]") {
			continue
		}
		lines = append(lines, clipLine(line, 140))
	}
	return lines
}

func appendLimited(lines []string, line string, max int) []string {
	if len(lines) >= max {
		return lines
	}
	return append(lines, line)
}

func clipLine(line string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(line)
	if len(runes) <= max {
		return line
	}
	if max <= 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
}

func statusColor(result Result) lipgloss.Color {
	switch {
	case result.Succeeded:
		return lipgloss.Color("#2EAD6B")
	case result.TimedOut:
		return lipgloss.Color("#E0A11B")
	default:
		return lipgloss.Color("#D95C5C")
	}
}

func renderKV(labelStyle, bodyStyle lipgloss.Style, key, value string) string {
	return labelStyle.Render(key) + "\n" + bodyStyle.Render(value)
}

func formatMillis(value int64) string {
	if value >= 1000 {
		return fmt.Sprintf("%.1fs", float64(value)/1000)
	}
	return fmt.Sprintf("%dms", value)
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
