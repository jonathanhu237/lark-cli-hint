## 1. CLI Surface

- [x] 1.1 Add `lark-cue eval report` routing and help text without changing existing `feedback` behavior.
- [x] 1.2 Add `--limit <N>` parsing with a sensible default and clear validation errors for invalid limits.
- [x] 1.3 Ensure the report command is read-only and does not invoke command execution, `lark-cli`, push sending, or LLM providers.

## 2. Evaluation Reading and Aggregation

- [x] 2.1 Implement evaluation JSONL reading that returns cue records, skips non-cue records, handles missing files as empty, and counts malformed lines.
- [x] 2.2 Implement recent-record limiting over valid cue records after reading the log.
- [x] 2.3 Aggregate run count, retrieval status counts, fixture count, source count, citation coverage, average query count, average latency, feedback counts, and warning count.
- [x] 2.4 Add unit tests for empty logs, malformed lines, feedback update records, mixed retrieval statuses, and limit behavior.

## 3. Report Rendering

- [x] 3.1 Render a terminal validation summary card for interactive stdout using the existing styled-output conventions.
- [x] 3.2 Render a readable plain-text report for non-TTY stdout, `NO_COLOR`, or style-disabled terminals.
- [x] 3.3 Include clear empty-state output when no cue records exist.
- [x] 3.4 Add CLI tests covering styled/plain output selection and stdout-only report behavior.

## 4. Demo Documentation and Verification

- [x] 4.1 Update demo documentation to show a recorded-demo flow: run cue, provide feedback, then run `lark-cue eval report`.
- [x] 4.2 Document how to use `LARK_CUE_EVAL_LOG` for isolated demo logs.
- [x] 4.3 Update the main spec purpose text if needed so the archived spec no longer reads as TBD.
- [x] 4.4 Run `go test ./...` and `openspec validate add-demo-validation-report --strict`.
