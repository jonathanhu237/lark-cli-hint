## Why

The FlowOps demo can show one successful knowledge card, but it still lacks a repeatable way to prove that seeded internal knowledge is actually recalled for expected real command failures. A benchmark runner gives the demo a concrete effectiveness signal: real failures should retrieve and cite the sources that the seed data declares as relevant.

## What Changes

- Add a `lark-cue benchmark run --cases <path>` command that actively runs real benchmark cases instead of reading only existing logs.
- Introduce a benchmark case JSON schema with real command, optional setup/teardown commands, `expect_failure`, `expected_sources`, and `min_expected_hits`.
- Run benchmark cases through the same `lark-cue run -- <command>` behavior, using an isolated temporary evaluation log so normal demo reports are not polluted.
- Score each case by exact citation-title matches against expected sources, running all cases before summarizing failures.
- Render a compact benchmark report with pass/fail counts, expected-source hit rate, source coverage, citation precision, average latency, and per-case details.
- Add FlowOps benchmark metadata under `examples/flowops-airflow/seed/eval-cases.json`.

## Capabilities

### New Capabilities
- `benchmark-runner`: Defines the `lark-cue benchmark run` command, benchmark case schema, real-command execution, scoring, reporting, and exit behavior.

### Modified Capabilities
- `flowops-airflow-demo`: Adds benchmark case metadata for the FlowOps seeded Wiki and documents how to run the benchmark against the real local FlowOps/Airflow demo.

## Impact

- Affected code: CLI routing in `internal/app`, new benchmark package or module, evaluation/log handling integration, tests, and documentation.
- Affected demo assets: `examples/flowops-airflow/seed/eval-cases.json` and FlowOps demo docs.
- External systems: benchmark runs can invoke Docker/Airflow through `flowctl`, call the configured LLM, and retrieve Feishu data through `lark-cli`; it must be explicit and user-triggered.
- No new third-party runtime dependency is expected.
