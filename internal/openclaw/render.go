package openclaw

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"lark-cue/internal/runner"
)

func RenderResult(result Result, wrappedExitCode int) string {
	var b strings.Builder
	b.WriteString("OpenClaw result\n")
	b.WriteString("Status\n")
	b.WriteString(resultStatus(result) + "\n\n")
	b.WriteString("Details\n")
	fmt.Fprintf(&b, "- agent: %s\n", DefaultAgent)
	fmt.Fprintf(&b, "- command: %s\n", runner.CommandString(result.Command))
	fmt.Fprintf(&b, "- OpenClaw exit code: %d\n", result.ExitCode)
	fmt.Fprintf(&b, "- duration: %s\n", formatMillis(result.LatencyMS))
	fmt.Fprintf(&b, "- wrapped command exit preserved: %d\n", wrappedExitCode)
	if strings.TrimSpace(result.Error) != "" {
		fmt.Fprintf(&b, "- error: %s\n", strings.TrimSpace(result.Error))
	}
	if excerpt := outputExcerpt(result.Output, 8); excerpt != "" {
		b.WriteString("\nOutput excerpt\n")
		b.WriteString(excerpt + "\n")
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
		"command: " + runner.CommandString(result.Command),
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
		renderKV(label, body, "Details", strings.Join(details, "\n")),
	}
	if excerpt := outputExcerpt(result.Output, 8); excerpt != "" {
		sections = append(sections, renderKV(label, body, "Output excerpt", excerpt))
	}
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
		return "Review the OpenClaw output above. If verification was not completed there, rerun the failed command or continue in the OpenClaw session."
	case result.TimedOut:
		return "OpenClaw did not finish within the timeout. Continue in the OpenClaw session or rerun with a narrower failure context."
	default:
		return "Review the OpenClaw error/output above. The original command exit code is preserved so the shell still reflects the failed workflow."
	}
}

func outputExcerpt(output string, maxLines int) string {
	output = strings.TrimSpace(output)
	if output == "" || maxLines <= 0 {
		return ""
	}
	lines := strings.Split(output, "\n")
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	return strings.Join(lines, "\n")
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
