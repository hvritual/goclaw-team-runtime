# Incremental server function migration — Auth member foundation

- Plan-ID: `server-function-migration`
- Version: `5`
- Status: `approved`
- Approval source: user instruction dated 2026-08-03 to begin implementing the
  Auth module
- Base commit: `0c4f848a8458e256dd4fe2ed51498af969aa3c59`
- Repository: `/Users/fworld/Hvritual/goclaw`
- Branch: `codex/multica-six-domain-baseline`
- Task type: `change`

## Goal

Implement the first independently verifiable Auth slice around human Workspace
membership: provision the initial Workspace owner, list members for an existing
member, and update roles while preserving manager and last-owner invariants.
Expose a stable Auth-owned capability that a later Workspace lifecycle workflow
can coordinate without allowing Workspace to write Auth tables.

## Context brief

- Context: Auth.
- Purpose: own human and Agent identity plus Workspace team membership and
  authorization attributes.
- Ubiquitous language: Auth user, member, initial owner, owner, admin, member,
  membership root.
- Owned state in this slice: Auth user profile projections, Workspace member
  rows, and the one-time Workspace membership root used to serialize initial
  owner provisioning.
- Commands: ProvisionWorkspaceOwner, UpdateMemberRole.
- Queries: ListMembers.
- Invariants: one initial-owner provisioning decision per Workspace; roles are
  lowercase owner/admin/member; only owner/admin manages roles; only an owner
  changes an owner role; the final owner cannot be demoted; all access is
  Workspace scoped.
- External dependencies: caller-supplied authenticated user context for
  List/Update and caller-supplied Workspace/User references for trusted initial
  owner provisioning.
- Consistency boundary: one Auth-native SQLite transaction per command/query.

## Current and target paths

- Current target backend path: generated MemberService adapter -> generated
  not-implemented application stub.
- Installed source path: HTTP/member boundary -> Auth application policy ->
  native SQLite/sqlc member store.
- Target path: generated local/gRPC adapter -> Auth application use case -> Auth
  member domain -> Auth repository/unit-of-work port <- SQLite adapter.

## Ordered steps

### P5-S1 — active Auth member foundation

1. Extend `auth.v1.MemberService` with idempotent
   `ProvisionWorkspaceOwner`, retaining existing RPCs and field numbers.
2. Reconcile the service through dddgen, apply the established exact-prefix
   import postprocessor, regenerate Buf bindings, and inspect generated changes.
3. Add Auth member domain roles/policies, authenticated actor context, stable
   errors, application orchestration, and transaction ports.
4. Add Auth-owned SQLite schema, migration runner, member unit of work, profile
   projection mapping, and explicit opt-in module composition.
5. Verify domain/application policy matrices, provisioning idempotency and
   conflict behavior, tenant isolation, last-owner rollback, local/gRPC
   contracts, generation idempotence, and full repository gates.

## Frozen behavior

- `ProvisionWorkspaceOwner` accepts trimmed non-empty Workspace and Auth User
  IDs. The User must already exist in Auth storage. It creates a Member with the
  fixed `owner` role and UTC timestamp in one Auth transaction.
- A Workspace membership-root row serializes the first provisioning decision.
  Repeating the same completed Workspace/User command returns the same owner and
  `created=false`; another User for that Workspace returns the stable
  already-initialized error. No foreign key or cascade is introduced.
- Provisioning is a trusted collaboration capability with permission code
  `auth.member.provision_workspace_owner`, audit enabled, and idempotency
  metadata. It does not create or validate a Workspace row and is not by itself
  authorization to cut over Workspace creation.
- `ListMembers` requires an authenticated User ID from Auth context and first
  verifies that User belongs to the request Workspace. Outsiders receive the
  stable hidden-Workspace error. Results order by creation time then Member ID.
- `UpdateMemberRole` requires an authenticated current Member and validates the
  lowercase owner/admin/member role before persistence. Owner or Admin may
  manage ordinary roles; only Owner may promote to or demote from Owner; the
  final Owner cannot be demoted. The policy check and write share one Auth
  transaction.
- Member responses join the Auth-owned User profile projection for name, email,
  and optional avatar URL. Auth stores those profiles but login/profile mutation
  is deferred.
