# lark-cue

`lark-cue` is an LLM-planned internal knowledge cue assistant for terminal-heavy enterprise workflows.

It wraps a command, preserves the command's stdout/stderr and exit code, and when the command fails it asks the configured LLM whether the failure should retrieve enterprise knowledge from Feishu. If retrieval is recommended, the LLM emits short keyword-style queries, `lark-cue` searches Feishu Docs/Wiki and IM through `lark-cli`, then renders a compact cited knowledge card. By default, the final card is handed to OpenClaw so a local agent can continue from the cited evidence.

## Requirements

- Go 1.24+
- `lark-cli` authenticated against a Feishu tenant
- Docker Desktop or Docker Compose for the FlowOps Airflow demo and benchmark
- LLM config is required before `lark-cue run` will execute wrapped commands:

```sh
export LARK_CUE_LLM_API_KEY="..."
export LARK_CUE_LLM_MODEL="..."
export LARK_CUE_LLM_BASE_URL="https://api.openai.com/v1"
```

- OpenClaw is required for the default `lark-cue run -- <command>` path. The run command preflights OpenClaw before executing the wrapped command, then hands the cited knowledge card to the local OpenClaw agent after card rendering when the planner selects internal-knowledge retrieval. OpenClaw output streams live, followed by a compact result card with status, duration, exit code, and output excerpt.

Ensure the `openclaw` binary is on `PATH` and the agent CLI is available:

```sh
openclaw agent --help
```

The MVP default is the local `main` agent with a 900 second timeout:

```sh
openclaw agent --local --agent main --timeout 900 --message "<task>"
```

Use `--no-openclaw` for card-only mode. That skips both the OpenClaw preflight and the post-card handoff.

Configure a seed-only `lark-cli` profile in `~/.lark-cue/config.toml` before writing demo knowledge:

```toml
[seed]
feishu_profile = "flowops-demo"
wiki_name = "星桥科技 FlowOps 知识库"
im_chat = "星桥科技 FlowOps 排障演示群"
```

Use the same profile for runtime retrieval when demoing against that test tenant:

```toml
[feishu]
profile = "flowops-demo"
```

## Install

Install `lark-cue` and the demo `flowctl` CLI yourself:

```sh
go install ./cmd/lark-cue
examples/flowops-airflow/scripts/install-flowctl  # installs only when flowctl is missing
lark-cue version
flowctl help
openclaw agent --help
```

If `lark-cue` or `flowctl` is not found after install, add Go's bin directory to `PATH`:

```sh
export PATH="$(go env GOPATH)/bin:$PATH"
```

For local development builds that should not touch `PATH`, use `go build -o ./bin/lark-cue ./cmd/lark-cue`.

## FlowOps Demo

The main demo uses a real local Airflow environment wrapped as 星桥科技's internal FlowOps platform.

```sh
# Dry-run first, then apply to the configured test Wiki/chat.
examples/flowops-airflow/scripts/seed-feishu
examples/flowops-airflow/scripts/seed-feishu --apply

# Reset a disposable broken workspace.
examples/flowops-airflow/scripts/reset-demo
cd examples/flowops-airflow/.demo-workspace
export PATH="$PWD:$PATH"
flowctl init

# Default demo path: card, then OpenClaw local main-agent handoff.
lark-cue run -- flowctl check billing_daily

# Local card-only path: skip OpenClaw preflight and handoff.
lark-cue run --no-openclaw -- flowctl check billing_daily

# IM retrieval path: show latest group discussion for source schema drift.
flowctl source reset billing_export_2026
flowctl check billing_export_2026
lark-cue run --no-openclaw -- flowctl check billing_export_2026
flowctl source refresh billing_export_2026 --alias customer_segment=segment
flowctl check billing_export_2026
```

See `docs/demo.md` and `examples/flowops-airflow/README.md` for the full recorded-demo flow.

## FlowOps Benchmark

The benchmark is the current effect-validation path. It runs real `flowctl` commands, lets `lark-cue` call the configured LLM to decide retrieval and generate Feishu keyword queries, searches the seeded Wiki/IM data, and then scores whether the final card cited the expected source.

The current benchmark has 10 scenarios:

| Case | Command | Expected source |
| --- | --- | --- |
| DAG import config | `flowctl check billing_daily` | Wiki FAQ + dev guide + postmortem |
| Source schema drift | `flowctl check billing_export_2026` | Wiki FAQ + IM group |
| Dataset freshness | `flowctl check orders_reconcile_2026` | Wiki FAQ |
| Secret rotation | `flowctl check ad_spend_daily` | Wiki FAQ |
| Executor capacity | `flowctl check inventory_snapshot` | Wiki FAQ |
| Feature gate rollout | `flowctl check churn_features` | IM group, with `EXP-883` and `cohort_v3` |
| Partner quota | `flowctl check payment_settlement` | Wiki FAQ, with `partner_api_v2` and `replay-window` |
| Egress allowlist | `flowctl check crm_sync` | Wiki FAQ, with `crm.internal.svc` and `revenue-egress` |
| Data governance | `flowctl check customer360_pii` | Wiki FAQ, with `PII-DLP-17` and `PRIV-2049` |
| Release freeze | `flowctl check revenue_forecast` | IM group, with `finclose-2026-05` and `RFC-7781` |

Run it locally:

```sh
REPO_ROOT="$(pwd)"
go build -o "$REPO_ROOT/bin/lark-cue" ./cmd/lark-cue
examples/flowops-airflow/scripts/seed-feishu --apply
examples/flowops-airflow/scripts/reset-demo --quiet
cd examples/flowops-airflow/.demo-workspace
export PATH="$PWD:$REPO_ROOT/bin:$PATH"
lark-cue benchmark run --no-openclaw --cases ../seed/eval-cases.json --verbose
```

The benchmark uses an isolated temporary evaluation log and returns `0` only when every case passes. Use `--no-openclaw` for card-only scoring; omit it when you want the default OpenClaw preflight/handoff path. Run `examples/flowops-airflow/scripts/reset-demo` when you need a full reset before another recording.

Expected healthy summary:

```text
cases: 10/10 passed
expected-source hit rate: 10/10
```

## Evaluation

```sh
lark-cue eval report
```

The report summarizes planner decisions, cue runs, retrieval status, citation coverage, latency, query count, and OpenClaw handoff attempts/results when present.

## Safety

OpenClaw receives the failed command context, planner output, knowledge card, action plan, and Feishu citations so it can inspect and repair the local workspace. The handoff is not permission to perform high-risk external actions without asking first. OpenClaw should ask before deleting data, changing production configuration, rotating secrets, sending messages, committing code, or pushing code.

## Current Limitation

`lark-cue run` captures stdout/stderr through pipes so it can analyze failed command output. It preserves streamed bytes and exit codes, but commands that change behavior based on `isatty(1)` or `isatty(2)` may render differently than when run directly.
