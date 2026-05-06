## Why

The MVP can now detect a Feishu API failure and render cited knowledge cards, but the demo still relies on a single visible card to imply product value. For the OpenClaw contest, `lark-cue` needs a terminal-native way to summarize recent cue runs so the demo can show retrieval quality, citation coverage, feedback, and latency as verifiable evidence.

## What Changes

- Add a `lark-cue eval report` flow that reads the local evaluation JSONL log and renders a human-readable terminal validation report.
- Summarize recent cue records by run count, retrieval status, fixture-vs-real usage, citation coverage, query count, latency, and feedback state.
- Keep the report read-only: it must not run commands, modify cue records, call `lark-cli`, send Feishu messages, or invoke an LLM.
- Preserve optional machine-friendly behavior only where it supports the demo, such as deterministic output and clear handling of missing or empty logs.
- Update demo documentation so the recorded demo can show the cue flow followed by a validation view.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `terminal-knowledge-cue`: add a validation report requirement over existing evaluation records.

## Impact

- CLI routing in `cmd/lark-cue` / `internal/app` for an `eval report` subcommand.
- Evaluation package additions for reading and aggregating JSONL records.
- Terminal rendering additions for a validation summary view, reusing the current styled/non-styled output behavior.
- Tests for report aggregation, empty-log behavior, malformed-record tolerance, and CLI output.
- Demo docs and OpenSpec spec text for the validation report behavior.
