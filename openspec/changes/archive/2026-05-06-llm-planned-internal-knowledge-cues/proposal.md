## Why

The current MVP is too narrowly shaped around a hardcoded Feishu API scope/token scenario, which does not match the intended product: a general internal-knowledge troubleshooting assistant for CLI-heavy enterprise workflows. We need to pivot the core loop so an LLM plans whether a failed command should trigger Feishu knowledge retrieval and which keyword queries should be searched.

## What Changes

- **BREAKING** Require LLM configuration before `lark-cue run` executes a wrapped command; without LLM config, the command is not run.
- **BREAKING** Remove the public `--demo-fixture` product path and stop presenting local fixture retrieval as a demo capability.
- Replace hardcoded scenario-triggered retrieval with an LLM planner that returns `should_retrieve`, `scenario`, `reason`, and keyword-style `queries`.
- When `should_retrieve=false`, preserve the wrapped command exit code, print a short non-intrusive message, skip Feishu retrieval, and record a planner decision event.
- When `should_retrieve=true`, send the planner's unified keyword queries to both Feishu Docs/Wiki search and IM search through `lark-cli`.
- Keep evidence-grounded card generation and citations; if retrieval yields weak or no evidence, show a transparent low-confidence card instead of inventing a cause.
- Replace the old Feishu API demo with a real local Airflow-based FlowOps demo that reproduces a DAG import error caused by parse-time `Variable.get("billing_region")`.
- Add a `scripts/seed-flowops-feishu-demo` setup script that seeds three FlowOps internal-knowledge Markdown documents into Feishu via `lark-cli`; it defaults to dry-run and requires `--apply` for writes.
- Update documentation, specs, and evaluation reporting so FlowOps and LLM-planned internal knowledge cues become the main product narrative.

## Capabilities

### New Capabilities
- `flowops-airflow-demo`: Real local FlowOps/Airflow demo environment and Feishu demo knowledge seeding.

### Modified Capabilities
- `terminal-knowledge-cue`: Replaces deterministic Feishu API scenario detection with mandatory LLM-planned internal knowledge retrieval, removes public fixture mode, and expands evaluation logging to planner decisions.

## Impact

- CLI behavior: `lark-cue run` now requires LLM config before executing wrapped commands.
- Public flags and docs: `--demo-fixture`, the Feishu API fixture demo, and related mock-data docs are removed from the main path.
- Go packages: app orchestration, LLM client, query planning, retrieval input, evidence/card fallback, eval event schema, and tests.
- Demo assets: new `examples/flowops-airflow/` Docker Compose demo, broken DAG, `flowctl` wrapper, and FlowOps demo README.
- Feishu setup: new seed script uses `lark-cli` to create/update demo Markdown documents; default dry-run protects real tenants.
- Branch: `codex/llm-planned-internal-knowledge-cues`.
