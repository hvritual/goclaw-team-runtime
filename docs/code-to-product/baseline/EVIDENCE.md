# 证据、范围与盘点方法

## 1. 快照信息

| 项目 | 值 |
| --- | --- |
| 仓库 | `F:\code\ai\goclaw-team-runtime` |
| 分支 | `codex/multica-six-domain-baseline` |
| 提交 | `3fab1050fb58a7dfea638b6c94f3b2e73745e9b4` |
| 提交时间 | 2026-08-12 18:53:55 +08:00 |
| 盘点日期 | 2026-08-13 |
| 模式 | current-state baseline，白盒静态证据 |
| 写入边界 | 仅 `docs/code-to-product/baseline/**` |

盘点开始时工作树已有用户未提交内容：`packages/ui/components/ui/input.tsx`、`packages/views/auth/input-controlled.test.tsx`、`.local-runtime/` 与 `ui/`。本次未修改或纳入这些内容。

## 2. 判定规则

1. 路由或页面存在，只证明界面入口存在；还要交叉检查 core query/mutation 或 API 客户端。
2. API 客户端存在，只证明客户端契约存在；没有 canonical server 处理器时不能判为完成迁移。
3. Proto、类型、目录或生成代码存在，但应用 service 返回 `NotImplemented` 时标为占位。
4. 显式 SQLite composition 能运行、默认 bootstrap 未选中时标为部分实现。
5. `server/**` 永远标为只读遗留证据；不能用于证明 `backend/**` 已完成。
6. README、旧全景文档和本地化文案只作为待核对声明，不能单独作为实现证据。
7. 未在本次实际启动或联通的能力，不声明运行验证通过。

## 3. 主要证据索引

### 3.1 仓库边界与启动方式

- `AGENTS.md`、`CLAUDE.md`：`backend/**` 是唯一可写后端，`server/**` 永久只读。
- `Makefile`：默认 `make dev/start/server` 启动 `server/cmd/sqlite-server`；根 build/test/CLI 也进入 `server/**`。
- `backend/Makefile`：canonical backend 有独立 check、test、race、vet 和 run。
- `backend/README.md`：canonical 模块、上下文和延期边界。

### 3.2 当前产品入口

- `apps/web/app/**/page.tsx`：Web 的实际路由清单。
- `apps/desktop/src/renderer/src/routes.tsx`：桌面工作区路由。
- `apps/desktop/src/renderer/src/components/window-overlay.tsx`：桌面 onboarding、工作区与邀请 overlay。
- `packages/core/paths/paths.ts`：共享工作区路径只有 issues、projects、tasks、knowledge、my-issues、skills、settings 等。
- `packages/views/layout/app-sidebar.tsx`：当前侧边栏入口。
- `packages/views/settings/components/settings-page.tsx`：当前设置标签的权威前端入口。

### 3.3 功能页面与客户端契约

- `packages/views/issues/`、`packages/core/issues/`、`packages/core/api/client.ts`：issue、评论、reaction、订阅、附件、搜索、属性和批量操作。
- `packages/views/projects/`、`packages/core/projects/`、`packages/core/project-requirements/`：项目、资源、需求基线和回顾。
- `packages/views/tasks/tasks-page.tsx`、`packages/core/tasks/`：轻量普通任务。
- `packages/views/skills/`、`packages/core/skills/`：skill 列表、详情、文件和权限。
- `packages/views/knowledge/`、`packages/core/knowledge/`：知识提议、审核、来源和修订。
- `packages/views/team-control/`、`packages/core/team-control/`：Team Control 页面、命令、投影和 SSE。
- `packages/views/settings/components/`：账号、偏好、PAT、工作区、成员、角色、标签和属性。
- `packages/views/auth/`、`apps/web/app/(auth)/`、`apps/web/app/auth/callback/`：验证码、OAuth、CLI 与桌面授权回调。

### 3.4 Canonical backend

