## 1. CLI Foundation

- [x] 1.1 Initialize Go module, CLI entrypoint, command routing, and baseline help output for `lark-cue`.
- [x] 1.2 Implement `lark-cue run -- <command>` argument parsing with clear errors for missing commands.
- [x] 1.3 Execute wrapped commands with real-time stdout/stderr passthrough and bounded output capture.
- [x] 1.4 Preserve wrapped command exit codes and add tests for success, failure, and non-TTY behavior.

## 2. Scenario Detection and Query Generation

- [x] 2.1 Implement Feishu API auth/scope/token detector for `missing required scope`, `docx:document:read`, `tenant_access_token invalid`, and `permission denied`.
- [x] 2.2 Ensure detector only runs after non-zero command exit and ignores unrelated command failures.
- [x] 2.3 Implement deterministic seed query extraction with deduplication, count limits, and length limits.
- [x] 2.4 Add thin LLM provider interface for query expansion and card generation with OpenAI-compatible environment/config loading.
- [x] 2.5 Implement constrained LLM query expansion using only command and captured output, with deterministic seed fallback when LLM is unavailable.

## 3. Feishu Retrieval

- [x] 3.1 Add `lark-cli` command adapter for JSON execution, error capture, timeout handling, and diagnostics.
- [x] 3.2 Implement Docs/Wiki search through `lark-cli` using generated queries.
- [x] 3.3 Implement IM message search through `lark-cli` using generated queries.
- [x] 3.4 Implement fetch/read handling for candidate Docs/Wiki and IM results before they can become evidence.
- [x] 3.5 Ensure primary runtime retrieval does not use hardcoded known document titles, URLs, or chat IDs.
- [x] 3.6 Add explicit `--demo-fixture` retrieval mode that labels fixture output and never activates silently.

## 4. Evidence and Card Generation

- [x] 4.1 Implement keyword evidence scoring for strong error signals, cause/action signals, and scenario signals.
- [x] 4.2 Filter unrelated or unfetched results from cited evidence.
- [x] 4.3 Implement confidence classification for strong, weak, and missing evidence.
- [x] 4.4 Implement LLM-backed knowledge card generation grounded only in fetched evidence snippets and source metadata.
- [x] 4.5 Implement deterministic template fallback for missing LLM, invalid LLM output, weak evidence, and retrieval failures.
- [x] 4.6 Render compact terminal cards with detected scenario, likely cause or caveat, next action, citations, and confidence.

## 5. Push, Feedback, and Evaluation

- [x] 5.1 Implement group push message preparation as a preview-only default.
- [x] 5.2 Implement explicit send flag support for Feishu group push through `lark-cli`.
- [x] 5.3 Implement local configuration loading from `~/.lark-cue/config.toml` with `LARK_CUE_*` environment variable precedence.
- [x] 5.4 Implement evaluation JSONL logging with card id, command, scenario, retrieval status, sources, latency, and feedback state.
- [x] 5.5 Implement interactive useful/not useful/skip feedback for TTY sessions and non-blocking skip behavior for non-TTY sessions.
- [x] 5.6 Implement `lark-cue feedback <card-id> useful|not-useful` to update or append evaluation feedback state.

## 6. Demo and Verification

- [x] 6.1 Add `examples/failing-feishu-api.js` that emits the MVP Feishu API permission/scope/token failure.
- [x] 6.2 Add tests covering detector, query generation, fixture labeling, evidence scoring, card fallback, and feedback logging.
- [x] 6.3 Add an integration smoke path that can run with real `lark-cli` when credentials are available and skip cleanly otherwise.
- [x] 6.4 Document demo prerequisites, LLM configuration, `lark-cli` health check, and the expected real Feishu mock-data flow.
- [x] 6.5 Run `openspec status --change build-feishu-scope-error-cue-mvp` and fix any artifact issues before implementation begins.