- The default `auth.New()` and process bootstrap remain generated stubs. The
  SQLite implementation is selected only through an explicit constructor.

## Scope

- `backend/api/auth/v1/member.proto` and its dddgen/Buf-generated outputs.
- User-owned Auth contract behavior, member domain, application use case,
  SQLite provider/migrations, composition, architecture documentation, and
  Auth tests under `backend/`.
- Append-only migration journal and current plan pointer.

## Non-goals

- No changes outside `backend/` and no writes to installed `server/` sources.
- No Workspace Create/List/Get/Update/Delete/Leave implementation or cross-module
  transaction/cutover in this slice.
- No login, verification codes, sessions/tokens, profile mutation, invitations,
  member removal/leave, owner transfer workflow, Agent identity, authorization
  middleware, compatibility HTTP routes, PostgreSQL/sqlc, realtime events, or
  production data migration.
- No default runtime switch and no direct Workspace table access from Auth.
- No OpenAPI or access-manifest generation beyond existing configured Buf
  outputs.

## Invariants

- Workspace remains the tenant and authorization boundary.
- Auth owns Member/User tables and never reads or writes Workspace module tables.
- Domain imports only the standard library; application depends only on domain
  and public Auth contracts/ports; SQLite owns SQL and native transactions.
- No foreign keys or cascading actions are added.
- Proto remains the public service source of truth. Generated files change only
  through dddgen/postprocess/Buf.
- Existing HTTP APIs, Chi/sqlc server, database, SQLite-local implementation,
  runtime behavior, and Workspace isolation semantics remain unchanged.

## Dependencies

- Accepted four-module Proto/generated scaffold and explicit extension registry.
- Installed source Auth member policies and test examples as behavior evidence.
- Installed dddgen module revision
  `7266939f0e295648593739c694ab4e614b141546`, existing exact-prefix
  postprocessor, Buf configuration, and local protoc plugins.
- Existing `modernc.org/sqlite` dependency and explicit provider pattern.

## Acceptance criteria

1. MemberService exposes ProvisionWorkspaceOwner/ListMembers/UpdateMemberRole
   through matching local and gRPC contracts without changing existing field
   numbers.
2. Initial owner provisioning is transactional, deterministic, idempotent for
   the same Workspace/User pair, rejects a different initializer, and rejects a
   missing Auth User.
3. List hides membership from outsiders and never returns a different
   Workspace's members.
4. Update enforces the installed manager/owner/last-owner policy matrix inside
   one transaction and maps missing/foreign Members to stable errors.
5. Persistence stores only Auth-owned tables, uses Workspace-scoped predicates,
   and introduces no foreign keys or cascades.
6. The explicit SQLite constructor fails closed on missing dependencies while
   default bootstrap/runtime behavior remains unchanged.
7. Generated output is content-idempotent, uses `rpc/pb`, and all deterministic
   gates pass.

## Deterministic verification

- Buf format/lint before and after generation.
- dddgen MemberService reconciliation, exact-prefix postprocessing, Buf
  generation, gofmt, and repeat-generation digest comparison.
- Domain/application table tests and SQLite local/bufconn gRPC integration.
- `go test -race ./internal/modules/auth/... -count=1`.
- `go test ./tests/contract/... -count=1`, `go test ./... -count=1`,
  `go vet ./...`, and `go mod verify`.
- Forbidden inward/cross-module imports, no-FK scan, gofmt, generated inventory,
  policy hashes, `git diff --check`, backend-only scope audit, and live runtime
  probes.

## Risks

- ProvisionWorkspaceOwner closes only the Auth capability gap. A later
  Workspace workflow must still define atomic cross-module orchestration or a
  verified compensation protocol before cutover.
- Auth user rows are seeded by tests in this slice because login/profile
  creation is deferred. The provider must reject missing users explicitly.
- A locally installed generator build is marked dirty; its module revision is
  frozen and repeated generated output must be identical.
- SQLite serializes the membership-root write, but concurrent conflict mapping
  must be tested without leaking driver errors.

## Rollback

Restore `member.proto`, rerun the same generation pipeline, remove only P5 Auth
domain/application/provider/composition/test additions and the additive Auth
migration, then restore the v4 plan pointer. No default runtime or production
database rollback is required.
