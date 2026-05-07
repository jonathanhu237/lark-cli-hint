package benchmark

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"lark-cue/internal/card"
	"lark-cue/internal/eval"
	"lark-cue/internal/runner"
)

type File struct {
	Cases []Case `json:"cases"`
}

type Case struct {
	ID                    string     `json:"id"`
	Command               []string   `json:"command"`
	Setup                 [][]string `json:"setup,omitempty"`
	Teardown              [][]string `json:"teardown,omitempty"`
	ExpectFailure         bool       `json:"expect_failure"`
	ExpectedSources       []string   `json:"expected_sources"`
	ExpectedEvidenceTerms []string   `json:"expected_evidence_terms,omitempty"`
	MinExpectedHits       int        `json:"min_expected_hits"`
}

type Observation struct {
	CommandExitCode int
	SetupError      string
	TeardownErrors  []string
	PlannerRecords  []eval.Record
	CueRecords      []eval.Record
	CommandOutput   string
	SetupOutput     string
	TeardownOutput  string
}

type CaseResult struct {
	Case                 Case
	Passed               bool
	Failures             []string
	CommandExitCode      int
	PlannerRetrieve      string
	QueryCount           int
	LatencyMS            int64
	CitedTitles          []string
	ExpectedHitCount     int
	ExpectedCitationHits int
	TotalCitations       int
	MatchedSources       []string
	MatchedEvidenceTerms []string
	CommandOutput        string
	SetupOutput          string
	TeardownOutput       string
}

type Summary struct {
	TotalCases              int
	PassedCases             int
	ExpectedSourceHitCases  int
	DistinctExpectedSources int
	DistinctMatchedSources  int
	ExpectedCitationHits    int
	TotalCitations          int
	AverageLatencyMS        float64
	Jobs                    int
	Results                 []CaseResult
	Verbose                 bool
}

func LoadCases(path string) ([]Case, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("--cases is required")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read cases file: %w", err)
	}
	var file File
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse cases JSON: %w", err)
	}
	cases, err := normalizeAndValidate(file.Cases)
	if err != nil {
		return nil, err
	}
	return cases, nil
}

func normalizeAndValidate(cases []Case) ([]Case, error) {
	if len(cases) == 0 {
		return nil, errors.New("cases file must contain at least one case")
	}
	seen := map[string]bool{}
	out := make([]Case, 0, len(cases))
	for i, c := range cases {
		c.ID = strings.TrimSpace(c.ID)
		if c.ID == "" {
			return nil, fmt.Errorf("case %d: id is required", i+1)
		}
		if seen[c.ID] {
			return nil, fmt.Errorf("case %q: duplicate id", c.ID)
		}
		seen[c.ID] = true
		if err := validateCommand(c.Command, fmt.Sprintf("case %q command", c.ID)); err != nil {
			return nil, err
		}
		for j, command := range c.Setup {
			if err := validateCommand(command, fmt.Sprintf("case %q setup[%d]", c.ID, j)); err != nil {
				return nil, err
			}
		}
		for j, command := range c.Teardown {
			if err := validateCommand(command, fmt.Sprintf("case %q teardown[%d]", c.ID, j)); err != nil {
				return nil, err
			}
		}
		if len(c.ExpectedSources) == 0 {
			return nil, fmt.Errorf("case %q: expected_sources must not be empty", c.ID)
		}
		for j, source := range c.ExpectedSources {
			c.ExpectedSources[j] = strings.TrimSpace(source)
			if c.ExpectedSources[j] == "" {
				return nil, fmt.Errorf("case %q: expected_sources[%d] is empty", c.ID, j)
			}
		}
		for j, term := range c.ExpectedEvidenceTerms {
			c.ExpectedEvidenceTerms[j] = strings.TrimSpace(term)
			if c.ExpectedEvidenceTerms[j] == "" {
				return nil, fmt.Errorf("case %q: expected_evidence_terms[%d] is empty", c.ID, j)
			}
		}
		if c.MinExpectedHits == 0 {
			c.MinExpectedHits = 1
		}
		if c.MinExpectedHits < 1 {
			return nil, fmt.Errorf("case %q: min_expected_hits must be at least 1", c.ID)
		}
		if c.MinExpectedHits > len(uniqueStrings(c.ExpectedSources)) {
			return nil, fmt.Errorf("case %q: min_expected_hits cannot exceed distinct expected_sources", c.ID)
		}
		out = append(out, c)
	}
	return out, nil
}

