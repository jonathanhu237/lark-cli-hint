## Why

The CLI runner currently produces only baseline hints, so the product does not yet demonstrate Recover or Next value in the agreed knowledge-base demo. The next step is to add deterministic docs workflow hints for the first real Feishu knowledge path: search docs/wiki, fetch document content, and prepare a group push command.

## What Changes

- Add domain-specific Hint Card analysis for successful `lark-cli docs +search` commands.
- Add domain-specific Hint Card analysis for successful `lark-cli docs +fetch` commands.
- Add domain-specific Recover analysis for failed `lark-cli docs +fetch` commands, including token/argument-shape and common setup/identity failures.
- Suggest real `lark-cli` command shapes:
  - `lark-cli docs +fetch --doc <document-url-or-token>`
  - `lark-cli im +messages-send --chat-id <oc_xxx> --markdown <message>`
- Keep suggestions deterministic and rule-backed; do not introduce YAML rule loading, LLMs, RAG, or automatic message sending.
- Update docs/guardrails that still reference outdated command forms such as `--doc-token` or `im message create`.

## Capabilities

### New Capabilities

- `docs-workflow-hints`: Produce deterministic Recover and Next hints for the docs/wiki knowledge-base workflow.

### Modified Capabilities

None.

## Impact

- Extends analyzer behavior beyond the baseline `cli-runner` fallback.
- Adds fixtures and tests for docs search, docs fetch success, and docs fetch failure cases.
- Updates product documentation and implementation guardrails to use current `lark-cli` command forms.
- Does not add a generic rule engine, persistent sessions, RAG, LLM SDKs, shell hooks, or automatic execution of suggested commands.
