# FE-W00 前台契约静态清单（预备证据）

- Wave：`FE-W00`
- Step：`FE-W00-S02`
- Evidence：`FE-EVID-W00-003`
- 日期：2026-07-26
- 状态：`collecting`
- 证据类型：源码静态对照

本清单只证明“当前前端引用可以在当前源码中找到 Gateway 注册或处理
位置”，不证明请求能在真实部署成功、权限语义正确、返回结构与页面一致，
也不证明页面交互可用。`FE-W00-S01` 的运行环境基线尚未完成，因此本文件
是 S02 的预备输入，不构成 S02 完成证据。

## 数量

| 类别 | 前端引用 | 找到 Gateway 路径 |
|---|---:|---:|
| Session HTTP 操作 | 3 | 3 |
| 查询 RPC | 16 | 16 |
| 命令 RPC | 23 | 23 |
| WebSocket 通知 | 1 | 1 |
| 合计 | 43 | 43 |

当前没有发现“前端调用某 RPC，但 Gateway 完全没有注册该方法”的静态
缺口。该结论不能排除参数、RBAC、项目归属、状态机或响应 envelope 不一致。

## Session HTTP

| 操作 | 前端调用 | Gateway 路径 |
|---|---|---|
| `GET /auth/session` | `ui/src/team/client.ts:33-45`，`TeamClient.resume` | `gateway/server.go:283`；`gateway/web_sessions.go:116-124,181-187` |
| `POST /auth/session` | `ui/src/team/client.ts:48-65`，`TeamClient.login` | `gateway/web_sessions.go:131-179` |
| `DELETE /auth/session` | `ui/src/team/client.ts:68-78`，`TeamClient.logout` | `gateway/web_sessions.go:190-204` |

浏览器 RPC 统一经 `POST /rpc`：前端位于
`ui/src/team/client.ts:80-109`，Gateway 的会话认证、CSRF 和 principal
解析位于 `gateway/server.go:1172-1220`。

## 查询 RPC

| # | RPC | 前端调用 | Gateway 注册/处理 | 静态项目边界 |
|---:|---|---|---|---|
| 1 | `work.items` | `OverviewPage.tsx:39`；`TeamPage.tsx:40` | `team_control.go:42,496` | handler 按 `project_id` |
| 2 | `issue.list` | `OverviewPage.tsx:40`；`TeamPage.tsx:41` | `team_control.go:37,415` | handler 按 `project_id` |
| 3 | `runner.list` | `OverviewPage.tsx:41`；`TeamPage.tsx:42` | `workstation.go:17,116-132` | 显式 `project.read` |
| 4 | `policy.status` | `OverviewPage.tsx:42`；`TeamPage.tsx:43` | `team_control.go:63,808` | handler 按项目 |
| 5 | `dev.tasks` | `OverviewPage.tsx:43`；`ApprovalsPage.tsx:49`；`DevelopmentPage.tsx:22`；`ProgressPage.tsx:27` | `development.go:17-29` | `work_item.read` |
| 6 | `harness.traces` | `OverviewPage.tsx:44`；`ProgressPage.tsx:28`；`HarnessPage.tsx:30` | `handler.go:498` | Harness 绑定项目 |
| 7 | `knowledge.proposals` | `OverviewPage.tsx:45`；`ApprovalsPage.tsx:46` | `handler.go:601` | Harness 绑定项目、`document.read` |
| 8 | `ouroboros.sessions` | `SpecPage.tsx:25`；`ApprovalsPage.tsx:50` | `ouroboros.go:18` | 请求项目、`project.read` |
| 9 | `memory.catalog.status` | `MemoryPage.tsx:34` | `memory_catalog.go:13` | 请求项目、`document.read` |
| 10 | `memory.catalog.list` | `MemoryPage.tsx:35-36`；`ApprovalsPage.tsx:47` | `memory_catalog.go:17` | 请求项目、`document.read` |
| 11 | `memory.catalog.search` | `MemoryPage.tsx:47` | `memory_catalog.go:34` | 请求项目、`document.read` |
| 12 | `team.members` | `TeamPage.tsx:39` | `team_control.go:23,736` | 项目成员列表 |
| 13 | `docs.summary` | `TeamPage.tsx:44` | `team_control.go:55,851` | handler 按项目鉴权 |
| 14 | `components.summary` | `TeamPage.tsx:45` | `team_control.go:58,903` | handler 按项目鉴权 |
| 15 | `harness.status` | `HarnessPage.tsx:28` | `handler.go:474` | Harness 绑定项目、`artifact.read` |
| 16 | `harness.experiments` | `ApprovalsPage.tsx:48`；`HarnessPage.tsx:29` | `handler.go:525` | Harness 绑定项目、`artifact.read` |

Harness/Knowledge 的集中授权位于 `gateway/team_guard.go:84-120`，
Memory 查询授权位于 `:123-168`，Ouroboros 查询授权位于 `:217-252`。

## 命令 RPC

