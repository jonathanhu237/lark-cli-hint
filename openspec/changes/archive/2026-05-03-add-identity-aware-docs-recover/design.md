## Context

The current MVP wraps `lark-cli`, captures command output, and renders Recover/Next hints through the terminal Hint Card and JSON envelope. The docs workflow analyzer already handles successful `docs +search`, successful `docs +fetch`, and several `docs +fetch` failures, including a narrow unsupported-identity recovery.

The gap is `docs +search`: a real `lark-cli docs +search --query demo` can fail when the resolved identity is `bot` while the command requires `user`. `lark-cli` may include evidence such as `only supports: user` or `hint: use --as user`, but `lark-cli-hint` currently falls back to the baseline failure hint instead of producing a domain-specific Recover command.

## Goals / Non-Goals

**Goals:**

- Detect unsupported-identity failures for `lark-cli docs +search` and `lark-cli docs +fetch` when the captured output indicates that `user` is the supported identity.
- Produce one Recover command by minimally rewriting the original wrapped command to use `--as user`.
- Preserve the original command intent, including query, document, paging, format, jq, and other non-identity arguments.
- Keep terminal and JSON output behavior compatible with the existing Hint Card and JSON envelope contracts.
- Keep the implementation deterministic and backed by captured command/output evidence.

**Non-Goals:**

- Do not implement OAuth, token storage, user login, permission indexing, or a remote authorization service.
- Do not build a full schema/help snapshot system in this change.
- Do not generalize identity recovery to all `lark-cli` commands.
- Do not auto-run the recovered command.

## Decisions

1. Scope identity-aware Recover to the current docs workflow.

   The implementation will cover only `lark-cli docs +search` and `lark-cli docs +fetch`. Broader command coverage needs command metadata from `lark-cli <command> --help` and API metadata from `lark-cli schema <service> --format json`; without that metadata, generic identity rewriting risks suggesting invalid commands.

2. Use runtime evidence as the trigger for this change.

   The analyzer will match captured stdout/stderr evidence such as `only supports: user`, `hint: use --as user`, or unsupported identity messages. This is enough for the current failure mode and avoids blocking the small Recover fix on a larger schema ingestion system.

3. Rewrite the original argv instead of constructing template commands.

   The recovered command will preserve the user's original arguments and insert or replace `--as user` after the docs operation. For example:

   ```bash
   lark-cli docs +search --query demo
   ```

   becomes:

   ```bash
   lark-cli docs +search --as user --query demo
   ```

   and:

   ```bash
   lark-cli docs +search --as bot --query demo
   ```

   becomes:

   ```bash
   lark-cli docs +search --as user --query demo
   ```

4. Treat schema/help snapshots as a future rule-source layer.

   If the product later expands identity recovery beyond the docs demo flow, it should first capture command metadata from high-level command help and API metadata from `lark-cli schema`. That future metadata layer can prevent broad hand-written rules from inventing unsupported identity flags or command forms.

## Risks / Trade-offs

- [Risk] Runtime error text may change across `lark-cli` versions. → Match multiple stable evidence fragments and keep tests fixture-based.
- [Risk] `--as user` may not solve every authorization failure. → Trigger only when `lark-cli` indicates user identity is supported or explicitly suggests `--as user`; do not treat generic permission errors as identity errors.
- [Risk] Rewriting argv can mishandle flags with values or existing identity flags. → Implement focused command-argument tests for no `--as`, existing `--as bot`, and preservation of other arguments.
- [Risk] The design still does not provide full permission-aware RAG. → Keep the scope explicit and document OAuth/ACL-aware retrieval as a later backend concern.
