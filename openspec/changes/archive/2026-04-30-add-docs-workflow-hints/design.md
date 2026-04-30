## Context

`cli-runner` now provides the executable wrapper, captured output, baseline Hint Cards, JSON envelope output, and i18n. However, the product still has no domain-specific Recover or Next behavior. The first demo-worthy domain path is the Feishu docs/wiki workflow described in `AGENTS.md`: search docs/wiki, fetch a document body, and prepare a group message without sending it automatically.

This change adds deterministic docs workflow hints directly to the analyzer layer. It should stay intentionally narrow and should not introduce a general rule engine.

## Goals / Non-Goals

**Goals:**

- Detect `lark-cli docs +search` success and suggest a real `lark-cli docs +fetch --doc <token-or-url>` command.
- Detect `lark-cli docs +fetch` success and suggest a real `lark-cli im +messages-send --chat-id <oc_xxx> --markdown <message>` command.
- Detect `lark-cli docs +fetch` failure and produce a Recover hint for common argument/resource/setup failures.
- Keep all suggestions deterministic, localized, and visible in both terminal Hint Card output and JSON envelope output.
- Keep baseline analyzer behavior as the fallback when docs workflow rules do not match.

**Non-Goals:**

- Add YAML rule loading or a generic rules DSL.
- Add LLM summarization, RAG, long-running sessions, or persistent memory.
- Automatically execute `Next` commands.
- Actually send Feishu messages.
- Fully support every possible `lark-cli docs` output shape.
- Resolve wiki node tokens through additional API calls in this change.

## Decisions

### Add a narrow docs workflow analyzer before baseline fallback

Add an analyzer path that receives the full command, exit status, stdout, and stderr. It should attempt docs-specific analysis first and return `null` when no docs rule matches. The existing baseline analyzer remains the fallback.

Alternative considered: implement a YAML-driven rule engine first. That is more extensible, but it creates rule DSL design work before the first product hint exists. A narrow analyzer is easier to validate and can later be replaced or backed by YAML.

### Parse command intent from argv, not from rendered text

Identify commands using argv segments such as `["lark-cli", "docs", "+search"]` and `["lark-cli", "docs", "+fetch"]`. Do not infer command type from localized prose or formatted terminal output.

Alternative considered: match raw command strings. String matching is fragile around quoting and option order.

### Extract the top document candidate conservatively

For `docs +search` success, parse stdout as JSON and inspect common result containers such as `data.items`, `items`, `data.results`, or `results`. Select the first candidate that exposes a usable document token, URL, or token-like field. If no usable candidate exists, fall back to the baseline hint.

Alternative considered: support every Feishu search response variation now. That is unnecessary for the demo and should be driven by fixtures as real output shapes are collected.

### Use placeholder chat IDs for prepared send commands

For `docs +fetch` success, suggest a command that prepares a group push but does not execute it. If no configured chat ID exists, use a visible placeholder such as `<chat_id>` rather than inventing a real target.

Alternative considered: add project-specific resource mapping or Recall now. That belongs to a later change; this change only prepares the command shape.

### Recover from observable failure evidence only

For failed `docs +fetch`, classify only failures supported by the wrapped command arguments or captured error text: missing/incorrect `--doc`, wiki-looking values passed to `--doc`, not configured errors, unsupported identity, and generic fetch failure. Every Recover hint must cite command args, stderr, or both.

Alternative considered: call `lark-cli` help/schema during analysis. That creates nested command execution and harder failure modes; static command templates are enough for this change.

## Risks / Trade-offs

- Search output shapes can vary -> use fixtures for known shapes and fall back conservatively when extraction fails.
- Suggested `im +messages-send` command could look actionable without a real chat ID -> use a placeholder unless a safe configured value exists; never auto-run it.
- Error classification can be overconfident -> keep confidence moderate and cite stderr/args as evidence.
- Current docs in the repo contain outdated command examples -> update docs/guardrails as part of the implementation tasks.
