# docs-workflow-hints Specification

## Purpose
Provide deterministic Recover and Next hints for Feishu docs/wiki workflows, including docs search follow-ups, docs fetch group-message preparation, and recoveries for common docs fetch failures.
## Requirements
### Requirement: Docs search success suggests document fetch

The system SHALL detect successful `lark-cli docs +search` executions with a usable document candidate and suggest one `lark-cli docs +fetch --doc <document>` command.

#### Scenario: Search result contains document token

- **WHEN** the wrapped command is `lark-cli docs +search` and stdout contains a successful JSON result with a document token or URL
- **THEN** the Hint Card `Next` section contains `lark-cli docs +fetch --doc <document>`

#### Scenario: Real search result contains nested result metadata URL

- **WHEN** the wrapped command is `lark-cli docs +search` and stdout contains a successful JSON result with `results[].result_meta.url`
- **THEN** the Hint Card `Next` section contains `lark-cli docs +fetch --doc <url>`

#### Scenario: Real search result contains nested result metadata token

- **WHEN** the wrapped command is `lark-cli docs +search` and stdout contains a successful JSON result with `results[].result_meta.token` but no candidate URL
- **THEN** the Hint Card `Next` section contains `lark-cli docs +fetch --doc <token>`

#### Scenario: Search result has no usable document candidate

- **WHEN** the wrapped command is `lark-cli docs +search` and stdout does not contain a usable document token or URL
- **THEN** the system falls back to the conservative baseline hint and does not invent a fetch command

### Requirement: Docs fetch success suggests group message preparation

The system SHALL detect successful `lark-cli docs +fetch` executions and suggest one `lark-cli im +messages-send` command that prepares a group message without executing it.

#### Scenario: Document content fetched successfully

- **WHEN** the wrapped command is `lark-cli docs +fetch` and exits successfully
- **THEN** the Hint Card `Next` section contains an `lark-cli im +messages-send --chat-id <chat_id> --markdown <message>` command

#### Scenario: Suggested group message uses placeholder target when chat is unknown

- **WHEN** no safe chat ID is configured for a successful document fetch
- **THEN** the suggested `im +messages-send` command uses an explicit placeholder instead of inventing a chat ID

### Requirement: Docs fetch failure produces Recover hint

The system SHALL detect failed `lark-cli docs +fetch` executions and produce a Recover-oriented Hint Card when the failure evidence matches known docs workflow problems.

#### Scenario: Fetch command uses outdated doc-token flag

- **WHEN** the wrapped command is `lark-cli docs +fetch` and includes `--doc-token`
- **THEN** the Hint Card explains that current `lark-cli docs +fetch` expects `--doc` and suggests a corrected command shape

#### Scenario: Fetch command is missing doc argument

- **WHEN** the wrapped command is `lark-cli docs +fetch` and does not include `--doc` or `--doc-token`
- **THEN** the Hint Card explains that `docs +fetch` needs a document URL or token and suggests running `docs +search`

#### Scenario: Fetch command appears to use a wiki node token

- **WHEN** the wrapped command is `lark-cli docs +fetch --doc <value>` and `<value>` appears to be a wiki node token
- **THEN** the Hint Card explains that the user may need an actual document token or URL and suggests returning to `docs +search` or resolving the wiki resource before fetching

#### Scenario: Fetch fails because lark-cli is not configured

- **WHEN** captured stderr or stdout includes a not configured error from `lark-cli docs +fetch`
- **THEN** the Hint Card suggests running `lark-cli config init --new`

#### Scenario: Fetch fails because identity is unsupported

- **WHEN** captured stderr or stdout includes an unsupported identity error from `lark-cli docs +fetch`
- **THEN** the Hint Card suggests retrying the fetch with a supported identity flag

#### Scenario: Fetch failure has no known classification

- **WHEN** the wrapped command is `lark-cli docs +fetch`, exits with a non-zero status, and no known Recover classification matches
- **THEN** the system returns a docs fetch failure hint that cites stderr or exit status without inventing a specific fix

### Requirement: Docs workflow hints preserve protocol and i18n behavior

The system SHALL render docs workflow hints through the existing Hint Card and JSON envelope contracts.

#### Scenario: Docs workflow hint in terminal mode

- **WHEN** a docs workflow rule matches in terminal mode
- **THEN** the output uses the existing `Status`, `Hint`, `Next`, `Why`, and `Sources` sections in the selected locale

#### Scenario: Docs workflow hint in JSON mode

- **WHEN** a docs workflow rule matches in `--json` mode
- **THEN** the JSON envelope keeps stable English field names and includes localized user-facing hint values

### Requirement: Docs unsupported identity failures suggest user retry

The system SHALL detect unsupported-identity failures for `lark-cli docs +search` and `lark-cli docs +fetch` when captured output indicates that `user` is the supported identity, and SHALL suggest a recovered command that uses `--as user`.

#### Scenario: Search fails because bot identity is unsupported

- **WHEN** the wrapped command is `lark-cli docs +search`, exits with a non-zero status, and captured stdout or stderr says the resolved identity is unsupported and the command supports `user`
- **THEN** the Hint Card `Next` section contains the same `docs +search` command with `--as user`

#### Scenario: Fetch fails because bot identity is unsupported

- **WHEN** the wrapped command is `lark-cli docs +fetch`, exits with a non-zero status, and captured stdout or stderr says the resolved identity is unsupported and the command supports `user`
- **THEN** the Hint Card `Next` section contains the same `docs +fetch` command with `--as user`

#### Scenario: Existing bot identity is replaced

- **WHEN** the wrapped docs command already includes `--as bot` and captured stdout or stderr indicates that `user` is required
- **THEN** the recovered command replaces the existing identity value with `user` instead of adding a second `--as` flag

#### Scenario: Original command arguments are preserved

- **WHEN** the system generates an identity Recover command for a failed docs command
- **THEN** the recovered command preserves non-identity arguments from the original wrapped command, including query, document, pagination, output format, and jq arguments

#### Scenario: Identity Recover remains evidence-backed

- **WHEN** a docs command fails without captured evidence that `user` is a supported or suggested identity
- **THEN** the system does not invent an identity Recover command

#### Scenario: Identity Recover is available in JSON mode

- **WHEN** a docs unsupported-identity failure matches in `--json` mode
- **THEN** the JSON envelope keeps stable English field names and includes the localized Recover hint with `hint.next.command` set to the recovered `--as user` command

