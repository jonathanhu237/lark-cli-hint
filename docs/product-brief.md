# Product Brief

## Background

This project is for the Feishu OpenClaw AI Campus Challenge, OpenClaw track.

Repository name and CLI name:

```text
lark-cue
```

The product is an active enterprise knowledge assistant for developers working in terminal-heavy internal workflows.

## Problem

Enterprise developers often debug internal platforms through CLI commands, but the answer is scattered across Feishu Docs, Wiki, group messages, postmortems, and checklists. The difficult part is not only searching; it is deciding whether the current failure depends on internal knowledge and which keywords will find the right fragment.

Manual search is slow because the developer must leave the terminal, explain the failure, guess keywords, and read long documents or noisy chat history.

## Product Positioning

`lark-cue` wraps a command and stays close to the terminal. When the command fails, it asks an LLM planner whether this failure should retrieve internal Feishu knowledge. If yes, the planner emits short keyword-style Feishu queries; `lark-cue` searches Docs/Wiki/Sheets and IM through `lark-cli`, fetches evidence, and prints a compact cited knowledge card.

The product is not a generic search box and not a hardcoded rule library. It is a contextual internal knowledge cue assistant.

## Target Users

- Enterprise developers using internal CLIs.
- New team members unfamiliar with internal platforms.
- On-call engineers who need historical fixes quickly.
- Teams that want Feishu knowledge to be reused instead of repeatedly asked in groups.

## Core Value

The assistant should answer:

1. Does this failure need internal knowledge?
2. What scenario is probably happening?
3. Which Feishu knowledge is relevant now?
4. What is the shortest safe next step?
5. Which sources support the suggestion?

Every confident suggestion must be grounded in fetched/read Feishu evidence.

## Main Demo Story

The main demo uses 星桥科技's internal scheduling platform, FlowOps, backed locally by a real Airflow container.

```text
developer runs flowctl check billing_daily
-> real Airflow reports a DAG import error
-> lark-cue planner decides this needs internal FlowOps knowledge
-> planner emits keyword queries such as billing_daily billing_region Variable.get
-> lark-cue searches Feishu Docs/Wiki/Sheets and IM through lark-cli
-> lark-cue cites FlowOps FAQ / historical incident / development standard docs
-> lark-cue recommends moving Variable.get out of DAG parse time or applying a short-term Variable unblock
-> benchmark/evaluation reports show planner decisions, retrieval, citations, and latency
```

This demo is more convincing than a hardcoded Feishu API error because it shows a real internal-app-style CLI failure and internal knowledge reuse.

## Knowledge Sources

Current retrieval routes:

- Docs/Wiki/Sheets through `lark-cli docs +search`
- IM through `lark-cli im +messages-search`

The FlowOps seed creates or reuses a complete team Wiki from a repository manifest and Markdown files. IM retrieval remains enabled for real tenants and future seeded group discussions.

## lark-cli Role

`lark-cli` is the Feishu bridge. `lark-cue` uses it to search and fetch internal sources, and optionally to send a prepared group push when the user explicitly requests sending.

`lark-cue` should not become a tutorial for `lark-cli`; it uses `lark-cli` as infrastructure for active internal knowledge delivery.

## LLM Role

LLM configuration is required.

The LLM planner decides:

- whether to retrieve internal knowledge;
- a short human-readable scenario;
- a short reason for the decision;
- keyword-style Feishu search queries.

The LLM may also draft card text, but card claims are accepted only when grounded in fetched snippets.

## Knowledge Card Shape

Cards should stay compact:

```text
Scenario
FlowOps DAG import error

Likely Cause
billing_daily reads billing_region while the DAG is imported, so FlowOps cannot finish DagBag parsing.

Action Plan
1. Move Variable.get("billing_region") into task runtime.
2. Set billing_region only as a short-term unblock if the pipeline is blocked.
3. Rerun flowctl check billing_daily.

Evidence
- FlowOps DAG Import Error 排障 FAQ
- billing_daily 历史故障复盘

Confidence
High, because the evidence mentions the same DAG, variable, and repair path.
```

If evidence is weak or absent, the card must say that internal evidence is insufficient.

## MVP Scope

```text
run command -> command fails -> LLM planner -> Feishu retrieval -> evidence-grounded card -> eval log
```

MVP requirements:

- command wrapper with stdout/stderr passthrough and exit-code preservation;
- mandatory LLM planner;
- real `lark-cli` retrieval;
- cited terminal card;
- optional explicit Feishu push;
- evaluation report;
- real-command benchmark that checks expected seeded source citations;
- reproducible FlowOps/Airflow demo and Feishu seed script.

## Out of Scope

- Semantic/vector search.
- Per-application hardcoded detector rules.
- Automatic repair command execution.
- Broad indexing across all Feishu surfaces.
- GUI terminal.
- Public local fixture mode.

## Effect Validation

The effect validation report should show:

- planner decisions and retrieve-vs-skip counts;
- cue runs;
- retrieval status;
- citation coverage;
- benchmark expected-source hit rate and citation precision;
- average query count;
- average latency.

The demo should be judged by whether `lark-cue` shortens the path from a real internal-style CLI failure to a cited internal answer.
