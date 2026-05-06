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
examples/flowops-airflow/scripts/seed-feishu
examples/flowops-airflow/scripts/seed-feishu --apply

# Reset a disposable broken workspace, then run the default demo path:
# card, then OpenClaw local main-agent handoff.
examples/flowops-airflow/scripts/reset-demo
examples/flowops-airflow/scripts/run-demo

# Local card-only path: skip OpenClaw preflight and handoff.
examples/flowops-airflow/scripts/run-demo --no-openclaw
```

See `docs/demo.md` and `examples/flowops-airflow/README.md` for the full recorded-demo flow.

To benchmark whether the seeded Wiki sources are actually cited for the real FlowOps failure:

```sh
examples/flowops-airflow/scripts/reset-demo
cd examples/flowops-airflow/.demo-workspace
lark-cue benchmark run --cases ../seed/eval-cases.json
```

The benchmark uses an isolated temporary evaluation log, runs real commands, and returns `0` only when every case passes. The FlowOps case runs `./flowctl init` as lightweight setup inside the disposable workspace. Use `--no-openclaw` for a card-only benchmark run. Run `examples/flowops-airflow/scripts/reset-demo` when you need a full reset.

## Evaluation

```sh
lark-cue eval report
```

The report summarizes planner decisions, cue runs, retrieval status, citation coverage, latency, query count, and OpenClaw handoff attempts/results when present.

## Safety

OpenClaw receives the failed command context, planner output, knowledge card, action plan, and Feishu citations so it can inspect and repair the local workspace. The handoff is not permission to perform high-risk external actions without asking first. OpenClaw should ask before deleting data, changing production configuration, rotating secrets, sending messages, committing code, or pushing code.

## Current Limitation

`lark-cue run` captures stdout/stderr through pipes so it can analyze failed command output. It preserves streamed bytes and exit codes, but commands that change behavior based on `isatty(1)` or `isatty(2)` may render differently than when run directly.
