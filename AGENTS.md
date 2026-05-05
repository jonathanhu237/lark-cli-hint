# AGENTS.md

This file is the working agreement for humans and AI agents contributing to this project.

## Product Source of Truth

This project is for the Feishu OpenClaw AI Campus Challenge.

The product is an immersive fragmented knowledge push assistant for enterprise developers. It lives close to the CLI workflow, observes developer signals such as terminal errors, specific commands, workflow checkpoints, or repeated failure patterns, then retrieves relevant enterprise knowledge from Feishu sources and turns it into short, actionable, cited knowledge cards.

The core value is not search. The core value is active, contextual knowledge delivery.

## Chosen Direction

The selected direction is:

**Direction C: immersive fragmented knowledge push assistant, focused on geek and developer experience.**

Target users are developers working in terminal-heavy enterprise workflows.

Typical knowledge includes:

- Internal troubleshooting guides.
- Historical bug discussions from Feishu groups.
- API usage notes and permission pitfalls.
- Deployment, configuration, and release checklist knowledge.
- Meeting conclusions or task context relevant to the current workflow.

## Core Product Promise

When a developer hits a problem or enters a risky workflow, the assistant should answer:

1. What is probably happening?
2. What internal knowledge is relevant now?
3. What should the developer do next?
4. Which Feishu sources support the suggestion?

The product should feel like a teammate who remembers the team's scattered knowledge and pushes the right fragment at the right moment.

## MVP Demo Focus

Prefer one strong, coherent demo over broad but shallow coverage.

The recommended MVP flow is:

```text
terminal command fails
-> assistant detects the scenario from the command and output
-> assistant searches or fetches relevant Feishu knowledge via lark-cli
-> assistant produces a short evidence-backed knowledge card
-> assistant displays it in the terminal
-> assistant optionally prepares or sends a Feishu group push according to explicit user/demo settings
-> assistant records whether the hint was useful for evaluation
```

Good first demo scenario:

```text
Developer hits a Feishu API auth/scope/token error
-> assistant retrieves internal permission setup docs and historical group discussion
-> assistant explains the likely cause
-> assistant suggests the next verification or repair step
-> assistant cites the exact docs/messages/minutes used
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
- Broad indexing across Docs, Minutes, messages, tasks, and mail before the core demo loop works.

## Feishu and lark-cli Role

`lark-cli` is the integration layer for Feishu data and delivery.

Use it to access Feishu Docs, wiki, messages, Minutes, tasks, and group messaging where useful. The product should not merely wrap `lark-cli` commands. It should use `lark-cli` to create an active enterprise knowledge assistant experience.

OpenClaw and the CLI should be presented as the agent runtime and Feishu ecosystem bridge, not as the product's end-user value by themselves.

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
- Pattern trigger: command output matches a known error pattern.
- Threshold trigger: repeated similar errors appear within a time window.
- Scheduled trigger: a periodic knowledge digest for an upcoming workflow.

For the first version, prefer event-driven terminal failure detection because it is easiest to demonstrate and evaluate.

## Evaluation Mindset

The project must support an effect validation report.

Design features so they can be evaluated by:

- Accuracy of the suggested knowledge.
- Quality and traceability of citations.
- Time saved compared with manual Feishu search.
- User acceptance, such as whether the suggested card was opened, copied, or marked useful.
- Task completion improvement in a controlled demo.

Do not add features that cannot be demonstrated or evaluated in the contest context.

## Implementation Bias

Keep implementation choices aligned with the product promise:

- Build the trigger -> retrieve -> compress -> push -> evaluate loop first.
- Prefer deterministic scenario detection for the first demo.
- Use fixtures when real Feishu access is unavailable, but keep command shapes and evidence realistic.
- Keep side effects explicit. Preparing a group push is safer than silently sending one.
- Keep the UI compact enough for an active terminal workflow.