func validateCommand(command []string, label string) error {
	if len(command) == 0 {
		return fmt.Errorf("%s must be a non-empty command array", label)
	}
	for i, part := range command {
		if strings.TrimSpace(part) == "" {
			return fmt.Errorf("%s[%d] must not be empty", label, i)
		}
	}
	return nil
}

func ScoreCase(c Case, observation Observation) CaseResult {
	result := CaseResult{
		Case:            c,
		CommandExitCode: observation.CommandExitCode,
		PlannerRetrieve: "unknown",
		CommandOutput:   observation.CommandOutput,
		SetupOutput:     observation.SetupOutput,
		TeardownOutput:  observation.TeardownOutput,
	}
	if observation.SetupError != "" {
		result.Failures = append(result.Failures, "setup failed: "+observation.SetupError)
	}
	for _, err := range observation.TeardownErrors {
		if strings.TrimSpace(err) != "" {
			result.Failures = append(result.Failures, "teardown failed: "+err)
		}
	}
	if c.ExpectFailure {
		if observation.CommandExitCode == 0 {
			result.Failures = append(result.Failures, "expected command failure but command exited 0")
		}
	} else if observation.CommandExitCode != 0 {
		result.Failures = append(result.Failures, fmt.Sprintf("expected command success but command exited %d", observation.CommandExitCode))
	}

	var planner *eval.Record
	if len(observation.PlannerRecords) > 0 {
		planner = &observation.PlannerRecords[len(observation.PlannerRecords)-1]
		result.QueryCount = planner.QueryCount
		switch {
		case planner.ShouldRetrieve == nil:
			result.PlannerRetrieve = "unknown"
		case *planner.ShouldRetrieve:
			result.PlannerRetrieve = "retrieve"
		default:
			result.PlannerRetrieve = "skip"
		}
	}

	var cue *eval.Record
	if len(observation.CueRecords) > 0 {
		cue = &observation.CueRecords[len(observation.CueRecords)-1]
		result.QueryCount = cue.QueryCount
		result.LatencyMS = cue.LatencyMS
		for _, source := range cue.Sources {
			label := benchmarkSourceLabel(source)
			if label != "" {
				result.CitedTitles = append(result.CitedTitles, label)
			}
		}
	}
	if cue == nil {
		result.Failures = append(result.Failures, "no scored card was available")
	}
	sourceCorpus := evidenceCorpus(result.CitedTitles, cue)

	expectedSet := makeSet(c.ExpectedSources)
	matchedSet := map[string]bool{}
	for _, title := range result.CitedTitles {
		if expectedSet[title] {
			result.ExpectedCitationHits++
			matchedSet[title] = true
		}
	}
	result.TotalCitations = len(result.CitedTitles)
	for title := range matchedSet {
		result.MatchedSources = append(result.MatchedSources, title)
	}
	slices.Sort(result.MatchedSources)
	result.ExpectedHitCount = len(result.MatchedSources)
	if result.ExpectedHitCount < c.MinExpectedHits {
		result.Failures = append(result.Failures, fmt.Sprintf("expected source hits %d below minimum %d", result.ExpectedHitCount, c.MinExpectedHits))
	}
	for _, term := range c.ExpectedEvidenceTerms {
		if strings.Contains(sourceCorpus, strings.ToLower(term)) {
			result.MatchedEvidenceTerms = append(result.MatchedEvidenceTerms, term)
			continue
		}
		result.Failures = append(result.Failures, fmt.Sprintf("expected evidence term %q was not present in cited sources", term))
	}
	result.Passed = len(result.Failures) == 0
	return result
}

func evidenceCorpus(labels []string, cue *eval.Record) string {
	var parts []string
	parts = append(parts, labels...)
	if cue != nil {
		for _, source := range cue.Sources {
			parts = append(parts,
				source.Title,
				source.ID,
				source.ChatName,
				source.Sender,
				source.Timestamp,
				source.Summary,
			)
		}
	}
	return strings.ToLower(strings.Join(parts, "\n"))
}

