## ADDED Requirements

### Requirement: LLM Retrieval Planner
The system SHALL use a configured LLM planner to decide whether a failed wrapped command should trigger Feishu internal knowledge retrieval and to produce keyword-style search queries.

#### Scenario: Planner recommends retrieval
- **WHEN** a wrapped command exits non-zero and the LLM planner returns `should_retrieve: true` with one or more queries
- **THEN** the system MUST search Feishu internal knowledge using those planner queries

#### Scenario: Planner skips retrieval
- **WHEN** a wrapped command exits non-zero and the LLM planner returns `should_retrieve: false`
- **THEN** the system MUST NOT call `lark-cli` retrieval, MUST NOT render a knowledge card, MUST print a short non-intrusive skip message, and MUST preserve the wrapped command exit code

#### Scenario: Planner output is recorded
- **WHEN** the planner returns a decision for a failed command
- **THEN** the system MUST record the planner scenario, reason, `should_retrieve` value, query count, and latency in the evaluation log

#### Scenario: Planner query style
- **WHEN** the planner returns queries
- **THEN** the queries MUST be treated as keyword-style Feishu search phrases and normalized only by trimming, dropping empty entries, de-duplicating, and limiting the count

## MODIFIED Requirements

### Requirement: Command Wrapper
The system SHALL provide a `lark-cue run -- <command>` CLI flow that requires LLM configuration before executing the wrapped command, streams stdout and stderr to the terminal in real time, captures a bounded output buffer for analysis, and returns the wrapped command exit code after any cue handling.

#### Scenario: Missing LLM configuration blocks execution
- **WHEN** the user runs `lark-cue run -- <command>` without required LLM configuration
- **THEN** the system MUST print a clear configuration error and MUST NOT execute the wrapped command

#### Scenario: Successful command does not trigger a cue
- **WHEN** a wrapped command exits with code `0`
- **THEN** the system MUST return exit code `0` and MUST NOT run the planner, Feishu knowledge retrieval, or display a knowledge card

#### Scenario: Failed command preserves exit code
- **WHEN** a wrapped command exits non-zero
- **THEN** the system MUST preserve and return the wrapped command exit code even if planning, retrieval, or card generation succeeds

#### Scenario: Output is streamed while captured
- **WHEN** a wrapped command writes stdout or stderr before exiting
- **THEN** the system MUST display that output to the terminal without waiting for planner analysis and MUST retain only a bounded recent buffer for analysis

### Requirement: Real Feishu Retrieval
The system SHALL use `lark-cli` at runtime to search real Feishu Docs/Wiki/Sheets and IM messages for candidate evidence using planner-generated keyword queries, without pinning known demo document titles, URLs, chat IDs, or final answers as the retrieval path.

#### Scenario: Docs/Wiki and IM routes are searched
- **WHEN** the LLM planner recommends retrieval and provides queries
- **THEN** the system MUST search the Docs/Wiki/Sheets route through `lark-cli docs +search` and the IM route through `lark-cli im +messages-search`

#### Scenario: Unified queries are used across routes
- **WHEN** planner queries are available
- **THEN** the system MUST use the same normalized query list for both retrieval routes

#### Scenario: Runtime retrieval is not pinned
- **WHEN** the system performs primary retrieval
- **THEN** it MUST NOT start from a hardcoded list of known source titles, document URLs, chat IDs, or expected demo answers

#### Scenario: Retrieval failure is transparent
- **WHEN** `lark-cli` authentication, network access, or command execution fails
- **THEN** the system MUST report that real Feishu retrieval failed and MUST NOT produce a confident card that implies real evidence was found

### Requirement: Evidence Fetching and Scoring
The system SHALL fetch or read candidate search results before using them as evidence and SHALL score fetched content for support of the planner scenario, command failure, and actionable next-step guidance.

#### Scenario: Search result title alone is insufficient
- **WHEN** a candidate result appears in search results but its content cannot be fetched or read
- **THEN** the system MUST NOT use that result as a supporting citation for a confident knowledge card

#### Scenario: Strong evidence enables a confident cause
- **WHEN** fetched evidence snippets support both the planner scenario and a concrete next action
- **THEN** the system MAY generate a high-confidence likely cause grounded in that evidence

#### Scenario: Weak evidence produces a low-confidence card
- **WHEN** fetched content has only weak or partial support for the planner scenario
- **THEN** the system MUST produce a low-confidence card that states evidence is limited and avoids a definitive cause

#### Scenario: No evidence is transparent
- **WHEN** retrieval returns no fetched evidence that supports the planner scenario
- **THEN** the system MUST render a low-confidence card that explains no strong internal evidence was found and MUST NOT invent a cause

#### Scenario: Unrelated content is filtered
- **WHEN** fetched content is about an unrelated topic and does not support the planner scenario or command failure
- **THEN** the system MUST exclude it from cited sources

### Requirement: Knowledge Card Generation
The system SHALL generate a compact terminal knowledge card using only the wrapped command, captured output, planner scenario, planner queries, fetched evidence snippets, source metadata, and card format contract.

