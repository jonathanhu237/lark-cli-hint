# Lark CLI Hint

English | [简体中文](README.zh.md)

A drop-in wrapper for [`lark-cli`](https://github.com/larksuite/cli) that surfaces contextual knowledge from your Feishu workspace when CLI commands fail or succeed.

## Why

`lark-cli` exposes 400+ commands across the Feishu ecosystem. Concepts like `chat_id` vs `doc_id` vs `app_token` vs `tenant_access_token`, scopes, ID formats, and entity relationships consistently trip up developers — especially newcomers. Meanwhile, the knowledge that would help is scattered across your team's Feishu Docs, meeting Minutes, tasks, and chat history. By the time you find the right doc someone wrote three months ago, you have already burned an hour.

`lch` bridges the gap. You run `lch <args>` instead of `lark-cli <args>`. When something goes wrong (or you ask for help explicitly), it retrieves from a local index of your Feishu workspace plus the `lark-cli` schema and surfaces the relevant knowledge inline in your terminal — with citations, and a concrete next-step command.

## Scenarios

### 1. First integration

A second-week engineer is asked to add Feishu CI notifications. He guesses a command:

```bash
$ lark-cli im messages send --chat-id="6843234567" "deployed v1.2"
Error: invalid chat_id format
```

Without `lch`, he spends ~90 minutes alternating between API docs and trial-and-error before pinging a senior teammate.

With `lch`:

```bash
$ lch im messages send --chat-id="6843234567" "deployed v1.2"
Error: invalid chat_id format

❌  The chat_id you passed looks like a group-link ID. send-message
    requires a chat_id starting with "oc_".

📚  Sources
    · [Doc] "lark-cli auth & ID quick reference" by Wang Fang · 2 weeks ago
    · [schema] im.messages.send · chat-id required, format=oc_xxx

▶️  Next
    $ lark-cli im chats list      # list chats and find the right chat_id
```

### 2. Cross-person knowledge resurfacing

Three months ago, a senior engineer wrote an internal Feishu Doc titled "lark-cli auth FAQ." It is excellent. Nobody on the team knows it exists.

Today, a new teammate hits a token-expiry error. Without `lch`, he interrupts the team chat and waits 30 minutes for a reply. With `lch`, the hint surfaces the doc the moment the error appears — the senior is never interrupted, and the doc starts paying off for everyone, not just whoever happens to remember it exists.

### 3. Concept confusion

A frontend engineer reading the Feishu API reference notices the same entity is called `doc_id` in some endpoints, `doc_token` in others, `file_token` in a third. He runs:

```bash
$ lch explain "doc_id, doc_token, file_token — same thing or not?"

📊  Same entity, different historical names
    (synthesized from 7 schema methods + 2 internal Docs)

    Name        Used in                    Note
    ───────────────────────────────────────────────────────
    doc_id      docs.documents.*           Current naming
    doc_token   drive.files.copy           Legacy naming
    file_token  drive.medias.upload        Legacy in drive

    In `lark-cli`, all three accept the same value; the wrapper normalizes.

📚  Sources
    · [schema] 7 method definitions
    · [Doc] "Feishu entity naming conventions"
```

## How it works

`lch` runs as a single Python process. It maintains a local index built from your team's Feishu workspace (Docs, Minutes, Tasks) plus the `lark-cli` command schema. Each indexed chunk is distilled into a citation-bearing card at index time. At runtime, `lch` matches command output against a rule table for common errors, falls back to retrieval for the long tail, and renders a hint inline (or as JSON, for agent callers).

Architecture and design rationale: see [`AGENTS.md`](AGENTS.md).

## License

MIT — see [`LICENSE`](LICENSE).
