## Context

`lark-cue` already records planner and cue events, and the FlowOps demo now has a realistic local command failure plus seeded Feishu Wiki knowledge. The missing piece is an active benchmark that runs real commands and checks whether the final cited knowledge card recalls the sources expected for each demo failure.

The benchmark must stay explicit because it can trigger Docker/Airflow through `flowctl`, LLM calls, Feishu retrieval through `lark-cli`, and local evaluation writes. It should reuse the same `lark-cue run` behavior rather than creating a separate retrieval path.

## Goals / Non-Goals

**Goals:**

- Add a reusable benchmark runner command: `lark-cue benchmark run --cases <path>`.
- Execute real commands from case definitions, including optional setup and teardown commands.
- Isolate benchmark evaluation records in a temporary log for each benchmark run.
- Score cases by exact source-title matches between card citations and declared expected sources.
- Run all cases before reporting, with clear exit codes for pass/fail/config errors.
- Add FlowOps benchmark cases beside the FlowOps demo seed data.

**Non-Goals:**

- Do not support output fixtures or fake command outputs in benchmark cases.
- Do not add per-app deterministic detection rules for benchmark scoring.
- Do not mutate the normal user evaluation log during benchmark runs.
- Do not make benchmark runs part of default unit tests.
- Do not implement fuzzy source matching in the first version.

## Decisions

1. **Add a top-level `benchmark run` command.**

   The runner belongs in the main CLI because benchmark execution can be reused by future internal app demos that provide their own cases files. Keeping it under `lark-cue` also makes the benchmark report feel like part of the evaluation story rather than a FlowOps-only helper.

2. **Use JSON case files with real command arrays.**

   Each case will declare `id`, `command`, optional `setup`, optional `teardown`, `expect_failure`, `expected_sources`, and `min_expected_hits`. Command arrays avoid shell quoting ambiguity. Setup and teardown commands are also arrays and run directly.

3. **Invoke the existing run pipeline with an isolated eval log.**

   Benchmark cases should execute through the same behavior as `lark-cue run -- <command>`, including LLM planning, Feishu retrieval, card construction, and original command exit preservation. The benchmark runner will set `LARK_CUE_EVAL_LOG` to a temporary path while running cases, then read only that log for scoring.

4. **Score from final citations, not search results.**

   A case passes only when the final knowledge card cites at least `min_expected_hits` sources whose citation titles exactly match `expected_sources`. This measures the user-visible result and avoids inflated scores from raw retrieval candidates that were not selected or cited.

5. **Run all cases and summarize.**

   The runner should not fail fast. It should complete every case, then report pass/fail count, expected-source hit rate, source coverage, citation precision, average latency, and per-case details. Exit code `0` means all cases passed, `1` means at least one case failed, and `2` means cases file or runner configuration is invalid.

6. **Keep output compact by default.**

   Default benchmark output should show summaries and per-case cited/expected source titles, not full Airflow tracebacks. `--verbose` can show or retain more command output details when debugging.

## Risks / Trade-offs

- **[Risk] Benchmark runs are slow or flaky because they depend on Docker, LLM, Feishu, and local network state.** → Mitigation: make the command explicit, document prerequisites, and keep default unit tests deterministic.
- **[Risk] Exact title matching can fail after benign document title edits.** → Mitigation: keep expected titles in `eval-cases.json` beside the seed manifest and update them together with seed content.
- **[Risk] Running setup repeatedly can be expensive.** → Mitigation: support setup but keep FlowOps initial setup lightweight with `flowctl init`; users can run `flowctl clean` manually when they need a full reset.
- **[Risk] Benchmark output can become noisy.** → Mitigation: summarize by default and reserve full command output for verbose/debug paths.
