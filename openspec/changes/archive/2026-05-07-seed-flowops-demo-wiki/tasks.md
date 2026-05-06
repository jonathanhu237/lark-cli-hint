## 1. Seed Content Model

- [x] 1.1 Create `examples/flowops-airflow/seed/wiki/manifest.json` with the default Wiki name, description, and nested page tree.
- [x] 1.2 Move all FlowOps seed page bodies into standalone Markdown files referenced by the manifest.
- [x] 1.3 Ensure directory pages have realistic internal-Wiki content and no demo-only prefixes or seed markers.

## 2. Seed Configuration

- [x] 2.1 Add `[seed]` config support for `feishu_profile` and `wiki_name`.
- [x] 2.2 Update seed script option parsing so CLI `--profile` overrides config and `--apply` fails when no seed profile is resolved.
- [x] 2.3 Keep dry-run usable without credentials while printing the resolved or placeholder Wiki/profile plan.

## 3. Wiki Apply Logic

- [x] 3.1 Teach the seed script to list Wiki spaces, find the configured team Wiki by exact name, and create it with `wiki spaces create --yes` when missing.
- [x] 3.2 Implement manifest traversal that creates content-bearing pages under the correct parent nodes.
- [x] 3.3 Implement idempotent updates for existing managed pages without creating duplicates.
- [x] 3.4 Implement managed-page moves when an existing page is in the target Wiki but under the wrong parent.
- [x] 3.5 Preserve pages outside the manifest and keep IM sends/deletes/pruning out of scope.

## 4. Verification And Docs

- [x] 4.1 Add deterministic script tests or shell checks for dry-run output, missing-profile failure, manifest parsing, and idempotent command planning.
- [x] 4.2 Update README and demo docs to describe config-based seed profile, automatic Wiki creation/reuse, and the seeded Wiki tree.
- [x] 4.3 Run `python3 -m py_compile examples/flowops-airflow/scripts/seed-feishu`, seed dry-run, `go test ./...`, and `openspec validate --all --strict`.
- [x] 4.4 Optionally run real `--apply` against the configured test profile and record the created/reused Wiki URL or clear skip reason. Ran real apply with profile `cli_a970f44fddb8dcb6`; target Wiki `星桥科技 FlowOps 知识库` was created/reused as space `7636831927403564251`, and a second apply reused the same space and updated managed pages in place.
