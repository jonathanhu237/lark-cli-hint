## MODIFIED Requirements

### Requirement: FlowOps Feishu Seed Script
The system SHALL provide a dry-run-first script that creates, updates, and repairs a complete FlowOps demo team Wiki in Feishu through `lark-cli`.

#### Scenario: Dry run is default
- **WHEN** the user runs the seed script without `--apply`
- **THEN** the script MUST print the planned Feishu Wiki and document write operations
- **AND** the script MUST NOT create or update Feishu content

#### Scenario: Apply requires explicit seed profile
- **WHEN** the user runs the seed script with `--apply`
- **THEN** the script MUST resolve a Feishu test profile from CLI `--profile` or local `[seed].feishu_profile` configuration
- **AND** the script MUST fail before any write when no seed profile is resolved

#### Scenario: Apply creates or reuses target team Wiki
- **WHEN** the user runs the seed script with `--apply`
- **THEN** the script MUST use `lark-cli` to find an existing team Wiki by the configured Wiki name
- **AND** the script MUST create the team Wiki when no exact match exists
- **AND** the script MUST NOT write to `my_library` or another personal document library

#### Scenario: Wiki seed is manifest-backed
- **WHEN** the seed script runs
- **THEN** the script MUST read the desired Wiki tree from a repository JSON manifest
- **AND** page content MUST come from repository Markdown files referenced by that manifest
- **AND** the Markdown content and titles MUST NOT expose demo-only prefixes or seed-management markers

#### Scenario: Seeded Wiki structure is complete
- **WHEN** the seed script applies successfully
- **THEN** the target Wiki MUST contain content-bearing pages for `FlowOps 调度平台`, `DAG 发布与巡检`, and `历史故障复盘`
- **AND** the target Wiki MUST contain documents for FlowOps DAG import error FAQ, `billing_daily` historical incident review, and FlowOps DAG development standards
- **AND** those pages MUST be arranged according to the manifest-defined parent-child structure

#### Scenario: Apply is idempotent
- **WHEN** the user runs the seed script repeatedly against the same profile and Wiki
- **THEN** the script MUST update managed page content in place instead of creating duplicate pages
- **AND** the script MUST move managed pages to their manifest-defined parent when they exist in the target Wiki under a different parent
- **AND** the script MUST preserve pages that are not listed in the manifest

#### Scenario: No automatic IM or pruning side effects
- **WHEN** the seed script runs
- **THEN** it MUST NOT send mock messages to Feishu chats unless a future explicit chat-send option is added and invoked
- **AND** it MUST NOT delete or prune pages outside the manifest

#### Scenario: Smoke commands are emitted
- **WHEN** the seed script finishes dry-run or apply mode
- **THEN** it MUST print smoke search commands that verify the seeded FlowOps keywords can be found through `lark-cli`