#### Scenario: Card contains required sections
- **WHEN** the planner recommends retrieval
- **THEN** the card MUST include detected scenario, likely cause or evidence status, one recommended next action when supported, evidence sources when available, and confidence or caveat

#### Scenario: Card is evidence grounded
- **WHEN** the LLM generates a card
- **THEN** the card MUST NOT cite sources that were not fetched/read and MUST NOT introduce claims unsupported by the provided evidence snippets

#### Scenario: Weak-evidence card is explicit
- **WHEN** evidence is weak or absent
- **THEN** the card MUST say that internal evidence is insufficient and SHOULD include compact retrieval context such as attempted query count or retrieval error

#### Scenario: Template fallback is used
- **WHEN** LLM card generation is unavailable or invalid after successful planning
- **THEN** the system MUST use a deterministic template card based on planner output, scored evidence, and source metadata

### Requirement: Feedback and Evaluation Logging
The system SHALL create local evaluation records for planner decisions and cue attempts and SHALL support useful/not useful feedback collection for generated knowledge cards.

#### Scenario: Planner event is recorded
- **WHEN** the planner decides whether to retrieve internal knowledge for a failed command
- **THEN** the system MUST append a JSONL evaluation record containing command, scenario, reason, `should_retrieve`, query count, planner latency, and created timestamp

#### Scenario: Cue event is recorded
- **WHEN** a knowledge card is generated
- **THEN** the system MUST append a JSONL evaluation record containing card id, command, scenario, retrieval status, cited sources, latency, query count, confidence, and feedback state

#### Scenario: Interactive feedback
- **WHEN** a knowledge card is displayed in an interactive TTY session
- **THEN** the system MUST allow the user to mark the card useful, not useful, or skipped

#### Scenario: Non-interactive execution does not block
- **WHEN** the command runs in a non-TTY context or feedback prompting is disabled
- **THEN** the system MUST NOT wait for interactive feedback and MUST record feedback as absent or skipped

#### Scenario: Feedback command updates an event
- **WHEN** the user runs a feedback command with a known card id and useful/not-useful value
- **THEN** the system MUST update or append evaluation state so the feedback can be included in validation reports

### Requirement: Evaluation Validation Report
The system SHALL provide a read-only `lark-cue eval report` flow that summarizes recent planner and cue evaluation records in a terminal-readable validation view.

#### Scenario: Report summarizes cue records
- **WHEN** the evaluation log contains recent `type: "cue"` records
- **THEN** the report MUST display run count, retrieval status counts, citation coverage, average source count, average query count, average latency, and feedback counts

#### Scenario: Report summarizes planner decisions
- **WHEN** the evaluation log contains recent planner decision records
- **THEN** the report MUST display planner decision count and retrieve versus skip counts

#### Scenario: Report uses existing evaluation records only
- **WHEN** the user runs `lark-cue eval report`
- **THEN** the system MUST NOT execute wrapped commands, call `lark-cli`, invoke an LLM, send Feishu messages, or modify evaluation records

#### Scenario: Missing or empty log is handled
- **WHEN** the configured evaluation log is missing or contains no cue or planner records
- **THEN** the report MUST render a clear empty-state message and exit successfully

#### Scenario: Malformed log lines are tolerated
- **WHEN** the evaluation log contains malformed JSONL lines mixed with valid records
- **THEN** the report MUST skip malformed lines, summarize valid records, and include a warning count

#### Scenario: Recent record limit is respected
- **WHEN** the user provides a report limit
- **THEN** the report MUST summarize only the most recent relevant records up to that limit

#### Scenario: Report output follows terminal behavior
- **WHEN** report output is written to an interactive terminal
- **THEN** the system MAY render a styled validation card
- **WHEN** report output is redirected, `NO_COLOR` is set, or the terminal is not style-capable
- **THEN** the system MUST produce readable plain text without ANSI styling

### Requirement: Runtime Feishu Profile
The system SHALL allow `lark-cue run` to pass a configured `lark-cli` profile to real Feishu retrieval and explicit push sending.

#### Scenario: Profile is shared with retrieval
- **WHEN** a Feishu profile is configured for `lark-cue run`
- **THEN** the system MUST pass that profile to `lark-cli` retrieval commands

#### Scenario: Profile is shared with explicit push sending
- **WHEN** a Feishu profile is configured and the user explicitly requests push sending
- **THEN** the system MUST pass that profile to the `lark-cli` push command

## REMOVED Requirements

### Requirement: Feishu Error Detection
**Reason**: Hardcoded Feishu API scope/token detection makes the product a scenario-specific rules helper instead of a general internal knowledge cue assistant.

**Migration**: Use the LLM Retrieval Planner to decide whether any failed command should retrieve Feishu knowledge.

### Requirement: Query Generation
**Reason**: Deterministic Feishu API seed extraction and LLM query expansion are replaced by a planner that jointly decides retrieval and emits keyword queries.

**Migration**: Use planner queries as the retrieval input after minimal normalization.

### Requirement: Configuration and Fixture Mode
**Reason**: Public fixture mode conflicts with the new product contract that requires LLM configuration and real `lark-cli` retrieval.

**Migration**: Remove public `--demo-fixture`; use mocked dependencies in unit tests and opt-in real Feishu E2E for integration coverage.
