# AGENTS.md

This file is the working agreement for humans and AI agents contributing to Lark CLI Hint.

## Product Source of Truth

Lark CLI Hint is a `lark-cli` knowledge-base command copilot for humans and AI agents.

Do not drift into building:

- A general shell assistant for npm, pnpm, pytest, git, or arbitrary terminal errors.
- A Warp-style GUI terminal.
- A generic chatbot.
- A full enterprise RAG/indexing platform in the first version.
- An agent that automatically sends Feishu messages or executes follow-up commands without explicit user action.

## MVP Surface

The first version has one primary entry point:

```bash
lark-cli-hint run -- lark-cli ...
```

It has two user-visible capabilities:

- `Recover`: if the wrapped `lark-cli` command fails, explain the likely cause and suggest one repair command.
- `Next`: if the wrapped `lark-cli` command succeeds, explain what was learned and suggest one next command.

`Plan`, `Recall`, shell hooks, and full RAG are deferred unless the README is deliberately updated to expand scope.

## Main Demo Flow

Implement and test around this knowledge-base flow first:

```text
docs/wiki search -> docs fetch -> suggest im push command
```

The key Recover demo is wiki token / doc token confusion:

```bash
lark-cli-hint run -- lark-cli docs +fetch --doc wiki_xxx
```

The key Next demo is:

```bash
lark-cli-hint run -- lark-cli docs +search --query "<project keyword>"
```

which should suggest:

```bash
lark-cli docs +fetch --doc <top_doc_token>
```

## Hint Card Contract

Default terminal output must use exactly these sections:

```text
Status
Hint
Next
Why
Sources
```

Keep the card short. It should be useful inside an active terminal workflow.

Rules:

- Prefer one best `Next` command.
- Do not auto-run `Next`.
- Include evidence in `Sources`.
- Support `--json` with the same semantics for AI-agent consumption.
- Command suggestions should be rule-backed first. LLM output may improve wording but must not invent unsupported commands or parameters.
- User-facing text must support i18n. Do not hard-code English or Chinese strings deep inside command logic; keep messages locale-aware and testable.

## Implementation Priorities

Prioritize deterministic handling for:

- `lark-cli docs +search` success.
- `lark-cli docs +fetch` success.
- `lark-cli docs +fetch` failures caused by token-type mismatch.
- Preparing, but not executing, an `lark-cli im +messages-send ...` push command.

Use mocked fixtures when real Feishu access is unavailable. Mocking is acceptable for the contest demo as long as the behavior reflects real `lark-cli` command shapes and the hint cites its fixture source.

## Technical Direction

Use this stack for the first implementation:

- Language/runtime: TypeScript on Node.js.
- Package manager and tooling: pnpm, tsup, vitest, and tsx.
- CLI framework: commander.
- Package name and executable name: `lark-cli-hint`.
- Architecture: a thin CLI shell over a core app library. Do not put analyzer logic directly inside commander handlers.
- Execution model: human mode streams the wrapped `lark-cli` output to the terminal and then appends the Hint Card; `--json` mode emits one JSON envelope and does not intermix terminal prose.
- i18n: default to English, switch to `zh-CN` when the user environment is Chinese, and allow explicit locale override later. JSON field names stay stable and English; user-facing values are localized.
- Data files: YAML for human-maintained rules/config, JSON for locale files, and JSON/TXT for fixtures that mirror original `lark-cli` output.
- LLM: do not introduce an LLM SDK in the MVP. Keep analyzer boundaries extensible so LLM/RAG can be added later.
