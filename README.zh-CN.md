# Lark CLI Hint

[English](README.md) | [简体中文](README.zh-CN.md)

Lark CLI Hint 是一个面向飞书知识库工作流的 `lark-cli` 命令副驾驶，服务于终端中的人类用户，也服务于调用 CLI 工具的 AI Agent。

它会包裹 `lark-cli` 命令，并把命令结果转化为简洁、带依据的提示：

- **Recover**：解释失败的 `lark-cli` 命令，并建议一条修复命令。
- **Next**：解释成功的 `lark-cli` 结果，并建议下一条有用命令。

初始产品重点是知识库探索：搜索 docs/wiki、读取文档正文，并准备飞书群推送命令，但不会自动发送。

终端输出面向人类阅读，并支持国际化。JSON 输出提供相同语义，便于 AI Agent 消费。

更多范围、MVP 行为和实现约束见 [AGENTS.md](AGENTS.md)。
