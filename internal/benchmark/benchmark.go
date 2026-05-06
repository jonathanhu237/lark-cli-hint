package benchmark

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"lark-cue/internal/eval"
	"lark-cue/internal/runner"
)

type File struct {
	Cases []Case `json:"cases"`
}

type Case struct {
	ID              string     `json:"id"`
	Command         []string   `json:"command"`
	Setup           [][]string `json:"setup,omitempty"`
	Teardown        [][]string `json:"teardown,omitempty"`
	ExpectFailure   bool       `json:"expect_failure"`
	ExpectedSources []string   `json:"expected_sources"`
	MinExpectedHits int        `json:"min_expected_hits"`
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
			title := strings.TrimSpace(source.Title)
			if title != "" {
				result.CitedTitles = append(result.CitedTitles, title)
			}
		}
	}
	if cue == nil {
		result.Failures = append(result.Failures, "no scored card was available")
	}

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
	result.Passed = len(result.Failures) == 0
	return result
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
	fmt.Fprintf(&b, "- cases: %d/%d passed\n", summary.PassedCases, summary.TotalCases)
	fmt.Fprintf(&b, "- expected-source hit rate: %d/%d\n", summary.ExpectedSourceHitCases, summary.TotalCases)
	fmt.Fprintf(&b, "- source coverage: %d/%d\n", summary.DistinctMatchedSources, summary.DistinctExpectedSources)
	fmt.Fprintf(&b, "- citation precision: %d/%d\n", summary.ExpectedCitationHits, summary.TotalCitations)
	fmt.Fprintf(&b, "- avg latency: %.0fms\n", summary.AverageLatencyMS)
	b.WriteString("\nCases\n")
	for _, result := range summary.Results {
		status := "FAIL"
		if result.Passed {
			status = "PASS"
		}
		fmt.Fprintf(&b, "%s %s\n", status, result.Case.ID)
		fmt.Fprintf(&b, "- command: %s\n", runner.CommandString(result.Case.Command))
		fmt.Fprintf(&b, "- expected hits: %d/%d (min %d)\n", result.ExpectedHitCount, len(uniqueStrings(result.Case.ExpectedSources)), result.Case.MinExpectedHits)
		fmt.Fprintf(&b, "- expected: %s\n", renderList(result.Case.ExpectedSources))
		fmt.Fprintf(&b, "- cited: %s\n", renderList(result.CitedTitles))
		fmt.Fprintf(&b, "- planner: %s\n", result.PlannerRetrieve)
		fmt.Fprintf(&b, "- queries: %d\n", result.QueryCount)
		if len(result.Failures) > 0 {
			fmt.Fprintf(&b, "- failures: %s\n", strings.Join(result.Failures, "; "))
		}
		if summary.Verbose {
			if output := excerpt(result.SetupOutput, 600); output != "" {
				fmt.Fprintf(&b, "- setup output: %s\n", output)
			}
			if output := excerpt(result.CommandOutput, 1200); output != "" {
				fmt.Fprintf(&b, "- command output: %s\n", output)
			}
			if output := excerpt(result.TeardownOutput, 600); output != "" {
				fmt.Fprintf(&b, "- teardown output: %s\n", output)
			}
		}
	}
	return b.String()
}

func RenderSummaryStyled(summary Summary, width int) string {
	if width <= 0 {
		width = 88
	}
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")).Render("lark-cue benchmark report")
	passStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	failStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("9"))
	muted := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	var b strings.Builder
	b.WriteString(title)
	b.WriteString("\n\n")
	fmt.Fprintf(&b, "%s %d/%d passed  %s %d/%d  %s %d/%d  %s %d/%d  %s %.0fms\n\n",
		muted.Render("cases"), summary.PassedCases, summary.TotalCases,
		muted.Render("hits"), summary.ExpectedSourceHitCases, summary.TotalCases,
		muted.Render("coverage"), summary.DistinctMatchedSources, summary.DistinctExpectedSources,
		muted.Render("precision"), summary.ExpectedCitationHits, summary.TotalCitations,
		muted.Render("avg latency"), summary.AverageLatencyMS)
	for _, result := range summary.Results {
		status := failStyle.Render("FAIL")
		if result.Passed {
			status = passStyle.Render("PASS")
		}
		fmt.Fprintf(&b, "%s %s\n", status, result.Case.ID)
		fmt.Fprintf(&b, "  command: %s\n", runner.CommandString(result.Case.Command))
		fmt.Fprintf(&b, "  expected hits: %d/%d (min %d)\n", result.ExpectedHitCount, len(uniqueStrings(result.Case.ExpectedSources)), result.Case.MinExpectedHits)
		fmt.Fprintf(&b, "  expected: %s\n", renderList(result.Case.ExpectedSources))
		fmt.Fprintf(&b, "  cited: %s\n", renderList(result.CitedTitles))
		fmt.Fprintf(&b, "  planner: %s, queries: %d\n", result.PlannerRetrieve, result.QueryCount)
		if len(result.Failures) > 0 {
			fmt.Fprintf(&b, "  failures: %s\n", strings.Join(result.Failures, "; "))
		}
		if summary.Verbose {
			if output := excerpt(result.CommandOutput, 1200); output != "" {
				fmt.Fprintf(&b, "  output: %s\n", output)
			}
		}
	}
	return lipgloss.NewStyle().Width(width).Render(b.String())
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
