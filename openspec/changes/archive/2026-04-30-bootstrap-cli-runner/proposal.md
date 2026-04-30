## Why

Lark CLI Hint needs a reliable executable foundation before Recover and Next rules can be useful. The first step is to prove that `lark-cli-hint run -- lark-cli ...` can wrap real `lark-cli` commands, preserve terminal usability, and expose machine-readable results for AI agents.

## What Changes

- Add the initial `lark-cli-hint` CLI surface with a `run` command that accepts a wrapped command after `--`.
- Execute the wrapped command while capturing `stdout`, `stderr`, and exit status.
- In human mode, stream wrapped command output to the terminal and append a short Hint Card after completion.
- In `--json` mode, emit one JSON envelope containing command metadata, captured output, exit status, and hint data without intermixing terminal prose.
- Add minimal locale-aware rendering for English by default and Simplified Chinese when the user environment is Chinese.
- Establish a testable core app boundary separate from the commander CLI shell.

## Capabilities

### New Capabilities

- `cli-runner`: Wrap `lark-cli` commands, capture execution results, and render human or JSON hint output.

### Modified Capabilities

None.

## Impact

- Introduces the TypeScript/Node.js project scaffold and CLI executable named `lark-cli-hint`.
- Adds runtime dependencies for command parsing and local development/test tooling.
- Establishes core modules for command running, analysis, i18n, and output rendering.
- Does not add docs/wiki Recover or Next domain rules yet.
- Does not introduce an LLM SDK, RAG, shell hooks, or automatic execution of suggested commands.
