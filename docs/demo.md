# lark-cue FlowOps Demo

This demo proves one loop:

```text
real internal-style CLI command fails
-> lark-cue asks the LLM whether internal knowledge should be retrieved
-> LLM produces short Feishu keyword queries
-> lark-cue searches Docs/Wiki/Sheets and IM through lark-cli
-> lark-cue fetches/reads evidence
-> lark-cue prints a short cited knowledge card
-> by default, lark-cue hands the card and evidence to OpenClaw local main agent
-> lark-cue records planner/cue evaluation data
```

## Prerequisites

Install `lark-cue` and the demo `flowctl` CLI yourself:

```sh
go install ./cmd/lark-cue
examples/flowops-airflow/scripts/install-flowctl  # installs only when flowctl is missing
lark-cue version
flowctl help
```

If `lark-cue` or `flowctl` is not found after install, add Go's bin directory to `PATH`:

```sh
export PATH="$(go env GOPATH)/bin:$PATH"
```

Configure LLM access. `lark-cue run` refuses to execute wrapped commands without it.

```sh
export LARK_CUE_LLM_API_KEY="..."
export LARK_CUE_LLM_MODEL="..."
export LARK_CUE_LLM_BASE_URL="https://api.openai.com/v1"
```

Configure OpenClaw for the default run path. `lark-cue run -- <command>` preflights OpenClaw before it executes the wrapped command unless you pass `--no-openclaw`.

```sh
openclaw agent --help
```

The MVP invokes the local `main` agent after the knowledge card is rendered:

```sh
openclaw agent --local --agent main --timeout 900 --message "<task>"
```

Use `lark-cue run --no-openclaw -- <command>` when you want card-only mode for local testing. This skips OpenClaw preflight and skips the post-card handoff.

Check Feishu access:

```sh
lark-cli doctor
```

Configure a test Feishu profile for seed writes and runtime retrieval in `~/.lark-cue/config.toml`.

```toml
[seed]
feishu_profile = "flowops-demo"
wiki_name = "星桥科技 FlowOps 知识库"

[feishu]
profile = "flowops-demo"
```

## Seed FlowOps Knowledge

Dry-run first:

```sh
examples/flowops-airflow/scripts/seed-feishu
```

Apply when the planned writes look correct:

```sh
examples/flowops-airflow/scripts/seed-feishu --apply
```

The script creates or reuses the team Wiki named `星桥科技 FlowOps 知识库`, then creates or updates this tree:

```text
星桥科技 FlowOps 知识库
└── FlowOps 调度平台
    ├── DAG 发布与巡检
    │   ├── FlowOps DAG Import Error 排障 FAQ
    │   └── FlowOps DAG 开发规范
    └── 历史故障复盘
        └── billing_daily 历史故障复盘
```

The seed content lives in `examples/flowops-airflow/seed/wiki/manifest.json` and sibling Markdown files. The script is idempotent: repeated runs update managed pages and move them back to the manifest-defined parents without creating duplicates. It does not send IM messages, delete resources, prune non-manifest pages, write personal-library documents, or write without `--apply`.

After apply, run the smoke searches printed by the script. Feishu indexing can lag, so wait and retry if the documents do not appear immediately.

## Start the Local FlowOps Demo

The demo uses real Airflow through Docker Compose.

```sh
cd examples/flowops-airflow
cp .env.example .env
flowctl init
```

Confirm the broken path:

```sh
flowctl check billing_daily
```

Expected output includes a DAG import error for `billing_daily`, `Variable.get("billing_region")`, or a missing `billing_region` variable. The exact traceback can vary by Airflow version.

## Run lark-cue

For a clean recorded demo, isolate evaluation records:

```sh
export LARK_CUE_EVAL_LOG="$(mktemp -t lark-cue-flowops-evaluations.XXXXXX)"
```

Run the default OpenClaw demo path:

```sh
lark-cue run -- flowctl check billing_daily
```

Expected behavior:

- before `flowctl check billing_daily` starts, `lark-cue` verifies that `openclaw agent --help` can run;
- the original FlowOps/Airflow error remains visible;
- `lark-cue` shows the planner-selected FlowOps scenario;
- the knowledge card shows the LLM planner reason and generated keyword queries;
- those queries search real Feishu Docs/Wiki/Sheets and IM;
- the knowledge card cites FlowOps seed documents when retrieval succeeds;
- if evidence is weak, the card says so instead of inventing a cause.
- when a card is generated, `lark-cue` invokes OpenClaw with the local `main` agent and a 900 second timeout, streaming OpenClaw output to stderr while preserving the wrapped command exit code.

Run the local card-only path when OpenClaw is not installed or when you only want to inspect the card:

```sh
lark-cue run --no-openclaw -- flowctl check billing_daily
```

In card-only mode, `lark-cue` still requires LLM configuration and Feishu access, but it does not preflight or invoke OpenClaw.

Then show validation:

```sh
lark-cue eval report
```

The report summarizes planner decisions, retrieve-vs-skip counts, cue runs, retrieval status, citation coverage, query count, and latency.

## Run the Benchmark

Use the benchmark when you want a repeatable score instead of a single recorded run:

```sh
lark-cue benchmark run --cases examples/flowops-airflow/seed/eval-cases.json
```

The benchmark runs real commands through the same `lark-cue run -- <command>` path, writes planner/cue records to a temporary evaluation log, and scores exact citation-title matches against the seeded Wiki titles declared in `eval-cases.json`. The FlowOps case uses `flowctl init` as lightweight setup. It does not run automatically from tests because it depends on Docker, LLM access, Feishu access, and local indexing state.

For local benchmark runs that should avoid OpenClaw, pass the benchmark opt-out:

```sh
lark-cue benchmark run --no-openclaw --cases examples/flowops-airflow/seed/eval-cases.json
```

Keep the default benchmark command for the contest story where OpenClaw is part of the active handoff.

## Optional Push Preview

Preview a Feishu group card without sending:

```sh
lark-cue run --prepare-push -- flowctl check billing_daily
```

Actual sending requires an explicit send flag and target:

```sh
lark-cue run --send-push --push-chat "oc_xxx" -- flowctl check billing_daily
```

OpenClaw does not make Feishu sending implicit. Sending a group push still requires the explicit send flag and target. The same safety boundary applies to other high-risk actions: OpenClaw should ask before deleting data, changing production configuration, rotating secrets, sending messages, committing code, pushing code, or performing similar external side effects.

## Cleanup

```sh
flowctl down
flowctl clean
```
