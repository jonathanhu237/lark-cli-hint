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
	PlannerRecords []Record
	Events         []Record
	MalformedLines int
}

type Summary struct {
	TotalRuns         int
	StatusCounts      map[string]int
	RealRuns          int
	RunsWithSources   int
	TotalSources      int
	AverageSources    float64
	AverageQueryCount float64
	AverageLatencyMS  float64
	FeedbackCounts    map[string]int
	PlannerRuns       int
	PlannerRetrieve   int
	PlannerSkip       int
	PlannerUnknown    int
	AveragePlannerMS  float64
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
	type cueIndex struct {
		record int
		event  int
	}
	indexByCardID := map[string]cueIndex{}
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
			result.Events = append(result.Events, record)
			if record.CardID != "" {
				indexByCardID[record.CardID] = cueIndex{record: len(result.Records) - 1, event: len(result.Events) - 1}
			}
		case "planner":
			result.PlannerRecords = append(result.PlannerRecords, record)
			result.Events = append(result.Events, record)
		case "feedback_update":
			if record.CardID == "" || record.Feedback == "" {
				return
			}
			if index, ok := indexByCardID[record.CardID]; ok {
				result.Records[index.record].Feedback = record.Feedback
				result.Events[index.event].Feedback = record.Feedback
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

func LimitReadResult(result ReadResult, limit int) ReadResult {
	if limit <= 0 || len(result.Events) <= limit {
		return result
	}
	limited := ReadResult{MalformedLines: result.MalformedLines}
	limited.Events = append(limited.Events, result.Events[len(result.Events)-limit:]...)
	for _, event := range limited.Events {
		switch event.Type {
		case "cue":
			limited.Records = append(limited.Records, event)
		case "planner":
			limited.PlannerRecords = append(limited.PlannerRecords, event)
		}
	}
	return limited
}

func Summarize(records []Record, malformedLines int, limit int) Summary {
	return SummarizeResult(ReadResult{Records: records, Events: records, MalformedLines: malformedLines}, limit)
}

func SummarizeResult(result ReadResult, limit int) Summary {
	summary := Summary{
		StatusCounts:   map[string]int{},
		FeedbackCounts: map[string]int{},
		MalformedLines: result.MalformedLines,
		Limit:          limit,
	}
	for _, record := range result.Records {
		summary.TotalRuns++
		status := strings.TrimSpace(record.RetrievalStatus)
		if status == "" {
			status = "unknown"
		}
		summary.StatusCounts[status]++
		summary.RealRuns++

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
	for _, record := range result.PlannerRecords {
		summary.PlannerRuns++
		if record.ShouldRetrieve == nil {
			summary.PlannerUnknown++
		} else if *record.ShouldRetrieve {
			summary.PlannerRetrieve++
		} else {
			summary.PlannerSkip++
		}
		summary.AveragePlannerMS += float64(record.LatencyMS)
	}
	if summary.PlannerRuns > 0 {
		summary.AveragePlannerMS /= float64(summary.PlannerRuns)
	}
	return summary
}

func RenderSummary(summary Summary) string {
	var b strings.Builder
	b.WriteString("lark-cue validation report\n\n")
	if summary.TotalRuns == 0 && summary.PlannerRuns == 0 {
		b.WriteString("No cue or planner records found.\n")
		if summary.MalformedLines > 0 {
			fmt.Fprintf(&b, "Warnings: skipped %d malformed log line(s).\n", summary.MalformedLines)
		}
		return b.String()
	}

	if summary.PlannerRuns > 0 {
		fmt.Fprintf(&b, "Planner\n")
		fmt.Fprintf(&b, "- decisions: %d", summary.PlannerRuns)
		if summary.Limit > 0 {
			fmt.Fprintf(&b, " (latest %d max)", summary.Limit)
		}
		b.WriteString("\n")
		fmt.Fprintf(&b, "- retrieve: %d\n", summary.PlannerRetrieve)
		fmt.Fprintf(&b, "- skip: %d\n", summary.PlannerSkip)
		if summary.PlannerUnknown > 0 {
			fmt.Fprintf(&b, "- unknown: %d\n", summary.PlannerUnknown)
		}
		fmt.Fprintf(&b, "- avg planner latency: %s\n\n", formatMillis(summary.AveragePlannerMS))
	}

	if summary.TotalRuns == 0 {
		if summary.MalformedLines > 0 {
			fmt.Fprintf(&b, "Warnings\n- skipped malformed log lines: %d\n", summary.MalformedLines)
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
	b.WriteString("\n")

	fmt.Fprintf(&b, "Evidence\n")
	fmt.Fprintf(&b, "- citation coverage: %d/%d (%s)\n", summary.RunsWithSources, summary.TotalRuns, percent(summary.RunsWithSources, summary.TotalRuns))
	fmt.Fprintf(&b, "- total sources: %d\n", summary.TotalSources)
	fmt.Fprintf(&b, "- avg sources/run: %.1f\n\n", summary.AverageSources)

	fmt.Fprintf(&b, "Runtime\n")
	fmt.Fprintf(&b, "- avg latency: %s\n", formatMillis(summary.AverageLatencyMS))
	fmt.Fprintf(&b, "- avg queries/run: %.1f\n", summary.AverageQueryCount)
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
	if summary.TotalRuns == 0 && summary.PlannerRuns == 0 {
		sections = append(sections, body.Render("No cue or planner records found. Run a cue demo first, then rerun this report."))
		if summary.MalformedLines > 0 {
			sections = append(sections, warnStyle.Render(fmt.Sprintf("Warnings: skipped %d malformed log line(s).", summary.MalformedLines)))
		}
		return box.Render(strings.Join(sections, "\n\n")) + "\n"
	}

	if summary.PlannerRuns > 0 {
		sections = append(sections,
			renderKV(label, body, "Planner", fmt.Sprintf("%d decisions%s\nRetrieve %d / skip %d / unknown %d\nAverage planner latency: %s", summary.PlannerRuns, limitSuffix(summary.Limit), summary.PlannerRetrieve, summary.PlannerSkip, summary.PlannerUnknown, formatMillis(summary.AveragePlannerMS))),
		)
	}

	if summary.TotalRuns == 0 {
		if summary.MalformedLines > 0 {
			sections = append(sections, warnStyle.Render(fmt.Sprintf("Warnings: skipped %d malformed log line(s).", summary.MalformedLines)))
		}
		return box.Render(strings.Join(sections, "\n\n")) + "\n"
	}

	sections = append(sections,
		renderKV(label, body, "Runs", fmt.Sprintf("%d cue runs%s", summary.TotalRuns, limitSuffix(summary.Limit))),
		renderKV(label, body, "Retrieval", renderStatusCounts(summary.StatusCounts)),
		renderKV(label, body, "Evidence", fmt.Sprintf("Citation coverage: %d/%d (%s)\nSources: %d total, %.1f per run", summary.RunsWithSources, summary.TotalRuns, percent(summary.RunsWithSources, summary.TotalRuns), summary.TotalSources, summary.AverageSources)),
		renderKV(label, body, "Runtime", fmt.Sprintf("Average latency: %s\nAverage queries/run: %.1f", formatMillis(summary.AverageLatencyMS), summary.AverageQueryCount)),
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
	ordered := []string{"ok", "partial", "failed", "unknown"}
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
