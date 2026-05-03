## Why

`lark-cli-hint` already turns docs workflow results into Recover and Next hints, but a common `lark-cli docs +search` failure still falls back to the generic baseline when the resolved identity is unsupported. This blocks the main docs search -> fetch -> push demo at the first step even when `lark-cli` has already provided enough evidence to retry with `--as user`.

## What Changes

- Add identity-aware Recover behavior for failed `lark-cli docs +search` commands when captured output indicates that the command only supports user identity or suggests `--as user`.
- Tighten the existing `lark-cli docs +fetch` unsupported-identity Recover behavior so it preserves the original command arguments and rewrites or inserts `--as user`.
- Generate one directly executable repair command by minimally changing the original command identity, while preserving the user's original query, document, paging, format, and other arguments.
- Keep the behavior deterministic and evidence-backed; do not introduce OAuth, token storage, permission indexing, RAG, or schema/help snapshot generation in this change.
- Record schema/help snapshots as a future direction for broader command coverage, not as an implementation requirement for this change.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `docs-workflow-hints`: Extend docs workflow Recover hints to handle unsupported identity failures for `docs +search` and strengthen identity repair behavior for `docs +fetch`.

## Impact

- Affected analyzer code: docs workflow failure classification and command-rewrite utilities.
- Affected locale files: English and Simplified Chinese identity Recover copy.
- Affected fixtures/tests: add docs search identity failure fixtures and tests for preserving arguments, replacing existing `--as bot`, terminal Hint Card rendering, JSON envelope behavior, and i18n.
- No dependency, CLI entry, output protocol, OAuth, remote backend, or RAG changes.
