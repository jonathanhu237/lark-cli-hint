package eval

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	DefaultReportLimit = 20
	maxReportLineBytes = 1024 * 1024
)

type ReadResult struct {
	Records        []Record
	MalformedLines int
}

type Summary struct {
	TotalRuns         int
	StatusCounts      map[string]int
	FixtureRuns       int
	RealRuns          int
	RunsWithSources   int
	TotalSources      int
	AverageSources    float64
	AverageQueryCount float64
	AverageLatencyMS  float64
	FeedbackCounts    map[string]int
	MalformedLines    int
	Limit             int
}

func ReadCueRecords(path string) (ReadResult, error) {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return ReadResult{Records: []Record{}}, nil
	}
	if err != nil {
		return ReadResult{}, err
	}
	defer file.Close()

	var result ReadResult
	indexByCardID := map[string]int{}
	processLine := func(raw []byte) {
		line := strings.TrimSpace(string(raw))
		if line == "" {
			return
		}
		var record Record
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			result.MalformedLines++
			return
		}
		switch record.Type {
		case "cue":
			result.Records = append(result.Records, record)
			if record.CardID != "" {
				indexByCardID[record.CardID] = len(result.Records) - 1
			}
		case "feedback_update":
			if record.CardID == "" || record.Feedback == "" {
				return
			}
			if index, ok := indexByCardID[record.CardID]; ok {
				result.Records[index].Feedback = record.Feedback
			}
		}
	}

	reader := bufio.NewReaderSize(file, 64*1024)
	var line []byte
	lineOverLimit := false
	for {
		fragment, err := reader.ReadSlice('\n')
		if len(fragment) > 0 {
			if lineOverLimit || len(line)+len(fragment) > maxReportLineBytes {
				lineOverLimit = true
				line = line[:0]
			} else {
				line = append(line, fragment...)
			}
		}

		switch {
		case errors.Is(err, bufio.ErrBufferFull):
			continue
		case errors.Is(err, io.EOF):
			if lineOverLimit {
				result.MalformedLines++
			} else if len(line) > 0 {
				processLine(line)
			}
			return result, nil
		case err != nil:
			return result, err
		}

		if lineOverLimit {
			result.MalformedLines++
		} else {
			processLine(line)
		}
		line = line[:0]
		lineOverLimit = false
	}
}

func LimitCueRecords(records []Record, limit int) []Record {
	if limit <= 0 || len(records) <= limit {
		return records
	}
	return records[len(records)-limit:]
}

func Summarize(records []Record, malformedLines int, limit int) Summary {
	summary := Summary{
		StatusCounts:   map[string]int{},
		FeedbackCounts: map[string]int{},
		MalformedLines: malformedLines,
		Limit:          limit,
	}
	for _, record := range records {
		summary.TotalRuns++
		status := strings.TrimSpace(record.RetrievalStatus)
		if status == "" {
			status = "unknown"
		}
		summary.StatusCounts[status]++
		if status == "fixture" {
			summary.FixtureRuns++
		} else {
			summary.RealRuns++
		}

		sourceCount := len(record.Sources)
		summary.TotalSources += sourceCount
		if sourceCount > 0 {
			summary.RunsWithSources++
		}
		summary.AverageQueryCount += float64(record.QueryCount)
		summary.AverageLatencyMS += float64(record.LatencyMS)

		feedback := strings.TrimSpace(record.Feedback)
		if feedback == "" {
			feedback = "unknown"
		}
		summary.FeedbackCounts[feedback]++
	}
	if summary.TotalRuns > 0 {
		count := float64(summary.TotalRuns)
		summary.AverageSources = float64(summary.TotalSources) / count
		summary.AverageQueryCount /= count
		summary.AverageLatencyMS /= count
	}
	return summary
}

func RenderSummary(summary Summary) string {
	var b strings.Builder
	b.WriteString("lark-cue validation report\n\n")
	if summary.TotalRuns == 0 {
		b.WriteString("No cue records found.\n")
		if summary.MalformedLines > 0 {
			fmt.Fprintf(&b, "Warnings: skipped %d malformed log line(s).\n", summary.MalformedLines)
		}
		return b.String()
	}

	fmt.Fprintf(&b, "Runs\n")
	fmt.Fprintf(&b, "- cue runs: %d", summary.TotalRuns)
	if summary.Limit > 0 {
		fmt.Fprintf(&b, " (latest %d max)", summary.Limit)
	}
	b.WriteString("\n")
	fmt.Fprintf(&b, "- retrieval status: %s\n", renderStatusCounts(summary.StatusCounts))
	fmt.Fprintf(&b, "- fixture runs: %d\n\n", summary.FixtureRuns)

	fmt.Fprintf(&b, "Evidence\n")
	fmt.Fprintf(&b, "- citation coverage: %d/%d (%s)\n", summary.RunsWithSources, summary.TotalRuns, percent(summary.RunsWithSources, summary.TotalRuns))
	fmt.Fprintf(&b, "- total sources: %d\n", summary.TotalSources)
	fmt.Fprintf(&b, "- avg sources/run: %.1f\n\n", summary.AverageSources)

	fmt.Fprintf(&b, "Runtime\n")
	fmt.Fprintf(&b, "- avg latency: %s\n", formatMillis(summary.AverageLatencyMS))
	fmt.Fprintf(&b, "- avg queries/run: %.1f\n\n", summary.AverageQueryCount)

	fmt.Fprintf(&b, "Feedback\n")
	fmt.Fprintf(&b, "- useful: %d\n", summary.FeedbackCounts["useful"])
	fmt.Fprintf(&b, "- not-useful: %d\n", summary.FeedbackCounts["not-useful"])
	fmt.Fprintf(&b, "- skipped: %d\n", summary.FeedbackCounts["skipped"])
	if unknown := summary.FeedbackCounts["unknown"]; unknown > 0 {
		fmt.Fprintf(&b, "- unknown: %d\n", unknown)
	}
	if summary.MalformedLines > 0 {
		fmt.Fprintf(&b, "\nWarnings\n- skipped malformed log lines: %d\n", summary.MalformedLines)
	}
	return b.String()
}

