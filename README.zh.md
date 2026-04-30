# Lark CLI Hint

[English](README.md) | 简体中文

为 [`lark-cli`](https://github.com/larksuite/cli) 提供的 drop-in wrapper，在 CLI 命令成功或失败时，从你的飞书工作空间检索相关知识并就地呈现。

## 为什么需要它

`lark-cli` 在飞书生态里暴露了 400+ 命令。`chat_id` 与 `doc_id`、`app_token` 与 `tenant_access_token`、scope、ID 格式、实体关系等概念，对开发者（尤其是新人）来说一直是踩坑高发区。而真正能解决问题的知识，往往散落在团队的飞书 Docs、会议妙记、任务和聊天历史里——等你翻到三个月前同事写的那篇文档时，已经过了一个小时。

`lch` 把这些信息接通。你用 `lch <args>` 替代 `lark-cli <args>`。当出错时（或显式提问时），它从本地索引（覆盖你的飞书工作空间 + `lark-cli` schema）检索相关内容，把知识就地呈现在终端——附带引用源，以及一条具体的下一步命令。

## 应用场景

### 1. 第一次接入

入职第二周的工程师被指派给 CI 加飞书通知。他猜了个命令：

```bash
$ lark-cli im messages send --chat-id="6843234567" "deployed v1.2"
Error: invalid chat_id format
```

没有 `lch`，他大约要在 API 文档和试错之间反复横跳 90 分钟，最后还是去 ping 老员工。

有 `lch`：

```bash
$ lch im messages send --chat-id="6843234567" "deployed v1.2"
Error: invalid chat_id format

❌  你传的 chat_id 看起来是群链接里的 ID。send-message 需要的是
    以 "oc_" 开头的 chat_id。

📚  来源
    · [Doc] 《lark-cli 鉴权与 ID 速查》· 王芳 · 2 周前
    · [schema] im.messages.send · chat-id required, format=oc_xxx

▶️  下一步
    $ lark-cli im chats list      # 列出群聊，找到正确的 chat_id
```

### 2. 跨人知识沉淀

三个月前，一位资深工程师写过一篇飞书 Doc 《lark-cli 鉴权 FAQ》。质量很好，但团队里没人知道它存在。

今天，新同事撞上 token 过期错误。没有 `lch`，他只能去群里 @ 一下、等回复 30 分钟。有 `lch`，hint 在他撞错的那一刻就把这篇 Doc 推到他面前——老员工没被打断，那篇 Doc 也不再绑定在某个人的记忆里，而是属于所有踩到同样坑的人。

### 3. 概念混淆

一位前端工程师在读飞书 API 文档时发现，同一个实体在不同接口里被叫做 `doc_id`、`doc_token`、`file_token`。他直接问：

```bash
$ lch explain "doc_id, doc_token, file_token，是同一个东西吗？"

📊  同一实体，历史命名不一致
    （基于 7 个 schema 方法 + 2 篇内部 Doc 合成）

    名称        出现于                     说明
    ───────────────────────────────────────────────────────
    doc_id      docs.documents.*           当前命名
    doc_token   drive.files.copy           遗留命名
    file_token  drive.medias.upload        drive 模块的遗留命名

    `lark-cli` 中三者接受同一个值；wrapper 内部统一规范化。

📚  来源
    · [schema] 7 处方法定义
    · [Doc] 《飞书实体命名约定》
```

## 工作原理

`lch` 是一个单进程 Python 应用。它在本地维护一份索引——内容来自你团队的飞书工作空间（Docs、Minutes、Tasks）以及 `lark-cli` 命令 schema。索引时每个 chunk 会被压缩成带引用的高密度卡片。运行时，`lch` 先用规则表匹配常见错误，未命中则走 retrieval，最终把 hint 就地渲染（人类模式：终端彩色输出；agent 模式：JSON 结构化输出）。

UI 与 LLM 输出语言由 `--lang` flag、`config.yaml` 或系统 `LANG` 环境变量决定，默认英文，检测到中文环境自动切中文。

架构与设计细节见 [`AGENTS.md`](AGENTS.md)。

## 许可证

MIT — 见 [`LICENSE`](LICENSE)。
