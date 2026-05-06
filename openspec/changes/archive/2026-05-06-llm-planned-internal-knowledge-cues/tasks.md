## 1. LLM Planner Contract

- [x] 1.1 Add planner data types for `should_retrieve`, `scenario`, `reason`, `queries`, and planner latency/status.
- [x] 1.2 Add an LLM provider planner method that returns structured JSON and prompts for keyword-style Feishu search phrases rather than natural-language questions.
- [x] 1.3 Add planner output normalization: trim, drop empty values, de-duplicate case-insensitively, and cap at eight queries.
- [x] 1.4 Add unit tests for planner JSON parsing, malformed output handling, query normalization, and `should_retrieve=false`.

## 2. Run Command Refactor

- [x] 2.1 Require LLM configuration before `lark-cue run` executes the wrapped command.
- [x] 2.2 Remove public `--demo-fixture` parsing, help text, and runtime branch.
- [x] 2.3 Replace detector-gated retrieval with the planner decision flow after non-zero command exit.
- [x] 2.4 Implement `should_retrieve=false` behavior: short stderr message, no `lark-cli` retrieval, no card, planner eval record, original exit code preserved.
- [x] 2.5 Implement `should_retrieve=true` behavior: render planning/search status, retrieve with planner queries, generate card, prompt feedback as before, original exit code preserved.
- [x] 2.6 Keep successful wrapped commands unchanged: no planner, no retrieval, no card.
- [x] 2.7 Add app-level tests proving missing LLM blocks command execution and failed commands use planner decisions.

## 3. Retrieval, Evidence, and Card Generalization

- [x] 3.1 Remove Feishu API-specific query seed dependency from the primary retrieval path.
- [x] 3.2 Ensure each planner query searches both `docs +search` and `im +messages-search`.
- [x] 3.3 Generalize evidence scoring so fetched snippets are evaluated against planner scenario/query/error context instead of only `docx:document:read` signals.
- [x] 3.4 Generalize card scenario, likely-cause fallback, next-action fallback, confidence text, and retrieval caveats away from Feishu API permission wording.
- [x] 3.5 Ensure weak or absent evidence still renders a transparent low-confidence card without inventing a cause.
- [x] 3.6 Add unit tests for generic FlowOps evidence, unrelated evidence filtering, and no-evidence low-confidence cards.

## 4. Evaluation Logging and Report

- [x] 4.1 Extend evaluation records to support planner decision events with command, scenario, reason, `should_retrieve`, query count, latency, and timestamp.
- [x] 4.2 Record cue confidence and planner-related fields needed for validation.
- [x] 4.3 Update `lark-cue eval report` to summarize planner decisions and retrieve-vs-skip counts.
- [x] 4.4 Preserve feedback updates for generated cue cards.
- [x] 4.5 Add tests for planner-only records, mixed planner/cue logs, malformed lines, and report output.

## 5. FlowOps Airflow Demo

- [x] 5.1 Add `examples/flowops-airflow/` with a local Airflow Docker Compose demo suitable for one-time initialization and repeated demo runs.
- [x] 5.2 Add a broken `billing_daily` DAG that triggers import failure from parse-time `Variable.get("billing_region")`.
- [x] 5.3 Add a `flowctl` wrapper that invokes the real local Airflow CLI and does not print hardcoded fake errors.
- [x] 5.4 Add demo README instructions for setup, expected failure output, `lark-cue run`, cleanup, and recording tips.
- [x] 5.5 Add `.gitignore` coverage for Airflow logs, generated config, local DB/state, and temporary files.

## 6. Feishu Seed Script

- [x] 6.1 Add `scripts/seed-flowops-feishu-demo` with default dry-run behavior.
- [x] 6.2 Implement `--apply` mode that uses `lark-cli` to create or update three FlowOps Markdown documents.
- [x] 6.3 Write seeded content as 星桥科技 internal FlowOps mock knowledge without external source-reference sections.
- [x] 6.4 Ensure the script does not send IM messages or delete real resources.
- [x] 6.5 Print smoke search commands for FlowOps keywords after dry-run and apply.
- [x] 6.6 Add documentation for using a test Feishu profile with the seed script.

## 7. Documentation and Product Narrative

- [x] 7.1 Rewrite `docs/demo.md` around the FlowOps/Airflow demo and real Feishu seed path.
- [x] 7.2 Remove old Feishu API fixture demo documentation and mock data files from the main product path.
- [x] 7.3 Update `docs/product-brief.md` and `AGENTS.md` so the main story is LLM-planned internal knowledge cues rather than Feishu API scope/token detection.
- [x] 7.4 Update README/help text to explain required LLM config, real `lark-cli` retrieval, and FlowOps demo usage.
- [x] 7.5 Update OpenSpec main spec purpose text if needed so archive reflects the new product contract.

## 8. E2E and Verification

- [x] 8.1 Add opt-in E2E test scaffolding gated by `LARK_CUE_E2E=1` and explicit LLM/lark-cli profile configuration.
- [x] 8.2 Ensure default `go test ./...` does not require Docker, network, LLM credentials, or Feishu login state.
- [x] 8.3 Run `go test ./...`.
- [x] 8.4 Run `openspec validate llm-planned-internal-knowledge-cues --strict`.
- [x] 8.5 Run a local FlowOps demo smoke check when Docker is available.
- [x] 8.6 Run real Feishu E2E only when the test profile and seeded data are available; otherwise document the skipped external verification.

Verification note: Docker Compose was available, so `examples/flowops-airflow/flowctl init` and `examples/flowops-airflow/flowctl check billing_daily` were run; the check reproduced the expected `Variable billing_region does not exist` DAG import failure. Real Feishu/LLM E2E was skipped because `LARK_CUE_E2E`, `LARK_CUE_LLM_API_KEY`, `LARK_CUE_LLM_MODEL`, and `LARK_CUE_FEISHU_PROFILE` were not set in this local environment.
