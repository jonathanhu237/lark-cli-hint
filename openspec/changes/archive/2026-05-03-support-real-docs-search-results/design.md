## Context

The current docs workflow analyzer can suggest `lark-cli docs +fetch --doc <document>` after a successful `docs +search` when the search result exposes a candidate in simple top-level fields such as `url`, `doc_token`, or `token`. Manual testing with a real Feishu document showed that current `lark-cli docs +search` returns candidates under `results[].result_meta`, for example `result_meta.url` and `result_meta.token`.

Because the extractor does not inspect that nested shape, the command succeeds but falls back to the baseline success hint. That breaks the real docs search -> fetch demonstration even though the data needed for the Next command is present.

## Goals / Non-Goals

**Goals:**

- Recognize real `lark-cli docs +search` result objects with nested `result_meta.url` or `result_meta.token`.
- Prefer `result_meta.url` over `result_meta.token` because a URL is directly accepted by `docs +fetch` and is easier for users to inspect.
- Preserve the existing conservative fallback when no usable candidate exists.
- Add regression tests and a fixture that represent the observed real output shape.

**Non-Goals:**

- Do not change `docs +fetch` behavior.
- Do not change identity-aware Recover behavior.
- Do not add schema ingestion, RAG, OAuth, or remote backend behavior.
- Do not sanitize highlighted title markup beyond extracting a useful display title for the Hint Card.

## Decisions

1. Extend candidate extraction paths rather than adding a new analyzer branch.

   The existing `docs +search` success analyzer already owns the behavior. The implementation should broaden `extractDocumentCandidate` so the same success path handles simple fixtures and real `result_meta` shapes.

2. Prefer URL fields over token fields.

   The observed real output includes both `result_meta.url` and `result_meta.token`. The suggested command should use the URL when available:

   ```bash
   lark-cli docs +fetch --doc https://.../docx/<token>
   ```

   If only `result_meta.token` is present, the system may fall back to that token.

3. Keep title extraction opportunistic.

   Real search output may include `title_highlighted` with highlight tags and may not include a plain title. The title is user-facing context only; failure to produce a perfect plain title must not block the Next command when a usable document exists.

## Risks / Trade-offs

- [Risk] Real `docs +search` shapes may vary across `lark-cli` versions. → Keep fixture-based regression coverage and add candidate paths incrementally when real outputs reveal them.
- [Risk] `title_highlighted` can include markup. → Allow it as a fallback title but keep command generation based only on URL/token fields.
- [Risk] Supporting nested fields could accidentally pick non-document wiki candidates first. → Preserve result order for now because the CLI search rank is the best available signal, but prefer document URL/token fields inside each candidate.