- `backend/cmd/server/main.go`、`backend/internal/bootstrap/runtime.go`：默认四模块 HTTP/gRPC 入口。
- `backend/docs/architecture/modules/auth.md`：Member opt-in 与默认 stub/延期项。
- `backend/docs/architecture/modules/workspace.md`：Workspace 各显式 SQLite slice 及 cutover 延期。
- `backend/internal/modules/*/internal/application/*_service.go`：生成占位服务。
- `backend/internal/modules/workspace/internal/application/*_usecase.go`：Project、Relationship、Todo、Issue、Knowledge、Requirement、Setting 的实际 use case。
- `backend/internal/modules/auth/sqlite_persistence.go`、`backend/internal/modules/workspace/sqlite_workspace_chain.go`：显式 SQLite 组合。
- `backend/internal/controlplane/http.go`：Team Control health、workspace、members、projection、SSE、commands。
- `backend/internal/controlplane/kernel.go`、`p2_flows.go`：事件内核与受治理状态流。
- `backend/cmd/controlplane/main.go`：控制面默认 SQLite 和身份配置。
- `backend/openapi/team-control.v1.yaml`：Team Control HTTP 契约。

### 3.5 测试与历史验收

- `backend/internal/controlplane/*_test.go`：HTTP、SSE、身份、内核、服务和 P2 流。
- `backend/internal/modules/auth/sqlite_member_services_test.go`：Auth Member SQLite 与 gRPC。
- `backend/internal/modules/workspace/**/*_test.go`：Workspace domain、use case、SQLite 和 adapter。
- `backend/tests/contract/`：生成服务的合同形状；注意合同存在不代表业务方法已实现。
- `packages/views/**/*.test.tsx`、`packages/core/**/*.test.ts`：共享前端行为。
- `e2e/*.spec.ts`：认证、issue、评论、设置、导航、属性与 Team Control 流。
- `backend/docs/plans/tc-w01-team-control/journal.md`：Team Control 历史生产构建、Playwright 和独立评审证据。
- `docs/six-domain-verification.md`：遗留六域的历史确定性验证，不能替代 canonical 验收。

## 4. 关键冲突证据

### 4.1 路由冲突

`docs/product-overview.md` 仍列出 agents、runtimes、autopilots、inbox 和 chat；当前 `apps/web/app`、桌面 `routes.tsx`、共享 `paths.ts` 和侧边栏没有这些入口。因此标为文档漂移/仅遗留。

### 4.2 Onboarding 冲突

旧产品全景描述五步 onboarding；`packages/views/onboarding/onboarding-flow.tsx` 的步骤类型只有 `workspace`，当前流程只创建或复用工作区。

### 4.3 Mobile 冲突

根 `package.json`、CI 和文档仍有 Mobile 命令/说明，但当前 `apps/` 只有 `web`、`desktop`、`docs`，不存在 `apps/mobile`。

### 4.4 六域基线冲突

`docs/six-domain-baseline.json` 的 database/backend/protocol/CLI 证据均指向 `server/**`。它证明的是遗留六域仍在树中，不证明新后端已完成六域迁移。

### 4.5 默认运行时冲突

`backend/**` 规则称其为 canonical backend，但根 Makefile 默认仍启动遗留 SQLite server；canonical `cmd/server` 默认又选择生成占位业务服务。因此当前没有“完整产品默认跑在 canonical backend”这一事实。

## 5. 本次实际验证

- 读取并交叉检查 Git 快照、路由、页面、core client、设置、文档、Makefile、backend 模块、计划与测试清单；
- 后端只读调查代理运行 `cd backend && go test ./...` 并报告所有包通过；
- 检查本次新增 Markdown 链接与路径存在性；
- 未启动 Web、Electron、PostgreSQL、Docker、Kubernetes 或任何第三方服务；
- 未运行全量 pnpm、Playwright、根遗留 Go 测试或真实 agent smoke test。

## 6. 未知与后续建议

- 无法从静态代码确认线上到底使用遗留后端、canonical 控制面还是混合部署；
- 无法确认真实 OAuth、邮件、GitHub/VCS、对象存储和 WebSocket 的部署配置；
- PostgreSQL constructor 存在，但控制面入口固定打开 SQLite，且本次未找到 PostgreSQL 集成测试，不能判定生产就绪；
- 若要把本基线提升为运行基线，下一步应分别启动：默认 SQLite 产品、canonical `cmd/server`、Team Control `cmd/controlplane`，并记录可复现的页面/API/数据库证据；
- 若要继续迁移，应先以状态矩阵为清单，将“仅遗留”能力逐项建立 `backend/**` 版本化计划，不能修改 `server/**`。
