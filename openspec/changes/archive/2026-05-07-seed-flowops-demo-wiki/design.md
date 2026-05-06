## Context

The current FlowOps seed script is a Bash script with Markdown content embedded inline. It can write Lark documents into a caller-supplied Wiki space, but the user still has to prepare or remember the Wiki space ID, and the script does not represent a complete Wiki structure. This makes the demo harder to reproduce across test Feishu tenants and makes the seed data feel like loose documents instead of an internal knowledge base.

The desired target is a single safe seed entrypoint that uses a test `lark-cli` profile, creates or reuses a named team Wiki, and applies a stable tree of content-bearing pages from repository files.

## Goals / Non-Goals

**Goals:**

- Make FlowOps Feishu seeding reproducible from one script.
- Require an explicit seed/test Feishu profile before any write.
- Create or reuse a team Wiki named `星桥科技 FlowOps 知识库` by default.
- Store seed content as standalone Markdown files plus a JSON manifest.
- Keep apply idempotent: repeat runs update, move, or create managed pages without duplicates.
- Seed a realistic Wiki structure with directory pages and troubleshooting pages.

**Non-Goals:**

- Do not build a generic Wiki migration/sync engine.
- Do not delete or prune pages that are not in the seed manifest.
- Do not seed IM messages in this change.
- Do not add runtime per-app deterministic detectors; retrieval remains LLM-planned keyword search.

## Decisions

1. **Use a single script entrypoint with file-backed content.**

   `examples/flowops-airflow/scripts/seed-feishu` remains the only command users run, but content moves to `examples/flowops-airflow/seed/wiki/manifest.json` and sibling Markdown files. This keeps the demo workflow simple while making content reviewable and maintainable.

2. **Use JSON for the seed manifest.**

   JSON is easy to parse from Python, Go, and shell workflows without adding dependencies. The manifest will describe Wiki defaults, page title, source file, and nested children.

3. **Resolve seed profile explicitly.**

   `--apply` must resolve a profile from `--profile` first, then `[seed].feishu_profile` in `~/.lark-cue/config.toml`. If neither exists, the script fails before calling any write command. This prevents accidental writes through the default `lark-cli` login state.

4. **Create or reuse a named team Wiki.**

   The script lists Wiki spaces by name and reuses the exact match. If none exists during `--apply`, it creates a new Wiki space with `wiki spaces create --yes`. `--wiki-name` can override the default, but `my_library` and personal-library fallbacks are not allowed.

5. **Identify pages by manifest path inside the target Wiki.**

   Idempotency is based on target Wiki plus parent path plus title. If a managed page exists in the target Wiki under the wrong parent, the script moves it to the manifest-defined parent. If content differs, the script overwrites the page. The script should not write seed markers into user-visible titles or page bodies.

6. **Create content-bearing directory pages.**

   Directory nodes are ordinary Lark Docs with short explanatory Markdown. This makes the generated Wiki feel like real internal knowledge and improves search context.

## Risks / Trade-offs

- **Search indexing lag** -> Use direct Wiki/node APIs for idempotency where possible, and print smoke search commands as post-apply checks rather than relying on immediate search results for correctness.
- **High-risk Wiki creation** -> Require explicit seed profile and call `wiki spaces create` only in `--apply` mode with `--yes`; dry-run prints planned writes only.
- **Duplicate titles in the same Wiki** -> Prefer path-based matching through parent node and title. If ambiguity remains, fail with a clear message rather than picking a random node.
- **Existing test-tenant drift** -> Only manage manifest pages and never delete outside pages. A future `--prune` can be added as a separate explicit dangerous change.
