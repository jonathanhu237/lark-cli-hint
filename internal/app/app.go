package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"

	"lark-cue/internal/benchmark"
	"lark-cue/internal/card"
	"lark-cue/internal/config"
	"lark-cue/internal/detector"
	"lark-cue/internal/eval"
	"lark-cue/internal/evidence"
	"lark-cue/internal/larkcli"
	"lark-cue/internal/llm"
	"lark-cue/internal/openclaw"
	"lark-cue/internal/push"
	"lark-cue/internal/retrieval"
	"lark-cue/internal/runner"
)

const version = "0.1.0"

type runOptions struct {
	command     []string
	preparePush bool
	sendPush    bool
	pushChat    string
	verbose     bool
	noOpenClaw  bool
}

type evalReportOptions struct {
	limit int
}

type benchmarkRunOptions struct {
	casesPath  string
	verbose    bool
	noOpenClaw bool
	jobs       int
}

type cueProvider interface {
	llm.Planner
	llm.CardProvider
}

var newPlannerProvider = func(cfg config.LLMConfig) (cueProvider, error) {
	provider := llm.NewOpenAICompatible(cfg)
	if !provider.Available() {
		return nil, errors.New("LLM configuration is required; set LARK_CUE_LLM_API_KEY and LARK_CUE_LLM_MODEL")
	}
	return provider, nil
}

var newRetriever = func(cfg config.FeishuConfig) retrieval.Retriever {
	return retrieval.NewLarkRetriever(larkcli.NewWithProfile("lark-cli", cfg.Profile))
}

type openClawClient interface {
	Preflight(context.Context) error
	Invoke(context.Context, string, io.Writer) openclaw.Result
}

var newOpenClawClient = func(cfg config.OpenClawConfig) openClawClient {
	return openclaw.New(cfg)
}

var readCueRecords = eval.ReadCueRecords

func Main(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printHelp(stdout)
		return 0
	}

	cfg, cfgErr := config.Load()
	if cfgErr != nil {
		fmt.Fprintf(stderr, "lark-cue: config warning: %v\n", cfgErr)
	}

	switch args[0] {
	case "-h", "--help", "help":
		printHelp(stdout)
		return 0
	case "-v", "--version", "version":
		fmt.Fprintf(stdout, "lark-cue %s\n", version)
		return 0
	case "run":
		opts, err := parseRunArgs(args[1:])
		if err == errHelp {
			printRunHelp(stdout)
			return 0
		}
		if err != nil {
			fmt.Fprintf(stderr, "lark-cue run: %v\n\n", err)
			printRunHelp(stderr)
			return 2
		}
		return runCommand(context.Background(), cfg, opts, stdin, stdout, stderr)
	case "benchmark":
		return runBenchmarkCommand(context.Background(), cfg, args[1:], stdout, stderr)
	case "eval":
		return runEval(cfg, args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "lark-cue: unknown command %q\n\n", args[0])
		printHelp(stderr)
		return 2
	}
}

func parseRunArgs(args []string) (runOptions, error) {
	opts := runOptions{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--":
			opts.command = append([]string{}, args[i+1:]...)
			if len(opts.command) == 0 {
				return opts, errors.New("missing command after --")
			}
			return opts, nil
		case "--prepare-push":
			opts.preparePush = true
		case "--send-push":
			opts.preparePush = true
			opts.sendPush = true
		case "--verbose":
			opts.verbose = true
		case "--no-openclaw":
			opts.noOpenClaw = true
		case "--push-chat":
			if i+1 >= len(args) {
				return opts, errors.New("--push-chat requires a chat id or chat name")
			}
			i++
			opts.pushChat = args[i]
		case "-h", "--help":
			return opts, errHelp
		default:
			if strings.HasPrefix(arg, "-") {
				return opts, fmt.Errorf("unknown flag %s", arg)
			}
			return opts, errors.New("missing -- before wrapped command")
		}
	}
	return opts, errors.New("missing wrapped command; use lark-cue run -- <command>")
}

