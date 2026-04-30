## 1. Documentation and Fixtures

- [x] 1.1 Update README/AGENTS examples that reference outdated `docs +fetch --doc-token` or `im message create` command forms.
- [x] 1.2 Add fixtures for successful `docs +search` output with a usable document token or URL.
- [x] 1.3 Add fixtures for successful `docs +fetch` output.
- [x] 1.4 Add fixtures for failed `docs +fetch` cases: outdated `--doc-token`, missing `--doc`, wiki-looking `--doc`, not configured, unsupported identity, and generic failure.

## 2. Command and Output Parsing

- [x] 2.1 Add command classification utilities for `lark-cli docs +search` and `lark-cli docs +fetch`.
- [x] 2.2 Add option extraction for `--doc`, `--doc-token`, `--query`, and relevant command arguments without relying on raw command-string matching.
- [x] 2.3 Add JSON parsing helpers that safely parse captured stdout and return `null` on invalid JSON.
- [x] 2.4 Add conservative document candidate extraction from known search result containers and token/url field names.

## 3. Docs Workflow Analyzer

- [x] 3.1 Add a docs workflow analyzer that runs before the baseline analyzer and returns `null` when no docs workflow rule matches.
- [x] 3.2 Implement `docs +search` success Next hint to suggest `lark-cli docs +fetch --doc <document>`.
- [x] 3.3 Implement `docs +fetch` success Next hint to suggest `lark-cli im +messages-send --chat-id <chat_id> --markdown <message>` with a placeholder chat ID when none is configured.
- [x] 3.4 Implement `docs +fetch` Recover hint for outdated `--doc-token`.
- [x] 3.5 Implement `docs +fetch` Recover hint for missing `--doc`.
- [x] 3.6 Implement `docs +fetch` Recover hint for wiki-looking `--doc` values.
- [x] 3.7 Implement `docs +fetch` Recover hint for not configured errors.
- [x] 3.8 Implement `docs +fetch` Recover hint for unsupported identity errors.
- [x] 3.9 Implement generic docs fetch failure hint when no known Recover classification matches.

## 4. i18n and Rendering

- [x] 4.1 Add English locale strings for docs workflow Status/Hint/Why/Sources messages.
- [x] 4.2 Add Simplified Chinese locale strings for docs workflow Status/Hint/Why/Sources messages.
- [x] 4.3 Ensure docs workflow hints render correctly in both terminal Hint Card output and JSON envelope output.

## 5. Verification

- [x] 5.1 Add tests for `docs +search` success suggesting `docs +fetch --doc`.
- [x] 5.2 Add tests for `docs +search` success without a usable document falling back to baseline.
- [x] 5.3 Add tests for `docs +fetch` success suggesting `im +messages-send`.
- [x] 5.4 Add tests for `docs +fetch --doc-token` Recover.
- [x] 5.5 Add tests for missing `--doc` Recover.
- [x] 5.6 Add tests for wiki-looking `--doc` Recover.
- [x] 5.7 Add tests for not configured Recover.
- [x] 5.8 Add tests for unsupported identity Recover.
- [x] 5.9 Add tests that docs workflow hints remain localized and JSON field names remain stable.
- [x] 5.10 Add CLI-level docs workflow integration tests with a fake `lark-cli`.
- [x] 5.11 Run build, typecheck, tests, and OpenSpec validation.
