## Context

The repository currently contains product documentation but no executable implementation. The product boundary is already fixed in `AGENTS.md`: first build a `lark-cli` knowledge-base command copilot with `Recover` and `Next`, but avoid domain-specific rules until the command wrapper, output capture, rendering, and JSON protocol are reliable.

This change establishes that foundation. It should produce a working `lark-cli-hint run -- lark-cli ...` command without adding docs/wiki analysis rules, RAG, LLM integration, or automatic follow-up execution.

## Goals / Non-Goals

**Goals:**

- Create a TypeScript/Node.js CLI executable named `lark-cli-hint`.
- Provide a `run` command that wraps a `lark-cli` command supplied after `--`.
- Stream wrapped command output in human mode while retaining captured output for analysis.
- Emit a single machine-readable JSON envelope in `--json` mode.
- Render a conservative five-section Hint Card in the selected locale.
- Keep the CLI shell separate from the core app logic so tests and future agent integrations can call the app directly.
- Preserve stdin availability for wrapped commands in human mode.

**Non-Goals:**

- Implement docs/wiki Recover or Next domain rules.
- Add YAML rule loading or a rule DSL.
- Add LLM SDKs, RAG, shell hooks, or long-running services.
- Automatically execute suggested `Next` commands.
- Send Feishu messages.
- Preserve stdout/stderr TTY semantics for wrapped commands. The MVP streams captured output back to the terminal, but it is not a PTY layer.

## Decisions

### Use TypeScript runtime with a thin CLI shell

Use `commander` for command parsing and keep it limited to argument handling and process exit behavior. The core `run` workflow should live in app-level modules that can be tested without invoking a real terminal.

Alternative considered: place all logic in `src/cli.ts`. This is faster initially, but it makes `--json`, i18n, runner behavior, and future analyzer rules harder to test independently.

### Stream in human mode, buffer in JSON mode

Human mode should behave like a lightweight wrapper around the underlying command: stdin remains available to the wrapped command, users see `lark-cli` output as it happens, and then receive the Hint Card. JSON mode should suppress streamed terminal prose and return one JSON envelope containing command metadata, captured output, exit status, and hint data.

Alternative considered: always stream output and append JSON. That is hostile to AI agents because the output is no longer a valid single JSON document.

Alternative considered: use a PTY or tee layer to preserve stdout/stderr TTY semantics while capturing output. That is more faithful to terminal behavior, but it adds platform complexity beyond the bootstrap runner. This can be revisited when interactive commands become a primary requirement.

### Start with a baseline analyzer

The analyzer should generate a conservative Hint Card based on exit status and captured output only. It should not invent domain-specific `Next` commands. Domain rules for docs/wiki can be added in a later change.

Alternative considered: implement the first docs/wiki rules immediately. That would blur the purpose of this change and make runner correctness harder to isolate.

### Localize user-facing strings, not protocol fields

The JSON envelope shape should use stable English field names. Human-readable field values and terminal labels should be localized. Default to English, and choose `zh-CN` when `LANG`, `LC_ALL`, or `LC_MESSAGES` indicates a Chinese environment.

Alternative considered: localize JSON keys. That would make AI-agent consumption and tests unnecessarily fragile.

## Risks / Trade-offs

- Wrapped commands can produce large output -> capture should be bounded so memory usage stays predictable.
- Some commands behave differently when stdout/stderr are not TTYs -> the bootstrap implementation explicitly does not guarantee TTY preservation; it only guarantees stdin inheritance and streamed output forwarding in human mode.
- JSON mode cannot both stream arbitrary `lark-cli` output and remain one valid JSON document -> choose valid JSON over real-time output for agent use.
- Locale detection can be imperfect -> keep the first heuristic simple and deterministic.