var errHelp = errors.New("help requested")

func parseEvalReportArgs(args []string) (evalReportOptions, error) {
	opts := evalReportOptions{limit: eval.DefaultReportLimit}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--limit":
			if i+1 >= len(args) {
				return opts, errors.New("--limit requires a positive integer")
			}
			i++
			limit, err := strconv.Atoi(args[i])
			if err != nil || limit <= 0 {
				return opts, errors.New("--limit requires a positive integer")
			}
			opts.limit = limit
		case "-h", "--help":
			return opts, errHelp
		default:
			return opts, fmt.Errorf("unknown flag %s", arg)
		}
	}
	return opts, nil
}

func parseBenchmarkRunArgs(args []string) (benchmarkRunOptions, error) {
	opts := benchmarkRunOptions{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--cases":
			if i+1 >= len(args) {
				return opts, errors.New("--cases requires a path")
			}
			i++
			opts.casesPath = args[i]
		case "--verbose":
			opts.verbose = true
		case "--no-openclaw":
			opts.noOpenClaw = true
		case "--jobs":
			if i+1 >= len(args) {
				return opts, errors.New("--jobs requires a positive integer")
			}
			i++
			jobs, err := strconv.Atoi(args[i])
			if err != nil || jobs < 1 {
				return opts, errors.New("--jobs requires a positive integer")
			}
			opts.jobs = jobs
		case "-h", "--help":
			return opts, errHelp
		default:
			return opts, fmt.Errorf("unknown flag %s", arg)
		}
	}
	if strings.TrimSpace(opts.casesPath) == "" {
		return opts, errors.New("--cases is required")
	}
	return opts, nil
}

