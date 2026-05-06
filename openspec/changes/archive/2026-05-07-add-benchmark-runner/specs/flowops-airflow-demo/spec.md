## ADDED Requirements

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
