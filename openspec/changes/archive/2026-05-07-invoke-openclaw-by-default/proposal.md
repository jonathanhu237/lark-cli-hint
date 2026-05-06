## Why

`lark-cue` currently stops at a cited knowledge card, so the OpenClaw part of the contest story is still mostly narrative. The next MVP step is to turn Feishu-backed knowledge into an execution handoff that OpenClaw can act on in the developer's real local workspace.

## What Changes

- **BREAKING** Make OpenClaw invocation the default `lark-cue run` behavior after an internal-knowledge cue is generated.
- Add `--no-openclaw` so users can explicitly keep the current card-only behavior.
- Require OpenClaw preflight before executing the wrapped command unless `--no-openclaw` is set.
- Generate a deterministic OpenClaw task message from cwd, command, exit code, terminal output excerpt, planner output, knowledge card, action plan, and Feishu evidence citations.
- Invoke `openclaw agent --local --agent main --timeout 900 --message <task>` after rendering the knowledge card when the planner recommends retrieval.
- Route OpenClaw output to stderr so wrapped command stdout remains clean.
- Preserve the wrapped command exit code even if the post-card OpenClaw invocation fails or times out.
- Record OpenClaw invocation status in evaluation data when a cue attempts execution handoff.

## Capabilities

### New Capabilities
- `openclaw-execution-handoff`: Covers OpenClaw preflight, task generation, default invocation, output routing, timeout, and failure semantics.

### Modified Capabilities
- `terminal-knowledge-cue`: Changes the default run workflow from card-only delivery to card plus OpenClaw execution handoff, adds `--no-openclaw`, and extends evaluation logging/configuration expectations.

## Impact

- CLI: `lark-cue run` default behavior changes and gains `--no-openclaw`.
- Config: add OpenClaw binary and timeout settings while keeping the MVP invocation fixed to the local `main` agent.
- Runtime: add an OpenClaw adapter that shells out to `openclaw`.
- Card/app orchestration: render card before invoking OpenClaw and preserve stdout/stderr/exit semantics.
- Evaluation: add OpenClaw attempt/result fields to cue records and report output.
- Docs/demo: update setup and local demo instructions to include OpenClaw installation, preflight, and opt-out mode.