func runCommand(ctx context.Context, cfg config.Config, opts runOptions, stdin io.Reader, stdout, stderr io.Writer) int {
	provider, err := newPlannerProvider(cfg.LLM)
	if err != nil {
		fmt.Fprintf(stderr, "lark-cue: %v\n", err)
		return 2
	}

	openClawEnabled := !opts.noOpenClaw
	var openClaw openClawClient
	if openClawEnabled {
		openClaw = newOpenClawClient(cfg.OpenClaw)
		if err := openClaw.Preflight(ctx); err != nil {
			fmt.Fprintf(stderr, "lark-cue: OpenClaw is required by default but is not available: %v\n", err)
			fmt.Fprintln(stderr, "lark-cue: install/configure OpenClaw, or rerun with --no-openclaw for card-only mode.")
			return 2
		}
	}

	result, err := runner.Run(ctx, opts.command, runner.Streams{
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
		Buffer: runner.NewBoundedBuffer(256 * 1024),
	})
	if err != nil {
		fmt.Fprintf(stderr, "lark-cue: failed to run command: %v\n", err)
		return 127
	}
	if result.ExitCode == 0 {
		return 0
	}

	analysisOutput := result.Output
	started := time.Now()
	decision, plannerErr := provider.PlanRetrieval(ctx, llm.PlanInput{
		Command:  opts.command,
		ExitCode: result.ExitCode,
		Output:   analysisOutput,
	})
	plannerLatency := time.Since(started).Milliseconds()
	if plannerErr != nil {
		fmt.Fprintf(stderr, "lark-cue: planner failed: %v\n", plannerErr)
		return result.ExitCode
	}
	decision.Queries = llm.NormalizeQueries(decision.Queries, 8, 30)
	if !decision.ShouldRetrieve {
		if strings.TrimSpace(decision.Reason) != "" {
			fmt.Fprintf(stderr, "\nlark-cue: no internal knowledge lookup recommended: %s\n", decision.Reason)
		} else {
			fmt.Fprintln(stderr, "\nlark-cue: no internal knowledge lookup recommended for this failure.")
		}
		if err := eval.Append(cfg.Evaluation.LogPath, eval.FromPlanner(runner.CommandString(opts.command), decision, plannerLatency)); err != nil {
			fmt.Fprintf(stderr, "lark-cue: failed to write evaluation log: %v\n", err)
		}
		return result.ExitCode
	}
	if len(decision.Queries) == 0 {
		if strings.TrimSpace(decision.Reason) == "" {
			decision.Reason = "planner recommended lookup but produced no keyword queries"
		}
		fmt.Fprintln(stderr, "\nlark-cue: planner recommended lookup but produced no keyword queries.")
		if err := eval.Append(cfg.Evaluation.LogPath, eval.FromPlanner(runner.CommandString(opts.command), decision, plannerLatency)); err != nil {
			fmt.Fprintf(stderr, "lark-cue: failed to write evaluation log: %v\n", err)
		}
		return result.ExitCode
	}

	if err := eval.Append(cfg.Evaluation.LogPath, eval.FromPlanner(runner.CommandString(opts.command), decision, plannerLatency)); err != nil {
		fmt.Fprintf(stderr, "lark-cue: failed to write evaluation log: %v\n", err)
	}

	scenario := scenarioFromDecision(decision)
	queries := decision.Queries
	if shouldStyleOutput(stderr) {
		fmt.Fprintln(stderr)
		fmt.Fprint(stderr, card.RenderPlannerStatusStyled(scenario, decision.Reason, queries, analysisOutput, terminalWidth(stderr)))
	} else {
		fmt.Fprint(stderr, card.RenderPlannerStatus(scenario, decision.Reason, queries))
	}

	if opts.verbose {
		fmt.Fprintf(stderr, "lark-cue: LLM configured model=%s base_url=%s\n", cfg.LLM.Model, cfg.LLM.BaseURL)
		fmt.Fprintf(stderr, "lark-cue: planner reason: %s\n", decision.Reason)
		fmt.Fprintf(stderr, "lark-cue: planner queries: %s\n", strings.Join(queries, " | "))
	}

	retriever := newRetriever(cfg.Feishu)
	sources, retrievalStatus, retrievalErr := retriever.Retrieve(ctx, queries)
	if retrievalErr != nil {
		if retrievalStatus == retrieval.StatusPartial {
			fmt.Fprintf(stderr, "lark-cue: Feishu retrieval partially failed: %v\n", retrievalErr)
		} else {
			fmt.Fprintf(stderr, "lark-cue: Feishu retrieval failed: %v\n", retrievalErr)
		}
	}

	scored := evidence.ScoreWithContext(sources, evidence.Context{
		Scenario: decision.Scenario,
		Queries:  queries,
		Output:   analysisOutput,
	})
	selected, confidence := evidence.Select(scored)
	var llmStatus card.LLMStatus
	kcard := card.Build(ctx, card.Input{
		Command:         opts.command,
		Output:          analysisOutput,
		Scenario:        scenario,
		PlannerReason:   decision.Reason,
		Queries:         queries,
		Evidence:        selected,
		Confidence:      confidence,
		RetrievalStatus: retrievalStatus,
		RetrievalError:  retrievalErr,
		Provider:        provider,
		LLMStatus:       &llmStatus,
	})
	if opts.verbose {
		switch {
		case llmStatus.Accepted:
			fmt.Fprintln(stderr, "lark-cue: LLM card draft accepted")
		case llmStatus.Attempted && llmStatus.Error != "":
			fmt.Fprintf(stderr, "lark-cue: LLM card draft fallback: %s\n", llmStatus.Error)
		case llmStatus.Attempted:
			fmt.Fprintln(stderr, "lark-cue: LLM card draft fallback")
		default:
			fmt.Fprintln(stderr, "lark-cue: LLM card draft not attempted")
		}
	}

	cueOutput := stderr
	fmt.Fprintln(cueOutput)
	if shouldStyleOutput(cueOutput) {
		fmt.Fprint(cueOutput, card.RenderStyled(kcard, terminalWidth(cueOutput)))
	} else {
		fmt.Fprint(cueOutput, card.Render(kcard))
	}

	if openClawEnabled {
		cwd, _ := os.Getwd()
		task := openclaw.BuildTask(openclaw.TaskInput{
			WorkingDir:      cwd,
			Command:         opts.command,
			ExitCode:        result.ExitCode,
			Output:          analysisOutput,
			PlannerScenario: decision.Scenario,
			PlannerReason:   decision.Reason,
			Queries:         queries,
			Card:            kcard,
			Evidence:        selected,
		})
		fmt.Fprintf(cueOutput, "\nlark-cue: invoking OpenClaw agent %s...\n", openclaw.DefaultAgent)
		openClawResult := openClaw.Invoke(ctx, task, io.Discard)
		kcard.OpenClaw = card.OpenClawHandoff{
			Attempted: true,
			Succeeded: openClawResult.Succeeded,
			TimedOut:  openClawResult.TimedOut,
			ExitCode:  openClawResult.ExitCode,
			Error:     openClawResult.Error,
			LatencyMS: openClawResult.LatencyMS,
		}
		fmt.Fprintln(cueOutput)
		if shouldStyleOutput(cueOutput) {
			fmt.Fprint(cueOutput, openclaw.RenderResultStyled(openClawResult, result.ExitCode, terminalWidth(cueOutput)))
		} else {
			fmt.Fprint(cueOutput, openclaw.RenderResult(openClawResult, result.ExitCode))
		}
	} else if opts.noOpenClaw {
		kcard.OpenClaw = card.OpenClawHandoff{SkippedReason: "--no-openclaw"}
	}

	if opts.preparePush || opts.sendPush || cfg.Feishu.SendPushDefault {
		target := firstNonEmpty(opts.pushChat, cfg.Feishu.DefaultPushChat)
		renderedPush := push.Prepare(kcard)
		fmt.Fprintln(cueOutput, "\nPush Preview")
		fmt.Fprintln(cueOutput, renderedPush)
		if opts.sendPush {
			if target == "" {
				fmt.Fprintln(stderr, "lark-cue: push send requested but no --push-chat or feishu.default_push_chat is configured")
			} else {
				sender := push.NewSender(larkcli.NewWithProfile("lark-cli", cfg.Feishu.Profile))
				if err := sender.Send(ctx, target, renderedPush); err != nil {
					fmt.Fprintf(stderr, "lark-cue: failed to send Feishu push: %v\n", err)
				} else {
					fmt.Fprintf(stderr, "lark-cue: Feishu push sent to %s\n", target)
				}
			}
		}
	}

	kcard.LatencyMS = time.Since(started).Milliseconds()
	kcard.Feedback = "skipped"

	if err := eval.Append(cfg.Evaluation.LogPath, eval.FromCard(kcard)); err != nil {
		fmt.Fprintf(stderr, "lark-cue: failed to write evaluation log: %v\n", err)
	}

	return result.ExitCode
}

