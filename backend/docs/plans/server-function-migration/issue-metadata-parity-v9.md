# Issue metadata parity matrix — plan v9

Status vocabulary: `Verified` is direct source evidence, `Inferred` is a
cross-layer conclusion, and `Unknown` is not claimed.

| Behavior | Legacy evidence | Canonical target | Frontend consumer | State |
|---|---|---|---|---|
| GET complete bag | `server/cmd/server/router.go:264`; `server/internal/handler/issue_metadata.go:130-136` | Workspace metadata service plus HTTP adapter | core client; Issue detail raw JSON | Verified source; target pending |
| PUT one key | `server/cmd/server/router.go:265`; handler `139-198` | atomic metadata Put | core client method | Verified source; target pending |
| PUT body | handler `36-42,147-155`; CLI `278-282` | exact `{"value": primitive}` | typed `IssueMetadataValue` | Verified; frozen |
| DELETE one key | router `266`; handler `201-240` | atomic metadata Delete | core client method | Verified source; target pending |
| Success body | handler `136,198,240` | exact `{"metadata":{...}}` | dedicated response schema | Verified; frozen |
| Error body | `server/internal/handler/handler.go:202-204` | exact `{"error":"..."}` | existing API error path | Verified; frozen |
| Auth and membership | router `192-194,228-233`; handler `449-487` | canonical authorizer and trusted actor | platform-injected client headers | Verified source; target pending |
| Workspace isolation | handler `454-487`; SQL `server/pkg/db/queries/issue.sql:136-153` | every read/write scoped by workspace | route-driven current workspace | Verified; frozen |
| UUID or identifier | handler `460-487` | canonical resolver | caller supplies Issue id | Verified; frozen |
| Key validation | handler `22-24,34,44-51` | domain rule | key path encoded by core | Verified; frozen |
| Primitive values | handler `54-72` | raw JSON primitive validation | string/number/bool type | Verified; frozen |
| 50-key limit | handler `30-32,166-172` | atomic repository rule | surfaced 400 | Verified; frozen |
| 8 KiB limit | handler `181-184`; DB constraint referenced at `18-29` | compact SQLite JSON bound | surfaced 400 | Verified behavior; provider measurement inferred from v8 |
| Delete missing key | SQL `146-153` | success and timestamp refresh | complete resulting bag | Verified; frozen |
| Malformed legacy bag | handler `75-88` | degrade/repair from empty | response parser defaults bag | Verified; frozen |
| Mutation event | handler `191-198,233-240`; `server/pkg/protocol/events.go:8` | complete snapshot event or declared deferred publisher | existing type/updater/refresh path | Verified source; canonical publisher pending |
| Read-only UI | `packages/views/issues/components/issue-detail.tsx:2018-2041` | unchanged shared projection | Web and Desktop reuse the view | Verified current |
| Core response validation | `packages/core/api/schemas.ts:573-604`; client `643-645` | dedicated metadata schema and parsed getIssue | shared CoreProvider | Partial current; target pending |
| New edit UI | no verified consumer | none | none | Out of scope |
| PostgreSQL/default cutover | no accepted canonical integration proof | deferred | no endpoint-origin change | Unknown/deferred |

## Error compatibility cases

| Case | Expected public result |
|---|---|
| Missing authentication | `401 {"error":"user not authenticated"}` |
| Missing workspace identity | `400 {"error":"workspace_id is required"}` |
| Invalid/missing/foreign issue | `404 {"error":"issue not found"}` |
| Invalid key | `400` with the legacy validation message |
| Invalid/missing/null/compound value | `400` with the legacy validation message |
| 51st key | `400 {"error":"metadata cannot exceed 50 keys"}` |
| Bag above limit | `400 {"error":"metadata exceeds the 8KB size limit"}` |
| Unexpected persistence failure | `500` operation-specific legacy error |

The CLI-only behavior that metadata `list` degrades a route-level 404 to `{}`
is verified in `server/cmd/multica/cmd_issue_metadata_test.go`; it remains a CLI
consumer concern and does not change HTTP 404 semantics.
