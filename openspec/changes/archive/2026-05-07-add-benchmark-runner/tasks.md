## 1. Benchmark Case Model

- [x] 1.1 Add benchmark case structs and JSON loading for `id`, `command`, optional `setup`, optional `teardown`, `expect_failure`, `expected_sources`, and `min_expected_hits`.
- [x] 1.2 Validate cases before execution: required `--cases`, readable JSON, unique non-empty ids, non-empty command arrays, non-empty expected sources, and valid `min_expected_hits`.
- [x] 1.3 Add unit tests for missing cases path, malformed JSON, invalid entries, and valid case loading.

## 2. Runner Integration

- [x] 2.1 Add `lark-cue benchmark run --cases <path> [--verbose]` routing, help text, and exit-code handling.
- [x] 2.2 Execute case setup commands in order before the case command and teardown commands after each case attempt.
- [x] 2.3 Run each case command through the existing `lark-cue run -- <command>` behavior while using an isolated temporary evaluation log.
- [x] 2.4 Ensure benchmark runs all cases even when one case fails, while returning `0` only when every case passes, `1` for case failures, and `2` for config/runner errors.

## 3. Scoring And Report

- [x] 3.1 Read benchmark planner/cue records from the isolated evaluation log and associate them with the executed case.
- [x] 3.2 Score exact citation title matches against `expected_sources` and enforce `min_expected_hits`.
- [x] 3.3 Enforce `expect_failure: true` by failing a case when the wrapped command exits `0`.
- [x] 3.4 Compute aggregate metrics: pass count, expected-source hit rate, source coverage, citation precision, and average latency.
- [x] 3.5 Render compact plain/styled benchmark output with aggregate metrics and per-case status, expected/cited titles, planner retrieve status, and query count.
- [x] 3.6 Add tests for pass, fail, missing cue record, expected-failure mismatch, run-all behavior, and report output.

## 4. FlowOps Benchmark Assets

- [x] 4.1 Add `examples/flowops-airflow/seed/eval-cases.json` with a real `flowctl check billing_daily` case and expected seeded Wiki source titles.
- [x] 4.2 Include lightweight setup for the FlowOps case using `flowctl init` and document that users can run `flowctl clean` manually for a full reset.
- [x] 4.3 Update README, demo docs, and FlowOps example docs with `lark-cue benchmark run --cases examples/flowops-airflow/seed/eval-cases.json`.

## 5. Verification

- [x] 5.1 Run `go test ./...`.
- [x] 5.2 Run `openspec validate --all --strict`.
- [x] 5.3 Run benchmark loader/report tests without requiring Docker, LLM credentials, Feishu login, or network access.
- [x] 5.4 Optionally run the real FlowOps benchmark against the configured local/Feishu environment and record the result or clear skip reason.