func runBenchmarkCommand(ctx context.Context, cfg config.Config, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printBenchmarkHelp(stdout)
		return 0
	}
	switch args[0] {
	case "-h", "--help", "help":
		printBenchmarkHelp(stdout)
		return 0
	case "run":
		opts, err := parseBenchmarkRunArgs(args[1:])
		if err == errHelp {
			printBenchmarkRunHelp(stdout)
			return 0
		}
		if err != nil {
			fmt.Fprintf(stderr, "lark-cue benchmark run: %v\n\n", err)
			printBenchmarkRunHelp(stderr)
			return 2
		}
		return runBenchmark(ctx, cfg, opts, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "lark-cue benchmark: unknown command %q\n\n", args[0])
		printBenchmarkHelp(stderr)
		return 2
	}
}

func runBenchmark(ctx context.Context, cfg config.Config, opts benchmarkRunOptions, stdout, stderr io.Writer) int {
	cases, err := benchmark.LoadCases(opts.casesPath)
	if err != nil {
		fmt.Fprintf(stderr, "lark-cue benchmark: %v\n", err)
		return 2
	}
	if _, err := newPlannerProvider(cfg.LLM); err != nil {
		fmt.Fprintf(stderr, "lark-cue benchmark: %v\n", err)
		return 2
	}
	tempDir, err := os.MkdirTemp("", "lark-cue-benchmark-*")
	if err != nil {
		fmt.Fprintf(stderr, "lark-cue benchmark: failed to create temporary evaluation log: %v\n", err)
		return 2
	}
	defer os.RemoveAll(tempDir)

	benchCfg := cfg
	benchCfg.Evaluation.LogPath = filepath.Join(tempDir, "evaluations.jsonl")
	restoreEvalLogEnv := setBenchmarkEvalLogEnv(benchCfg.Evaluation.LogPath)
	defer restoreEvalLogEnv()

	jobs := normalizedBenchmarkJobs(opts.jobs, len(cases))
	results, err := runBenchmarkCases(ctx, benchCfg, cases, opts, tempDir, jobs)
	if err != nil {
		fmt.Fprintf(stderr, "lark-cue benchmark: %v\n", err)
		return 2
	}
	summary := benchmark.Summarize(results, opts.verbose)
	summary.Jobs = jobs
	if shouldStyleOutput(stdout) {
		fmt.Fprint(stdout, benchmark.RenderSummaryStyled(summary, terminalWidth(stdout)))
	} else {
		fmt.Fprint(stdout, benchmark.RenderSummary(summary))
	}
	if summary.AllPassed() {
		return 0
	}
	return 1
}

