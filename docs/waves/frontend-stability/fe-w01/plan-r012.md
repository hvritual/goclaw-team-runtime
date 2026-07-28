---
schema: goclaw.wave/v1
wave_id: FE-W01
track_id: FE-STABILITY-2026-07
title: Recovered-base browser and credential closure
revision: 12
supersedes:
  - plan-r011.md
plan_status: approved
wave_state: active
approved_by:
  - user-directive-2026-07-28
owner: Codex root agent
reviewers:
  - recovery_code_review
  - recovery_security_review
  - recovery_docs_review
depends_on:
  - FE-W00
  - MVP-W00
created_at: 2026-07-28
updated_at: 2026-07-28
steps:
  - FE-W01-S13A
  - FE-W01-S13B
  - FE-W01-S13C
  - FE-W01-S13D
  - FE-W01-S13E
allowed_change_scope:
  - docs/waves/frontend-stability/**
  - docs/recovery/**
  - ui/src/team/**
  - ui/tests/**
  - ui/vite.config.ts
  - gateway/ui.go
  - gateway/team_runtime_test.go
  - gateway/web_sessions_test.go
  - gateway/server_auth_test.go
  - session/**
  - test-plans/**
product_code_changes_allowed: true
---

# FE-W01 r012 — recovered base 浏览器与凭据闭环

本 revision 是 MVP-W01。它从
`v0.8.0-pilot.1-recovered.1` 的已验证源码继续，只关闭 FE-W01 技术/浏览器
门禁和外部凭据责任人门禁；不进入 FE-W02、新页面、视觉改版或双应用拆分。

## Target flow

The flow under test is: `/dashboard/` -> synthetic login and project selection
-> connected/chat/refresh/logout/401 states render without runtime errors.

Desktop 固定 `1440×1000`，Mobile 固定 `390×844`。至少验证：

1. 页面 identity、非空内容、无框架错误 overlay、Console 无相关 error；
2. 登录与 same-origin WebSocket connected；
3. 刷新后会话/项目作用域恢复；
4. 401 清理 session 并回到登录态；
5. 退出后 Token 不残留在 URL、WebSocket protocol、Vault 或持久化日志；
6. 项目切换不显示旧项目数据；
7. 三个独立 BrowserContext 不串项目或 session。

## 浏览器执行政策

- Browser plugin 可用时必须先用 plugin；不得先启动外部 Chrome；
- Browser plugin 缺失时使用仓库既有 Playwright；
- Browser plugin 若因 localhost/运行时能力失败，本 Plan 明确允许回退到
  仓库外临时 Playwright 脚本，必须记录精确失败原因；
- 不安装新浏览器依赖，不写入真实凭据，不把截图/trace/临时脚本提交仓库；
- 任何 UI-changing action 后必须验证可见状态、DOM、Console 和截图；
- build/test 通过不能替代真实渲染验证。

## 稳定 Issue 与外部阻断

- 技术回归：`FE-ISSUE-003`–`006`、`008`；
- credential owner：`FE-ISSUE-007` / `FE-EVID-W01-011`；
- syscall 零出站：`FE-ISSUE-010` / `FE-EVID-W01-018`。

历史技术 Issue 已标记 fixed，但必须在 recovered base 上重新验证。若出现新
失败，只登记稳定 Issue 和证据；未经新 Plan revision 不修改产品代码。

## 顺序合同

### S13A — 激活与冻结

Recovery completion、r012 Plan、Registry 和 r012 Policy manifest 先形成
docs-only transition commit。该 commit 是 Task base；下一个 commit 才冻结
完整 Task tuple。产品代码和 runtime 在 freeze 前禁止修改或启动。

### S13B — 安全环境预检

- 核对 Go 1.25.5、Node 24.14.0、npm 11.9.0 与 clean worktree；
- 确认 Browser availability 并记录 Browser/plugin/fallback 路径；
- 只用 synthetic project/principal/token marker；
- Gateway/Vite 只绑定批准的 loopback；
- provider 采用既有 inert contract；不使用真实 provider/API key；
- syscall Gate 必须先通过 `/bin/true` capability test，失败则保持
  `environment-blocked`，不以 socket polling 替代。

### S13C — 确定性与真实浏览器回归

先运行：

```bash
(cd ui && npm ci && npm test && npm run build)
go test -race -count=1 ./gateway ./session ./agent
```

再执行 Desktop、Mobile 和三个独立 context 的目标 flow。证据必须包含 URL/
title、DOM 非空、overlay、Console、交互状态与脱敏截图；不保存 raw HAR、
Token、Cookie、完整 profile 或业务数据。

### S13D — 外部 owner closure

credential owner 必须证明历史 credential-shaped material 已撤销、轮换或
从未有效。Codex 不能代替 owner 生成该证明；没有证明时技术 Gate 可以
`passed`，FE-W01 仍保持 `blocked`。

### S13E — 独立验收

确定性、浏览器、security/docs review 全部通过且 S13D 完成，才允许把
FE-W01 标记 complete。若 Browser、ptrace 或 owner 任一缺失，必须输出明确
blocked Evidence，禁止进入 MVP-W02 真实三 Runner 放行。

## Acceptance criteria

- [ ] Task base 可解析 active r012 Plan/Registry/Policy；
- [ ] activation、freeze、runtime/evidence 分离；
- [ ] deterministic UI 与 Gateway/session/agent race 通过；
- [ ] Browser path 与 fallback 原因可追溯；
- [ ] Desktop/Mobile/3-context 目标 flow 有真实渲染证据；
- [ ] ptrace capability 与 Gateway syscall 零出站 Gate 通过；
- [ ] credential owner closure 通过；
- [ ] code/security/docs final review P0=0/P1=0；
- [ ] 未修改未复现 Issue 的产品代码，未泄露 credential。

## 回滚与停止条件

- Browser plugin/fallback 均不可运行：记录 environment-blocked，停止；
- ptrace capability 失败：不启动带 provider 的 Gateway；
- 发现 Token/OAuth/device key/业务数据：立即停止并清理；
- 出现跨项目/session 泄露：登记 S0，停止所有 MVP 运行；
- 任何范围、证明机制或产品修复变化：先新建 Plan revision。
