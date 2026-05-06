# terminal-knowledge-cue Specification

## Purpose
`terminal-knowledge-cue` covers the `lark-cue` terminal workflow: wrapping developer commands, requiring LLM-planned failed-command knowledge decisions, retrieving cited Feishu knowledge through `lark-cli`, rendering actionable knowledge cards, handing generated cards to OpenClaw by default, controlling optional Feishu push side effects, and recording planner/cue/OpenClaw evaluation data for demo validation.

## Requirements
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
The system SHALL fetch or read candidate search results before using them as evidence and SHALL score fetched content for support of the planner scenario, command failure, and actionable troubleshooting context.

#### Scenario: Search result title alone is insufficient
- **WHEN** a candidate result appears in search results but its content cannot be fetched or read
- **THEN** the system MUST NOT use that result as a supporting citation for a confident knowledge card

#### Scenario: Strong evidence enables a confident cause
- **WHEN** fetched evidence snippets support both the planner scenario and relevant troubleshooting context
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
- **THEN** the card MUST include detected scenario, likely cause or evidence status, an ordered action plan, evidence sources when available, and confidence or caveat

#### Scenario: Card source claims are evidence grounded
- **WHEN** the LLM generates a card
- **THEN** the card MUST NOT cite sources that were not fetched/read and MUST keep likely-cause and caveat claims grounded in the provided evidence snippets

#### Scenario: Weak-evidence card is explicit
- **WHEN** evidence is weak or absent
- **THEN** the card MUST say that internal evidence is insufficient and SHOULD include compact retrieval context such as attempted query count or retrieval error

#### Scenario: Template fallback is used
- **WHEN** LLM card generation is unavailable or invalid after successful planning
- **THEN** the system MUST use a deterministic template card based on planner output, scored evidence, and source metadata

### Requirement: Source Citations
The system SHALL cite Feishu sources with enough metadata to support traceability while keeping terminal output compact.

#### Scenario: Document citation
- **WHEN** a Docs/Wiki source supports the card
- **THEN** the citation MUST include the source title and available URL or identifier

#### Scenario: IM citation
- **WHEN** an IM message supports the card
- **THEN** the citation MUST include the group name when available, speaker when available, timestamp when available, and a short summary rather than a long raw message transcript

### Requirement: Feishu Push Control
The system SHALL provide terminal card delivery as the default required path and SHALL make Feishu group push side effects explicit.

#### Scenario: Push is prepared by request
- **WHEN** a user requests group push preparation without the explicit send flag
- **THEN** the system MUST render or store a preview of the group message and MUST NOT send it

#### Scenario: Push requires explicit send
- **WHEN** a user provides the explicit send flag and a target can be resolved
- **THEN** the system MAY send the prepared message through `lark-cli`

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

### Requirement: Evaluation Validation Report
The system SHALL provide a read-only `lark-cue eval report` flow that summarizes recent planner and cue evaluation records in a terminal-readable validation view.

#### Scenario: Report summarizes cue records
- **WHEN** the evaluation log contains recent `type: "cue"` records
- **THEN** the report MUST display run count, retrieval status counts, citation coverage, average source count, average query count, and average latency

#### Scenario: Report summarizes OpenClaw handoffs
- **WHEN** cue records contain OpenClaw handoff fields
- **THEN** the report MUST display OpenClaw attempted, succeeded, failed, skipped, and average OpenClaw latency metrics

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