func normalizedBenchmarkJobs(requested int, caseCount int) int {
	if caseCount < 1 {
		return 1
	}
	if requested > 0 {
		if requested > caseCount {
			return caseCount
		}
		return requested
	}
	if caseCount < 4 {
		return caseCount
	}
	return 4
}

type benchmarkCaseRun struct {
	index  int
	result benchmark.CaseResult
	err    error
}

func runBenchmarkCases(ctx context.Context, cfg config.Config, cases []benchmark.Case, opts benchmarkRunOptions, tempDir string, jobs int) ([]benchmark.CaseResult, error) {
	results := make([]benchmark.CaseResult, len(cases))
	indexes := make(chan int)
	out := make(chan benchmarkCaseRun, len(cases))
	var wg sync.WaitGroup

	for worker := 0; worker < jobs; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range indexes {
				c := cases[index]
				caseCfg := cfg
				caseCfg.Evaluation.LogPath = filepath.Join(tempDir, fmt.Sprintf("%03d-%s.jsonl", index+1, sanitizeScenarioID(c.ID)))
				observation, err := runBenchmarkCase(ctx, caseCfg, c, opts)
				if err != nil {
					out <- benchmarkCaseRun{index: index, err: err}
					continue
				}
				out <- benchmarkCaseRun{index: index, result: benchmark.ScoreCase(c, observation)}
			}
		}()
	}

	go func() {
		for index := range cases {
			indexes <- index
		}
		close(indexes)
		wg.Wait()
		close(out)
	}()

	for item := range out {
		if item.err != nil {
			return nil, item.err
		}
		results[item.index] = item.result
	}
	return results, nil
}

func setBenchmarkEvalLogEnv(path string) func() {
	old, hadOld := os.LookupEnv("LARK_CUE_EVAL_LOG")
	_ = os.Setenv("LARK_CUE_EVAL_LOG", path)
	return func() {
		if hadOld {
			_ = os.Setenv("LARK_CUE_EVAL_LOG", old)
		} else {
			_ = os.Unsetenv("LARK_CUE_EVAL_LOG")
		}
	}
}

