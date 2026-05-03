## MODIFIED Requirements

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
