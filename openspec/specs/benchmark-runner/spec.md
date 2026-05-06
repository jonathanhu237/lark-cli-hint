# benchmark-runner Specification

## Purpose
`benchmark-runner` covers explicit real-command benchmark execution for `lark-cue`, using isolated evaluation logs to score whether final knowledge-card citations recall expected seeded internal sources.

## Requirements
### Requirement: Benchmark Case Loading
The system SHALL load benchmark cases from an explicit JSON file passed to `lark-cue benchmark run --cases <path>`.

#### Scenario: Cases path is required
- **WHEN** the user runs `lark-cue benchmark run` without `--cases`
- **THEN** the system MUST print a clear usage error and exit with code `2`

#### Scenario: Cases file is validated
- **WHEN** the cases file is missing, malformed JSON, or contains invalid case entries
- **THEN** the system MUST fail before running any benchmark command and exit with code `2`

#### Scenario: Case command is real command array
- **WHEN** a benchmark case is loaded
- **THEN** the case MUST define a non-empty `command` array and MUST NOT use fixture output as the benchmark input

### Requirement: Benchmark Real Command Execution
The system SHALL execute each benchmark case through real local commands and the same `lark-cue run` behavior used by normal users.

#### Scenario: Setup commands run before case command
- **WHEN** a case defines one or more `setup` commands
- **THEN** the system MUST execute those commands in order before running the case command

#### Scenario: Case command runs through lark-cue
- **WHEN** a case command is executed
- **THEN** the system MUST run it as the wrapped command path equivalent to `lark-cue run -- <command>`

#### Scenario: Benchmark card-only mode skips OpenClaw
- **WHEN** the user runs `lark-cue benchmark run --no-openclaw --cases <path>`
- **THEN** each benchmark case MUST run through the wrapped command path with OpenClaw preflight and post-card handoff disabled

#### Scenario: Teardown runs after case attempt
- **WHEN** a case defines one or more `teardown` commands
- **THEN** the system MUST attempt those commands after the case attempt even when the case fails

#### Scenario: Expected failure is enforced
- **WHEN** a case defines `expect_failure: true`
- **THEN** the case MUST fail scoring if the wrapped command exits with code `0`

#### Scenario: Benchmark runs all cases
- **WHEN** one benchmark case fails
- **THEN** the system MUST continue running remaining cases before producing the final report

### Requirement: Benchmark Evaluation Isolation
The system SHALL isolate benchmark evaluation records from the user's normal evaluation log.

#### Scenario: Temporary evaluation log is used
- **WHEN** benchmark execution starts
- **THEN** the system MUST create or choose a benchmark-specific temporary evaluation log and use it for all case runs

#### Scenario: Normal evaluation log is not polluted
- **WHEN** benchmark cases run
- **THEN** the system MUST NOT append benchmark planner or cue records to the user's configured normal evaluation log

### Requirement: Benchmark Scoring
The system SHALL score benchmark cases from the final knowledge card citations written by the normal cue pipeline.

#### Scenario: Expected source title hit
- **WHEN** a case finishes and a cue card record is available
- **THEN** the system MUST compare citation titles against `expected_sources` using exact title matching

#### Scenario: Minimum expected hits controls pass
- **WHEN** the number of matched expected sources is at least `min_expected_hits`
- **THEN** the source-hit portion of the case MUST pass

#### Scenario: Missing cue record fails case
- **WHEN** a case expects retrieval but no cue card record is produced
- **THEN** the case MUST fail and report that no scored card was available

#### Scenario: Citation precision is reported
- **WHEN** the benchmark report is rendered
- **THEN** it MUST include how many cited sources were expected sources among all cited sources

#### Scenario: Source coverage is reported
- **WHEN** the benchmark report is rendered
- **THEN** it MUST include how many distinct expected sources were cited across all cases

### Requirement: Benchmark Report and Exit Codes
The system SHALL render a compact benchmark report and use stable exit codes.

#### Scenario: Passing benchmark exits zero
- **WHEN** all benchmark cases pass
- **THEN** the command MUST exit with code `0`

#### Scenario: Failing benchmark exits one
- **WHEN** at least one benchmark case fails after all cases have run
- **THEN** the command MUST exit with code `1`

#### Scenario: Configuration error exits two
- **WHEN** the benchmark cannot load or validate cases or runner options
- **THEN** the command MUST exit with code `2`

#### Scenario: Report includes aggregate metrics
- **WHEN** benchmark execution finishes
- **THEN** the report MUST include total cases, passed cases, expected-source hit rate, source coverage, citation precision, and average latency

#### Scenario: Report includes per-case details
- **WHEN** benchmark execution finishes
- **THEN** the report MUST include each case id, pass/fail status, command, expected source hit count, cited source titles, planner retrieval status, and query count when available

#### Scenario: Verbose output is optional
- **WHEN** the user does not pass `--verbose`
- **THEN** the report MUST avoid dumping full wrapped command output
- **WHEN** the user passes `--verbose`
- **THEN** the report MAY include additional per-case output or diagnostics needed for debugging
