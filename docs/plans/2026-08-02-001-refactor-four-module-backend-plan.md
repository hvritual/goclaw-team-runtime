---
title: Four-Module Backend and Proto Contract Plan
type: refactor
date: 2026-08-02
topic: four-module-backend
artifact_contract: multica-ddd-plan/v1
artifact_readiness: decision-required
product_contract_source: confirmed-conversation
execution: incremental
---

# Four-Module Backend and Proto Contract Plan

## Goal Capsule

- **Objective:** 将后端长期能力划分为 `workspace`、`auth`、`space`、`system` 四个模块，并为每个模块建立清晰的 Proto 服务、数据所有权、依赖方向和渐进迁移顺序。
- **Current authorized action:** Proto 基础阶段已经完成；后续授权仅扩展到 Space 的 S1a 文件上传入口迁移，包括 Go 分层实现、现有 Chi/sqlc/storage 适配和测试。S1 的可重试清理/finalization 以及其他模块和 Space S2–S4 仍保持计划状态。
- **Business invariant:** 工作区始终是租户和授权边界；任何工作区级数据访问都必须以 `workspace_id` 和成员资格为边界。
- **Compatibility invariant:** 保留现有 Chi 路由、JSON 形状、错误状态、WebSocket 事件、PostgreSQL/sqlc 行为和 SQLite-local 行为，除非后续执行文档获得单独授权。
- **Migration strategy:** 一个可验证用例一个 tracer slice；不做四模块大爆炸迁移，不保留永久转发包、双写或平行业务模型。

## Confirmed Decisions

1. 四个顶层模块为 `workspace`、`auth`、`space`、`system`。
2. 工作区模块包含项目、待办、issue、知识、需求、设置和关系服务。
3. 待办服务承接当前普通 `task` 业务，明确排除智能体执行生命周期。
4. 认证模块拥有团队成员、成员资格、工作区角色和未来智能体身份。
5. 素材模块只拥有素材的上传、存储、校验、版本、元数据和访问生命周期；消费服务拥有附件、资源和证据的业务关系。
6. 系统模块拥有智能体版本发布、升级策略，以及系统级、可版本化的 skill 目录。
7. 工作区只保存 skill 的租户级启用、配置和智能体绑定引用。
8. `RelationshipService` 只管理项目与成员、智能体的权威关系，模型为 `ProjectActorRelation(project_id, actor_type, actor_id, role)`。
9. 项目参与者角色使用 `lead`、`member`、`agent`；关系写入必须验证参与者与项目属于同一工作区。
10. 项目删除与其 `ProjectActorRelation` 清理属于工作区模块内的一致性边界。

## Target Context and Service Map

| Module | Proto services | Owned truth | Explicit non-ownership |
| --- | --- | --- | --- |
| Workspace | Project, Todo, Issue, Knowledge, Requirement, Setting, Relationship | collaboration objects, delivery state, project-actor relations, tenant-level Skill activation references | Member/Agent identity, asset bytes, global Skill versions, Agent releases |
| Auth | Member, Agent | users, memberships, roles, last-owner rule, Agent identity and authorization | Project relationships, Agent release binaries, Agent execution, Skill content |
| Space | Asset | asset identity, versions, storage object metadata, checksums, access lifecycle | attachments' business meaning, Project resource meaning, Knowledge evidence meaning |
| System | AgentRelease, Skill | Agent versions/releases/upgrades, Skill definitions/versions/publication | Agent team identity, Workspace Skill enablement/binding, Asset business links |

## Target Proto Layout

```text
server/api/
  workspace/v1/
    project.proto
    todo.proto
    issue.proto
    knowledge.proto
    requirement.proto
    setting.proto
    relationship.proto
  auth/v1/
    member.proto
    agent.proto
  space/v1/
    asset.proto
  system/v1/
    agent_release.proto
    skill.proto
```

Package names use `<module>.v1`. Go package destinations are reserved as `github.com/multica-ai/multica/server/gen/go/<module>/v1`, but generated files are outside the current scope.

## Dependency Direction

```text
Workspace -> Auth contracts
Workspace -> Space contracts
Workspace -> System contracts
System    -> Auth contracts
System    -> Space contracts

interfaces -> application -> domain
dependency -> application/domain ports
composition root -> interfaces + dependency
```

- Workspace validates Member and Agent references through Auth contracts.
- Workspace stores Asset IDs returned by Space, never Space database rows.
- Workspace references published Skill versions from System and owns only tenant-level activation/configuration.
- System targets Auth-owned Agent IDs for rollout; it does not own membership or authorization.
- System references Space-owned Asset IDs for release artifacts and versioned Skill content; it never reads Space storage rows.
- In-process collaboration uses local contract adapters; no network loopback is introduced.
- Cross-module database-table reads are not valid integration contracts.

