---
schema: goclaw.wave/v1
wave_id: FE-W01
track_id: FE-STABILITY-2026-07
title: Session transport repair
revision: 3
plan_status: approved
wave_state: planned
approved_by: user-directive-2026-07-26-start-repair
supersedes:
  - plan-r002
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
  - docs/waves/**
product_code_changes_allowed: true
---

# FE-W01 r003 — 会话 Transport 首批修复

## 目标与冻结任务

恢复可验证的开发态登录和 WebSocket transport，同时恢复 Gateway session
测试编译。严格 Origin、CSRF、Cookie、Gateway/Team 双层身份和项目 RBAC
不变。

| 字段 | 冻结值 |
|---|---|
| Project-ID | `goclaw-team-runtime` |
| Task-ID | `FE-W01-TRANSPORT-R1` |
| Task-Revision | `1` |
| Work-Item | `FE-W01-S01`、`FE-W01-S02`、`FE-W01-S03`、`FE-W01-S04` |
| Issue | `FE-ISSUE-002`、`FE-ISSUE-003`、`FE-ISSUE-004` |
| Assignee | Codex root agent |
| Base commit | `b288564361fac4f09d65e2a6a7ff80362a5cc12e` |
| Plan | `FE-W01 plan-r003` |
| Policy bundle | `wave-governance-v1` |
| Auto product commit | 禁止；确定性验证与独立验收后再决定 |

本 revision 不覆盖项目/Topic/hash、过期会话、双标签、业务 loader 或页面命令。

## 入口门禁

- [ ] FE-W00 r003 为 `complete`，Registry 只把 FE-W01 标为 active。
- [x] 三个 Issue 均为 `root-caused`，并有独立 Reviewer 认可的证据。
- [x] Task base、Plan revision、Step、Issue 和 allowed paths 已冻结。
- [x] 先失败测试、成功条件、安全负例和停止条件已定义。
- [x] 不增加 npm 依赖；`package-lock.json` 不允许变化。
- [ ] 独立 Reviewer 批准本 revision 可执行。
- [ ] 激活 Journal 已记录完整冻结元组。

入口未全部通过前不得修改产品代码。

## 影响分析

| 变更 | 直接影响 | 不应影响 |
|---|---|---|
| `/auth.changeOrigin=false` | Vite 开发代理保留页面 Host，使严格同源 login 可达 | 生产 embedded/Caddy、Cookie 内容、CSRF |
| `/ws.changeOrigin=false` 显式化 | 冻结 Vite WebSocket proxy 的同源 Host 行为 | Gateway Origin 实现和允许来源 |
| TeamClient 使用页面 host `/ws` | DEV 经 Vite proxy，生产仍同源 | RPC URL、session 数据、项目/Topic |
| 修复测试三返回值 | 恢复测试编译并断言 personal-token 无 browser session | 认证实现 |
| 新增 Node/Go 负例 | 锁定 transport 与安全合同 | 生产数据和外部服务 |

上游依赖是浏览器同源 Cookie 与 Vite proxy；下游依赖是所有页面的
`TeamClient` 实时事件。若 transport 仍失败，W02/W03 不得开始。

## 分步计划

| Step ID | 动作 | 先失败证据 | 通过条件 | 状态 |
|---|---|---|---|---|
| `FE-W01-S01` | 新增 Node transport 与 Go session/origin 安全回归；修正旧测试接收值 | 原 client 捕获 `:28789`；auth proxy 403；Gateway tests 编译失败 | 每个失败分别由 URL、Host/Origin、签名测试锁定 | planned |
| `FE-W01-S02` | `/auth` 保留页面 Host；显式 `/ws.changeOrigin=false` | 实际 Vite auth probe 403 | auth probe 204，Host 等于 Origin.host | planned |
| `FE-W01-S03` | TeamClient DEV/生产统一使用当前页面 host 的 `/ws` | client transport test 失败；direct upgrade 403 | client URL 同源、protocol 精确、真实 Vite WS proxy 101 | planned |
| `FE-W01-S04` | 全 Gateway/RBAC、race、UI build、范围与锁文件回归 | 修复前 Gateway test package blocked | 所有确定性 Gate 通过 | planned |
| `FE-W01-S05` | 在可达 Browser 中验证登录、connected、刷新与退出 | Cloud Browser localhost blocked | 页面级回归通过 | blocked-external |

## Node/Vite 测试合同

`ui/tests/team-transport.test.mjs` 只使用 Node 内置 test 与已有 Vite：

1. SSR 加载真实 TeamClient；mock browser transport，断言 URL 使用
   `window.location.host`、路径 `/ws`、subprotocol 精确为 `["goclaw.v1"]`，
   URL 和 subprotocol 均不含 Gateway/Team Token；
2. 程序化读取真实 `vite.config.ts`，以 loopback 临时端口启动 Vite 与 HTTP
   target，断言 `/auth` 转发后 `Host == Origin.host` 且返回 204；
3. 以 loopback 临时端口启动真实 HTTP Upgrade target，经真实 `/ws` proxy
   发起 Upgrade，断言 target 观察到 `Host == Origin.host` 且返回 101；
4. 仅记录 method、path、host、origin 和 status；不读取 `.env`、真实配置、
   body、Cookie、Authorization 或任何凭据；
5. `finally` 关闭所有 server、socket 并恢复 globals；测试串行运行。

不新增依赖，`package-lock.json` SHA 必须与 base 完全相同。

## Go 安全与身份矩阵

| 用例 | 必须结果 |
|---|---|
| WS Host `localhost:28789` + Origin `http://localhost:5173` | `false` |
| `/auth/session` 同 hostname、不同 port | 403 |
| Host/Origin 相同但 `Sec-Fetch-Site: cross-site` | 403 |
| CSRF missing / wrong / correct | false / false / true |
| Browser session RPC missing / wrong CSRF | 403 / 403 |
| Personal Token 成功 | principal 正确，`browserSession == nil` |
| Personal Token 缺失 | authentication error |
| 全 Gateway tests | 包含现有 TeamGuard/RBAC 全部通过 |

Gateway 的 `checkWebSocketOrigin`、`validCSRFHeader`、认证实现和 RBAC handler
不在允许产品修改范围。

## 验证命令

```text
npm --prefix ui run test:transport
npm --prefix ui run build
go test ./gateway -count=1
go test -race ./gateway -count=1
go test ./... -count=1
gofmt -d gateway/team_runtime_test.go gateway/web_sessions_test.go
git diff --check b288564361fac4f09d65e2a6a7ff80362a5cc12e
git diff --name-only b288564361fac4f09d65e2a6a7ff80362a5cc12e
git ls-files --others --exclude-standard
sha256sum ui/package-lock.json
```

范围 Gate 将 `git diff --name-only <base>` 与 untracked 列表合并、去重，再与
frontmatter 的精确 allowlist 比对；出现任何其他路径即失败。不能只比较
`<base>...HEAD`。

## 风险、停止和回滚

| 风险/停止信号 | 立即动作 | 可执行回滚 |
|---|---|---|
| 跨 Origin 请求被接受 | 停止，不进入 S04 | 保留失败 worktree 只读作为 Evidence；从 base 新建 revision worktree，不部署当前分支 |
| Token 出现在 URL/subprotocol/log | 停止并按安全事件处理 | 销毁一次性夹具，轮换受影响测试凭据；当前分支不部署 |
| CSRF、身份或 RBAC 回归 | 停止并新建 Plan revision | 不合并产品 patch；从 base 建立干净 worktree，保留失败证据 |
| 范围外文件或 lockfile 改变 | 停止范围审查 | 由 Reviewer 识别并用反向 patch 撤销本任务产生的精确变更；不使用 reset/checkout 覆盖用户文件 |
| UI/Go build 回归 | 停止，不做“顺手修复” | 当前分支不合并；根因另建 Issue/revision |

修复在隔离 branch/worktree 中执行。回滚默认是“不合并、不部署并从冻结 base
建立新 revision”，不删除旧 worktree，不覆盖旧 0.6.0 脏工作树。

## 退出门禁

- [ ] 先观察到三个独立失败，再观察到修复后的对应通过。
- [ ] Node auth 与真实 WS proxy（101）测试通过。
- [ ] 所有 Origin、CSRF、browser session 和 personal-token 负例通过。
- [ ] 全 Gateway、Gateway race、全 Go 与 UI build 通过。
- [ ] package lock 未改变，范围只包含 allowlist。
- [ ] 确定性 Evidence 已索引并通过独立复核。
- [ ] Browser 页面级回归通过。

Browser 仍不可达时，前六项可完成，但 W01 必须保持 active/blocked，不能
标记 complete，也不能宣称前台已经恢复。

