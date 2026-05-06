# FlowOps Airflow Demo

This example is a local FlowOps demo for `lark-cue`. FlowOps is 星桥科技's internal scheduling platform in the demo story, backed here by a real local Airflow container.

The broken `billing_daily` DAG calls `Variable.get("billing_region")` while Airflow imports the DAG file. On a fresh local metadata database that Variable does not exist, so Airflow reports a real DAG import error.

## One-Time Setup

From this directory:

```sh
cp .env.example .env
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

After seeding the Feishu demo knowledge with `../../scripts/seed-flowops-feishu-demo`, run:

```sh
lark-cue run -- ./flowctl check billing_daily
```

For the contest recording, keep the command prompt visible and show the generated knowledge card with:

- detected FlowOps/Airflow DAG import scenario;
- likely parse-time Variable lookup cause;
- one next action, such as moving Variable access into task runtime or configuring the missing Variable only as a short-term unblock;
- citations to the 星桥科技 FlowOps mock docs.

## Reset or Cleanup

Stop containers:

```sh
./flowctl down
```

Return the demo to a clean state:

```sh
./flowctl clean
```

To demonstrate a temporary fix manually, set the Variable and rerun the check:

```sh
./flowctl airflow variables set billing_region cn-north
./flowctl check billing_daily
```

Then run `./flowctl clean && ./flowctl init` before recording the broken path again.

## Recording Tips

- Run `./flowctl init` before recording so the first captured command is fast.
- Keep Feishu seed search results ready, because indexing can lag after writes.
- Use a test Feishu profile and avoid real team chats; this demo does not send IM messages.
- If the Airflow UI is shown, keep it secondary. The main demo is the terminal failure -> `lark-cue` cue -> cited internal knowledge loop.
