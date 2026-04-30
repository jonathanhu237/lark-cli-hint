# Lark CLI Hint

[English](README.md) | [简体中文](README.zh-CN.md)

Lark CLI Hint is a `lark-cli` command copilot for Feishu knowledge-base workflows, built for both humans in the terminal and AI agents calling CLI tools.

It wraps `lark-cli` and turns command results into concise, evidence-backed guidance:

- **Recover**: explain failed `lark-cli` commands and suggest a repair command.
- **Next**: explain successful `lark-cli` results and suggest the next useful command.

The initial product focus is knowledge-base exploration: search docs/wiki, fetch document content, and prepare a Feishu group push command without sending it automatically.

Terminal output is human-readable and locale-aware. JSON output exposes the same semantics for AI-agent consumption.

See [AGENTS.md](AGENTS.md) for scope, MVP behavior, and implementation guardrails.