func runBenchmarkCase(ctx context.Context, cfg config.Config, c benchmark.Case, opts benchmarkRunOptions) (benchmark.Observation, error) {
	observation := benchmark.Observation{CommandExitCode: -1}
	setupOutput, setupErr := runBenchmarkSetup(ctx, c.Setup)
	observation.SetupOutput = setupOutput
	if setupErr != "" {
		observation.SetupError = setupErr
		observation.TeardownOutput, observation.TeardownErrors = runBenchmarkTeardown(ctx, c.Teardown)
		return observation, nil
	}

	before, err := readCueRecords(cfg.Evaluation.LogPath)
	if err != nil {
		observation.TeardownOutput, observation.TeardownErrors = runBenchmarkTeardown(ctx, c.Teardown)
		return observation, fmt.Errorf("failed to read benchmark evaluation log before case %q: %w", c.ID, err)
	}
	var stdout, stderr bytes.Buffer
	observation.CommandExitCode = runCommand(ctx, cfg, runOptions{
		command:    c.Command,
		verbose:    opts.verbose,
		noOpenClaw: opts.noOpenClaw,
	}, strings.NewReader(""), &stdout, &stderr)
	observation.CommandOutput = stdout.String() + stderr.String()
	after, err := readCueRecords(cfg.Evaluation.LogPath)
	observation.TeardownOutput, observation.TeardownErrors = runBenchmarkTeardown(ctx, c.Teardown)
	if err != nil {
		return observation, fmt.Errorf("failed to read benchmark evaluation log after case %q: %w", c.ID, err)
	}
	if len(after.PlannerRecords) >= len(before.PlannerRecords) {
		observation.PlannerRecords = append([]eval.Record(nil), after.PlannerRecords[len(before.PlannerRecords):]...)
	}
	if len(after.Records) >= len(before.Records) {
		observation.CueRecords = append([]eval.Record(nil), after.Records[len(before.Records):]...)
	}
	return observation, nil
}

func runBenchmarkSetup(ctx context.Context, commands [][]string) (string, string) {
	var output strings.Builder
	for _, command := range commands {
		out, exitCode, err := runBenchmarkAuxCommand(ctx, command)
		output.WriteString(out)
		if err != nil {
			return output.String(), err.Error()
		}
		if exitCode != 0 {
			return output.String(), fmt.Sprintf("%s exited %d", runner.CommandString(command), exitCode)
		}
	}
	return output.String(), ""
}

func runBenchmarkTeardown(ctx context.Context, commands [][]string) (string, []string) {
	var output strings.Builder
	var errs []string
	for _, command := range commands {
		out, exitCode, err := runBenchmarkAuxCommand(ctx, command)
		output.WriteString(out)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		if exitCode != 0 {
			errs = append(errs, fmt.Sprintf("%s exited %d", runner.CommandString(command), exitCode))
		}
	}
	return output.String(), errs
}

func runBenchmarkAuxCommand(ctx context.Context, command []string) (string, int, error) {
	var stdout, stderr bytes.Buffer
	result, err := runner.Run(ctx, command, runner.Streams{
		Stdin:  strings.NewReader(""),
		Stdout: &stdout,
		Stderr: &stderr,
		Buffer: runner.NewBoundedBuffer(256 * 1024),
	})
	return stdout.String() + stderr.String(), result.ExitCode, err
}

func scenarioFromDecision(decision llm.PlanDecision) detector.Scenario {
	name := strings.TrimSpace(decision.Scenario)
	if name == "" {
		name = "Internal knowledge cue"
	}
	return detector.Scenario{
		ID:      sanitizeScenarioID(name),
		Name:    name,
		Matched: decision.Queries,
	}
}

func sanitizeScenarioID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastUnderscore := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			lastUnderscore = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastUnderscore = false
		default:
			if !lastUnderscore && b.Len() > 0 {
				b.WriteByte('_')
				lastUnderscore = true
			}
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "internal_knowledge_cue"
	}
	return out
}

