# M1-S0 frozen compatibility and runtime inventory

- Plan-ID: `canonical-sqlite-runtime-cutover`
- Inventory version: `2`
- Evidence base: `e20114cc7f401b503c6506d1b99cf0eddf894780`
- Status: `approved and frozen`

This inventory freezes the first Canonical-only local Issue journey. Legacy
paths are evidence only and remain read-only. Request and response bodies below
do not change.

## Shared HTTP boundary

Every Core request uses `credentials: include`, JSON content type, and a request
ID. When available, it sends `Authorization: Bearer <token>`,
`X-Workspace-Slug`, `X-CSRF-Token`, and client identity headers. Metadata also
sends `X-Workspace-ID`. Non-2xx is an error; 401 clears the client session.
Canonical identity comes from the accepted session or bearer token before
Workspace resolution. Actor headers are never trusted as identity.

## Accepted endpoint inventory

| ID | Method and path | Frozen request | Frozen success | Canonical story |
| --- | --- | --- | --- | --- |
| `AUTH-01` | `POST /auth/send-code` | `{"email":string}` | existing empty success | S2 |
| `AUTH-02` | `POST /auth/verify-code` | `{"email":string,"code":string}` | `{"token":string,"user":User}` plus accepted session | S2 |
| `AUTH-03` | `GET /api/me` | no body | existing `User` body | S2 |
| `AUTH-04` | `POST /auth/logout` | no body | existing empty success and invalidated session | S2 |
| `WS-01` | `GET /api/workspaces` | no body | existing `Workspace[]` body | S3 |
| `ISSUE-01` | `POST /api/issues/table/facets` | existing `{query,facets,include_total}` | existing facets envelope | S4 |
| `ISSUE-02` | `POST /api/issues/table/rows` | existing `{query,group,group_key,hierarchy,parent_id,page}` | existing rows envelope | S4 |
| `ISSUE-03` | `POST /api/issues/table/groups` | existing `{query,group,page}` | existing groups envelope | S4 grouped view |
| `ISSUE-04` | `GET /api/issues` | existing filter/paging query | `{"issues":Issue[],"total":number}` | S4 detail dependency |
| `ISSUE-05` | `POST /api/issues/query` | same list query as JSON | same list envelope | S4 large-ID compatibility |
| `ISSUE-06` | `GET /api/issues/{id}` | UUID or Workspace identifier | existing `Issue` body | S4 |
| `META-01` | `GET /api/issues/{id}/metadata` | no body | `{"metadata":primitive-map}` | S5 |
| `META-02` | `PUT /api/issues/{id}/metadata/{key}` | `{"value":primitive}` | complete metadata envelope | S5 |
| `META-03` | `DELETE /api/issues/{id}/metadata/{key}` | no body | complete metadata envelope | S5 |
| `RUN-01` | `GET /healthz` | no body | `{"status":"ok"}` when alive | S1/S7 |
| `RUN-02` | `GET /readyz` | no body | `{"status":"ready"}` after DB/migrations/providers | S1/S7 |
| `RUN-03` | `GET /api/config` | no body | existing config plus additive capability flags | S4/S7 |

Invalid input keeps existing public status/error JSON. Missing/expired identity
is 401. Missing and foreign Workspace resources are indistinguishable. S2 and
S3 add schemas and malformed-response tests to the currently raw login and
Workspace-list calls without changing successful bodies.

## Honest Issue-detail boundary

The installed detail mounts timeline, members, reactions, subscribers,
attachments, labels, properties, pins, children, project, child-progress, and
acceptance-history consumers. They are not required by M1 and are not
Canonical-ready.

Canonical local mode advertises explicit additive capability flags from
`/api/config`. Shared views do not mount a disabled consumer and never fabricate
empty success. M1 enables Issue list/table, base detail fields, the read-only
metadata projection, metadata Get/Put/Delete, and minimum refresh. Disabled
capabilities stay enabled in the retained legacy profile. Browser acceptance
fails on any unexpected disabled or legacy-only request.

## Realtime contract

- URL: `/ws?workspace_slug=<slug>` with optional client identity query fields.
- Cookie mode authenticates during upgrade. Token mode sends first frame
  `{"type":"auth","payload":{"token":"..."}}` and requires `auth_ack`.
- Frames are `{"type":event,"payload":object,"actor_id"?:string,
  "actor_type"?:string}`.
- M1 publishes `issue:created`, `issue:updated`, `issue:deleted`, and
  `issue_metadata:changed`; metadata sends the complete bag snapshot.
- Publication is after SQLite commit. Duplicate delivery is tolerated. M1 has
  no durable resume cursor; reconnect authoritatively refetches accepted active
  Workspace Issue/detail/metadata keys.
- Unauthorized connections receive no events. Event payloads are schema-checked
  before cache work.

## Runtime ownership and rollback

- One Canonical `backend/cmd/server` process owns one shared product `*sql.DB`
  for Auth and Workspace migrations/providers.
- Default product DB is `data/multica-canonical.db`, distinct from retained
  legacy `data/multica-local.db`.
- Ports: Web `127.0.0.1:3000`, Canonical HTTP `127.0.0.1:8000`, gRPC
  `127.0.0.1:9000`.
- Team Control remains a separate optional process and database.
- Empty DB runs migrations only. Any acceptance fixture is explicit,
  local/test-only and idempotent; the normal profile never silently seeds.
- Missing required providers fail startup/readiness. Health is liveness only.
- An explicit `canonical`/`legacy` selector is retained until S7. Rollback stops
  Canonical, selects legacy, and preserves both databases and logs.

## Verification map

| Behavior | Executable evidence |
| --- | --- |
| Auth/session | Core schemas plus Canonical HTTP/session matrix |
| Workspace | schema plus HTTP/SQLite role/isolation matrix |
| Issue | body/schema/order/cursor tests plus browser trace |
| Metadata | accepted v9 tests plus real-runtime browser readback |
| Realtime | publisher ordering, handshake/reconnect and Core cache tests |
| Runtime | empty/retained/restart/readiness/close and PID/port audit |
| Cutover | browser network trace and scripted rollback/readback |
