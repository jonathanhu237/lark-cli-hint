## MODIFIED Requirements

### Requirement: Command Wrapper
The system SHALL provide a `lark-cue run -- <command>` CLI flow that requires LLM configuration before executing the wrapped command, requires OpenClaw preflight before executing the wrapped command unless `--no-openclaw` is set, streams stdout and stderr to the terminal in real time, captures a bounded output buffer for analysis, and returns the wrapped command exit code after any cue and OpenClaw handling.

#### Scenario: Missing LLM configuration blocks execution
- **WHEN** the user runs `lark-cue run -- <command>` without required LLM configuration
- **THEN** the system MUST print a clear configuration error and MUST NOT execute the wrapped command

#### Scenario: Missing OpenClaw configuration blocks default execution
- **WHEN** the user runs `lark-cue run -- <command>` without `--no-openclaw` and OpenClaw preflight fails
- **THEN** the system MUST print a clear OpenClaw configuration error and MUST NOT execute the wrapped command
- **AND** the system MUST exit with code `2`

#### Scenario: Card-only mode does not require OpenClaw
- **WHEN** the user runs `lark-cue run --no-openclaw -- <command>`
- **THEN** the system MUST NOT require OpenClaw preflight before executing the wrapped command

#### Scenario: Successful command does not trigger a cue
- **WHEN** a wrapped command exits with code `0`
- **THEN** the system MUST return exit code `0` and MUST NOT run the planner, Feishu knowledge retrieval, display a knowledge card, or invoke OpenClaw

#### Scenario: Failed command preserves exit code
- **WHEN** a wrapped command exits non-zero
- **THEN** the system MUST preserve and return the wrapped command exit code even if planning, retrieval, card generation, or post-card OpenClaw invocation succeeds

#### Scenario: Output is streamed while captured
- **WHEN** a wrapped command writes stdout or stderr before exiting
- **THEN** the system MUST display that output to the terminal without waiting for planner analysis and MUST retain only a bounded recent buffer for analysis

### Requirement: LLM Retrieval Planner
The system SHALL use a configured LLM planner to decide whether a failed wrapped command should trigger Feishu internal knowledge retrieval and to produce keyword-style search queries.

#### Scenario: Planner recommends retrieval
- **WHEN** a wrapped command exits non-zero and the LLM planner returns `should_retrieve: true` with one or more queries
- **THEN** the system MUST search Feishu internal knowledge using those planner queries

#### Scenario: Planner skips retrieval
- **WHEN** a wrapped command exits non-zero and the LLM planner returns `should_retrieve: false`
- **THEN** the system MUST NOT call `lark-cli` retrieval, MUST NOT render a knowledge card, MUST NOT invoke OpenClaw, MUST print a short non-intrusive skip message, and MUST preserve the wrapped command exit code

#### Scenario: Planner output is recorded
- **WHEN** the planner returns a decision for a failed command
- **THEN** the system MUST record the planner scenario, reason, `should_retrieve` value, query count, and latency in the evaluation log

#### Scenario: Planner query style
- **WHEN** the planner returns queries
- **THEN** the queries MUST be treated as keyword-style Feishu search phrases and normalized by trimming, dropping empty entries, de-duplicating, and limiting the count
- **AND** each normalized query MUST be capped to Feishu document search limits

### Requirement: Configuration
The system SHALL support environment-variable and local-file configuration, with environment variables taking precedence, SHALL require LLM configuration for `lark-cue run`, SHALL require OpenClaw configuration for default `lark-cue run`, SHALL allow `--no-openclaw` to disable OpenClaw for a run, and SHALL allow runtime Feishu retrieval to use the same `lark-cli` profile used for demo seeding.

#### Scenario: Environment overrides config
- **WHEN** a value is present both in the local config file and in a supported `LARK_CUE_*` environment variable
- **THEN** the system MUST use the environment variable value

#### Scenario: Feishu profile is forwarded
- **WHEN** a Feishu profile is configured for `lark-cue`
- **THEN** runtime retrieval and explicit push sending MUST pass that profile to `lark-cli`

#### Scenario: OpenClaw defaults are available
- **WHEN** OpenClaw configuration is not explicitly set
- **THEN** the system MUST default to binary `openclaw`, agent `main`, local mode enabled, and timeout `900` seconds

#### Scenario: OpenClaw can be disabled per run
- **WHEN** the user provides `--no-openclaw`
- **THEN** the system MUST skip OpenClaw preflight and OpenClaw invocation for that run

#### Scenario: Fixture mode is not available as product path
- **WHEN** the user runs `lark-cue run`
- **THEN** the system MUST NOT expose or silently activate local fixture retrieval

### Requirement: Evaluation Logging
The system SHALL create local evaluation records for planner decisions, cue attempts, and OpenClaw handoff attempts and SHALL NOT interrupt `run` with interactive feedback prompts.

#### Scenario: Planner event is recorded
- **WHEN** the planner decides whether to retrieve internal knowledge for a failed command
- **THEN** the system MUST append a JSONL evaluation record containing command, scenario, reason, `should_retrieve`, query count, planner latency, and created timestamp

#### Scenario: Cue event is recorded
- **WHEN** a knowledge card is generated
- **THEN** the system MUST append a JSONL evaluation record containing card id, command, scenario, retrieval status, cited sources, latency, query count, confidence, feedback state, and OpenClaw handoff status

#### Scenario: OpenClaw result is recorded when attempted
- **WHEN** OpenClaw handoff is attempted for a cue
- **THEN** the cue evaluation record MUST include whether OpenClaw was attempted, whether it succeeded, latency, and any compact error or timeout status

#### Scenario: OpenClaw skip reason is recorded
- **WHEN** OpenClaw handoff is skipped for a generated cue because `--no-openclaw` is set
- **THEN** the cue evaluation record MUST include that OpenClaw was not attempted and MUST include the skip reason

#### Scenario: Run never prompts for feedback
- **WHEN** a knowledge card is displayed
- **THEN** the system MUST NOT wait for interactive feedback and MUST record feedback as absent or skipped
