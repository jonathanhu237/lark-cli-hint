package app

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
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
	"lark-cue/internal/query"
	"lark-cue/internal/retrieval"
	"lark-cue/internal/runner"
)

const version = "0.1.0"

type runOptions struct {
	command          []string
	demoFixture      bool
	preparePush      bool
	sendPush         bool
	pushChat         string
	noFeedbackPrompt bool
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
		case "--demo-fixture":
			opts.demoFixture = true
		case "--prepare-push":
			opts.preparePush = true
		case "--send-push":
			opts.preparePush = true
			opts.sendPush = true
		case "--no-feedback-prompt":
			opts.noFeedbackPrompt = true
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

func runCommand(ctx context.Context, cfg config.Config, opts runOptions, stdin io.Reader, stdout, stderr io.Writer) int {
	signalBuffer := detector.NewSignalBuffer(8192)
	result, err := runner.Run(ctx, opts.command, runner.Streams{
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
		Buffer: runner.NewBoundedBuffer(256 * 1024),
		Tap:    signalBuffer,
	})
	if err != nil {
		fmt.Fprintf(stderr, "lark-cue: failed to run command: %v\n", err)
		return 127
	}
	if result.ExitCode == 0 {
		return 0
	}

	analysisOutput := outputWithSignals(result.Output, signalBuffer.String())
	scenario, ok := detector.Detect(analysisOutput)
	if !ok {
		return result.ExitCode
	}

	started := time.Now()
	fmt.Fprintf(stderr, "\nlark-cue: detected %s; searching Feishu knowledge...\n", scenario.Name)

	provider := llm.NewOpenAICompatible(cfg.LLM)
	queries := query.Build(ctx, opts.command, analysisOutput, scenario, provider)

	var retriever retrieval.Retriever
	if opts.demoFixture {
		retriever = retrieval.NewFixtureRetriever()
	} else {
		retriever = retrieval.NewLarkRetriever(larkcli.New("lark-cli"))
	}

	sources, retrievalStatus, retrievalErr := retriever.Retrieve(ctx, queries)
	if retrievalErr != nil {
		if retrievalStatus == retrieval.StatusPartial {
			fmt.Fprintf(stderr, "lark-cue: Feishu retrieval partially failed: %v\n", retrievalErr)
		} else {
			fmt.Fprintf(stderr, "lark-cue: Feishu retrieval failed: %v\n", retrievalErr)
		}
	}

	scored := evidence.Score(sources)
	selected, confidence := evidence.Select(scored)
	kcard := card.Build(ctx, card.Input{
		Command:         opts.command,
		Output:          analysisOutput,
		Scenario:        scenario,
		Queries:         queries,
		Evidence:        selected,
		Confidence:      confidence,
		RetrievalStatus: retrievalStatus,
		RetrievalError:  retrievalErr,
		Fixture:         opts.demoFixture,
		Provider:        provider,
	})

	cueOutput := stderr
	fmt.Fprintln(cueOutput)
	fmt.Fprint(cueOutput, card.Render(kcard))

	if opts.preparePush || opts.sendPush || cfg.Feishu.SendPushDefault {
		target := firstNonEmpty(opts.pushChat, cfg.Feishu.DefaultPushChat)
		renderedPush := push.Prepare(kcard)
		fmt.Fprintln(cueOutput, "\nPush Preview")
		fmt.Fprintln(cueOutput, renderedPush)
		if opts.sendPush {
			if target == "" {
				fmt.Fprintln(stderr, "lark-cue: push send requested but no --push-chat or feishu.default_push_chat is configured")
			} else {
				sender := push.NewSender(larkcli.New("lark-cli"))
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

func outputWithSignals(output string, signals string) string {
	signals = strings.TrimSpace(signals)
	if signals == "" || strings.Contains(output, signals) {
		return output
	}
	if strings.TrimSpace(output) == "" {
		return signals
	}
	return output + "\n" + signals
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
  lark-cue feedback <card-id> useful|not-useful

Commands:
  run       Run a command and show an evidence-backed cue on Feishu API failures
  feedback  Record useful/not-useful feedback for a generated cue
  help      Show this help
  version   Show version`)
}

func printRunHelp(w io.Writer) {
	fmt.Fprintln(w, `Usage:
  lark-cue run [flags] -- <command>

Flags:
  --demo-fixture        Use labeled local fixture retrieval instead of real lark-cli
  --prepare-push        Print a Feishu group message preview
  --send-push           Send the prepared message through lark-cli
  --push-chat <target>  Feishu chat id or chat name for push sending
  --no-feedback-prompt  Do not ask for interactive useful/not-useful feedback`)
}
