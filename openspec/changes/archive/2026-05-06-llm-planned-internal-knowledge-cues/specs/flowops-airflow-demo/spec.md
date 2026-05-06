## ADDED Requirements

### Requirement: FlowOps Airflow Demo Environment
The system SHALL provide a reproducible local FlowOps demo environment based on real Airflow that can produce a DAG import failure from an internal scheduling platform CLI workflow.

#### Scenario: Demo environment is documented
- **WHEN** a developer opens the FlowOps demo documentation
- **THEN** the documentation MUST explain one-time setup, demo execution, cleanup, and expected failure output

#### Scenario: Broken DAG reproduces import error
- **WHEN** the demo Airflow environment parses the `billing_daily` DAG without a configured `billing_region` Variable
- **THEN** Airflow MUST report a DAG import error caused by parse-time `Variable.get("billing_region")`

#### Scenario: FlowOps wrapper uses real Airflow
- **WHEN** the user runs the FlowOps demo wrapper
- **THEN** the wrapper MUST call the local Airflow CLI rather than printing a hardcoded fake error

#### Scenario: Demo command is compatible with lark-cue
- **WHEN** the user runs `lark-cue run -- <flowctl command>` with LLM configured
- **THEN** the command failure MUST provide enough terminal context for the LLM planner to generate FlowOps/Airflow keyword queries

### Requirement: FlowOps Feishu Seed Script
The system SHALL provide a dry-run-first script that creates or updates FlowOps demo knowledge documents in Feishu through `lark-cli`.

#### Scenario: Dry run is default
- **WHEN** the user runs the seed script without `--apply`
- **THEN** the script MUST print the planned Feishu write operations and MUST NOT create or update Feishu content

#### Scenario: Apply writes demo documents
- **WHEN** the user runs the seed script with `--apply`
- **THEN** the script MUST use `lark-cli` to create or update the FlowOps demo Markdown documents

#### Scenario: Seeded documents are internal mock knowledge
- **WHEN** the script creates or updates demo documents
- **THEN** the documents MUST be written as 星桥科技 internal FlowOps knowledge and MUST NOT include external Airflow source-reference sections

#### Scenario: Seeded document set is complete
- **WHEN** the seed script applies successfully
- **THEN** Feishu MUST contain documents for FlowOps DAG import error FAQ, `billing_daily` historical incident review, and FlowOps DAG development standards

#### Scenario: No automatic IM side effects
- **WHEN** the seed script runs
- **THEN** it MUST NOT send mock messages to Feishu chats unless a future explicit chat-send option is added and invoked

#### Scenario: Smoke commands are emitted
- **WHEN** the seed script finishes dry-run or apply mode
- **THEN** it MUST print smoke search commands that verify the seeded FlowOps keywords can be found through `lark-cli`

### Requirement: Real Environment E2E Gate
The system SHALL keep real LLM, Feishu, and Airflow integration checks opt-in so default tests remain deterministic while official demo verification can exercise real services.

#### Scenario: E2E is opt-in
- **WHEN** default unit tests run
- **THEN** they MUST NOT require Docker, Airflow, network access, LLM credentials, or Feishu login state

#### Scenario: E2E uses explicit environment
- **WHEN** `LARK_CUE_E2E=1` and required LLM/lark-cli profile settings are present
- **THEN** integration tests MAY exercise the real FlowOps demo, Feishu seed data, retrieval, and card generation path

#### Scenario: Missing E2E configuration skips safely
- **WHEN** E2E tests are requested but required external configuration is missing
- **THEN** tests MUST skip or fail with a clear setup message rather than silently using local fixture data