func benchmarkSourceLabel(source card.Citation) string {
	title := strings.TrimSpace(source.Title)
	if title != "" {
		return title
	}
	if source.Type == "im" {
		chatName := strings.TrimSpace(source.ChatName)
		if chatName != "" {
			return chatName
		}
	}
	id := strings.TrimSpace(source.ID)
	if id != "" {
		return id
	}
	return strings.TrimSpace(source.URL)
}

func Summarize(results []CaseResult, verbose bool) Summary {
	summary := Summary{
		TotalCases: len(results),
		Results:    append([]CaseResult(nil), results...),
		Verbose:    verbose,
	}
	expectedSet := map[string]bool{}
	matchedSet := map[string]bool{}
	var latencyTotal int64
	var latencyCount int
	for _, result := range results {
		if result.Passed {
			summary.PassedCases++
		}
		if result.ExpectedHitCount >= result.Case.MinExpectedHits {
			summary.ExpectedSourceHitCases++
		}
		for _, source := range result.Case.ExpectedSources {
			expectedSet[source] = true
		}
		for _, source := range result.MatchedSources {
			matchedSet[source] = true
		}
		summary.ExpectedCitationHits += result.ExpectedCitationHits
		summary.TotalCitations += result.TotalCitations
		if result.LatencyMS > 0 {
			latencyTotal += result.LatencyMS
			latencyCount++
		}
	}
	summary.DistinctExpectedSources = len(expectedSet)
	summary.DistinctMatchedSources = len(matchedSet)
	if latencyCount > 0 {
		summary.AverageLatencyMS = float64(latencyTotal) / float64(latencyCount)
	}
	return summary
}

func (s Summary) AllPassed() bool {
	return s.TotalCases > 0 && s.PassedCases == s.TotalCases
}

func RenderSummary(summary Summary) string {
	var b strings.Builder
	b.WriteString("lark-cue benchmark report\n\n")
	b.WriteString("Summary\n")
	for _, line := range summaryMetricLines(summary) {
		fmt.Fprintf(&b, "- %s\n", line)
	}
	b.WriteString("\nCases\n")
	for _, result := range summary.Results {
		b.WriteString(renderPlainCase(result, summary.Verbose))
	}
	return b.String()
}

func RenderSummaryStyled(summary Summary, width int) string {
	if width <= 0 {
		width = 88
	}
	cardWidth := clamp(width-4, 72, 118)
	contentWidth := cardWidth - 6
	accent := lipgloss.Color("10")
	status := "PASS"
	if !summary.AllPassed() {
		accent = lipgloss.Color("9")
		status = "CHECK"
	}
	box := lipgloss.NewStyle().
		Width(cardWidth).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accent).
		Padding(1, 2)
	title := lipgloss.NewStyle().Bold(true).Foreground(accent)
	badge := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("0")).Background(accent).Padding(0, 1)
	label := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("8"))
	passStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	failStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("9"))
	muted := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	var caseLines []string
	for _, result := range summary.Results {
		caseStatus := failStyle.Render("FAIL")
		if result.Passed {
			caseStatus = passStyle.Render("PASS")
		}
		primary := fmt.Sprintf("%s %-45s %s %d/%d  %s %d  %s %s",
			caseStatus,
			clipRunes(result.Case.ID, 45),
			muted.Render("hits"), result.ExpectedHitCount, len(uniqueStrings(result.Case.ExpectedSources)),
			muted.Render("q"), result.QueryCount,
			muted.Render("lat"), formatMillis(result.LatencyMS),
		)
		caseLines = append(caseLines, primary)
		if !result.Passed {
			caseLines = append(caseLines, "     "+failStyle.Render(clipRunes(strings.Join(result.Failures, "; "), contentWidth-5)))
		} else if summary.Verbose {
			caseLines = append(caseLines, "     "+muted.Render(clipRunes("cited: "+compactList(result.CitedTitles, 3), contentWidth-5)))
		}
		if summary.Verbose && !result.Passed {
			if output := firstNonEmpty(excerpt(result.CommandOutput, 220), excerpt(result.SetupOutput, 220), excerpt(result.TeardownOutput, 220)); output != "" {
				caseLines = append(caseLines, "     "+muted.Render(clipRunes("output: "+output, contentWidth-5)))
			}
		}
	}

	overview := strings.Join(summaryMetricLines(summary), "\n")
	header := title.Render("lark-cue benchmark") + "  " + badge.Render(status)
	sections := []string{
		header,
		renderKV(label, overview, "Summary"),
		renderKV(label, strings.Join(caseLines, "\n"), "Cases"),
	}
	return box.Render(strings.Join(sections, "\n\n")) + "\n"
}

