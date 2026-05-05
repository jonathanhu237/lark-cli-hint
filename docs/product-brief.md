# Product Brief

## Background

This project is for the Feishu OpenClaw AI Campus Challenge, OpenClaw track.

The repository name and CLI name are both:

```text
lark-cue
```

The track asks participants to build an office agent with long-term memory and active service capability using Feishu OpenClaw, `lark-cli`, the Feishu ecosystem, and large model capabilities.

The selected topic is:

**Enterprise office knowledge integration and distribution agent.**

The selected sub-direction is:

**Direction C: immersive fragmented knowledge push assistant for geek and developer experience.**

## Problem

In an enterprise, valuable knowledge is scattered across Feishu Docs, wiki pages, meeting Minutes, group chats, tasks, and mail. The problem is usually not that knowledge does not exist. The problem is that developers do not know where to search, what keyword to use, or which historical discussion contains the answer.

Traditional knowledge management is passive:

- The user must know that a relevant document exists.
- The user must guess the right keyword.
- The user must leave the current workflow to search.
- The user must read long documents or noisy chats to extract the real answer.

This is especially painful for developers working in the terminal. When a command fails, the useful answer may be hidden in an old Feishu group discussion, an onboarding document, or a meeting conclusion. Manual search interrupts the workflow and often leads to repeated questions in team groups.

## Product Positioning

`lark-cue` is an active enterprise knowledge assistant for developers.

It stays close to the CLI workflow. When a developer hits a terminal error, executes a risky command, or enters a known workflow checkpoint, the assistant proactively retrieves relevant Feishu knowledge, compresses it into a high-density knowledge card, and delivers it in the terminal or Feishu group.

The product is not a general search box. It is a contextual knowledge delivery assistant.

## Target Users

Primary users:

- Developers in enterprise teams.
- New team members who are unfamiliar with internal systems.
- AI coding agents or human developers using CLI-heavy workflows.

Secondary users:

- Team leads who want to reduce repeated troubleshooting questions.
- On-call engineers who want known fixes to be distributed faster.
- Knowledge maintainers who want docs and historical discussions to be reused.

## Core Value

The assistant should reduce the distance between a developer's current problem and the team's existing knowledge.

It should provide:

- Faster troubleshooting.
- Fewer repeated questions in Feishu groups.
- Better reuse of internal Docs, Minutes, and chat history.
- More reliable answers because every suggestion cites sources.
- A stronger sense that enterprise knowledge can actively serve the workflow.

## Why Search or Chatbot Is Not Enough

Generic search is not enough because the developer may not know the right keyword or source.

Generic Q&A is not enough because it waits for the user to ask and may answer without enough context.

A normal chatbot is not enough because it is outside the developer's flow. The user must switch context, explain the error, paste logs, and verify sources manually.

This product should instead use the terminal event itself as context, then proactively push a small, sourced answer.

## Demo Story

Recommended demo:

```text
A developer runs a command that interacts with Feishu APIs.
The command fails with an auth/scope/token error.
The assistant detects the error pattern from the terminal output.
It searches Feishu Docs and historical group messages through lark-cli.
It finds an internal permission setup guide and a past discussion of the same error.
It produces a short knowledge card:
  - detected issue
  - likely cause
  - next action
  - source citations
The card appears in the terminal.
Optionally, a summarized card is pushed to a Feishu group for team visibility.
The user can mark the hint useful or not useful for evaluation.
```

Example command shape:

```bash
lark-cue run -- pnpm dev
```

This demo fits the contest because it shows:

- Long-term memory: the assistant reuses past documents and discussions.
- Active service: the assistant triggers from the terminal error.
- Knowledge integration: it connects Docs, messages, and possibly Minutes.
- Knowledge distribution: it pushes a compact card to terminal or Feishu group.
- Measurable value: it can compare manual search time against automatic hint time.

## Knowledge Sources

Potential Feishu sources:

- Docs and wiki pages for official troubleshooting guides.
- Group messages for historical bug discussions.
- Minutes for meeting conclusions and action items.
- Tasks for implementation context and ownership.
- Mail only if a later scenario clearly needs it.

For the first demo, Docs plus group messages are enough.

## lark-cli Role

`lark-cli` should be used as the bridge to Feishu.

Useful capabilities may include:

- Search or fetch Docs/wiki content.
- Search group messages or retrieve message context.
- Read Minutes summaries or transcripts.
- Send or prepare Feishu group messages.
- Check auth state and available scopes.

The product should not center on teaching `lark-cli` usage. It should use `lark-cli` as infrastructure to build an active knowledge assistant.

## Knowledge Card Shape

A terminal knowledge card should be compact:

```text
Scenario
Detected Feishu API permission error.

Likely Cause
The app token is valid, but the required scope was not granted after the last permission change.

Next Action
Run the scope check, then re-login with recommended permissions if the scope is missing.

Sources
- Feishu Doc: Internal API Permission Setup Guide
- Group: Backend Troubleshooting, 2026-04-28

Confidence
High, because both the error output and historical discussion mention the same missing scope pattern.
```

The card should not be a long answer. It should be a workflow aid.

## MVP Scope

Build one complete loop:

```text
trigger -> classify scenario -> retrieve Feishu knowledge -> compress into card -> deliver -> record feedback
```

MVP requirements:

- A CLI wrapper or watcher that can capture command output.
- At least one deterministic trigger, preferably command failure with known error pattern.
- Retrieval from at least one Feishu source through `lark-cli`.
- A knowledge card with citations.
- A terminal delivery path.
- Optional Feishu group delivery path.
- A simple evaluation log.

## Out of Scope for MVP

The MVP should not attempt:

- Full enterprise-wide indexing.
- Support for every terminal error.
- Automatic execution of repair commands.
- A complex GUI terminal.
- A general-purpose RAG platform.
- Broad multi-source ranking before the first demo is convincing.

## Effect Validation

The effect validation report should answer three questions.

### Accuracy

How often does the assistant produce a correct and source-supported hint?

Possible metric:

- Human rating from 1 to 5.
- Whether cited sources actually support the suggestion.
- Whether the suggested next action is safe and relevant.

### Efficiency

How much time does it save compared with manual search?

Possible metric:

- Manual search time for the same issue.
- Assistant response time.
- Number of documents or messages the user had to open.

### Acceptance

Do users actually use the pushed knowledge?

Possible metric:

- Useful / not useful feedback.
- Card click or copy count.
- Whether the user follows the suggested next action.
- Whether the issue is resolved faster in the demo task.

## Product Principle

The best version of this project is not the one that connects the most APIs. It is the one that makes one developer moment clearly better:

```text
I hit a problem.
The assistant recognized it.
It found the team's existing answer.
It gave me the shortest safe next step.
It showed where the answer came from.
```
