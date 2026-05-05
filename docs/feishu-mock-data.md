# Feishu Mock Data Setup

Use this document to recreate the MVP demo knowledge in a Feishu enterprise.
Do not paste real tenant URLs or chat IDs back into this file.

## Enterprise Context

- Enterprise name: 星桥科技
- Knowledge base name: 星桥科技知识库
- Demo project: Atlas 文档摘要服务
- Demo group chat: 星桥开放平台排障群
- Demo failure:
  - `missing required scope: docx:document:read`
  - `tenant_access_token invalid`
  - `permission denied`

## Group Chat Messages

Create a Feishu group named `星桥开放平台排障群` and add this discussion:

```text
张三：
我本地调试 Atlas 文档摘要服务时，调用飞书文档 API 报错：
missing required scope: docx:document:read
tenant_access_token invalid or permission denied
这个是代码问题吗？

李四：
这个我之前踩过，不是代码逻辑问题。一般是应用没有加 docx:document:read 权限，或者权限加了但没有发布权限变更。

王五：
补充一下，飞书开放平台新增权限后，需要发布应用权限变更；本地开发身份也要重新授权。旧 token 不会自动包含新 scope。

张三：
重新发布权限变更并重新授权后好了。我把这个结论记到《飞书应用权限配置避坑指南》。
```

## Knowledge Base Structure

Create a knowledge base named `星桥科技知识库` with this structure:

```text
星桥科技知识库
├── 开放平台集成
│   ├── Atlas 文档摘要服务接入说明
│   ├── 飞书应用权限配置避坑指南
│   └── 开放平台权限变更同步会纪要
└── Atlas 产品研发
    └── Atlas 前端主题色调整记录
```

## Document: Atlas 文档摘要服务接入说明

```markdown
# Atlas 文档摘要服务接入说明

Atlas 是星桥科技内部的知识运营平台。Atlas 文档摘要服务会读取飞书 Docs
内容，生成可复用的摘要片段，供研发、产品和客服团队在日常排障中使用。

本服务调用飞书文档 API 时需要应用具备文档读取权限：

- docx:document:read

如果本地开发时看到以下错误，通常优先排查飞书开放平台应用权限、授权状态和
token 状态，而不是先怀疑 Atlas 业务逻辑：

- missing required scope: docx:document:read
- tenant_access_token invalid
- permission denied
```

## Document: 飞书应用权限配置避坑指南

```markdown
# 飞书应用权限配置避坑指南

当 Atlas 文档摘要服务报错 `missing required scope: docx:document:read` 时，
优先检查以下事项：

1. 飞书开放平台应用是否已经添加 `docx:document:read`。
2. 新增权限后是否已经发布应用权限变更。
3. 本地开发身份是否已经重新授权。

旧 token 不会自动包含新发布的 scope。如果权限已经添加但仍然报错，建议清理
旧 token 缓存或刷新 token 后重试。

推荐处理步骤：

1. 打开飞书开放平台应用权限页面。
2. 确认 `docx:document:read` 已添加。
3. 发布权限变更。
4. 重新授权本地开发身份。
5. 必要时清理旧 token 或刷新 token。
6. 重新执行本地 API 调用。
```

## Document: 开放平台权限变更同步会纪要

```markdown
# 开放平台权限变更同步会纪要

会议结论：

- Atlas 文档摘要服务的 `missing required scope: docx:document:read` 问题不是
  业务逻辑错误。
- 新增飞书 API scope 后，必须把 scope 加入权限 checklist。
- 权限变更只有发布后才会对应用生效。
- 开发者在权限变更后需要重新授权。
- 发布 checklist 应包含飞书 API scope、权限发布状态、token 刷新状态。
```

## Negative-Control Document: Atlas 前端主题色调整记录

```markdown
# Atlas 前端主题色调整记录

本文记录 Atlas 前端主题色、深色模式和知识卡片样式的调整。

本记录只涉及前端视觉样式，不涉及飞书开放平台、docx:document:read、
missing required scope、tenant_access_token invalid、permission denied、
权限变更、重新授权或旧 token。
```

## Expected Retrieval Keywords

Positive keywords:

- `docx:document:read`
- `missing required scope`
- `tenant_access_token invalid`
- `permission denied`
- `权限变更`
- `重新授权`
- `旧 token`
- `刷新 token`
- `Atlas 文档摘要服务`
- `飞书开放平台`

Negative-control keywords:

- `主题色`
- `前端`
- `深色模式`
- `知识卡片样式`

## Smoke Commands

After the Feishu data is copied, run:

```bash
lark-cli doctor
lark-cli docs +search --query "missing required scope" --page-size 10 --format json
lark-cli docs +search --query "权限变更 重新授权" --page-size 10 --format json
lark-cli im +messages-search --query "docx:document:read" --page-size 10 --format json
```

Then run the CLI demo:

```bash
go build -o ./bin/lark-cue ./cmd/lark-cue
./bin/lark-cue run -- node examples/failing-feishu-api.js
```
