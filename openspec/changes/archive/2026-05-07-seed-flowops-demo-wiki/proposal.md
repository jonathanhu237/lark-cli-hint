## Why

The FlowOps demo seed currently behaves like a few document writes rather than a reproducible internal Wiki setup. For the recorded demo and future test tenants, seeding must be idempotent, profile-safe, and able to build the same complete team Wiki structure from one command.

## What Changes

- Replace the current flat document seed model with a single seed entrypoint that creates or reuses a named team Wiki.
- Add a file-backed Wiki seed manifest plus standalone Markdown files for all seed-managed pages.
- Require an explicit seed/test Feishu profile for `--apply`, resolved from CLI `--profile` or local config.
- Make seed apply idempotent: create missing pages, update managed page content, and move managed pages to the manifest-defined parent when needed.
- Seed a stable Wiki tree with content-bearing directory pages and FlowOps troubleshooting documents.
- Keep seed side effects bounded: no IM sends, no personal-library writes, and no automatic deletion/pruning of pages outside the manifest.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `flowops-airflow-demo`: Upgrade the Feishu seed requirement from flat team-Wiki document writes to a complete, idempotent FlowOps demo Wiki seeding workflow.

## Impact

- Affected files: `examples/flowops-airflow/scripts/seed-feishu`, demo documentation, config loading for seed profile/name, OpenSpec `flowops-airflow-demo` requirements, and tests for seed dry-run/apply planning behavior.
- Affected systems: real Feishu test tenants through `lark-cli wiki spaces`, `lark-cli docs`, and Wiki node operations.
- No runtime `lark-cue run` behavior should change except where existing query/retrieval fixes are already required for the FlowOps demo to use seeded Wiki content reliably.
