## 1. Configuration and CLI

- [x] 1.1 Add OpenClaw config defaults for binary `openclaw` and timeout 900 seconds, with environment-variable overrides, while keeping local `main` as the fixed MVP invocation target.
- [x] 1.2 Add `--no-openclaw` to `lark-cue run` parsing and help output.
- [x] 1.3 Add app-level tests proving default mode preflights OpenClaw before executing a command and `--no-openclaw` skips that preflight.

## 2. OpenClaw Adapter

- [x] 2.1 Add an internal OpenClaw package or adapter that can preflight `openclaw agent --help`.
- [x] 2.2 Implement OpenClaw invocation using `openclaw agent --local --agent main --timeout <seconds> --message <task>`.
- [x] 2.3 Route OpenClaw stdout and stderr to the caller-provided stderr writer.
- [x] 2.4 Add adapter unit tests for command shape, preflight failure, invocation failure, and timeout behavior using fakes.

## 3. Task Message Generation

- [x] 3.1 Add a deterministic OpenClaw task builder from cwd, command, exit code, output excerpt, planner decision, knowledge card, and scored evidence.
- [x] 3.2 Include safety constraints and verification requirements in the generated task.
- [x] 3.3 Add task builder tests proving required fields, citations/snippets, action plan, and constraints are present.

## 4. App Orchestration

- [x] 4.1 Run OpenClaw preflight after LLM config validation and before wrapped command execution in default mode.
- [x] 4.2 Invoke OpenClaw only after planner retrieval and knowledge card rendering.
- [x] 4.3 Do not invoke OpenClaw for successful commands, planner skip decisions, retrieval with no queries, or `--no-openclaw`.
- [x] 4.4 Preserve wrapped command exit codes when OpenClaw succeeds, fails, or times out.
- [x] 4.5 Add app-level tests for invocation order, stdout cleanliness, stderr routing, planner skip behavior, and exit-code preservation.

## 5. Evaluation and Reporting

- [x] 5.1 Extend cue evaluation records with OpenClaw attempted/succeeded/skipped/error/latency fields.
- [x] 5.2 Update eval report output to summarize OpenClaw attempts without requiring OpenClaw at report time.
- [x] 5.3 Add eval tests for OpenClaw fields and report summaries.

## 6. Documentation and Validation

- [x] 6.1 Update README and demo docs with OpenClaw installation, PATH, model configuration, verification, default invocation, and `--no-openclaw` usage.
- [x] 6.2 Run `go test ./...`, `go build -o ./bin/lark-cue ./cmd/lark-cue`, and `openspec validate invoke-openclaw-by-default --strict`.
- [x] 6.3 Run a local manual smoke test with `--no-openclaw`; if OpenClaw is available, also run an explicit OpenClaw preflight smoke test.
