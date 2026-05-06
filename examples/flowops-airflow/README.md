# FlowOps Airflow Demo

This example is a local FlowOps demo for `lark-cue`. FlowOps is 星桥科技's internal scheduling platform in the demo story, backed here by a real local Airflow container.

The broken `billing_daily` DAG calls `Variable.get("billing_region")` while Airflow imports the DAG file. On a fresh local metadata database that Variable does not exist, so Airflow reports a real DAG import error.

The reproducible demo path uses a disposable workspace under `.demo-workspace`. OpenClaw is allowed to edit files there; the fixture under `fixtures/broken` remains unchanged for the next recording.

Install the demo CLI from the repository root if you want a global `flowctl` command:

```sh
examples/flowops-airflow/scripts/install-flowctl
```

The install script is safe to run repeatedly: it skips installation when a `flowctl` command already exists in `PATH`.

Default `lark-cue run -- <command>` also requires OpenClaw. Ensure `openclaw` is on `PATH` and verify the agent CLI before recording the default demo:

```sh
openclaw agent --help
```

The default post-card handoff uses the local `main` agent with a 900 second timeout. Use `--no-openclaw` when you want to run only the terminal knowledge card locally; that mode skips OpenClaw preflight and skips the post-card handoff.

## One-Time Setup

From the repository root, create a fresh broken workspace:

```sh
examples/flowops-airflow/scripts/reset-demo
cd examples/flowops-airflow/.demo-workspace
./flowctl init
```

First startup downloads the Airflow image and initializes a local SQLite metadata database. This may take several minutes.

Optional inspection UI:

```sh
./flowctl up
```

Open `http://localhost:8080` and sign in with `admin` / `admin`.

## Expected Failure

Run the failing FlowOps check:

```sh
./flowctl check billing_daily
```

Expected terminal context includes Airflow import-error output for `billing_daily.py` and `Variable billing_region does not exist`. The exact traceback can vary by Airflow version, but it should identify the parse-time `Variable.get("billing_region")` failure.

## lark-cue Demo Run

After seeding the Feishu demo knowledge from the repository root, run the default OpenClaw path:

```sh
lark-cue run -- ./flowctl check billing_daily
```

For local card-only inspection without OpenClaw:

```sh
lark-cue run --no-openclaw -- ./flowctl check billing_daily
```

For the contest recording, keep the command prompt visible and show the generated knowledge card with:

- detected FlowOps/Airflow DAG import scenario;
- likely parse-time Variable lookup cause;
- an ordered action plan, such as moving Variable access into task runtime, using configuration only as a short-term unblock, and rerunning the failing check;
- citations to the 星桥科技 FlowOps mock docs.
- the default OpenClaw handoff after the card, using the local `main` agent.

OpenClaw should inspect local state and verify changes, but it is not allowed to silently perform high-risk actions. It should ask before deleting data, changing production configuration, rotating secrets, sending messages, committing code, pushing code, or performing similar external side effects.

## Benchmark

From the repository root:

```sh
examples/flowops-airflow/scripts/reset-demo
cd examples/flowops-airflow/.demo-workspace
lark-cue benchmark run --cases ../seed/eval-cases.json
```

Use `--no-openclaw` for a local card-only benchmark run:

```sh
lark-cue benchmark run --no-openclaw --cases ../seed/eval-cases.json
```

The benchmark runs the real `./flowctl check billing_daily` failure through `lark-cue run`, uses a temporary evaluation log, and checks whether the final card cites the expected seeded Wiki titles. The case runs `./flowctl init` first as lightweight setup. It does not reset the workspace after running; use `examples/flowops-airflow/scripts/reset-demo` when you want a full reset before recording.

## Reset or Cleanup

Stop containers:

```sh
./flowctl down
```

Return the demo to a clean state:

```sh
examples/flowops-airflow/scripts/reset-demo
```

To demonstrate a temporary fix manually, set the Variable and rerun the check:

```sh
./flowctl airflow variables set billing_region cn-north
./flowctl check billing_daily
```

Then run `examples/flowops-airflow/scripts/reset-demo` before recording the broken path again.

## Recording Tips

- Run `./flowctl init` in `.demo-workspace` before recording so the first captured command is fast.
- Keep Feishu seed search results ready, because indexing can lag after writes.
- Use a test Feishu profile and avoid real team chats; this demo does not send IM messages.
- If the Airflow UI is shown, keep it secondary. The main demo is the terminal failure -> `lark-cue` cue -> cited internal knowledge loop.
