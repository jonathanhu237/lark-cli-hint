## Context

`lark-cue` is an active enterprise knowledge assistant for terminal-heavy developers. The MVP focuses on one strong demo moment: a developer command fails with a Feishu API permission/scope/token error, and the CLI retrieves real Feishu-hosted mock enterprise knowledge through `lark-cli` before presenting a short cited knowledge card.

The repository currently contains product documentation and local mock-data notes, but no implementation. The local notes verify that the Starbridge/星桥 mock knowledge exists in real Feishu Docs/Wiki and IM, and that `lark-cli` can search/fetch/read those sources. Those notes are design/test context only; runtime retrieval must not pin known document titles, URLs, or chat IDs as the answer path.

## Goals / Non-Goals

**Goals:**

- Build a Go CLI MVP for `lark-cue run -- <command>`.
- Stream wrapped command output in real time while capturing a bounded buffer for analysis.
- Trigger analysis only after a non-zero command exit and a Feishu API auth/scope/token error match.
- Use real `lark-cli` search/read/fetch commands for Docs/Wiki and IM messages.
- Use deterministic seed query extraction plus constrained LLM query expansion to improve recall.
- Score fetched evidence with transparent keyword rules before generating any conclusion.
- Generate a compact knowledge card grounded only in fetched evidence.
- Support explicit group push preparation/sending and feedback/evaluation logging.
- Keep OpenClaw integration replaceable through a thin provider boundary.

**Non-Goals:**

- No generic terminal assistant behavior.
- No broad enterprise indexing or vector database in the MVP.
- No pinned runtime source list for the demo answer.
- No automatic repair command execution.
- No silent fixture fallback.
- No GUI terminal or long-form chatbot interface.

## Decisions

### Use Go for the CLI

`lark-cue` is a terminal wrapper whose core behavior is process execution, stdout/stderr streaming, signal handling, external command invocation, local file logging, and stable binary distribution. Go fits that shape better than Node.js or Python for the long-term CLI product.

Alternatives considered:

- Node.js/TypeScript: natural for `pnpm`/Node demo projects, but weaker for single-binary distribution and process/signal robustness.
- Python: fast to script, but less compelling as a polished developer CLI dependency.

### Analyze only after command failure

The MVP SHALL trigger only when the wrapped command exits with a non-zero code. Runtime output is streamed immediately, but knowledge retrieval starts after process completion.

This avoids interrupting long-running commands, duplicate live detections, and transient warning noise. A future `--live-watch` mode can add pattern-triggered analysis while the process is still running.

### Preserve command semantics

The wrapper SHALL return the original command exit code. `lark-cue` is an advisory layer; it must not make a failed command appear successful or break shell/CI semantics.

### Generate queries with deterministic seeds plus LLM expansion

Query generation uses two layers:

1. Deterministic extraction from command output, such as scope tokens, known Feishu token phrases, and permission error phrases.
2. Constrained LLM query expansion using only the command and captured output. The LLM may generate short additional Chinese/English search queries but must not receive mock-data titles, known source URLs, or final-answer background.

Seed queries are always retained. LLM queries improve recall but do not decide the answer.

### Search real Feishu sources at runtime

The primary path invokes `lark-cli` to search Docs/Wiki and IM messages. Runtime retrieval must not start from the known local mock document titles, URLs, or chat ID. The system can use the local mock-data file for tests, demo notes, and manual troubleshooting, but not as a hidden answer map.

Docs/Wiki provide official or semi-official troubleshooting knowledge. IM messages provide historical troubleshooting discussion. Minutes, tasks, and mail are deferred until a later scenario needs them.

### Use keyword evidence scoring before LLM card generation

The first evidence gate is transparent keyword scoring, not vector search or a reranker. Fetched documents and IM snippets are scored against:

- strong error signals, such as `docx:document:read`, `missing required scope`, `tenant_access_token invalid`, and `permission denied`
- cause/action signals, such as permission changes, re-authorization, old tokens, scope, authorization, and their Chinese equivalents
- scenario signals, such as Atlas, document summary, Feishu API, and Open Platform

Only fetched/read content can become evidence. Search result titles alone are insufficient for a confident card.

### Keep the LLM grounded

The LLM participates in two places:

- query expansion
- final card compression

For card generation, the LLM receives only the command, captured output, fetched evidence snippets, source metadata, and card format contract. It must not cite sources that were not fetched/read, and the system must fall back to a low-confidence or template card if evidence is weak or LLM output is invalid.

### Make Feishu group push explicit

Terminal delivery is required. Group push is optional:

- default: prepare a group message preview only
- explicit flag: send the prepared message

This keeps side effects visible and avoids silently posting to a Feishu group during development or tests.

### Record evaluation locally

Each generated card creates an evaluation event in local JSONL state, including command, scenario, query count, evidence source metadata, latency, card id, and feedback state. TTY sessions can prompt for useful/not useful/skip. Non-TTY sessions must not block and can be updated later through a feedback command.

### Provide explicit fixture mode only

The main path depends on real `lark-cli` access. If `lark-cli` fails, the system reports retrieval/auth/network failure and does not pretend to have real evidence.

An explicit `--demo-fixture` mode may use local fixture data for tests or demo recovery. Fixture cards must be labeled as fixture/simulated content.

### Keep OpenClaw behind a provider boundary

The MVP can use an OpenAI-compatible LLM provider for query expansion and card compression. The design keeps this behind a thin provider interface so OpenClaw agent/runtime integration can replace or wrap the provider later without rewriting command wrapping, retrieval, evidence scoring, or logging.

## Risks / Trade-offs

- Feishu search may miss colon-delimited scope tokens → Use seed query fan-out plus LLM expansion and multiple source types.
- LLM may generate poor or over-specific queries → Retain deterministic seed queries, limit query count/length, and require fetched evidence scoring before conclusions.
- Search results may include unrelated documents → Score fetched content, require evidence thresholds, and output low-confidence cards when support is weak.
- Real `lark-cli` auth or network may fail during demo → Surface the failure clearly and provide explicit labeled `--demo-fixture` recovery.
- IM evidence may expose too much chat content → Cite group name, speaker, timestamp, and short summary only; do not print long raw chat logs.
- Feedback prompts may break automation → Prompt only in interactive TTY sessions and provide a non-blocking feedback command.
- Go implementation adds provider/JSON/config plumbing upfront → The CLI product benefits from robust process behavior and binary distribution.

## Migration Plan

No existing application behavior needs migration. This change introduces the first implementation path for `lark-cue`.

Rollout sequence:

1. Add the Go CLI skeleton and `run` command.
2. Add deterministic detector/query/evidence/card components with tests.
3. Integrate real `lark-cli` retrieval behind command adapters.
4. Add LLM provider support and template fallback.
5. Add evaluation logging, feedback update command, and optional push preparation/sending.
6. Add the demo failing command and documentation for real Feishu mock-data prerequisites.

Rollback is to remove or disable the new CLI binary; no shared service or data migration is required.

## Open Questions

- Which OpenAI-compatible endpoint/model will be used for the first public demo?
- What exact `lark-cli` IM search response fields are stable enough for sender/time citation formatting?
- Should the first implementation include a health-check command for `lark-cli` and LLM configuration, or rely on error reporting from `run`?
