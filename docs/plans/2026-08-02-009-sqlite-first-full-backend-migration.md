---
title: SQLite-First Full Backend Migration
type: refactor
date: 2026-08-02
topic: sqlite-first-full-backend-migration
artifact_contract: multica-ddd-execution/v1
execution_status: in-progress
depends_on:
  - dddgen-native-scaffold
---

# SQLite-First Full Backend Migration

## Completion Contract

The backend is complete only when every frontend-used API, authorization rule,
side effect, and durable workflow is owned by the native Workspace, Auth, Space,
or System module; SQLite is the selected runtime database; Kratos v3 serves the
public transport; superseded Chi/sqlc/SQLite-local business paths are removed;
and the frontend typecheck, unit tests, and end-to-end suite pass against it.

Generated directories, compiling stubs, or one green module test are not proof
of full migration.

## Execution Order

1. Characterize one frontend-used use case and its current SQLite behavior.
   Default to one route; group routes only when they share one business rule,
   persistence seam, verification path and safe rollback boundary.
2. Add its Proto RPC, HTTP/access metadata, domain rule, application use case,
   and native SQLite provider.
3. Switch the SQLite route to the module contract and remove the old business
   branch so the use case has one owner.
4. Run module, contract, full SQLite, frontend focused, frontend full, and
   generated-clean gates.
5. Repeat until a service is complete, then move its transport to Kratos and
   delete the superseded Chi registration.

## Migration Matrix

| Module / service | SQLite native status | Remaining legacy work |
| --- | --- | --- |
| Auth / Member | A1 role change, A2 delete/leave, A3 member list, A4 invitation revoke/workspace-list/create, A5 personal invitation reads, and A6 invitation decisions switched | PostgreSQL parity |
| Auth / Agent | scaffold only | identity, authorization, persistence, routes |
| Workspace / Project | scaffold only | lifecycle, resources, events |
| Workspace / Todo | scaffold only | full task API and persistence |
| Workspace / Issue | generated update stub only | all Issue behavior, evidence, events, metadata |
| Workspace / Knowledge | scaffold only | durable store, outbox, MCP, fail-open behavior |
| Workspace / Requirement | native scaffold only | move current requirement application/provider into native module |
| Workspace / Setting | scaffold only | workspace lifecycle, settings, permissions |
| Workspace / Relationship | contract model only | Project–Member/Agent source of truth |
| Space / Asset | legacy DDD upload slice exists | native SQLite asset lifecycle, download, relations, cleanup |
| System / Skill | scaffold only | catalog/version publication and workspace activation split |
| System / AgentRelease | scaffold only | version, artifact, policy, rollout state |
| Runtime transport | scaffold bootstrap only | Kratos server lifecycle, middleware, realtime and route cutover |

## Checkpoint 2026-08-02 — Auth A1 SQLite

- Added the native `MemberService.UpdateMemberRole` Proto/HTTP/access contract.
- Moved role parsing, authorization policy, and the last-Owner invariant into Auth domain/application layers.
- Added an Auth-owned SQLite schema migration and transactional provider, assembled only through bootstrap.
- Replaced the SQLite-local role-change business branch with the native service while preserving the public JSON and error contract.
- Added Kratos transport error mapping and frontend response-schema validation.
- Passed DDD lint/vet/race gates, full Go test/vet/build, frontend typecheck, and the serial frontend test suite.

This checkpoint completes only the SQLite A1 tracer slice. The matrix and full migration goal remain in progress.

## Checkpoint 2026-08-02 — Auth A2 SQLite

- Added native DeleteMember and LeaveWorkspace Proto/HTTP/access contracts.
- Reused the Auth removal/departure policy and transactional SQLite membership repository.
- Replaced both SQLite-local business branches while preserving the frontend-facing 204 behavior.
- Extended the generation chain with a Proto-owned HTTP success-status option so Kratos server/client and OpenAPI preserve 204/no-body semantics, including generated-client path binding.
- Added domain, application, real SQLite rollback, HTTP lifecycle, transport error, generated HTTP client, and gRPC round-trip coverage.