| # | RPC | 前端调用 | Gateway 注册/处理 | 静态权限解析 |
|---:|---|---|---|---|
| 1 | `agent` | `ChatPage.tsx:56` | `handler.go:243` | 请求项目；当前 guard 使用 `project.read` |
| 2 | `ouroboros.session.start` | `SpecPage.tsx:47` | `ouroboros.go:42` | 请求项目、`work_item.write` |
| 3 | `ouroboros.session.answer` | `SpecPage.tsx:117` | `ouroboros.go:60` | Session 反查项目 |
| 4 | `ouroboros.session.reassess` | `SpecPage.tsx:127` | `ouroboros.go:77` | Session 反查项目 |
| 5 | `ouroboros.session.crystallize` | `SpecPage.tsx:128` | `ouroboros.go:88` | Session 反查项目 |
| 6 | `ouroboros.session.compile` | `SpecPage.tsx:129` | `ouroboros.go:115` | Session 反查项目 |
| 7 | `ouroboros.session.evaluate` | `SpecPage.tsx:130` | `ouroboros.go:129`；`team_guard.go:295-304` | Session 与 Task 项目交叉校验 |
| 8 | `ouroboros.session.evolve` | `SpecPage.tsx:131` | `ouroboros.go:171` | Session 反查项目 |
| 9 | `knowledge.proposal.approve` | `ApprovalsPage.tsx:101` | `handler.go:634` | Harness 绑定项目、`document.write` |
| 10 | `knowledge.proposal.reject` | `ApprovalsPage.tsx:102` | `handler.go:643` | Harness 绑定项目、`document.write` |
| 11 | `memory.catalog.candidate.approve` | `ApprovalsPage.tsx:113` | `memory_catalog.go:112` | Record 反查项目 |
| 12 | `memory.catalog.candidate.reject` | `ApprovalsPage.tsx:114` | `memory_catalog.go:120` | Record 反查项目 |
| 13 | `ouroboros.seed.approve` | `ApprovalsPage.tsx:123` | `ouroboros.go:99` | Session 反查项目、`artifact.write` |
| 14 | `ouroboros.seed.reject` | `ApprovalsPage.tsx:124` | `ouroboros.go:107` | Session 反查项目、`artifact.write` |
| 15 | `dev.task.review` | `ApprovalsPage.tsx:144-145` | `development.go:134-165` | Task 项目、Reviewer 角色 |
| 16 | `dev.task.freeze` | `ApprovalsPage.tsx:150` | `development.go:167-189` | Task 项目、assignee/manager |
| 17 | `dev.task.accept` | `ApprovalsPage.tsx:157` | `development.go:294-330` | Task 项目、manage、TaskAccept Reviewer |
| 18 | `dev.task.revise` | `ApprovalsPage.tsx:158`；`DevelopmentPage.tsx:79` | `development.go:191-292` | owner/manager、revision CAS |
| 19 | `dev.task.enqueue` | `DevelopmentPage.tsx:78` | `development.go:483,1258` | Task 反查项目、owner、冻结状态 |
| 20 | `harness.experiment.approve` | `ApprovalsPage.tsx:171`；`HarnessPage.tsx:85` | `handler.go:566` | Harness 项目、Reviewer |
| 21 | `harness.experiment.promote` | 同上，按状态选择 | `handler.go:584` | Harness 项目、Reviewer |
| 22 | `harness.experiment.reject` | `ApprovalsPage.tsx:172`；`HarnessPage.tsx:86` | `handler.go:575` | Harness 项目、Reviewer |
| 23 | `harness.rollback` | `HarnessPage.tsx:95` | `handler.go:593` | Harness 项目、Reviewer |

Ouroboros action 分类位于 `gateway/team_guard.go:254-307`，Memory
资源反查位于 `:200-214`，Harness/Knowledge 权限分类位于 `:102-120`。

## WebSocket 通知

`chat.event` 的前端订阅位于 `ui/src/team/ChatPage.tsx:13`，建连位于
`ui/src/team/client.ts:132-149`。Gateway 广播入口位于
`gateway/server.go:888`，channel 通知组装位于 `:929-940`，其他来源通知
位于 `:965-974`，项目读取过滤位于 `:925,961,992-1005`。

## 尚未验证的契约观察

以下不得直接登记为 Bug：

1. Harness/Knowledge 的 UI 调用多数不显式传当前项目，Gateway 使用单一
   `harnessSvc.ProjectID()`；必须运行验证 Shell 项目与绑定项目不一致时的
   预期行为。
2. `agent` 会发布项目消息，但当前 guard 使用 `project.read`；这是待确认
   的权限产品决策，不是仅凭静态代码即可宣布的越权。
3. Memory、Ouroboros 和 Dev 多由资源 ID 反查项目；应以跨项目负例验证
   反查和 RBAC，不能简单要求 UI 补传可伪造的项目参数。
4. channel 生成的 `chat.event` 顶层含 `project_id/topic_id`，其他来源的
   项目数据位于 `metadata`；`ChatPage` 读取顶层字段，需用真实跨渠道事件
   验证预期范围。
5. `ouroboros.evolution.approve/reject` 已在 Gateway 注册，但当前 Team Web
   Console 没有调用；是否属于本轮强制控制面范围必须由产品决策确认。

## 下一证据

- 以真实、脱敏 response fixture 对照 16 个查询的 envelope、enum 和可选字段；
- 用 authorized project A / unauthorized project B 验证每类资源归属；
- 用一次性数据跑 23 个命令的允许、拒绝、重复、超时和刷新矩阵；
- 用真实 `chat.event` 验证跨来源、项目和 Topic 过滤；
- 在 `FE-W00-S01` 环境门禁通过前，不把本文件状态改为 `passed`。
