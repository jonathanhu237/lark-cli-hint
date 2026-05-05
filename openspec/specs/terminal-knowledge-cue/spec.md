# terminal-knowledge-cue Specification

## Purpose
TBD - created by archiving change build-feishu-scope-error-cue-mvp. Update Purpose after archive.
## Requirements
### Requirement: Command Wrapper
The system SHALL provide a `lark-cue run -- <command>` CLI flow that executes the wrapped command, streams stdout and stderr to the terminal in real time, captures a bounded output buffer for analysis, and returns the wrapped command exit code.

#### Scenario: Successful command does not trigger a cue
- **WHEN** a wrapped command exits with code `0`
- **THEN** the system MUST return exit code `0` and MUST NOT run Feishu knowledge retrieval or display a knowledge card

#### Scenario: Failed command preserves exit code
- **WHEN** a wrapped command exits with code `1`
- **THEN** the system MUST preserve and return exit code `1` even if cue detection, retrieval, or card generation succeeds

#### Scenario: Output is streamed while captured
- **WHEN** a wrapped command writes stdout or stderr before exiting
- **THEN** the system MUST display that output to the terminal without waiting for cue analysis and MUST retain only a bounded recent buffer for analysis

### Requirement: Feishu Error Detection
The system SHALL analyze captured command output after a non-zero exit and identify the MVP Feishu API auth/scope/token error scenario when matching permission, scope, or token failure patterns are present.

#### Scenario: Scope error is detected
- **WHEN** captured output contains `missing required scope: docx:document:read`
- **THEN** the system MUST classify the scenario as a Feishu API scope/permission error

#### Scenario: Token and permission error is detected
- **WHEN** captured output contains `tenant_access_token invalid` or `permission denied`
- **THEN** the system MUST classify the output as a candidate Feishu API auth/token/permission error

#### Scenario: Unrelated command failure is ignored
- **WHEN** a command exits non-zero but captured output does not match Feishu API auth/scope/token patterns
- **THEN** the system MUST NOT search Feishu knowledge sources or display a knowledge card

### Requirement: Query Generation
The system SHALL generate search queries from the command and captured output using deterministic seed extraction plus constrained LLM query expansion when an LLM provider is configured.

#### Scenario: Seed queries are retained
- **WHEN** captured output includes exact phrases such as `docx:document:read`, `missing required scope`, `tenant_access_token invalid`, or `permission denied`
- **THEN** the system MUST include those extracted phrases, subject to deduplication and length limits, in the final query set

#### Scenario: LLM expansion is constrained
- **WHEN** the system invokes the LLM to expand queries
- **THEN** the LLM prompt MUST include only the wrapped command, captured output, and query-generation instructions, and MUST NOT include known mock document titles, URLs, chat IDs, or final-answer background

#### Scenario: LLM is unavailable
- **WHEN** no LLM provider is configured or LLM query expansion fails
- **THEN** the system MUST continue retrieval using deterministic seed queries

### Requirement: Real Feishu Retrieval
The system SHALL use `lark-cli` at runtime to search real Feishu Docs/Wiki and IM messages for candidate evidence, without pinning known demo document titles, URLs, or chat IDs as the retrieval path.

#### Scenario: Docs and IM are searched
- **WHEN** a Feishu API error scenario is detected and queries are generated
- **THEN** the system MUST search both Docs/Wiki and IM message sources through `lark-cli`

#### Scenario: Runtime retrieval is not pinned
- **WHEN** the system performs primary retrieval
- **THEN** it MUST NOT start from a hardcoded list of known source titles, document URLs, or chat IDs for the expected demo answer

#### Scenario: Retrieval failure is transparent
- **WHEN** `lark-cli` authentication, network access, or command execution fails
- **THEN** the system MUST report that real Feishu retrieval failed and MUST NOT produce a confident card that implies real evidence was found

