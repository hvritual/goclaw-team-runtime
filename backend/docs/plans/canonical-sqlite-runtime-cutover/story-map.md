# Canonical SQLite runtime story map

## XP policy

- Mode: strict XP.
- Maximum active stories: one.
- Current story: `M1-S7-C4` (core update/move candidate green; clean-candidate browser integration active).
- Every implementation story follows Story Ready -> Test Ready -> RED Proven ->
  GREEN Proven -> Refactor Safe -> Integrated -> Navigator Reviewed -> Customer
  Accepted.
- A story cannot be promoted using static inspection as runtime evidence.

## Activity map

| User activity | S0 contract | First usable slice | Later/deferred |
| --- | --- | --- | --- |
| Start locally | Freeze ports, DB path, process ownership, rollback | `M1-S1`, `M1-S7` | Production deployment |
| Authenticate | Freeze login/session/current-user contract | `M1-S2` | Invitations, password recovery, SSO |
| Enter Workspace | Freeze slug/ID/member/role behavior | `M1-S3`, `M1-S7-C1` | Workspace administration |
| Work with Issues | Freeze actual list/detail calls and fields | `M1-S4`, `M1-S7-C3` | `M1-S7-C4` through `C9`: full local detail |
| Inspect metadata | Reuse accepted v9 body contract | `M1-S5` | New editing UI |
| Observe changes | Freeze handshake/event/cache behavior | `M1-S6` | Complete domain event parity |
| Cut over safely | Freeze proof and rollback procedure | `M1-S7` | Legacy retirement |

## Story cards

### M1-S0 — Compatibility and runtime contract

**As the delivery owner**, I want the real journey dependencies and runtime
boundaries frozen so that implementation cannot hide missing behavior.

Given the existing frontend and read-only legacy evidence, when the accepted
journey is inventoried, then every critical API/event/runtime behavior has an
approved parity decision, executable test approach, exact write boundary, and
rollback path.

- Size: medium
- Risk: high; hidden frontend calls and split runtime ownership
- Demonstration: reviewed parity matrix and planned failing tests
- Rollback: documentation-only; supersede with `plan_v2.md`

### M1-S1 — Real Canonical SQLite composition

**As a developer**, I want one Canonical backend command using real SQLite
providers so that local behavior does not depend on generated stubs.

Given an empty or retained database, when the Canonical local profile starts,
then migrations and required providers are ready, health reflects dependency
state, and restart preserves data.

- Dependency: accepted `M1-S0`
- Size: medium
- Risk: high; two runtime entrypoints and provider graph
- Rollback: retain previous startup selector and database

### M1-S2 — Trusted authentication

**As a user**, I want to log in and restore my session so that Canonical APIs
can derive my identity without trusting caller-supplied actor headers.

Given valid or invalid session inputs, when login/current-user/logout is used,
then frozen success and failure bodies are returned and protected endpoints
fail closed.

- Dependency: accepted `M1-S1`
- Size: medium
- Risk: high; cookie/token/CSRF contract
- Rollback: disable Canonical auth route selection

### M1-S3 — Authorized Workspace selection

**As an authenticated member**, I want to list and select my Workspace so that
all following access is tenant-scoped.

Given member, non-member, missing identity, and foreign Workspace cases, when
Workspace APIs are called, then only authorized Workspaces and roles are
observable with frozen public errors.

- Dependency: accepted `M1-S2`
- Size: medium
- Risk: high; slug/UUID mapping and authorization order
- Rollback: disable Canonical Workspace route selection

### M1-S4 — Issue list/detail

**As a Workspace member**, I want the current Issue list and detail page to use
Canonical SQLite so that the core work surface no longer needs the legacy
server.

Given the UI's frozen filters, identifiers, and fields, when list/detail and
required mutations run, then the same bodies and visible states are produced
without cross-Workspace access.

- Dependency: accepted `M1-S3`
- Size: large; split further only through a plan revision if not demonstrable
- Risk: high; hidden Issue-owned projections
- Rollback: route the complete Issue slice back; do not split read/write owners

### M1-S5 — Metadata end to end

**As a Workspace member or agent-facing client**, I want Issue metadata reads
and single-key mutations to use the real Canonical runtime so that the accepted
v9 slice is usable from the existing frontend.

Given valid, invalid, concurrent, missing, and foreign cases, when metadata is
read or mutated, then the v9 body/status/error and atomicity contract holds
through HTTP, identity, application, SQLite, and the shared Core client.

- Dependency: accepted v9 merge and `M1-S4`
- Size: small
- Risk: medium; composition and identity drift
- Rollback: disable metadata Canonical route selection

### M1-S6 — Minimum realtime refresh

**As a user viewing an Issue**, I want committed Issue and metadata changes to
refresh the correct cache so that the UI does not display stale state.

Given an authorized connection, mutation, disconnect, and reconnect, when an
event is published, then publication follows commit, ordering/duplication rules
hold, and only the matching Workspace cache changes.

- Dependency: accepted `M1-S5`
- Size: medium
- Risk: high; durable delivery and resume semantics
- Rollback: fall back to explicit query invalidation/polling only if S0 approved
  that as compatible; otherwise revert the slice

### M1-S7 — Canonical-only local acceptance

**As a developer**, I want one supported startup and rollback workflow so that
I can use the accepted Issue journey without starting the legacy server.

Given clean and retained local states, when the documented workflow starts,
then the browser journey passes, no accepted request reaches a legacy process,
and rollback preserves both data sets.

- Dependency: accepted `M1-S1` through `M1-S6`
- Size: medium
- Risk: high; process ownership and data transition
- Rollback: stop Canonical processes and restore previous selector without data
  deletion

### M1-S7-C4 through C9 — Full local Issue detail

**As a Workspace member**, I want every local control mounted by the current
Issue detail page to use Canonical persistence and realtime so that moving an
Issue and opening/editing its detail never produces a missing request or a
title-only placeholder.

The approved sequence is deliberately split while retaining one final product
journey: C4 core update/move and public actor identity; C5 hierarchy/batch; C6
timeline/comments/reactions/subscribers; C7 labels/properties/acceptance; C8
attachments; C9 complete capability cutover and clean-candidate acceptance.

- Dependency: technically accepted `M1-S7-C3` and approved plan v7
- Size: extra large; strict single-story execution is mandatory
- Risk: high; multi-owner transactions, public/private actor identity, files,
  realtime and retained-data migration
- Demonstration: the current full detail page performs every approved local
  interaction, reloads/restarts with retained state and produces no missing
  route or legacy-network call
- Rollback: disable only the unaccepted story capability and preserve every
  SQLite/file/log artifact

## Promotion record

Story status changes are recorded append-only in `journal.md`. `plan.md` may
advance the active step only after the prior story has deterministic evidence,
independent review, and Human Customer acceptance. Material contract changes
require a new immutable plan version before promotion.
