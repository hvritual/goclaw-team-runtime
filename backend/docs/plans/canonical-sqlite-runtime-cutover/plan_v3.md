# Canonical SQLite runtime cutover — execution plan v3

- Plan-ID: `canonical-sqlite-runtime-cutover`
- Version: `3`
- Status: `approved`
- Approval source: Human Customer confirmation dated `2026-08-14`
- Supersedes: `plan_v2.md`
- Base commit: `46e55d5`
- Branch and integration target: `codex/multica-six-domain-baseline`
- Active step: `M1-S7`

## Reason for revision

M1-S7 runtime discovery proved the Canonical local-auth projection always
returns `onboarded_at:null`: the Canonical `auth_users` schema has no persisted
field and the store hardcodes the projection to nil. The installed Web
Workspace layout treats that value as not onboarded and redirects every
otherwise valid Canonical login away from the frozen Issue browser journey.
The browser acceptance gate is therefore impossible without a narrow Auth
compatibility correction. A fixture-only bypass would fabricate product state
and is rejected.

## Scope amendment

All v2 goals, invariants, completed-story evidence, non-goals and S7 gates stay
unchanged. S7 additionally permits only:

- an additive Canonical Auth SQLite migration for `auth_users.onboarded_at`;
- the Canonical Auth SQLite store/projection required to read that field;
- focused Auth/bootstrap tests proving null for a new user, persisted non-null
  for a fixture/member, restart retention and unchanged exact User response;
- the explicit Canonical browser fixture setting `onboarded_at` consistently
  with its pre-created Workspace membership.

No general onboarding API, new UI, production profile, legacy `server/**`
change, or unrelated Auth behavior is authorized. Existing Auth/Workspace
identity, session, CSRF and isolation contracts remain immutable.

## S7 execution and acceptance

S7 retains the v2 allowed selector/command, local documentation, `e2e/**` and
plan paths. It must still prove Web 3000, Canonical HTTP 8000, gRPC 9000,
`data/multica-canonical.db`, clean and retained startup, browser login/Issue/
metadata/realtime journey, no legacy process/request, restart readback and a
non-destructive selector rollback preserving both databases and logs.

Approval of this version authorizes only the scope amendment above and
continued M1-S7 execution. It does not accept the milestone; final browser,
rollback, independent review and Customer acceptance remain required.
