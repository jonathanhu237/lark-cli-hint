package app

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/term"

	"lark-cue/internal/card"
	"lark-cue/internal/config"
	"lark-cue/internal/detector"
	"lark-cue/internal/eval"
	"lark-cue/internal/evidence"
	"lark-cue/internal/larkcli"
	"lark-cue/internal/llm"
	"lark-cue/internal/push"
	"lark-cue/internal/retrieval"
	"lark-cue/internal/runner"
)

const version = "0.1.0"

type runOptions struct {
	command          []string
	preparePush      bool
	sendPush         bool
	pushChat         string
	noFeedbackPrompt bool
	verbose          bool
}

type evalReportOptions struct {
	limit int
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
		if err != nil {
			fmt.Fprintf(stderr, "lark-cue run: %v\n\n", err)
			printRunHelp(stderr)
			return 2
		}
		return runCommand(context.Background(), cfg, opts, stdin, stdout, stderr)
	case "eval":
		return runEval(cfg, args[1:], stdout, stderr)
	case "feedback":
		return runFeedback(cfg, args[1:], stdout, stderr)
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
		case "--no-feedback-prompt":
			opts.noFeedbackPrompt = true
		case "--verbose":
			opts.verbose = true
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

func runCommand(ctx context.Context, cfg config.Config, opts runOptions, stdin io.Reader, stdout, stderr io.Writer) int {
	provider, err := newPlannerProvider(cfg.LLM)
	if err != nil {
		fmt.Fprintf(stderr, "lark-cue: %v\n", err)
		return 2
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
	decision.Queries = llm.NormalizeQueries(decision.Queries, 8, 120)
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
	if shouldStyleOutput(stderr) {
		fmt.Fprintln(stderr)
		fmt.Fprint(stderr, card.RenderStatusStyled(scenario, analysisOutput, terminalWidth(stderr)))
	} else {
		fmt.Fprintf(stderr, "\nlark-cue: planner selected %s; searching Feishu knowledge...\n", scenario.Name)
	}

	queries := decision.Queries
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

	feedbackState := "skipped"
	kcard.LatencyMS = time.Since(started).Milliseconds()
	if !opts.noFeedbackPrompt && isInteractive(stdin) && isInteractive(cueOutput) {
		feedbackState = promptFeedback(stdin, cueOutput)
	}
	kcard.Feedback = feedbackState

	if err := eval.Append(cfg.Evaluation.LogPath, eval.FromCard(kcard)); err != nil {
		fmt.Fprintf(stderr, "lark-cue: failed to write evaluation log: %v\n", err)
	}

	return result.ExitCode
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

func runFeedback(cfg config.Config, args []string, stdout, stderr io.Writer) int {
	if len(args) != 2 {
		fmt.Fprintln(stderr, "usage: lark-cue feedback <card-id> useful|not-useful")
		return 2
	}
	value := args[1]
	if value != "useful" && value != "not-useful" {
		fmt.Fprintln(stderr, "feedback must be useful or not-useful")
		return 2
	}
	if err := eval.AppendFeedback(cfg.Evaluation.LogPath, args[0], value); err != nil {
		fmt.Fprintf(stderr, "lark-cue: failed to write feedback: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Recorded feedback for %s: %s\n", args[0], value)
	return 0
}

func promptFeedback(stdin io.Reader, stdout io.Writer) string {
	fmt.Fprint(stdout, "\nWas this cue useful? [y/n/skip] ")
	line, err := bufio.NewReader(stdin).ReadString('\n')
	if err != nil && len(line) == 0 {
		return "skipped"
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes", "useful":
		return "useful"
	case "n", "no", "not-useful", "not_useful":
		return "not-useful"
	default:
		return "skipped"
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
  lark-cue eval report [flags]
  lark-cue feedback <card-id> useful|not-useful

Commands:
  run       Run a command and show an LLM-planned evidence-backed internal knowledge cue on failures
  eval      Summarize local cue evaluation records
  feedback  Record useful/not-useful feedback for a generated cue
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

Flags:
  --prepare-push        Print a Feishu group message preview
  --send-push           Send the prepared message through lark-cli
  --push-chat <target>  Feishu chat id or chat name for push sending
  --no-feedback-prompt  Do not ask for interactive useful/not-useful feedback
  --verbose             Print LLM/retrieval diagnostics without secrets`)
}

func printEvalHelp(w io.Writer) {
	fmt.Fprintln(w, `Usage:
  lark-cue eval report [flags]

Commands:
  report  Summarize local cue evaluation records`)
}

func printEvalReportHelp(w io.Writer) {
	fmt.Fprintf(w, `Usage:
  lark-cue eval report [flags]

Flags:
  --limit <N>  Summarize the latest N cue records (default: %d)
`, eval.DefaultReportLimit)
}
