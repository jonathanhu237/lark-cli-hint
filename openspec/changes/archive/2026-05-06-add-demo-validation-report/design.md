## Context

`lark-cue` already appends evaluation records for cue attempts to a local JSONL log. Each cue record contains command, card id, scenario, retrieval status, sources, latency, query count, feedback state, and timestamp. The current demo can show one detection/card interaction, but it cannot summarize whether recent runs used real Feishu retrieval, carried citations, stayed fast enough, or received useful feedback.

The validation report is a demo-facing read path over this existing log. It should make the evaluation loop visible in the terminal without changing the cue pipeline, running commands, calling Feishu, or invoking an LLM.

## Goals / Non-Goals

**Goals:**

- Add a terminal-native validation report for recent cue records.
- Aggregate retrieval status, real-vs-fixture runs, citation coverage, feedback state, latency, and query count.
- Keep output useful in a recorded demo: concise, styled for TTY, and readable as plain text when redirected.
- Handle missing, empty, and partially malformed evaluation logs without failing the demo.
- Preserve existing cue behavior and evaluation append semantics.

**Non-Goals:**

- Do not build an automatic benchmark runner or YAML demo-case executor.
- Do not score semantic correctness with LLMs or external judges.
- Do not add new failure scenarios beyond the Feishu API scope/token MVP.
- Do not mutate existing evaluation records.
- Do not send or prepare Feishu messages from the report command.

## Decisions

1. **Report over existing JSONL, not active execution.**

   `lark-cue eval report` will read the configured evaluation log and summarize `type: "cue"` records. This keeps the feature safe and fast for demos. Automatic case execution can be a later change if we need negative-case proof in the report.

2. **Terminal report first, Markdown export out of scope for the first pass.**

   The primary consumer is a recorded terminal demo, so the first implementation should render a compact validation card to stdout. Plain text fallback should remain readable when stdout is redirected or `NO_COLOR` is set. A file export can be added later if contest submission requirements demand it.

3. **Aggregate only objective fields already recorded.**

   The report will not infer whether a card was semantically correct. It will aggregate objective fields: run count, retrieval statuses, fixture count, source count, citation coverage, feedback counts, query count, and latency. This avoids overstating product quality.

4. **Malformed records are warnings, not hard failures.**

   A JSONL log can accumulate partial writes or old records. The reader should skip malformed lines, count them as warnings, and still report valid cue records. A completely missing or empty log should produce a clear empty-state report.

5. **Recent-window controls are simple.**

   The first pass should support a small count limit such as `--limit <N>` with a sensible default, so demos can focus on recent runs without needing date filtering. Time-window filtering can come later.

## Risks / Trade-offs

- **Report can be misread as accuracy proof** -> Label metrics as operational validation and avoid words like "accuracy" unless there is human labeling.
- **Fixture-heavy demos look weaker** -> Show fixture counts explicitly so real Feishu runs are distinguishable.
- **Old logs pollute demo output** -> Support a recent record limit and document how to use `LARK_CUE_EVAL_LOG` for isolated demo logs.
- **Malformed JSONL could break the demo** -> Tolerate malformed lines and surface a warning count.
- **Styled output could hurt scripting** -> Follow existing TTY/`NO_COLOR` behavior and keep stdout plain when non-interactive.
