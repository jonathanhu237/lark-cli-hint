# AGENTS.md

This file is the working agreement for humans and AI agents contributing to this project.

## Product Source of Truth

This project is for the Feishu OpenClaw AI Campus Challenge.

Repository name and CLI name:

```text
lark-cue
```

The product is an LLM-planned internal knowledge cue assistant for enterprise developers. It lives close to CLI workflows, observes failed internal-platform commands, asks an LLM whether enterprise knowledge should be retrieved, searches Feishu through `lark-cli`, and turns fetched evidence into short, actionable, cited knowledge cards.

The core value is not search. The core value is active, contextual internal knowledge delivery.

## Chosen Direction

The selected direction is:

**Direction C: immersive fragmented knowledge push assistant, focused on geek and developer experience.**

Target users are developers working in terminal-heavy enterprise workflows.

Typical knowledge includes:

- Internal troubleshooting guides.
- Historical bug discussions from Feishu groups.
- Internal platform usage notes and permission/configuration pitfalls.
- Deployment, scheduling, configuration, and release checklist knowledge.
- Meeting conclusions or task context relevant to the current workflow.

## Core Product Promise

When a developer hits a problem in an internal CLI workflow, the assistant should answer:

1. Does this failure need internal knowledge?
2. What is probably happening?
3. What Feishu knowledge is relevant now?
4. What should the developer do next?
5. Which Feishu sources support the suggestion?

The product should feel like a teammate who remembers the team's scattered knowledge and pushes the right fragment at the right moment.

## MVP Demo Focus

Prefer one strong, coherent demo over broad but shallow coverage.

The recommended MVP flow is:

```text
terminal command fails
-> LLM planner decides whether internal knowledge retrieval is useful
-> LLM planner generates short keyword-style Feishu queries
-> assistant searches or fetches relevant Feishu knowledge via lark-cli
-> assistant produces a short evidence-backed knowledge card
-> assistant displays it in the terminal
-> assistant optionally prepares or sends a Feishu group push according to explicit user/demo settings
-> assistant records planner/cue events and feedback for evaluation
```

Good first demo scenario:

```text
Developer runs FlowOps check for billing_daily
-> real Airflow reports a DAG import error caused by parse-time Variable.get("billing_region")
-> assistant retrieves internal FlowOps FAQ and historical incident docs
-> assistant explains the likely cause
-> assistant suggests the next verification or repair step
-> assistant cites the exact docs/messages used
```

## Non-Goals

Do not drift into building:

- A generic shell assistant for arbitrary terminal problems.
- A generic chatbot.
- A generic RAG or enterprise search platform.
- A Warp-style GUI terminal.
- A lark-cli command tutorial or command copilot.
- A tool that only explains how to use `lark-cli`.
- An agent that automatically performs risky follow-up actions without explicit user consent.
- Per-application hardcoded detector rules before the LLM-planned loop is convincing.
- Semantic/vector search before keyword retrieval has been validated.

## Feishu and lark-cli Role

`lark-cli` is the integration layer for Feishu data and delivery.

Use it to access Feishu Docs, Wiki, messages, Minutes, tasks, and group messaging where useful. The product should not merely wrap `lark-cli` commands. It should use `lark-cli` to create an active enterprise knowledge assistant experience.

OpenClaw and the CLI should be presented as the agent runtime and Feishu ecosystem bridge, not as the product's end-user value by themselves.

## LLM Role

LLM configuration is required for `lark-cue run`.

The LLM planner decides:

- whether a failed command should retrieve internal knowledge;
- a short scenario name;
- a short reason;
- keyword-style Feishu search queries.

The LLM may also draft card text, but final claims must be grounded in fetched/read snippets.

## Knowledge Card Contract

A knowledge card should be short, actionable, and cited.

It should usually contain:

- Detected scenario.
- Likely cause or relevant knowledge fragment.
- One recommended next action.
- Evidence sources.
- Confidence or caveat when evidence is weak.

Avoid unsupported claims. If the assistant cannot find strong evidence, it should say so and avoid pretending certainty.

## Active Service Requirement

At least one proactive trigger must exist in the demo.

Acceptable triggers include:

- Event-driven trigger: a wrapped terminal command fails.
- LLM-planned trigger: the command failure is judged to need internal knowledge.
- Threshold trigger: repeated similar errors appear within a time window.
- Scheduled trigger: a periodic knowledge digest for an upcoming workflow.

For the first version, prefer event-driven terminal failure plus LLM planning because it is easiest to demonstrate and evaluate.

## Evaluation Mindset

The project must support an effect validation report.

Design features so they can be evaluated by:

- Planner precision: retrieve versus skip decisions.
- Accuracy of the suggested knowledge.
- Quality and traceability of citations.
- Time saved compared with manual Feishu search.
- User acceptance, such as whether the suggested card was opened, copied, or marked useful.
- Task completion improvement in a controlled demo.

Do not add features that cannot be demonstrated or evaluated in the contest context.

## Implementation Bias

Keep implementation choices aligned with the product promise:

- Build the run -> plan -> retrieve -> compress -> push -> evaluate loop first.
- Require LLM configuration before running wrapped commands.
- Use real `lark-cli` and real Feishu data in product/demo paths.
- Use mocks/fakes only for unit tests; real E2E is explicit and opt-in.
- Keep side effects explicit. Preparing a group push is safer than silently sending one.
- Keep the UI compact enough for an active terminal workflow.
