## Why

Developers working in terminal-heavy Feishu integration workflows lose time when API permission, scope, or token errors force them to leave the terminal and manually search scattered enterprise knowledge. The MVP needs to prove that `lark-cue` can actively recognize one concrete failure moment, retrieve real Feishu-hosted knowledge through `lark-cli`, and return a short cited knowledge card inside the CLI workflow.

## What Changes

- Add an MVP CLI flow for `lark-cue run -- <command>` that wraps a developer command, streams its output, preserves its exit code, and analyzes captured output only after a non-zero exit.
- Detect Feishu API auth/scope/token errors from terminal output, including the demo pattern `missing required scope: docx:document:read`, `tenant_access_token invalid`, and `permission denied`.
- Generate search queries from the command and error output using deterministic seed extraction plus constrained LLM query expansion.
- Search real Feishu-hosted mock enterprise knowledge through `lark-cli`, covering Docs/Wiki and IM messages, without pinning specific document titles, URLs, or chat IDs at runtime.
- Fetch/read candidate results, score evidence with transparent keyword-based rules, and filter unrelated results before answer generation.
- Generate a short knowledge card grounded only in fetched evidence, with source citations, confidence, and a safe low-confidence response when evidence is weak or missing.
- Provide terminal delivery as the required output path and support optional Feishu group push preparation, with actual sending guarded by an explicit user/demo flag.
- Record evaluation events and collect useful/not useful feedback through an interactive prompt when appropriate and a feedback command for later updates.
- Provide an explicit `--demo-fixture` mode for local fallback testing or demo recovery; fixture mode must be labeled and must never silently impersonate real Feishu retrieval.

## Capabilities

### New Capabilities
- `terminal-knowledge-cue`: Covers the active terminal failure trigger, real Feishu retrieval, evidence-grounded knowledge card generation, optional group push preparation, and feedback/evaluation logging for the MVP.

### Modified Capabilities

None.

## Impact

- Adds a Go-based `lark-cue` CLI entrypoint and related packages for command wrapping, detection, retrieval, evidence scoring, LLM provider integration, rendering, push preparation, configuration, and evaluation logging.
- Depends on a locally authenticated `lark-cli` profile for the primary MVP path and OpenAI-compatible environment/config values for optional LLM query expansion and card compression.
- Uses real Feishu-hosted mock data for the main demo path and local fixtures only when explicitly requested.
- Introduces local user state under `~/.lark-cue/`, including configuration and evaluation JSONL logs.
