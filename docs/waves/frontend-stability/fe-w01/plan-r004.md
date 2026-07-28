---
schema: goclaw.wave/v1
wave_id: FE-W01
track_id: FE-STABILITY-2026-07
title: Session transport and dashboard shell repair
revision: 4
plan_status: approved
wave_state: active
approved_by: user-directive + wave_transition_review + transport_security_review
supersedes:
  - plan-r003
depends_on:
  - FE-W00
created_at: 2026-07-26
updated_at: 2026-07-26
allowed_change_scope:
  - ui/src/team/client.ts
  - ui/vite.config.ts
  - ui/package.json
  - ui/tests/team-transport.test.mjs
  - gateway/team_runtime_test.go
  - gateway/web_sessions_test.go
  - gateway/ui.go
  - gateway/server_auth_test.go
  - docs/waves/**
product_code_changes_allowed: true
---

# FE-W01 r004 — 会话 Transport 与 Dashboard Shell 修复

## 目标与冻结任务候选

保留 r003 已按红测完成的 transport patch，同时修复 S04 全 Gateway Gate
确认的 `/dashboard/` 301 自循环。严格 Origin、CSRF、Cookie、
Gateway/Team 双层身份、项目 RBAC、shell 安全头和缓存分层不变。

| 字段 | 冻结值 |
|---|---|
| Project-ID | `goclaw-team-runtime` |
| Task-ID | `FE-W01-TRANSPORT-R1` |
| Task-Revision | `2` |
| Work-Item | `FE-W01-S01`、`FE-W01-S02`、`FE-W01-S03`、`FE-W01-S06`、`FE-W01-S04` |
| Issue | `FE-ISSUE-002`、`FE-ISSUE-003`、`FE-ISSUE-004`、`FE-ISSUE-005` |
| Assignee | Codex root agent |
| Cumulative W01 diff base | `697f50e5f428769b75061dfd859d2549dd1c330d` |
| Task Base | 待 r004 获批后的 docs-only activation commit；创建 R2 worktree 时必须在 Journal 冻结精确 SHA |
| Plan | `FE-W01 plan-r004` |
| Policy bundle | `wave-governance-v1` |
| Auto product commit | 禁止；确定性验证与独立验收后再决定 |

本 revision 不覆盖项目/Topic/hash、过期会话、双标签、业务 loader、页面命令、
旧 `DashboardHandler`、认证 handler 或浏览器策略绕行。

## r003 迁移规则

r003 worktree 中的 transport 变更保持未提交，并且已观察到对应红测与修复后
通过。`697f50e` 是累计 W01 产品 diff 的稳定比较基线，不是 Revision 2
worktree 的 Task Base。r004 获批后必须：

1. 先将获批 Plan、Registry、Track、Issue、Decision、Evidence 与 Journal 的
   原子切换形成一个 docs-only activation commit；
2. 以该 activation commit 作为 Task Base，创建 Revision 2
   branch/worktree，创建时 `HEAD == Task Base`；
3. 在 R2 Journal 冻结 activation SHA、branch、worktree 和完整任务元组，
   并先形成 docs-only freeze commit；
4. 只迁移 r004 allowlist 内的未提交 transport patch；
5. 不复用 r003 的 `node_modules`、进程或可变测试输出；
6. 在 R2 重新执行全部测试，不把 R1 的通过结果当成最终验收。

r003 worktree 保持只读证据，不 reset、不删除。

## 入口门禁

- [x] FE-W00 r003 为 `complete`，Registry 只把 FE-W01 标为 active。
- [x] `FE-ISSUE-002`–`005` 均有确定性 root-cause Evidence。
- [x] 新 Issue 的产品文件与冻结 base 相同，不是当前 transport patch 引入。
- [x] Task、Revision、Plan、Steps、Issues、累计 base 与精确 allowed paths 已候选冻结。
- [x] shell 先失败测试、成功条件、安全负例、停止条件和回滚已定义。
- [x] Wave Reviewer 批准 revision 迁移与 Step/范围语义。
- [x] Security Reviewer 批准 shell、缓存、身份与 transport 不变量。
- [x] Registry、Track index、README、Issue、全局 Decision、Evidence 与 Journal 原子切换到 r004。
- [x] `FE-ISSUE-002`–`004` 的 Task 迁移到 revision 2；`FE-ISSUE-005` 进入 `fixing`。
- [x] `FE-EVID-W01-007` reviewer 已记录；`FE-EVID-W01-008` 已建立。
- [ ] Revision 2 专用 branch/worktree 从 activation commit 创建，Task Base 精确 SHA 已记录。

入口未全部通过前，不得修改 `gateway/ui.go` 或
`gateway/server_auth_test.go`，也不得把 r003 transport patch 迁移到 R2。

## 影响分析

| 变更 | 直接影响 | 不应影响 |
|---|---|---|
| `/auth.changeOrigin=false` | Vite 开发代理保留页面 Host，严格同源 login 可达 | 生产 embedded/Caddy、Cookie、CSRF |
| `/ws.changeOrigin=false` | 冻结 Vite WebSocket proxy 的同源 Host | Gateway Origin 实现与允许来源 |
| TeamClient 使用页面 host `/ws` | DEV 经 Vite proxy，生产仍同源 | RPC URL、session、项目/Topic |
| 测试适配三返回值 | 恢复 Gateway 测试并锁定 personal-token 无 browser session | 认证实现 |
| `/dashboard/` 直接返回 embedded index | shell 从 301 loop 恢复为 200 | `/dashboard` canonical redirect、assets、auth/RBAC |
| 新增 route/cache 负例 | 锁定 shell 与静态资源合同 | 外部服务和生产数据 |

上游依赖是 Go embedded UI 与浏览器同源 Cookie；下游是所有 Team Web Console
页面。shell 不返回 200 时，页面功能无法进行真实交互验收。

## 分步计划

| Step ID | 动作 | 先失败证据 | 通过条件 | 状态 |
|---|---|---|---|---|
| `FE-W01-S01` | 保留 r003 的 Node transport 与 Go session/origin 测试语义；迁移后在 R2 重跑 | R1 中两个 transport red tests；旧 Gateway test 编译失败 | R2 中同一测试全部通过 | complete-in-r1; reverify-in-r2 |
| `FE-W01-S02` | 保留 r003 的 Vite `/auth` 与 `/ws` proxy 修复语义；迁移后在 R2 重跑 | R1 auth probe 403 | R2 auth 204、WS 101、Host 等于 Origin.host | complete-in-r1; reverify-in-r2 |
| `FE-W01-S03` | 保留 r003 的 TeamClient 同源 `/ws` 修复语义；迁移后在 R2 重跑 | R1 client 捕获 `:28789` | R2 URL/protocol/token 边界通过 | complete-in-r1; reverify-in-r2 |
| `FE-W01-S06` | 先扩展 route/cache 红测，再最小修复 `/dashboard/` shell | `/dashboard/` 稳定返回 301 `Location: ./` | slash shell 200 且无 Location；安全头和缓存分层保持 | ready-after-task-base |
| `FE-W01-S04` | 重跑全 Gateway/RBAC、race、全 Go、UI build、范围与 lockfile Gate | r003 全 Gateway 被 `FE-ISSUE-005` 阻断 | 所有确定性 Gate 通过 | blocked-by-S06 |
| `FE-W01-S05` | 在可达 Browser 验证登录、connected、刷新与退出 | Cloud Browser localhost blocked | 页面级回归通过 | blocked-external |

S06 是新增稳定 Step；不改写 S04 的全量 Gate 或 S05 的 Browser 语义。

## S06 测试合同

`gateway/server_auth_test.go` 必须先锁定：

1. `GET /dashboard` 只返回一次 `301` 到 `/dashboard/`；
2. `GET /dashboard/` 返回 `200`、无 `Location`、包含
   `<div id="root"></div>`，并精确返回
   `Content-Type: text/html; charset=utf-8`；
3. `HEAD /dashboard/` 返回 `200`、空 body，与 GET 相同的 Content-Type、
   Content-Length、安全头和 `no-store`；
4. GET/HEAD shell 保持 `X-Frame-Options: DENY`、
   `Referrer-Policy: no-referrer`、`X-Content-Type-Options: nosniff` 与
   `Cache-Control: no-store`；
5. GET/HEAD shell 的 CSP 必须与当前完整值精确相同：

   ```text
   default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self' ws: wss:; frame-ancestors 'none'; base-uri 'self'; form-action 'self'
   ```

6. 真实 embedded hashed JS 与 CSS asset 均返回 `200`、正确 Content-Type、
   `X-Content-Type-Options: nosniff` 和
   `Cache-Control: public, max-age=31536000, immutable`；
7. `/dashboard/<unknown>` 返回 `404`、无 `Location`，body 不含 shell marker，
   不允许“404 + shell”伪 fallback；
8. 不修改或绕过 session、CSRF、Origin、Gateway/Team 身份与项目 RBAC tests。

红测只允许修改测试文件；必须先观察 slash route 因 301 精确失败，再修改
`gateway/ui.go`。

## S06 实现合同

- 只对公开 route `/dashboard/` 读取
  `ui_dist/index.html` 并直接提供内容；
- 不把内部 `/ui_dist/index.html` URL path 交给 `http.FileServer`；
- `/dashboard`、`/assets/*` 与未知 path 继续走各自现有合同；
- 读取 embedded index 失败时返回 500，不泄露文件系统或凭据；
- 不引入依赖，不改变 embedded 文件内容；
- 不修改旧 `DashboardHandler` 或任何认证/RBAC handler。

本 Issue 不新增非 GET/HEAD 的 405 行为。现有静态 handler 的方法语义不是
`FE-ISSUE-005` 根因；若需收紧，必须另建 Issue 和 Plan revision。

## Transport 与安全矩阵

| 用例 | 必须结果 |
|---|---|
| WS Host `localhost:28789` + Origin `http://localhost:5173` | `false` |
| `/auth/session` 同 hostname、不同 port | 403 |
| Host/Origin 相同但 `Sec-Fetch-Site: cross-site` | 403 |
| CSRF missing / wrong / correct | false / false / true |
| Browser session RPC missing / wrong CSRF | 403 / 403 |
| Personal Token 成功 | principal 正确，`browserSession == nil` |
| Personal Token 缺失 | authentication error |
| `/dashboard/` | 200、no-store、安全头、无 redirect |
| `/assets/<embedded-hash>` | 200、immutable |
| 全 Gateway tests | 现有 TeamGuard/RBAC 全部通过 |

Gateway 的 Origin、CSRF、认证与 RBAC 实现不在允许产品修改范围。

## 验证命令

```text
npm --prefix ui run test:transport
npm --prefix ui run build
go test ./gateway -run 'TestTeamConsole' -count=1 -v
go test ./gateway -count=1
go test -race ./gateway -count=1
go test ./... -count=1
test -z "$(gofmt -l gateway/team_runtime_test.go gateway/web_sessions_test.go \
  gateway/ui.go gateway/server_auth_test.go)"
git diff --check <TASK_BASE_SHA>
git diff --check 697f50e5f428769b75061dfd859d2549dd1c330d
git diff --name-only <TASK_BASE_SHA>
git diff --name-only 697f50e5f428769b75061dfd859d2549dd1c330d
git ls-files --others --exclude-standard
sha256sum ui/package-lock.json
```

范围 Gate 必须分别检查：

1. 相对 Task Base 的 Revision 2 实际 patch；
2. 相对 `697f50e` 的累计 W01 diff；
3. 全部 untracked 路径。

三组路径合并、去重后与 frontmatter 精确 allowlist 比对。Journal 冻结真实
Task Base 后，执行命令必须把 `<TASK_BASE_SHA>` 替换为该精确值，不允许
保留 placeholder。`package-lock.json` SHA 必须保持：

```text
46fd937f66b1b7a16950df8347619831948e9dded477b7d4ba8139018974bdbb
```

## 风险、停止和回滚

| 风险/停止信号 | 立即动作 | 可执行回滚 |
|---|---|---|
| shell 仍 redirect 或出现 redirect chain | 停止 S04 | 不合并 R2；保留失败 Evidence，从 activation base 建新 revision |
| CSP/XFO/no-store 变弱或 asset 不再 immutable | 按安全回归停止 | 反向撤销 R2 的精确 `ui.go` patch；不 reset 用户文件 |
| 跨 Origin、CSRF、身份或 RBAC 回归 | 停止并新建 Plan revision | 不合并、不部署；保留失败 worktree |
| Token 出现在 URL/subprotocol/log/HTML | 按安全事件处理 | 销毁夹具并轮换受影响测试凭据；当前分支永久不部署 |
| 范围外文件或 lockfile 改变 | 停止范围审查 | Reviewer 识别后反向撤销本任务精确变更 |
| 其他全量测试失败 | 停止，不做“顺手修复” | 新建 Issue、Evidence 和 Plan revision 后再处理 |

修复在 Revision 2 专用 branch/worktree 中执行。旧 0.6.0 脏工作树与 R1
worktree 均不得修改、reset 或删除。

## 退出门禁

- [ ] transport 与 shell 均有先失败、后通过证据。
- [ ] `/dashboard/`、canonical redirect、unknown route、security headers 与缓存分层通过。
- [ ] Node auth 204、真实 WS proxy 101 和 TeamClient 同源 URL 通过。
- [ ] Origin、CSRF、session、token、TeamGuard/RBAC 安全矩阵通过。
- [ ] 全 Gateway、Gateway race、全 Go 与 UI build 通过。
- [ ] package lock 未改变；tracked/untracked 路径只包含 allowlist。
- [ ] `FE-EVID-W01-008` 已索引；全部 Evidence 通过独立代码与安全复核。
- [ ] Browser 页面级回归通过。

Browser 仍不可达时，前七项可以完成，但 W01 必须保持
`active/blocked-external`，不能标记 complete，也不能宣称全部前台功能已恢复。
