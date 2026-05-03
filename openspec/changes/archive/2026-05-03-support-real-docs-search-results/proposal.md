## Why

Real `lark-cli docs +search` output from the current CLI returns document candidates under `results[].result_meta.url` or `results[].result_meta.token`. `lark-cli-hint` currently recognizes only simpler fixture shapes, so a successful real search can fall back to the baseline hint instead of suggesting `docs +fetch`.

## What Changes

- Add support for real `docs +search` result candidates nested under `results[].result_meta`.
- Prefer a document URL when available, and fall back to the result token when no URL is present.
- Extract a useful title from real search output, including highlighted title fields, without requiring the title to be perfect.
- Add a fixture based on the real `docs +search` response shape observed during manual testing.
- Keep scope limited to candidate extraction for `docs +search`; do not change CLI entry behavior, identity recovery, fetch behavior, RAG, OAuth, or schema ingestion.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `docs-workflow-hints`: Extend the existing docs search success requirement to include real `lark-cli docs +search` result shapes with nested `result_meta` document fields.

## Impact

- Affected analyzer code: document candidate extraction in the docs workflow analyzer.
- Affected fixtures/tests: add real search output fixture and regression tests proving `results[].result_meta.url/token` produces a `docs +fetch --doc` Next command.
- No dependency, output protocol, OpenClaw, authentication, backend, or RAG changes.
