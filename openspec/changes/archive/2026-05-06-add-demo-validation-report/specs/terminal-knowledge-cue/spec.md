## ADDED Requirements

### Requirement: Evaluation Validation Report
The system SHALL provide a read-only `lark-cue eval report` flow that summarizes recent cue evaluation records in a terminal-readable validation view.

#### Scenario: Report summarizes cue records
- **WHEN** the evaluation log contains recent `type: "cue"` records
- **THEN** the report MUST display run count, retrieval status counts, fixture run count, citation coverage, average source count, average query count, average latency, and feedback counts

#### Scenario: Report distinguishes real retrieval from fixture runs
- **WHEN** cue records contain retrieval statuses such as `ok`, `partial`, `failed`, and `fixture`
- **THEN** the report MUST group and label those statuses so fixture-based demo recovery is not presented as real Feishu retrieval

#### Scenario: Report uses existing evaluation records only
- **WHEN** the user runs `lark-cue eval report`
- **THEN** the system MUST NOT execute wrapped commands, call `lark-cli`, invoke an LLM, send Feishu messages, or modify evaluation records

#### Scenario: Missing or empty log is handled
- **WHEN** the configured evaluation log is missing or contains no cue records
- **THEN** the report MUST render a clear empty-state message and exit successfully

#### Scenario: Malformed log lines are tolerated
- **WHEN** the evaluation log contains malformed JSONL lines mixed with valid cue records
- **THEN** the report MUST skip malformed lines, summarize valid cue records, and include a warning count

#### Scenario: Recent record limit is respected
- **WHEN** the user provides a report limit
- **THEN** the report MUST summarize only the most recent cue records up to that limit

#### Scenario: Report output follows terminal behavior
- **WHEN** report output is written to an interactive terminal
- **THEN** the system MAY render a styled validation card
- **WHEN** report output is redirected, `NO_COLOR` is set, or the terminal is not style-capable
- **THEN** the system MUST produce readable plain text without ANSI styling
