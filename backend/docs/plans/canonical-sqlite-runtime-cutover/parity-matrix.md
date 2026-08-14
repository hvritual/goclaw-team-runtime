# Canonical SQLite runtime parity matrix

Status after M1-S0 discovery: proposed plan v2 freezes every critical target.
This is target design, not implementation or runtime evidence.

Classifications: `Exact`, `Compatible`, `SemanticallyEquivalent`, `Improved`,
`Deferred`, `Unknown`.

| ID | Critical behavior | Reference evidence to freeze in S0 | Target decision | Body parity | Verification | Current status |
| --- | --- | --- | --- | --- | --- | --- |
| `PAR-AUTH-01` | Login establishes a usable session | frozen auth methods/bodies/cookie/bearer | trusted Canonical session resolver | Exact | HTTP/auth matrix | Target frozen; pending S2 |
| `PAR-AUTH-02` | Current user/session restoration | `GET /api/me`; missing/expired 401 | current-user endpoint + schema | Exact | HTTP/schema/malformed tests | Target frozen; pending S2 |
| `PAR-AUTH-03` | Logout invalidates the session | repeat-safe `POST /auth/logout` | session revocation | Exact | HTTP integration | Target frozen; pending S2 |
| `PAR-WS-01` | List authorized Workspaces | `GET /api/workspaces`; installed array | membership-backed SQLite | Exact | schema/order/empty tests | Target frozen; pending S3 |
| `PAR-WS-02` | Select Workspace by frontend identity | route slug and standard headers | slug to canonical ID | Compatible | HTTP isolation matrix | Target frozen; pending S3 |
| `PAR-WS-03` | Resolve membership and role | member/agent; fail closed | Auth/Workspace adapter | SemanticallyEquivalent | table-driven auth tests | Target frozen; pending S3 |
| `PAR-ISSUE-01` | List Issues used by current page | table APIs plus list/query dependency | Issue SQLite + HTTP | Exact | body/schema/cursor/browser | Target frozen; pending S4 |
| `PAR-ISSUE-02` | Load Issue by UUID or identifier | base detail; foreign hidden | scoped resolver | Exact | HTTP/SQLite isolation | Target frozen; pending S4 |
| `PAR-ISSUE-03` | Full local Issue detail | installed detail controls and fan-out | real local routes; only external VCS remains gated | Exact/Compatible | config/view/network trace | Plan v7 approved; C4-C9 pending |
| `PAR-ISSUE-04` | Update and relative move | current Core PUT/move contracts and retained behavior | atomic Canonical update/move with public actor IDs | Exact | HTTP/SQLite/event/browser | C4 technically accepted; committed clean candidate plus user-performed browser mutation persisted without a new move 404 |
| `PAR-ISSUE-05` | Hierarchy and batch operations | current children/progress/batch contracts | cycle-safe transactional Canonical operations | Exact | hierarchy/concurrency/browser | Plan v7 approved; pending C5 |
| `PAR-COLLAB-01` | Timeline and comment lifecycle | current Core schemas plus retained handlers | Canonical activity/comment persistence | Exact/Compatible | HTTP/order/restart/browser | Plan v7 approved; pending C6 |
| `PAR-COLLAB-02` | Comment and Issue reactions | current Core schemas plus retained handlers | idempotent actor-scoped reactions | Exact | HTTP/concurrency/realtime | Plan v7 approved; pending C6 |
| `PAR-COLLAB-03` | Subscribers | current list/subscribe/unsubscribe contract | member-scoped Canonical subscriptions | Exact | HTTP/realtime/restart | Plan v7 approved; pending C6 |
| `PAR-LABEL-01` | Label catalog and Issue associations | current Core/View contract plus retained handlers | Workspace-owned definitions and atomic links | Exact | schema/isolation/realtime | Plan v7 approved; pending C7 |
| `PAR-PROP-01` | Property catalog and Issue values | current typed property contract | validated definitions and atomic complete bag | Exact | type matrix/concurrency/realtime | Plan v7 approved; pending C7 |
| `PAR-ACCEPT-01` | Acceptance conclusions | current detail contract plus SQLite-local evidence | append-only Canonical conclusions | Exact/Compatible | state/restart/browser | Plan v7 approved; pending C7 |
| `PAR-ASSET-01` | Upload and bind local attachment | current multipart/Core contract plus retained evidence | Space-owned metadata/files and Workspace refs | Compatible | size/security/rollback/browser | Plan v7 approved; pending C8 |
| `PAR-ASSET-02` | List/preview/download/delete attachment | current Core/View contract | authorized Canonical local file lifecycle | Compatible | content/MIME/hash/restart | Plan v7 approved; pending C8 |
| `PAR-META-01` | Get complete metadata bag | accepted v9 commit `e20114c` | v9 in real runtime | Exact | v9 + restart/browser | Contract integrated; pending S5 |
| `PAR-META-02` | Put one primitive metadata value | exact body/envelope | v9 in real runtime | Exact | v9 + browser | Contract integrated; pending S5 |
| `PAR-META-03` | Delete one metadata key | v9 status/error/absent | v9 in real runtime | Exact | v9 + browser | Contract integrated; pending S5 |
| `PAR-RT-01` | Authenticate realtime connection | cookie or auth frame + ack | authorized WS boundary | Compatible | connect/deny/reconnect | Target frozen; pending S6 |
| `PAR-RT-02` | Issue change refreshes correct cache | four M1 events after commit | validated event | Compatible | publisher + cache tests | Target frozen; pending S6 |
| `PAR-RT-03` | Metadata change refreshes complete bag | complete snapshot | post-commit metadata event | Exact | schema/publisher/cache | Target frozen; pending S6 |
| `PAR-RUN-01` | Empty SQLite startup | migrations only; explicit fixture | shared Canonical product DB | Improved | clean runtime probe | Target frozen; pending S1 |
| `PAR-RUN-02` | Restart retains data | canonical DB and graceful close | one product DB owner | SemanticallyEquivalent | restart/readback | Target frozen; pending S1 |
| `PAR-RUN-03` | Health and readiness reflect dependencies | liveness plus dependency readiness | bootstrap probes | Compatible | dependency failure test | Target frozen; pending S1 |
| `PAR-CUT-01` | Accepted journey has no legacy network call | frozen inventory and disabled fan-out | Web 3000 to Canonical 8000 | Exact bodies | browser/PID/port audit | Target frozen; pending S7 |
| `PAR-CUT-02` | Failed cutover is reversible | selector and two retained DBs | non-destructive switch | Improved | rollback/readback | Target frozen; pending S7 |

## Explicitly deferred parity

After plan v7, the remaining explicit deferrals are external GitHub/VCS
integration, inbox, agents, skills browsing/administration, requirements,
tasks, general administration, production PostgreSQL, desktop packaging and
mobile. Comment knowledge proposals are included only through the accepted
local evidence/candidate boundary; they do not authorize a new external agent
runtime.

Deferred endpoints must not return fabricated success. They remain visibly
unsupported or on an explicitly retained legacy route outside the Canonical-
only accepted journey.

## S0 completion rule

For every critical row:

1. link the precise legacy/reference evidence without copying implementation;
2. freeze method, URL decision, headers/context, body, status/error, state
   transition, concurrency/retry, and observable side effects;
3. assign a non-`Unknown` target classification;
4. name the executable test and expected evidence;
5. obtain Human approval for any non-Exact body behavior. Body changes are not
   allowed by this plan and therefore require stopping, not approval-in-place.
