# Team Control Connection Plan v4

- Plan-ID: `TC-W01-TEAM-CONTROL-001`
- Version: `v4`
- Status: `approved-for-execution`
- Supersedes: `plan_v3.md`
- Base commit: `2fc411461e2ad7ba45730b2fcb05cb5f650660ac`
- Branch: `agent/tc-w01-team-control-001`
- Project-ID: `goclaw-team-runtime`
- Task-ID: `TC-W01-TEAM-CONTROL-001`
- Task-Revision: `r004`
- Policy bundle: `backend/AGENTS.md@ffe45b83ef3884d9b8bf66e6f7994b3a43d4f86e`

This immutable amendment inherits all applicable decisions from `plan_v1.md`
through `plan_v3.md`, including the permanent `server/**` prohibition, the
single-authority membership model, local-only S05 verification, and the
authorized S06 merge, except for the explicit scope change below.

## Amendment G — Team Control localization scope

S04 surfaced a deterministic repository rule: shared Views must not ship
literal user-facing strings. The v1 allowed paths included the shared Team
Control page but omitted its locale resources. S04 therefore adds only these
paths:

- `packages/views/locales/en/projects.json`
- `packages/views/locales/zh-Hans/projects.json`
- `packages/views/locales/ja/projects.json`
- `packages/views/locales/ko/projects.json`

Team Control copy must live below `projects.team_control`, preserve namespace
parity across all four supported locales, follow the conventions and Chinese
voice guide, and remain shared by Web and Desktop. No new namespace, locale,
dependency, generated resource, or product area is authorized.

## Active step

`TC-W01-S04` remains active. S05 and S06 remain ordered dependencies.
