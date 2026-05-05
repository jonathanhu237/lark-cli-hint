# lark-cue MVP Demo

This demo proves one loop:

```text
terminal command fails
-> lark-cue detects a Feishu API scope/token/permission error
-> lark-cue searches real Feishu Docs/Wiki and IM through lark-cli
-> lark-cue fetches/reads evidence
-> lark-cue prints a short cited knowledge card
-> lark-cue records feedback/evaluation data
```

## Prerequisites

Check local Feishu access:

```bash
lark-cli doctor
```

Expected health is a valid Feishu profile with token verification passing. The primary MVP path uses real `lark-cli` calls. It does not silently fall back to local fixtures.

Optional LLM configuration:

```bash
export LARK_CUE_LLM_BASE_URL="https://api.openai.com/v1"
export LARK_CUE_LLM_API_KEY="..."
export LARK_CUE_LLM_MODEL="..."
```

If LLM variables are absent, deterministic seed queries and template card generation are used.

The Feishu-side mock Docs and IM messages needed for the demo are listed in
`docs/feishu-mock-data.md`.

## Run The Demo

Build the CLI first if you want to avoid `go run` printing `exit status 1` when the wrapped failing command correctly returns code `1`:

```bash
go build -o ./bin/lark-cue ./cmd/lark-cue
```

```bash
./bin/lark-cue run -- node examples/failing-feishu-api.js
```

The wrapped command emits:

```text
LarkApiError: missing required scope: docx:document:read
tenant_access_token invalid or permission denied
```

`lark-cue` should search Docs/Wiki and IM messages using generated queries, fetch/read candidate sources, filter evidence, and print a compact terminal card.

## Demo Fixture Recovery

Fixture mode is explicit and labeled:

```bash
./bin/lark-cue run --demo-fixture --no-feedback-prompt -- node examples/failing-feishu-api.js
```

Use fixture mode only for offline tests or demo recovery. It is not the primary MVP path.

## Push Preview

Preview a group push without sending:

```bash
./bin/lark-cue run --prepare-push -- node examples/failing-feishu-api.js
```

Send requires an explicit flag and target:

```bash
./bin/lark-cue run --send-push --push-chat "oc_xxx" -- node examples/failing-feishu-api.js
```

## Feedback

Interactive terminals can mark a card useful, not useful, or skipped. Feedback can also be recorded later:

```bash
./bin/lark-cue feedback <card-id> useful
./bin/lark-cue feedback <card-id> not-useful
```

Evaluation records default to:

```text
~/.lark-cue/evaluations.jsonl
```