### Requirement: Evidence Fetching and Scoring
The system SHALL fetch or read candidate search results before using them as evidence and SHALL score fetched content with transparent keyword-based evidence rules.

#### Scenario: Search result title alone is insufficient
- **WHEN** a candidate result appears in search results but its content cannot be fetched or read
- **THEN** the system MUST NOT use that result as a supporting citation for a confident knowledge card

#### Scenario: Strong evidence enables a confident cause
- **WHEN** fetched content contains a strong combination of error signals and cause/action signals such as missing scope plus permission publication or re-authorization guidance
- **THEN** the system MAY generate a high-confidence likely cause grounded in that evidence

#### Scenario: Weak evidence produces a low-confidence card
- **WHEN** fetched content has only weak or partial support for the detected scenario
- **THEN** the system MUST produce a low-confidence card that states evidence is limited and avoids a definitive cause

#### Scenario: Unrelated content is filtered
- **WHEN** fetched content is about an unrelated topic and does not meet the evidence threshold for the detected error
- **THEN** the system MUST exclude it from cited sources

### Requirement: Knowledge Card Generation
The system SHALL generate a compact terminal knowledge card using only the wrapped command, captured output, fetched evidence snippets, source metadata, and card format contract.

#### Scenario: Card contains required sections
- **WHEN** sufficient evidence exists
- **THEN** the card MUST include detected scenario, likely cause or relevant knowledge, one recommended next action, evidence sources, and confidence or caveat

#### Scenario: Card is evidence grounded
- **WHEN** the LLM generates a card
- **THEN** the card MUST NOT cite sources that were not fetched/read and MUST NOT introduce claims unsupported by the provided evidence snippets

#### Scenario: Template fallback is used
- **WHEN** LLM card generation is unavailable or invalid
- **THEN** the system MUST use a deterministic template card based on scored evidence and source metadata

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

#### Scenario: Push is prepared by default
- **WHEN** a user requests group push preparation without the explicit send flag
- **THEN** the system MUST render or store a preview of the group message and MUST NOT send it

#### Scenario: Push requires explicit send
- **WHEN** a user provides the explicit send flag and a target can be resolved
- **THEN** the system MAY send the prepared message through `lark-cli`

### Requirement: Feedback and Evaluation Logging
The system SHALL create local evaluation records for cue attempts and SHALL support useful/not useful feedback collection.

#### Scenario: Evaluation event is recorded
- **WHEN** a cue analysis is attempted
- **THEN** the system MUST append a JSONL evaluation record containing card id when available, command, scenario, retrieval status, cited sources, latency, and feedback state

#### Scenario: Interactive feedback
- **WHEN** a knowledge card is displayed in an interactive TTY session
- **THEN** the system MUST allow the user to mark the card useful, not useful, or skipped

#### Scenario: Non-interactive execution does not block
- **WHEN** the command runs in a non-TTY context or feedback prompting is disabled
- **THEN** the system MUST NOT wait for interactive feedback and MUST record feedback as absent or skipped

#### Scenario: Feedback command updates an event
- **WHEN** the user runs a feedback command with a known card id and useful/not-useful value
- **THEN** the system MUST update or append evaluation state so the feedback can be included in validation reports

### Requirement: Configuration and Fixture Mode
The system SHALL support environment-variable and local-file configuration, with environment variables taking precedence, and SHALL allow fixture retrieval only through an explicit demo flag.

#### Scenario: Environment overrides config
- **WHEN** a value is present both in the local config file and in a supported `LARK_CUE_*` environment variable
- **THEN** the system MUST use the environment variable value

#### Scenario: Fixture mode is explicit
- **WHEN** the user runs with `--demo-fixture`
- **THEN** the system MAY use local fixture data and MUST label the resulting retrieval/card output as fixture or simulated content

#### Scenario: Fixture mode is not silent
- **WHEN** real Feishu retrieval fails without `--demo-fixture`
- **THEN** the system MUST NOT silently fall back to local fixture data

