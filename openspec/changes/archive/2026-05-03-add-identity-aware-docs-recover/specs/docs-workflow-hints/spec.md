## ADDED Requirements

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