This checkpoint completes only the SQLite A2 tracer slice. Auth invitations/create/list and the remaining migration matrix are still pending.

## Checkpoint 2026-08-02 — Auth A3 SQLite

- Added the native `MemberService.ListMembers` Proto/HTTP/access contract with an explicit top-level-array response body.
- Moved member-list authorization, projection, workspace scoping, and ordering into Auth application and SQLite provider layers.
- Replaced the SQLite-local member-list SQL branch with the Auth contract while preserving the existing public JSON response.
- Extended generation normalization so `response_body` RPCs use standard JSON in generated clients and emit the unwrapped OpenAPI schema.
- Added application, real SQLite, raw/generated HTTP, gRPC, SQLite-local, and frontend response-schema coverage.

This checkpoint completes only the SQLite A3 tracer slice. Auth invitations/create and the remaining migration matrix are still pending.

## Checkpoint 2026-08-02 — Auth A4 SQLite

- Added the native `MemberService.RevokeInvitation` Proto/HTTP/access contract with generated 204/no-body behavior.
- Introduced a reusable invitation transaction/repository seam alongside membership authorization in the Auth application layer.
- Added pending-only, workspace-scoped SQLite revocation with stable not-found mapping.
- Replaced the SQLite-local invitation-revoke SQL branch with the Auth contract and removed its duplicate role-policy branch.
- Added application ordering/policy tests, real SQLite isolation tests, transport mapping, raw/generated HTTP, gRPC, and existing invitation-lifecycle regression coverage.

This checkpoint completes only invitation revocation. Invitation creation, listing, lookup, acceptance, decline, expiry and the remaining migration matrix are still pending.

## Checkpoint 2026-08-02 — Auth A4 Workspace Invitation List

- Generated the Auth Invitation aggregate and Workspace aggregate/SQLite provider
  skeletons through native `dddgen`, then completed their provider-owned behavior.
- Added the Workspace public identity reader and registered the intentional
  Auth-to-Workspace contract edge; Auth never queries the Workspace table.
- Added the Proto-owned `ListWorkspaceInvitations` REST/gRPC/access contract with
  the existing top-level array and nullable invitee identity representation.
- Moved Owner/Admin authorization, pending expiry, workspace scoping, ordering,
  inviter projection and response mapping into Auth application/SQLite layers.
- Replaced the SQLite-local invitation-list SQL branch with the Auth contract.
- Added Workspace provider, Auth application/SQLite, bootstrap composition,
  raw/generated HTTP, gRPC, SQLite-local and frontend schema coverage.

This checkpoint completes only the workspace-scoped invitation list. Personal
listing, lookup, acceptance, decline and the remaining migration matrix are
still pending.

## Checkpoint 2026-08-02 — Auth A4 Invitation Creation

- Added the Proto-owned `CreateInvitation` operation on the compatible
  `POST /api/workspaces/{workspace_id}/members` route, with generated 201
  response-body and access-manifest behavior.
- Added the pending Invitation constructor for normalized email, allowed invite
  roles, timestamps and seven-day expiry.
- Moved Owner/Admin authorization, member/pending conflict checks, expired
  invitation rollover, registered-user resolution and insertion into the Auth
  application transaction and SQLite provider.
- Kept Workspace name lookup behind the Workspace public contract and outside
  the Auth transaction, preserving the single-connection SQLite runtime.
- Replaced the SQLite-local creation SQL/policy branch with the Auth contract and
  added domain, application, real SQLite, raw/generated HTTP, gRPC and frontend
  response-schema coverage.
- Generalized the generated-status postprocessor so 204 remains no-content while
  201 preserves its response body and OpenAPI schema.
- Added the Proto-driven `authorize_before_body` adapter hook so both Kratos and
  Chi resolve Owner/Admin authorization before decoding invitation JSON; the
  mutation repeats the check inside its transaction to remain authoritative.

This checkpoint completes invitation creation. Personal listing, lookup,
acceptance, decline and the remaining migration matrix are still pending.

