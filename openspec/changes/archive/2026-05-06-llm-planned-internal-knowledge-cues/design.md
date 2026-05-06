## Context

`lark-cue` currently behaves like a specialized Feishu API scope/token helper: a deterministic detector decides whether to trigger retrieval, query generation starts from Feishu API-specific seeds, fixture mode is exposed as a public demo path, and deterministic fallback card text is tied to `docx:document:read`. That path no longer matches the product direction.

The new product contract is broader and stricter: `lark-cue` is an LLM-planned internal knowledge cue assistant. A wrapped command failure is useful only when an LLM can judge whether enterprise knowledge is relevant, produce short keyword-style Feishu search queries, and let the rest of the pipeline retrieve real Feishu evidence through `lark-cli`.

## Goals / Non-Goals

**Goals:**

- Require LLM configuration before `lark-cue run` executes any wrapped command.
- Replace hardcoded scenario detection as the primary trigger with an LLM planner.
- Use planner output `{should_retrieve, scenario, reason, queries}` to decide retrieval and card flow.
- Treat planner queries as keyword phrases for Feishu search; only trim, de-duplicate, drop empty values, and cap the list.
- Search the same planner queries across Docs/Wiki/Sheets and IM routes through `lark-cli`.
- Keep card claims evidence-grounded and show a transparent low-confidence card when evidence is weak or absent.
- Remove public fixture mode and the old Feishu API demo from the product path.
- Add a real FlowOps demo based on local Airflow Docker Compose and a broken `billing_daily` DAG.
- Add a dry-run-first Feishu seed script for FlowOps internal knowledge documents.

**Non-Goals:**

- No semantic/vector search in this change.
- No per-application hardcoded detector rules for FlowOps.
- No route-specific query generation for Docs versus IM.
- No automatic repair command execution.
- No automatic posting of mock IM messages into real groups.
- No production-grade Airflow deployment.

## Decisions

### Mandatory LLM Before Command Execution

`lark-cue run` will validate LLM config before executing the wrapped command. If `LARK_CUE_LLM_API_KEY` and `LARK_CUE_LLM_MODEL` or equivalent config file values are missing, it returns a CLI usage/config error and does not run the command.

Rationale: the user explicitly wants LLM to be required. Running a potentially side-effectful command and only then discovering that analysis cannot happen would be misleading.

Alternative considered: execute first and only fail analysis after command failure. Rejected because it weakens the "must configure LLM" contract and can still perform side effects.

### LLM Planner Replaces Scenario Detector as Main Trigger

Add a planner method to the LLM provider. It receives the wrapped command, exit code, and bounded output excerpt, then returns JSON:

```json
{
  "should_retrieve": true,
  "scenario": "FlowOps DAG import error",
  "reason": "The failure mentions a DAG import error and billing_region variable lookup.",
  "queries": [
    "FlowOps DAG import error billing_daily",
    "billing_daily billing_region Variable.get"
  ]
}
```

If `should_retrieve=false`, `lark-cue` prints a short message, records a planner decision event, skips Feishu retrieval, and returns the wrapped command exit code.

Rationale: this generalizes to new internal applications without writing one detector per app.

Alternative considered: generic deterministic gate plus per-application app profiles. Deferred because the user wants a simpler MVP and will add query filtering or app-specific profiles only after evidence of need. A plain `lark-cli` runtime profile is still supported so the seeded test tenant and `lark-cue run` retrieval use the same Feishu identity.

### Minimal Query Cleanup Only

Planner queries are keyword-style Feishu search phrases, not natural-language questions. The program will trim whitespace, remove empty values, de-duplicate case-insensitively, and cap at eight queries. It will not enforce anchor-token filtering in this change.

Rationale: `lark-cli docs +search` and `im +messages-search` expose keyword query inputs, and the MVP should first validate whether prompt discipline is enough before adding filtering complexity.

### Unified Retrieval Routes

For every planner query, retrieval will search:

- Docs/Wiki/Sheets route via `lark-cli docs +search`
- IM route via `lark-cli im +messages-search`

Docs and Wiki remain one route because `lark-cli docs +search` uses the unified doc/wiki search surface. IM remains a separate route because chat history has different noise and metadata characteristics.

### Evidence-Grounded Card Remains the Safety Boundary

LLM can plan retrieval and draft card text, but final card claims must be grounded in fetched/read snippets. If no reliable evidence is found, the card must say that internal evidence is insufficient, list the query/retrieval context compactly, and avoid a definitive cause.

Rationale: the planner may correctly infer that internal knowledge is relevant while Feishu retrieval fails or the knowledge base lacks coverage.

### Fixture Mode Removed From Product

The public `--demo-fixture` flag and fixture retriever should be removed from runtime UX and docs. Tests may still use mocks/fakes at the dependency boundary, but product/demo execution must use real LLM and real `lark-cli`.

### FlowOps Demo Is Real Airflow, Internally Framed

Add `examples/flowops-airflow/` containing a lightweight local Airflow Docker Compose demo, a `flowctl` wrapper, and a broken `billing_daily` DAG that calls `Variable.get("billing_region")` at parse time. The user-facing story is FlowOps, 星桥科技's internal scheduling platform based on Airflow.

The Feishu seed script creates or updates three internal Markdown documents:

- `[lark-cue-demo] FlowOps DAG Import Error 排障 FAQ`
- `[lark-cue-demo] billing_daily 历史故障复盘`
- `[lark-cue-demo] FlowOps DAG 开发规范`

Seeded documents must be written as internal mock knowledge and must not include external Airflow source-reference sections.

## Risks / Trade-offs

- [Risk] LLM planner may over-trigger retrieval. → Mitigation: prompt clearly distinguishes enterprise-knowledge failures from local errors; evaluation logs record planner decisions for later tuning.
- [Risk] LLM may generate broad keyword queries. → Mitigation: cap query count and keep query output inspectable; defer stricter filters until observed failures justify them.
- [Risk] Feishu search indexing may lag after seeding documents. → Mitigation: seed script prints smoke search commands and demo docs mention waiting/retrying before recording.
- [Risk] Airflow Docker setup may be slow on first run. → Mitigation: demo README separates one-time initialization from the recorded demo command.
- [Risk] Removing fixture mode reduces offline demo recovery. → Mitigation: default unit tests use mocks; real E2E uses explicit `LARK_CUE_E2E=1` with a test Feishu profile.
- [Risk] Wrapped commands see stdout/stderr as pipes while output is captured. → Mitigation: document the limitation, preserve streamed bytes/stdin/exit codes, and use non-interactive CLI checks for the MVP demo; full TTY-preserving capture is deferred to a future terminal-runtime hardening change.

## Migration Plan

1. Add planner types and prompt to the LLM package.
2. Refactor `runCommand` to require LLM config before command execution and use planner output after failures.
3. Remove public fixture flag, fixture docs, and old Feishu API demo assets.
4. Generalize evidence/card fallback away from Feishu API permission-specific wording.
5. Extend evaluation logging/reporting for planner decisions and low/no-evidence cue outcomes.
6. Add FlowOps Airflow demo assets and Feishu seed script.
7. Update docs, product brief, and OpenSpec main spec narrative.
8. Verify with default unit tests plus opt-in real Feishu/Airflow E2E.

Rollback is a normal git revert of this change before archive; after archive, rollback should restore the previous archived spec and implementation commit if needed.

## Open Questions

None for this proposal. Query filtering, route-specific queries, and semantic retrieval are intentionally deferred.
