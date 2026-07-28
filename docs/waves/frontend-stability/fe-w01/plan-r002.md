---
schema: goclaw.wave/v1
wave_id: FE-W01
track_id: FE-STABILITY-2026-07
title: Session transport repair
revision: 2
plan_status: approved
wave_state: planned
approved_by: user-directive-2026-07-26-start-repair
supersedes:
  - plan-r001
depends_on:
  - FE-W00
created_at: 2026-07-26
updated_at: 2026-07-26
allowed_change_scope:
  - ui/src/team/client.ts
  - ui/src/team/transport.ts
  - ui/vite.config.ts
  - ui/package.json
  - ui/package-lock.json
  - ui/tests/**
  - gateway/team_runtime_test.go
  - gateway/web_sessions_test.go
  - docs/waves/**
product_code_changes_allowed: true
---

# FE-W01 r002 — 会话 Transport 首批修复

## 目标

先恢复可验证的开发态登录与 WebSocket 基础，同时修复 Gateway session 测试
编译阻断。严格 Origin、CSRF、Cookie、Gateway/Team 双层身份和项目 RBAC
保持不变。

本 revision 只覆盖 `FE-ISSUE-002`、`FE-ISSUE-003` 和 `FE-ISSUE-004`。
原 r001 中项目/Topic/hash、过期会话、双标签和完整断线恢复仍留在后续 W01
revision；不得在本任务顺带修改。

## 冻结问题与验收

| Issue | 当前失败 | 修复后验收 |
|---|---|---|
| `FE-ISSUE-002` | Gateway 测试按双返回值调用三返回值函数，包编译失败 | Gateway session/origin 目标测试可编译并通过 |
| `FE-ISSUE-003` | Vite `/auth` 代理把页面 Host 改成 target Host，登录被 403 | 实际 Vite proxy 保留 Host，严格同源探针返回 204 |
| `FE-ISSUE-004` | DEV client 直连 28789，被严格 Origin 拒绝 | client 使用页面 host 的 `/ws`；代理探针返回 101 |

## 冻结任务包

| 字段 | 值 |
|---|---|
| Project-ID | `goclaw-team-runtime` |
| Task-ID | `FE-W01-TRANSPORT-R1` |
| Task-Revision | `1` |
| Work-Item | `FE-W01-S01-S03` |
| Assignee | Codex root agent |
| Base commit | 由 W01 激活 Journal 冻结为 activation commit |
| Policy bundle | `wave-governance-v1` |
| Allowed paths | 仅 frontmatter `allowed_change_scope` |
| Auto commit | 禁止；验证完成后再交给人工验收 |

## 分步计划

| Step ID | 动作 | 先失败的证明 | 通过条件 | 状态 |
|---|---|---|---|---|
| `FE-W01-S01` | 建立 Node transport tests，并恢复 Gateway session 测试编译 | 新测试在原代码失败；现有 Go 编译错误保留原始输出 | 失败原因分别锁定 URL、proxy Host 和旧签名 | planned |
| `FE-W01-S02` | 将 `/auth` proxy 改为保留页面 Host；显式冻结 `/ws.changeOrigin=false` | 实际 proxy 探针当前 403 | `/auth` 204，Host 等于 Origin.host | planned |
| `FE-W01-S03` | 让 DEV 与生产都使用当前页面 host 的 `/ws` | TeamClient 当前捕获 `:28789` | 捕获 URL 为页面 host；proxy Upgrade 101 | planned |
| `FE-W01-S04` | 执行确定性回归和范围检查 | build/test/gofmt/git diff | 所有确定性 Gate 通过，无范围外变更 | planned |
| `FE-W01-S05` | 在可达 Browser 中执行登录与连接回归 | 当前 Cloud Browser localhost blocked | 页面登录、connected、刷新、退出通过 | blocked-external |

## 测试设计

`ui/tests/team-transport.test.mjs` 使用 Node 内置 test 与项目已有 Vite，
不增加测试框架依赖：

1. Vite SSR 加载真实 TeamClient，mock `window`、`fetch`、`WebSocket`，
   断言 DEV URL 为当前页面 host 的 `/ws`；
2. 读取真实 `vite.config.ts`，启动一次性 Vite 和临时 target，断言 `/auth`
   代理后 `Host == Origin.host`；
3. 每个测试在 `finally` 关闭 server 并恢复 globals，不使用固定凭据或真实数据。

Go 安全回归增加跨端口 Origin 拒绝用例，`want` 必须保持 `false`。

## 停止条件

- 任何方案要求把跨 Origin 请求改成允许；
- 需要持久化 Gateway/Team/Reviewer Token；
- 变更进入本 revision 未列出的页面、业务 loader 或命令；
- 新测试只能通过降低 CSRF、Cookie 或双层身份要求；
- Gateway 或 UI product build 回归。

触发时停止代码修改，创建新 Issue/Plan revision。

## 验证命令

```text
npm --prefix ui run test:transport
npm --prefix ui run build
go test ./gateway -run 'Test(WebSession|WebSocketOrigin|TeamPersonalTokenAuthenticationLayers)' -count=1
go test -race ./gateway -run 'Test(WebSession|WebSocketOrigin|TeamPersonalTokenAuthenticationLayers)' -count=1
gofmt -d gateway/team_runtime_test.go gateway/web_sessions_test.go
git diff --check
git diff --name-only <base_commit>...HEAD
```

## 退出门禁

- [ ] 三个 Issue 的失败测试已先观察到，且修复后通过。
- [ ] 严格跨端口 Origin 拒绝测试仍通过。
- [ ] UI build、目标 Go test 和 race test 通过。
- [ ] diff 只落在允许路径。
- [ ] 确定性 Evidence 已写入索引。
- [ ] 独立 Reviewer 完成代码、安全与测试复核。
- [ ] Browser gate 通过；若仍被外部策略阻断，W01 保持 active/blocked，不得 complete。

