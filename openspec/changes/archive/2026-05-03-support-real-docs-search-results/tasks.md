## 1. Fixtures

- [x] 1.1 Add a `docs-search.real-result-meta-url.json` fixture based on real `lark-cli docs +search` output containing `results[].result_meta.url`.
- [x] 1.2 Add or derive a token-only fixture for a real result shape containing `results[].result_meta.token` without a candidate URL.

## 2. Candidate Extraction

- [x] 2.1 Extend docs search candidate extraction to inspect nested `result_meta.url` fields.
- [x] 2.2 Extend docs search candidate extraction to inspect nested `result_meta.token` fields when no URL is available.
- [x] 2.3 Preserve existing candidate extraction behavior for simple fixture shapes.
- [x] 2.4 Extract a useful title from real search output, including `title_highlighted` or nested metadata fields, without blocking command generation if the title is imperfect.

## 3. Verification

- [x] 3.1 Add analyzer tests proving real `result_meta.url` output suggests `lark-cli docs +fetch --doc <url>`.
- [x] 3.2 Add analyzer tests proving token-only real metadata output suggests `lark-cli docs +fetch --doc <token>`.
- [x] 3.3 Add regression tests proving simple existing docs search fixtures still work.
- [x] 3.4 Add a test or assertion that search results with no usable URL/token still fall back to baseline.
- [x] 3.5 Run `pnpm build`, `pnpm typecheck`, `pnpm test`, and `openspec validate support-real-docs-search-results --strict`.
