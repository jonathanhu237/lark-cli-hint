## ADDED Requirements

### Requirement: Docs search success suggests document fetch

The system SHALL detect successful `lark-cli docs +search` executions with a usable document candidate and suggest one `lark-cli docs +fetch --doc <document>` command.

#### Scenario: Search result contains document token

- **WHEN** the wrapped command is `lark-cli docs +search` and stdout contains a successful JSON result with a document token or URL
- **THEN** the Hint Card `Next` section contains `lark-cli docs +fetch --doc <document>`

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