## Current-State Mapping

| Current capability | Target owner | Compatibility note |
| --- | --- | --- |
| `workspace`, workspace settings | Workspace/Setting | Keep workspace identity, slug and request boundary |
| `project` | Workspace/Project | Keep public API and workspace filters |
| `task` from migration 235 | Workspace/Todo | Keep table/API naming during initial migration if required |
| `issue` and extensions | Workspace/Issue | First tracer slice remains Issue status update |
| Knowledge capability | Workspace/Knowledge | Preserve separate durable store and fail-open isolation |
| Project requirements | Workspace/Requirement | Reuse existing application-facing repository pattern |
| Project lead/member/agent links | Workspace/Relationship | Remove duplicate Project lead truth only in an explicitly authorized slice |
| `member`, invitations, roles | Auth/Member | Preserve last-owner invariant and workspace authorization |
| future Agent identity | Auth/Agent | Distinct from Agent Release and execution lifecycle |
| attachment/avatar/media objects | Space/Asset | Consumer contexts retain relation semantics |
| Skill definition/content/version | System/Skill | Workspace retains enablement/config/binding refs |
| Agent release/version/upgrade | System/AgentRelease | Distinct from Auth-owned Agent identity |

## Execution Sequence

| Order | Document | Deliverable | Dependency |
| ---: | --- | --- | --- |
| 1 | `2026-08-02-002-refactor-proto-contract-foundation-execution.md` | four-module Proto source layout and validation | none |
| 2 | `2026-08-02-003-refactor-workspace-module-execution.md` | Workspace services migrated one use case at a time | Proto foundation; W4 depends on Auth actor contracts; W8 depends on System Skill contracts |
| 3 | `2026-08-02-004-refactor-auth-module-execution.md` | Member and Agent identity boundaries | Proto foundation |
| 4 | `2026-08-02-005-refactor-space-module-execution.md` | Asset lifecycle and adapters | Proto foundation |
| 5 | `2026-08-02-006-refactor-system-module-execution.md` | Agent release and Skill catalog | Proto foundation; Auth Agent and Space Asset contracts |
| 6 | `2026-08-02-007-refactor-module-integration-cutover-execution.md` | composition, architecture gates, legacy removal | all module slices |

Execution may interleave independent tracer slices, but no slice may bypass its ownership and compatibility checks.

## Open Decision Checkpoint

智能体执行生命周期明确不属于待办，也不等同于认证模块的智能体身份或系统模块的智能体发布。其所有权仍需产品负责人确认；确认前不创建 `AgentExecutionService` Proto，不规划持久化或运行时迁移，也不将其计入任何模块的验收范围。

## Current Scope Boundaries

### Included now

- Context Map, context glossaries and the accepted boundary ADR.
- This master plan and all execution documents.
- `.proto` service skeletons and the existing Issue status contract relocation.
- Standard `protoc` syntax validation and diff review.
- Space S1a 文件上传的 `domain → application ← dependency` 分层、Chi 接口适配和 composition-root 接线。
- dddgen 原生四模块脚手架、Buf/生成配置、访问清单、bootstrap、架构测试与 contract tests。

### Explicitly deferred

- Workspace、Auth、System 以及 Space S2–S4 的 Go 模块迁移。
- 将生成的 Kratos HTTP/gRPC 服务接入现有 Chi 生产运行时。
- Database schema or data migration.
- Route, response, event, authorization or SQLite-local behavior changes.
- Real Agent execution, release, deployment or upgrade operations.

## Acceptance Evidence

- Context Map and glossaries use one canonical meaning for Workspace, Todo, Agent, Asset, Skill Version and Project Actor Relation.
- Every planned service has exactly one owning module and one Proto source path.
- All Proto files compile together with `protoc --descriptor_set_out=/dev/null`.
- Generated Proto/OpenAPI/access output is reproducible through `make generate`.
- Future execution documents name the tracer slices, dependencies, verification, stop conditions and safe rollback boundary.

## Stop Conditions

- Stop if a slice would create a second source of truth for membership, Project actors, assets, Skills or Agent releases.
- Stop if public API compatibility, workspace isolation, last-owner safety, event ordering, or PostgreSQL/SQLite parity cannot be characterized.
- Stop before overwriting unrelated worktree changes.
- Stop before destructive schema work, deployment, real Agent execution or release operations without separate authorization.
- Stop before assigning Agent execution lifecycle ownership without explicit product-owner confirmation.
