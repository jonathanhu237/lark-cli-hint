# cli-runner Specification

## Purpose
Provide the executable foundation for `lark-cli-hint run`, including wrapped command execution, output capture, Hint Card rendering, JSON envelope output, and locale-aware user-facing messages.

## Requirements
### Requirement: Run command accepts wrapped lark-cli invocation

The system SHALL provide a `lark-cli-hint run` command that accepts a wrapped `lark-cli` invocation after `--`.

#### Scenario: Wrapped command is provided

- **WHEN** the user runs `lark-cli-hint run -- lark-cli --version`
- **THEN** the system executes the wrapped `lark-cli --version` command

#### Scenario: Wrapped command is missing

- **WHEN** the user runs `lark-cli-hint run` without a wrapped command
- **THEN** the system exits with an error explaining that a command must be provided after `--`

### Requirement: Human mode streams output and appends Hint Card

The system SHALL stream the wrapped command's terminal output in human mode and append a Hint Card after the wrapped command exits.

#### Scenario: Successful wrapped command in human mode

- **WHEN** the wrapped command exits successfully
- **THEN** the user sees the wrapped command output followed by a Hint Card with `Status`, `Hint`, `Next`, `Why`, and `Sources` sections

#### Scenario: Failed wrapped command in human mode

- **WHEN** the wrapped command exits with a non-zero status
- **THEN** the user sees the wrapped command output followed by a Hint Card that reports the failure status

### Requirement: JSON mode emits one envelope

The system SHALL support `--json` mode that emits a single JSON envelope instead of intermixing human-readable terminal prose.

#### Scenario: Wrapped command completes in JSON mode

- **WHEN** the user runs `lark-cli-hint run --json -- lark-cli --version`
- **THEN** stdout contains one valid JSON object with command metadata, exit status, captured stdout, captured stderr, and hint data

#### Scenario: Wrapped command fails in JSON mode

- **WHEN** the wrapped command exits with a non-zero status in `--json` mode
- **THEN** the JSON envelope includes the non-zero exit status and failure-oriented hint data

### Requirement: Baseline analyzer produces conservative hints

The system SHALL produce a baseline hint when no domain-specific Recover or Next rule is available.

#### Scenario: No domain rule is available for successful command

- **WHEN** a wrapped command succeeds and no domain-specific rule matches
- **THEN** the Hint Card reports that the command completed and does not invent a confident next command

#### Scenario: No domain rule is available for failed command

- **WHEN** a wrapped command fails and no domain-specific rule matches
- **THEN** the Hint Card reports that the command failed and cites the exit status or captured stderr as evidence

### Requirement: User-facing output is locale-aware

The system SHALL render user-facing hint text in English by default and in Simplified Chinese when the user's environment indicates Chinese.

#### Scenario: Non-Chinese locale environment

- **WHEN** the locale environment does not indicate Chinese
- **THEN** terminal Hint Card labels and baseline messages are rendered in English

#### Scenario: Chinese locale environment

- **WHEN** `LANG`, `LC_ALL`, or `LC_MESSAGES` indicates a Chinese locale
- **THEN** terminal Hint Card labels and baseline messages are rendered in Simplified Chinese

#### Scenario: JSON protocol fields remain stable

- **WHEN** output is rendered in any supported locale
- **THEN** JSON envelope field names remain stable and are not localized
