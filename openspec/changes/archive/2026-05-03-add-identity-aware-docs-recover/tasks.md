## 1. Fixtures and Locale Copy

- [x] 1.1 Add a fixture for failed `docs +search` output where the resolved identity is unsupported and `lark-cli` suggests `--as user`.
- [x] 1.2 Review the existing `docs +fetch` unsupported-identity fixture and adjust it if needed to reflect the same evidence pattern.
- [x] 1.3 Add or update English locale strings for docs unsupported-identity Recover status, hint, why, and source labels.
- [x] 1.4 Add or update Simplified Chinese locale strings for docs unsupported-identity Recover status, hint, why, and source labels.

## 2. Command Identity Rewriting

- [x] 2.1 Add a focused command utility that rewrites a classified docs command to use `--as user`.
- [x] 2.2 Ensure the utility inserts `--as user` immediately after `lark-cli docs +search` or `lark-cli docs +fetch` when no identity flag exists.
- [x] 2.3 Ensure the utility replaces an existing `--as <value>` pair with `--as user` instead of appending a second identity flag.
- [x] 2.4 Ensure the utility preserves all non-identity arguments and quotes shell-sensitive values in the rendered command.

## 3. Analyzer Behavior

- [x] 3.1 Add unsupported-identity evidence detection for captured stdout/stderr that indicates `user` is supported or suggested.
- [x] 3.2 Extend failed `docs +search` analysis to produce an identity Recover hint when the evidence matches.
- [x] 3.3 Tighten failed `docs +fetch` identity Recover behavior to use the shared command rewrite utility and preserve original arguments.
- [x] 3.4 Ensure docs commands without user-identity evidence do not produce identity Recover hints.

## 4. Verification

- [x] 4.1 Add analyzer tests for `docs +search` identity failure with no existing `--as`.
- [x] 4.2 Add analyzer tests for replacing an existing `--as bot` with `--as user`.
- [x] 4.3 Add analyzer tests proving query, document, paging, format, jq, and other non-identity arguments are preserved.
- [x] 4.4 Add tests that unmatched docs failures do not invent identity Recover commands.
- [x] 4.5 Add or update CLI-level tests for terminal Hint Card output and `--json` output.
- [x] 4.6 Add i18n tests or assertions for English and Simplified Chinese identity Recover output.
- [x] 4.7 Run `pnpm build`, `pnpm typecheck`, `pnpm test`, and `openspec validate add-identity-aware-docs-recover --strict`.
