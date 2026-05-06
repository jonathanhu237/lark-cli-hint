# lark-cue FlowOps Demo

This demo proves one loop:

```text
real internal-style CLI command fails
-> lark-cue asks the LLM whether internal knowledge should be retrieved
-> LLM produces short Feishu keyword queries
-> lark-cue searches Docs/Wiki/Sheets and IM through lark-cli
-> lark-cue fetches/reads evidence
-> lark-cue prints a short cited knowledge card
-> lark-cue records planner/cue evaluation data
```

## Prerequisites

Build the CLI:

```sh
go build -o ./bin/lark-cue ./cmd/lark-cue
```

Configure LLM access. `lark-cue run` refuses to execute wrapped commands without it.

```sh
export LARK_CUE_LLM_API_KEY="..."
export LARK_CUE_LLM_MODEL="..."
export LARK_CUE_LLM_BASE_URL="https://api.openai.com/v1"
```

Check Feishu access:

```sh
lark-cli doctor
```

Use a test Feishu profile for the demo.

```sh
export LARK_CUE_FEISHU_PROFILE="<test-profile>"
```

## Seed FlowOps Knowledge

Dry-run first:

```sh
scripts/seed-flowops-feishu-demo --profile "$LARK_CUE_FEISHU_PROFILE"
```

Apply when the planned writes look correct:

```sh
scripts/seed-flowops-feishu-demo --apply --profile "$LARK_CUE_FEISHU_PROFILE"
```

The script creates or updates three Markdown documents:

- `[lark-cue-demo] FlowOps DAG Import Error 排障 FAQ`
- `[lark-cue-demo] billing_daily 历史故障复盘`
- `[lark-cue-demo] FlowOps DAG 开发规范`

It does not send IM messages, delete resources, or write without `--apply`.

After apply, run the smoke searches printed by the script. Feishu indexing can lag, so wait and retry if the documents do not appear immediately.

## Start the Local FlowOps Demo

The demo uses real Airflow through Docker Compose.

```sh
cd examples/flowops-airflow
cp .env.example .env
./flowctl init
```

Confirm the broken path:

```sh
./flowctl check billing_daily
```

Expected output includes a DAG import error for `billing_daily`, `Variable.get("billing_region")`, or a missing `billing_region` variable. The exact traceback can vary by Airflow version.

## Run lark-cue

For a clean recorded demo, isolate evaluation records:

```sh
export LARK_CUE_EVAL_LOG="$(mktemp -t lark-cue-flowops-evaluations.XXXXXX)"
```

Run:

```sh
../../bin/lark-cue run --no-feedback-prompt -- ./flowctl check billing_daily
```

Expected behavior:

- the original FlowOps/Airflow error remains visible;
- `lark-cue` shows the planner-selected FlowOps scenario;
- the LLM-generated keyword queries search real Feishu Docs/Wiki/Sheets and IM;
- the knowledge card cites FlowOps seed documents when retrieval succeeds;
- if evidence is weak, the card says so instead of inventing a cause.

Record feedback after the card id is printed:

```sh
../../bin/lark-cue feedback <card-id> useful
```

Then show validation:

```sh
../../bin/lark-cue eval report
```

The report summarizes planner decisions, retrieve-vs-skip counts, cue runs, retrieval status, citation coverage, query count, latency, and feedback.

## Optional Push Preview

Preview a Feishu group card without sending:

```sh
../../bin/lark-cue run --prepare-push --no-feedback-prompt -- ./flowctl check billing_daily
```

Actual sending requires an explicit send flag and target:

```sh
../../bin/lark-cue run --send-push --push-chat "oc_xxx" --no-feedback-prompt -- ./flowctl check billing_daily
```

## Cleanup

```sh
./flowctl down
./flowctl clean
```