func runEval(cfg config.Config, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printEvalHelp(stdout)
		return 0
	}
	switch args[0] {
	case "-h", "--help", "help":
		printEvalHelp(stdout)
		return 0
	case "report":
		opts, err := parseEvalReportArgs(args[1:])
		if err == errHelp {
			printEvalReportHelp(stdout)
			return 0
		}
		if err != nil {
			fmt.Fprintf(stderr, "lark-cue eval report: %v\n\n", err)
			printEvalReportHelp(stderr)
			return 2
		}
		result, err := eval.ReadCueRecords(cfg.Evaluation.LogPath)
		if err != nil {
			fmt.Fprintf(stderr, "lark-cue: failed to read evaluation log: %v\n", err)
			return 1
		}
		result = eval.LimitReadResult(result, opts.limit)
		summary := eval.SummarizeResult(result, opts.limit)
		if shouldStyleOutput(stdout) {
			fmt.Fprint(stdout, eval.RenderSummaryStyled(summary, terminalWidth(stdout)))
		} else {
			fmt.Fprint(stdout, eval.RenderSummary(summary))
		}
		return 0
	default:
		fmt.Fprintf(stderr, "lark-cue eval: unknown command %q\n\n", args[0])
		printEvalHelp(stderr)
		return 2
	}
}

func isInteractive(value any) bool {
	file, ok := value.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(file.Fd()))
}

func shouldStyleOutput(value any) bool {
	return isInteractive(value) && os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb"
}

func terminalWidth(value any) int {
	file, ok := value.(*os.File)
	if !ok {
		return 88
	}
	width, _, err := term.GetSize(int(file.Fd()))
	if err != nil || width <= 0 {
		return 88
	}
	return width
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func printHelp(w io.Writer) {
	fmt.Fprintln(w, `lark-cue: active Feishu knowledge cues for terminal workflows

Usage:
  lark-cue run [flags] -- <command>
  lark-cue benchmark run --cases <path> [flags]
  lark-cue eval report [flags]

Commands:
  run       Run a command and show an LLM-planned evidence-backed internal knowledge cue on failures
  benchmark Run real benchmark cases and score cited expected sources
  eval      Summarize local cue evaluation records
  help      Show this help
  version   Show version`)
}

func printRunHelp(w io.Writer) {
	fmt.Fprintln(w, `Usage:
  lark-cue run [flags] -- <command>

Requires LLM config:
  LARK_CUE_LLM_API_KEY and LARK_CUE_LLM_MODEL
Optional Feishu profile:
  LARK_CUE_FEISHU_PROFILE to pass --profile to lark-cli retrieval and push sending
OpenClaw is required by default:
  install/configure openclaw, or pass --no-openclaw for card-only mode

Flags:
  --prepare-push        Print a Feishu group message preview
  --send-push           Send the prepared message through lark-cli
  --push-chat <target>  Feishu chat id or chat name for push sending
  --no-openclaw         Skip OpenClaw preflight and post-card handoff
  --verbose             Print LLM/retrieval diagnostics without secrets`)
}

func printEvalHelp(w io.Writer) {
	fmt.Fprintln(w, `Usage:
  lark-cue eval report [flags]

Commands:
  report  Summarize local cue evaluation records`)
}

func printBenchmarkHelp(w io.Writer) {
	fmt.Fprintln(w, `Usage:
  lark-cue benchmark run --cases <path> [flags]

Commands:
  run  Run real benchmark cases and score cited expected sources`)
}

func printBenchmarkRunHelp(w io.Writer) {
	fmt.Fprintln(w, `Usage:
  lark-cue benchmark run --cases <path> [flags]

Flags:
  --cases <path>  Benchmark case JSON file
  --jobs <N>      Run up to N cases concurrently (default: 4)
  --no-openclaw   Skip OpenClaw preflight and post-card handoff for benchmark cases
  --verbose       Include compact failure diagnostics in the report`)
}

func printEvalReportHelp(w io.Writer) {
	fmt.Fprintf(w, `Usage:
  lark-cue eval report [flags]

Flags:
  --limit <N>  Summarize the latest N cue records (default: %d)
`, eval.DefaultReportLimit)
}