func makeSet(values []string) map[string]bool {
	set := map[string]bool{}
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			set[strings.TrimSpace(value)] = true
		}
	}
	return set
}

func uniqueStrings(values []string) []string {
	set := makeSet(values)
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	slices.Sort(out)
	return out
}

func renderList(values []string) string {
	values = uniqueStrings(values)
	if len(values) == 0 {
		return "(none)"
	}
	return strings.Join(values, ", ")
}

func renderPlainCase(result CaseResult, verbose bool) string {
	status := "FAIL"
	if result.Passed {
		status = "PASS"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s  hits %d/%d  citations %d  queries %d  latency %s\n",
		status,
		result.Case.ID,
		result.ExpectedHitCount,
		len(uniqueStrings(result.Case.ExpectedSources)),
		result.TotalCitations,
		result.QueryCount,
		formatMillis(result.LatencyMS),
	)
	fmt.Fprintf(&b, "  command: %s\n", runner.CommandString(result.Case.Command))
	fmt.Fprintf(&b, "  planner: %s, queries: %d\n", result.PlannerRetrieve, result.QueryCount)
	if result.Passed {
		fmt.Fprintf(&b, "  cited: %s\n", compactList(result.CitedTitles, 5))
		return b.String()
	}
	fmt.Fprintf(&b, "  expected: %s\n", compactList(result.Case.ExpectedSources, 5))
	fmt.Fprintf(&b, "  cited: %s\n", compactList(result.CitedTitles, 5))
	if len(result.Failures) > 0 {
		fmt.Fprintf(&b, "  failures: %s\n", strings.Join(result.Failures, "; "))
	}
	if verbose {
		if output := firstNonEmpty(excerpt(result.CommandOutput, 500), excerpt(result.SetupOutput, 300), excerpt(result.TeardownOutput, 300)); output != "" {
			fmt.Fprintf(&b, "  output: %s\n", output)
		}
	}
	return b.String()
}

func summaryMetricLines(summary Summary) []string {
	lines := []string{
		fmt.Sprintf("cases: %d/%d passed", summary.PassedCases, summary.TotalCases),
		fmt.Sprintf("expected-source hit rate: %d/%d", summary.ExpectedSourceHitCases, summary.TotalCases),
		fmt.Sprintf("source coverage: %d/%d", summary.DistinctMatchedSources, summary.DistinctExpectedSources),
		fmt.Sprintf("citation precision: %d/%d", summary.ExpectedCitationHits, summary.TotalCitations),
		fmt.Sprintf("avg latency: %s", formatMillis(int64(summary.AverageLatencyMS))),
	}
	if summary.Jobs > 0 {
		lines = append(lines, fmt.Sprintf("parallel jobs: %d", summary.Jobs))
	}
	return lines
}

func compactList(values []string, maxItems int) string {
	values = uniqueStrings(values)
	if len(values) == 0 {
		return "(none)"
	}
	if maxItems < 1 || len(values) <= maxItems {
		return strings.Join(values, ", ")
	}
	return strings.Join(values[:maxItems], ", ") + fmt.Sprintf(", +%d more", len(values)-maxItems)
}

func formatMillis(value int64) string {
	if value <= 0 {
		return "0ms"
	}
	if value >= 1000 {
		return fmt.Sprintf("%.1fs", float64(value)/1000)
	}
	return fmt.Sprintf("%dms", value)
}

func renderKV(label lipgloss.Style, value, key string) string {
	return label.Render(key) + "\n" + value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func clipRunes(value string, limit int) string {
	if limit <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	if limit <= 3 {
		return string(runes[:limit])
	}
	return string(runes[:limit-3]) + "..."
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

func excerpt(value string, limit int) string {
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if value == "" {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	if limit <= 1 {
		return string(runes[:limit])
	}
	return string(runes[:limit-1]) + "..."
}
