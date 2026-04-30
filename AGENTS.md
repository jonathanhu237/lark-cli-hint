# Lark CLI Hint — Agent Guide

For the product overview and motivating scenarios, see [`README.md`](README.md). This document adds agent-specific guidance — read it before proposing changes or implementing features.

## Stack

- **Language**: Python 3.11+
- **CLI framework**: typer
- **Vector store**: chromadb (embedded, local persistent)
- **Embeddings**: `BAAI/bge-m3` via sentence-transformers (multilingual; required because content is bilingual zh/en)
- **LLM client**: LiteLLM (provider-agnostic; common choices include Doubao, Claude, GPT, DeepSeek, local Ollama; configured via `~/.config/lark-cli-hint/config.yaml`)
- **Tests**: pytest

The system runs as a **single Python process** — no daemon, no client-server. Internally there is a logical frontend (`cli/`, `runner/`, `render/`) and backend (`engine/`, `index/`) split inside one package.

## Internationalization

The product ships bilingual (Chinese + English). UI labels, canned hint copy from the rule table, and LLM output language all follow the same locale resolution:

1. CLI `--lang=zh|en` (highest)
2. `config.yaml` `language` field
3. System `LANG` / `LC_*` env vars (`zh*` → Chinese, otherwise English)
4. Fallback: English

Translation files live in `locales/zh.toml` and `locales/en.toml`. Document titles, command names, and schema field names are NOT translated.

## Cross-source knowledge requirement

The product's value comes from integrating knowledge across the user's Feishu workspace. Features that consume only lark-cli's own information (schema, error strings, hand-curated rules) reduce the tool to a generic CLI error helper and lose the cross-source value.

Every hint-generating feature must combine **at least**:

- The user's Feishu tenant content (Docs / Minutes / Tasks), retrieved from the local index
- The lark-cli `schema --format json` command surface

If a proposed feature pulls from only one of these, redesign or reject.

## Out of scope (explicit non-goals)

- Group chat ingestion as a knowledge source — intentionally cut from MVP to bound work
- Shell hook installation (zsh `preexec` / bash `DEBUG`) — rejected in favor of the drop-in wrapper model
- Daemon / client-server split — rejected; single-process is sufficient at our scale
- Pre-execution argument validation — only post-execution attribution is in scope

## Conventions

- **Commits**: conventional commits (`feat:`, `fix:`, `docs:`, `chore:`, `test:`).
- **Branches**: trunk-based; small PRs targeting `main`.
- **Secrets**: never commit Feishu credentials, LLM API keys, tenant-specific IDs, or seed content with real names. Configuration goes in `~/.config/lark-cli-hint/config.yaml` (out of repo) or environment variables.
- **Style**: enforced by `ruff` (formatting + linting); type-checked with `mypy` where it pays off (engine and index modules).

## Where to find more

- **Active change in flight**: `openspec/changes/<change-name>/proposal.md`
- **Accepted specs**: `openspec/specs/`
- **Upstream lark-cli**: <https://github.com/larksuite/cli> (Go; provides `lark-cli schema --format json` for the machine-readable command surface we depend on)
