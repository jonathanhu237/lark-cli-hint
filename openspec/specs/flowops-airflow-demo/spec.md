# flowops-airflow-demo Specification

## Purpose
`flowops-airflow-demo` covers the reproducible local FlowOps demo environment, the Feishu seed script for internal mock knowledge, benchmark case metadata, and the opt-in gates for real LLM/Feishu/Airflow E2E validation.

## Requirements
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
The system SHALL provide a dry-run-first script that creates, updates, and repairs a complete FlowOps demo team Wiki in Feishu through `lark-cli`.

#### Scenario: Dry run is default
- **WHEN** the user runs the seed script without `--apply`
- **THEN** the script MUST print the planned Feishu Wiki and document write operations
- **AND** the script MUST NOT create or update Feishu content

#### Scenario: Apply requires explicit seed profile
- **WHEN** the user runs the seed script with `--apply`
- **THEN** the script MUST resolve a Feishu test profile from CLI `--profile` or local `[seed].feishu_profile` configuration
- **AND** the script MUST fail before any write when no seed profile is resolved

#### Scenario: Apply creates or reuses target team Wiki
- **WHEN** the user runs the seed script with `--apply`
- **THEN** the script MUST use `lark-cli` to find an existing team Wiki by the configured Wiki name
- **AND** the script MUST create the team Wiki when no exact match exists
- **AND** the script MUST NOT write to `my_library` or another personal document library

#### Scenario: Wiki seed is manifest-backed
- **WHEN** the seed script runs
- **THEN** the script MUST read the desired Wiki tree from a repository JSON manifest
- **AND** page content MUST come from repository Markdown files referenced by that manifest
- **AND** the Markdown content and titles MUST NOT expose demo-only prefixes or seed-management markers

#### Scenario: Seeded Wiki structure is complete
- **WHEN** the seed script applies successfully
- **THEN** the target Wiki MUST contain content-bearing pages for `FlowOps 调度平台`, `DAG 发布与巡检`, and `历史故障复盘`
- **AND** the target Wiki MUST contain documents for FlowOps DAG import error FAQ, `billing_daily` historical incident review, and FlowOps DAG development standards
- **AND** those pages MUST be arranged according to the manifest-defined parent-child structure

#### Scenario: Apply is idempotent
- **WHEN** the user runs the seed script repeatedly against the same profile and Wiki
- **THEN** the script MUST update managed page content in place instead of creating duplicate pages
- **AND** the script MUST move managed pages to their manifest-defined parent when they exist in the target Wiki under a different parent
- **AND** the script MUST preserve pages that are not listed in the manifest

#### Scenario: No automatic IM or pruning side effects
- **WHEN** the seed script runs
- **THEN** it MUST NOT send mock messages to Feishu chats unless a future explicit chat-send option is added and invoked
- **AND** it MUST NOT delete or prune pages outside the manifest

#### Scenario: Smoke commands are emitted
- **WHEN** the seed script finishes dry-run or apply mode
- **THEN** it MUST print smoke search commands that verify the seeded FlowOps keywords can be found through `lark-cli`

### Requirement: FlowOps Benchmark Cases
The FlowOps demo SHALL include benchmark case metadata that maps real FlowOps command failures to the seeded Wiki sources expected to be cited.

#### Scenario: Eval cases file exists
- **WHEN** the repository contains the FlowOps demo seed assets
- **THEN** it MUST include `examples/flowops-airflow/seed/eval-cases.json`

#### Scenario: Eval cases use real commands
- **WHEN** a FlowOps benchmark case is defined
- **THEN** it MUST use a real `flowctl` command array and MUST NOT use fixture output

#### Scenario: Eval cases declare expected sources
- **WHEN** a FlowOps benchmark case targets a seeded troubleshooting scenario
- **THEN** it MUST declare the seeded Wiki source titles expected to be cited by the generated knowledge card

#### Scenario: FlowOps benchmark command is documented
- **WHEN** a developer opens the FlowOps demo documentation
- **THEN** the documentation MUST show how to run `lark-cue benchmark run --cases examples/flowops-airflow/seed/eval-cases.json`

#### Scenario: Demo benchmark remains explicit
- **WHEN** default tests or normal demo setup commands run
- **THEN** they MUST NOT automatically run the benchmark because it depends on real Docker/Airflow, LLM, and Feishu access

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