## Checkpoint 2026-08-02 — Auth A5 Personal Invitation Reads

- Added Proto-owned `ListMyInvitations` and `GetMyInvitation` operations for the
  existing `GET /api/invitations` and `GET /api/invitations/{id}` routes. They
  form one read-only frontend invitation-inbox use case and share one Auth
  ownership/projection path and rollback boundary.
- Moved authenticated-user lookup, ID-or-email ownership, pending expiry,
  ordering and inviter projection into the Auth application transaction and
  native SQLite provider. Workspace display names remain behind the Workspace
  public contract and are resolved only after the Auth transaction closes.
- Replaced both SQLite-local read branches while preserving top-level list and
  detail JSON, nullable invitee identity, missing-user/not-found/forbidden
  statuses, and non-pending detail visibility.
- Kept historical reads forward-compatible: persisted role, status and
  timestamp strings are projected verbatim, while newly created invitations
  continue to validate roles/status transitions and emit canonical RFC3339Nano
  timestamps.
- Generalized generated `response_body` normalization for singular message
  fields and made repeated optional-field processing idempotent; raw/generated
  HTTP, gRPC, OpenAPI and access-manifest paths are covered.
- Passed `make generate`, `make generated-clean`, `make lint-ddd
  DDD_BASE_REV=HEAD`, `make vet-ddd`, `make test-race-ddd`, `(cd server && go
  test ./...)`, Core typecheck, all 522 Core tests and `pnpm turbo test
  --concurrency=1` (five workspace test tasks). The root `pnpm test` wrapper
  cannot resolve its literal `--filter=!@multica/mobile` while the unrelated
  mobile-package removal is in progress, so the equivalent repository package
  tests were run through Turbo directly.

This checkpoint completes personal invitation listing and lookup. Acceptance,
decline, PostgreSQL parity and the remaining migration matrix are still pending.

## Checkpoint 2026-08-02 — Auth A6 Invitation Decisions

- Added Proto-owned `AcceptInvitation` and `DeclineInvitation` operations for
  the existing decision routes. They are one invitation-decision use case: both
  share ownership, lifecycle, workspace-existence, transaction and rollback
  rules; acceptance preserves the top-level member response while decline
  preserves 204/no-body behavior.
- Moved invitee ownership, pending-state and expiry policy into the Invitation
  domain. Auth re-reads authoritative state inside the mutation transaction;
  expired acceptance persists `expired`, while repeated or conflicting
  decisions cannot consume the invitation.
- Kept Workspace existence behind its public contract and outside the Auth
  transaction. The SQLite runtime intentionally uses a two-phase preflight so
  its single connection cannot deadlock, followed by an authoritative Auth
  recheck before mutation.
- Made member creation, invitation compare-and-set, and onboarding completion
  atomic in the native Auth SQLite provider. Conflict and onboarding failures
  roll back all three effects, and persisted future invitation roles remain
  readable for legacy compatibility.
- Replaced the SQLite-local accept/decline SQL branches with the Auth contract,
  preserving legacy status/message mappings, and added domain, application,
  real SQLite rollback/lifecycle, raw/generated HTTP, gRPC, SQLite-local and
  frontend response-schema coverage.
- Passed focused Auth/contract/SQLite-local tests, full `go test ./...`, `make
  lint-ddd DDD_BASE_REV=HEAD`, `make vet-ddd`, `make test-race-ddd`, the full
  Core suite (524 tests), Core typecheck, and the serial frontend workspace
  suite (five workspace test tasks). `make generated-clean` is run after the
  generated slice is committed so it can distinguish stale output from the
  intentional generated changes.

This checkpoint completes SQLite invitation acceptance and decline. PostgreSQL
parity and the remaining migration matrix are still pending.

## Required Gates

```sh
make generate
make generated-clean
make lint-ddd
make vet-ddd
make test-race-ddd
(cd server && go test ./...)
pnpm typecheck
pnpm test
pnpm exec playwright test
```

Each checkpoint records commands actually run and keeps this goal active until
every matrix row and runtime cutover is complete.