func RenderSummaryStyled(summary Summary, width int) string {
	if width < 60 {
		return RenderSummary(summary)
	}
	cardWidth := clamp(width-4, 72, 104)
	contentWidth := cardWidth - 6

	accent := lipgloss.Color("#5A7CFF")
	muted := lipgloss.Color("#7C8798")
	text := lipgloss.Color("#E7EAF0")
	warn := lipgloss.Color("#E0A11B")

	box := lipgloss.NewStyle().
		Width(cardWidth).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accent).
		Padding(1, 2)
	title := lipgloss.NewStyle().Bold(true).Foreground(accent)
	label := lipgloss.NewStyle().Bold(true).Foreground(muted)
	body := lipgloss.NewStyle().Foreground(text).Width(contentWidth)
	warnStyle := lipgloss.NewStyle().Foreground(warn).Width(contentWidth)

	sections := []string{
		title.Render("lark-cue validation"),
	}
	if summary.TotalRuns == 0 {
		sections = append(sections, body.Render("No cue records found. Run a cue demo first, then rerun this report."))
		if summary.MalformedLines > 0 {
			sections = append(sections, warnStyle.Render(fmt.Sprintf("Warnings: skipped %d malformed log line(s).", summary.MalformedLines)))
		}
		return box.Render(strings.Join(sections, "\n\n")) + "\n"
	}

	sections = append(sections,
		renderKV(label, body, "Runs", fmt.Sprintf("%d cue runs%s", summary.TotalRuns, limitSuffix(summary.Limit))),
		renderKV(label, body, "Retrieval", fmt.Sprintf("%s\nFixture: %d", renderStatusCounts(summary.StatusCounts), summary.FixtureRuns)),
		renderKV(label, body, "Evidence", fmt.Sprintf("Citation coverage: %d/%d (%s)\nSources: %d total, %.1f per run", summary.RunsWithSources, summary.TotalRuns, percent(summary.RunsWithSources, summary.TotalRuns), summary.TotalSources, summary.AverageSources)),
		renderKV(label, body, "Runtime", fmt.Sprintf("Average latency: %s\nAverage queries/run: %.1f", formatMillis(summary.AverageLatencyMS), summary.AverageQueryCount)),
		renderKV(label, body, "Feedback", renderFeedback(summary.FeedbackCounts)),
	)
	if summary.MalformedLines > 0 {
		sections = append(sections, warnStyle.Render(fmt.Sprintf("Warnings: skipped %d malformed log line(s).", summary.MalformedLines)))
	}
	return box.Render(strings.Join(sections, "\n\n")) + "\n"
}

func renderKV(labelStyle, bodyStyle lipgloss.Style, key, value string) string {
	return labelStyle.Render(key) + "\n" + bodyStyle.Render(value)
}

func renderStatusCounts(counts map[string]int) string {
	ordered := []string{"ok", "partial", "failed", "fixture", "unknown"}
	seen := map[string]bool{}
	parts := make([]string, 0, len(ordered))
	for _, key := range ordered {
		seen[key] = true
		parts = append(parts, fmt.Sprintf("%s %d", key, counts[key]))
	}
	var extras []string
	for key := range counts {
		if !seen[key] {
			extras = append(extras, key)
		}
	}
	sort.Strings(extras)
	for _, key := range extras {
		parts = append(parts, fmt.Sprintf("%s %d", key, counts[key]))
	}
	return strings.Join(parts, " / ")
}

func renderFeedback(counts map[string]int) string {
	parts := []string{
		fmt.Sprintf("useful %d", counts["useful"]),
		fmt.Sprintf("not-useful %d", counts["not-useful"]),
		fmt.Sprintf("skipped %d", counts["skipped"]),
	}
	if unknown := counts["unknown"]; unknown > 0 {
		parts = append(parts, fmt.Sprintf("unknown %d", unknown))
	}
	var extras []string
	for key := range counts {
		if key != "useful" && key != "not-useful" && key != "skipped" && key != "unknown" {
			extras = append(extras, key)
		}
	}
	sort.Strings(extras)
	for _, key := range extras {
		parts = append(parts, fmt.Sprintf("%s %d", key, counts[key]))
	}
	return strings.Join(parts, " / ")
}

func formatMillis(value float64) string {
	if value >= 1000 {
		return fmt.Sprintf("%.1fs", value/1000)
	}
	return fmt.Sprintf("%.0fms", value)
}

func percent(value, total int) string {
	if total == 0 {
		return "0%"
	}
	return fmt.Sprintf("%.0f%%", float64(value)*100/float64(total))
}

func limitSuffix(limit int) string {
	if limit <= 0 {
		return ""
	}
	return fmt.Sprintf(" (latest %d max)", limit)
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
