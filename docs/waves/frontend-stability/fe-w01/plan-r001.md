---
schema: goclaw.wave/v1
wave_id: FE-W01
track_id: FE-STABILITY-2026-07
title: Session and shell foundation
revision: 1
plan_status: draft
wave_state: planned
depends_on:
  - FE-W00
created_at: 2026-07-26
updated_at: 2026-07-26
allowed_change_scope:
  - ui/src/App.tsx
  - ui/src/team/AppShell.tsx
  - ui/src/team/LoginScreen.tsx
  - ui/src/team/client.ts
  - ui/src/team/context.tsx
  - gateway web-session and Team Console routes
  - directly related tests and documentation
product_code_changes_allowed: true
---

# FE-W01 — 会话、项目上下文与 Shell 基础

## 目标

在不改变 RBAC 和凭据边界的前提下，让登录、会话恢复、退出、项目/Topic
切换、hash 导航、WebSocket 生命周期和同源部署成为九个页面可依赖的稳定基础。

此计划只是候选范围。W00 未完成、Issue 未绑定、计划未独立复核前不得实施。

## 入口门禁

- [ ] `FE-W00` 为 `complete`。
- [ ] W00 已提供真实 Git base、构建哈希和部署拓扑。
- [ ] 纳入问题均为 `reproduced` 或 `root-caused`。
- [ ] 每个 Step 已绑定 Issue、WorkItem、Task、验证和回滚。
- [ ] Plan revision 已由非 assignee 复核。
- [ ] Registry 已把唯一 active Wave 切换为 `FE-W01`。

## 范围

### 包含

- `GET/POST/DELETE /auth/session`；
- HttpOnly/SameSite/Secure Cookie 与页面内存 CSRF；
- session 过期、撤销、401/403 与 React Provider 同步；
- `/ws` 建连、断线、指数退避、退出后停止重连；
- 项目 ID、Topic ID、Reviewer Token 的生命周期和切换；
- hash 路由、移动导航和刷新恢复；
- 直连开发环境与 Caddy 同源生产路径。

### 不包含

- 页面业务 loader 与命令语义；
- 放宽 Origin、CSRF、Gateway/Team 双层身份或项目 RBAC；
- 把 Token 写入 URL、LocalStorage、SessionStorage、Markdown 或 Trace；
- 搜索、通知等尚未确认的产品功能。

## 分步计划

| Step ID | 依赖 | 计划动作 | 验证 | 状态 |
|---|---|---|---|---|
| `FE-W01-S01` | W00 | 为 session HTTP、Provider 和 TeamClient 建立确定性契约测试 | fresh/expired/revoked、401/403、CSRF | `planned` |
| `FE-W01-S02` | S01 | 修复已确认的登录/恢复/退出状态同步问题 | 刷新、双标签、服务端撤销、退出 | `planned` |
| `FE-W01-S03` | S01 | 修复已确认的 WebSocket 建连、断线和重连问题 | 断网/恢复、退出无重连、无重复 listener | `planned` |
| `FE-W01-S04` | S02–S03 | 修复已确认的项目/Topic/hash 导航隔离问题 | A→B、Topic 切换、刷新、后退/前进 | `planned` |
| `FE-W01-S05` | S04 | 验证直连与反向代理的 `/dashboard`、`/assets`、`/auth/session`、`/rpc`、`/ws` | 同源、Cookie、CSP、Origin | `planned` |
| `FE-W01-S06` | S05 | 独立回归并冻结 W02 可依赖的 client/context 契约 | 证据包与 reviewer 结论 | `planned` |

## 必须保持的安全不变量

- 长期 Gateway Token 只参与登录交换，不复制到浏览器 Cookie。
- Team/Reviewer Token 不进入持久浏览器存储。
- 未授权项目不能以 empty 伪装成功。
- 401 不能继续展示可操作的已认证状态。
- 项目切换不能复用旧项目命令参数或实时事件。

## 验证与证据计划

| Evidence ID | 类型 | 通过条件 | 状态 |
|---|---|---|---|
| `FE-EVID-W01-001` | unit/contract | TeamClient 与 Provider 状态机覆盖 | `planned` |
| `FE-EVID-W01-002` | Gateway tests | Cookie、CSRF、Origin、撤销与 perimeter 回归 | `planned` |
| `FE-EVID-W01-003` | browser | 登录、刷新、切项目、断线、退出在桌面/移动通过 | `planned` |
| `FE-EVID-W01-004` | proxy | Caddy 同源拓扑完整通过 | `planned` |

## 风险与回滚

| 风险 | 触发信号 | 回滚 |
|---|---|---|
| 会话变更锁死全部页面 | 登录或恢复成功率回归 | 回滚本 Wave session/client 变更，保留 W00 证据 |
| 误放宽安全边界 | 跨站或无 CSRF 请求成功 | 立即停止并回滚；按 S0 登记 |
| 项目上下文串线 | A 项目事件/数据出现在 B | 停止后续 Wave，回滚并轮换测试凭据 |

## 退出门禁

- [ ] 所有纳入 Issue 独立验证通过。
- [ ] 登录、恢复、退出、401/403、过期与撤销通过。
- [ ] WebSocket 断线/重连无重复事件和退出后重连。
- [ ] 项目/Topic/hash 导航无状态串线。
- [ ] 直连和生产同源代理均通过。
- [ ] 跨项目、CSRF、Origin 和凭据存储回归通过。
- [ ] `FE-W02` 读取测试可使用冻结的 client/context 契约。
