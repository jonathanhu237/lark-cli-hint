## ADDED Requirements

### Requirement: OpenClaw Preflight
The system SHALL verify OpenClaw CLI availability before executing a wrapped command when OpenClaw handoff is enabled.

#### Scenario: Default run checks OpenClaw before command execution
- **WHEN** the user runs `lark-cue run -- <command>` without `--no-openclaw`
- **THEN** the system MUST check that the configured OpenClaw binary can run the agent help command before executing `<command>`

#### Scenario: Missing OpenClaw blocks default run
- **WHEN** OpenClaw handoff is enabled and the OpenClaw preflight fails
- **THEN** the system MUST print a clear error telling the user to install/configure OpenClaw or run with `--no-openclaw`
- **AND** the system MUST NOT execute the wrapped command
- **AND** the system MUST exit with code `2`

#### Scenario: Card-only mode skips OpenClaw preflight
- **WHEN** the user runs `lark-cue run --no-openclaw -- <command>`
- **THEN** the system MUST NOT require OpenClaw preflight before executing `<command>`

### Requirement: OpenClaw Task Message
The system SHALL generate a deterministic OpenClaw task message from the failed command, planner decision, knowledge card, and fetched Feishu evidence.

#### Scenario: Task includes execution context
- **WHEN** the system builds an OpenClaw task message
- **THEN** the task MUST include the current working directory, wrapped command, wrapped command exit code, and captured terminal output excerpt

#### Scenario: Task includes knowledge context
- **WHEN** the system builds an OpenClaw task message after Feishu retrieval
- **THEN** the task MUST include planner scenario, planner reason, planner queries, likely cause or evidence status, action plan, confidence, caveat when present, and Feishu evidence citations/snippets

#### Scenario: Task includes safety and verification constraints
- **WHEN** the system builds an OpenClaw task message
- **THEN** the task MUST instruct OpenClaw to inspect local state before changing files, adapt to actual findings, rerun the failed command or an equivalent verification, and ask before deleting data, changing production configuration, rotating secrets, sending messages, committing code, or performing other risky external side effects

### Requirement: OpenClaw Invocation
The system SHALL invoke OpenClaw after rendering a knowledge card only when the planner has selected the Feishu internal-knowledge path.

#### Scenario: Planner retrieval invokes OpenClaw after card rendering
- **WHEN** a wrapped command exits non-zero, the planner returns `should_retrieve: true`, and the system generates a knowledge card
- **THEN** the system MUST render the knowledge card before invoking OpenClaw
- **AND** the system MUST invoke OpenClaw with the generated task message

#### Scenario: Invocation uses local main agent
- **WHEN** the system invokes OpenClaw in the MVP
- **THEN** the command MUST target the local `main` agent with `openclaw agent --local --agent main --timeout <seconds> --message <task>`
- **AND** the default timeout MUST be 900 seconds

#### Scenario: Planner skip does not invoke OpenClaw
- **WHEN** a wrapped command exits non-zero and the planner returns `should_retrieve: false`
- **THEN** the system MUST NOT invoke OpenClaw

#### Scenario: Successful command does not invoke OpenClaw
- **WHEN** a wrapped command exits with code `0`
- **THEN** the system MUST NOT invoke OpenClaw

#### Scenario: Card-only mode does not invoke OpenClaw
- **WHEN** the user runs with `--no-openclaw`
- **THEN** the system MUST NOT invoke OpenClaw after card generation

### Requirement: OpenClaw Output and Exit Semantics
The system SHALL keep wrapped command semantics intact while streaming OpenClaw execution output visibly.

#### Scenario: OpenClaw output is routed to stderr
- **WHEN** OpenClaw writes stdout or stderr during invocation
- **THEN** the system MUST forward that output to `lark-cue` stderr
- **AND** the system MUST NOT write OpenClaw output to `lark-cue` stdout

#### Scenario: OpenClaw success preserves command failure code
- **WHEN** the wrapped command exits non-zero and OpenClaw invocation succeeds
- **THEN** `lark-cue run` MUST return the original wrapped command exit code

#### Scenario: OpenClaw failure preserves command failure code
- **WHEN** the wrapped command exits non-zero and OpenClaw invocation fails after card generation
- **THEN** the system MUST print an OpenClaw failure diagnostic
- **AND** `lark-cue run` MUST return the original wrapped command exit code

#### Scenario: OpenClaw timeout preserves command failure code
- **WHEN** the wrapped command exits non-zero and OpenClaw invocation exceeds the configured timeout
- **THEN** the system MUST print an OpenClaw timeout diagnostic
- **AND** `lark-cue run` MUST return the original wrapped command exit code
