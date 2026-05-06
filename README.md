# lark-cue

`lark-cue` is an LLM-planned internal knowledge cue assistant for terminal-heavy enterprise workflows.

It wraps a command, preserves the command's stdout/stderr and exit code, and when the command fails it asks the configured LLM whether the failure should retrieve enterprise knowledge from Feishu. If retrieval is recommended, the LLM emits short keyword-style queries, `lark-cue` searches Feishu Docs/Wiki/Sheets and IM through `lark-cli`, then renders a compact cited knowledge card.

## Requirements

- Go 1.24+
- `lark-cli` authenticated against a Feishu tenant
- LLM config is required before `lark-cue run` will execute wrapped commands:

```sh
export LARK_CUE_LLM_API_KEY="..."
export LARK_CUE_LLM_MODEL="..."
export LARK_CUE_LLM_BASE_URL="https://api.openai.com/v1"
```

Use the same `lark-cli` profile for seeding and runtime retrieval when demoing against a test tenant:

```sh
export LARK_CUE_FEISHU_PROFILE="<test-profile>"
```

## Build

```sh
go build -o ./bin/lark-cue ./cmd/lark-cue
```

## FlowOps Demo

The main demo uses a real local Airflow environment wrapped as 星桥科技's internal FlowOps platform.

```sh
scripts/seed-flowops-feishu-demo
scripts/seed-flowops-feishu-demo --apply --profile "$LARK_CUE_FEISHU_PROFILE"

cd examples/flowops-airflow
cp .env.example .env
./flowctl init

../../bin/lark-cue run -- ./flowctl check billing_daily
```

See `docs/demo.md` and `examples/flowops-airflow/README.md` for the full recorded-demo flow.

## Evaluation

```sh
./bin/lark-cue eval report
```

The report summarizes planner decisions, cue runs, retrieval status, citation coverage, latency, query count, and feedback.

## Current Limitation

`lark-cue run` captures stdout/stderr through pipes so it can analyze failed command output. It preserves streamed bytes and exit codes, but commands that change behavior based on `isatty(1)` or `isatty(2)` may render differently than when run directly.
