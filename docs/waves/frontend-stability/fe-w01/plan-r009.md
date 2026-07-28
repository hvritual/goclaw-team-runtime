---
schema: goclaw.wave/v1
wave_id: FE-W01
track_id: FE-STABILITY-2026-07
title: Traceable R7 reconstruction and local Playwright gate
revision: 9
plan_status: approved
wave_state: active
approved_by: user-directive + wave_transition_review + transport_security_review + wave_docs_validate
supersedes:
  - plan-r008
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
  - channels/weworkwsbot_test.go
  - memory/catalog/ingest.go
  - memory/catalog/service_test.go
  - docs/waves/**
product_code_changes_allowed: true
---

# FE-W01 r009 — 可追溯 R7 重建与本地 Playwright Gate

## 变更原因

`FE-EVID-W01-019` 在 R6 deterministic Gate 后、任何 synthetic
credential/runtime 前发现：

- r008 freeze tuple 缺 Repository-ID 与 policy bundle hash；
- r008 activation、R6 freeze、Evidence 017 三个 commit 缺 mandatory
  trailers。

R6 产品 patch 仍未 staged/commit，且技术 Gate 已通过，但该执行链不满足
AGENTS Traceability，不能继续 S11。r009 不改产品合同或产品 allowlist，只
新增稳定 Step `FE-W01-S12`：保留 R6 为只读失败证据，从 R5 最后一个带
trailers 的 Evidence commit 重建可追溯 R7，再重新执行全部 Gate。

## 冻结 Task 候选

| 字段 | 冻结值 |
|---|---|
| Project-ID | `goclaw-team-runtime` |
| Repository-ID | `repo-goclaw-source-review` |
| Repository authority | 本地 Git 审阅仓；无 remote；R7 revision-specific worktree |
| Task-ID | `FE-W01-TRANSPORT-R1` |
| Task-Revision | `7` |
| Work-Item | `FE-W01-S01`、`FE-W01-S02`、`FE-W01-S03`、`FE-W01-S06`、`FE-W01-S07`、`FE-W01-S08`、`FE-W01-S09`、`FE-W01-S04`、`FE-W01-S12`、`FE-W01-S11`、`FE-W01-S10`、`FE-W01-S05` |
| Issue | `FE-ISSUE-002`–`FE-ISSUE-009` |
| Assignee | `Codex root agent` |
| Cumulative W01 diff base | `697f50e5f428769b75061dfd859d2549dd1c330d` |
| Reconstruction base | `5160273fb17502cf02cd10e1a17f5a47b7eb30be` |
| Task Base | 待 r009 获批后，由 reconstruction base 直接形成的 docs-only activation commit；R7 在 Journal 冻结 |
| Policy bundle ID | `wave-governance-v1` |
| Policy bundle source | repository-root `AGENTS.md` |
| Policy bundle SHA-256 | `98bacd6013032cbaffd15095012ed6fc7cd274b62a78d3fdd738aeeadff94ebf` |
| Acceptance criteria | R7 traceability/source-first/deterministic Gate；S11 inert/zero-outbound；S05 Desktop/Mobile login/connected/refresh/logout；S08 owner closure |
| Deterministic verification | r008 全部 source、manifest、Go/race/vet、UI、scope、lockfile、download manifest、config、syscall、sentinel 与 cleanup Gate |
| Auto product commit | 禁止；S08 和全部退出门禁通过后另行决定 |

Repository-ID 是本地审阅仓的稳定逻辑身份，不声称已登记到 TeamControl。
Policy hash 只覆盖当前 repository-root `AGENTS.md` 原始字节；r009 期间该文件
不得修改。若内容 SHA 漂移，停止并新建 plan revision。

## S12 恢复顺序

1. r009 只在 draft 中接受独立代码、安全和文档复核；Registry 保持 r008；
2. 获批后，不在 R6 branch 上 commit、amend、rebase、reset 或继续测试；
3. 从 reconstruction base `5160273...` 新建
   `repair/fe-w01-transport-r7`；不得从 `047306b/90278f4/d761721` 派生；
4. 将批准的 r009 Plan、批准的 r008 源计划、EVID016/017/019 和所有权威
   投影作为一个 docs-only activation commit 原子写入 R7；其中
   `plan-r008.md` SHA-256 必须精确为
   `dd25cb6397aeef4db1442ef79fea4e0a36fd3dcc2a11f5db87919f4904993392`，
   缺失或漂移即停止；commit 必须包含本计划的完整 trailers；
5. Journal 是追加式例外处理对象：必须以 reconstruction base 的 Journal
   原始 `26641` bytes 为精确前缀，其 SHA-256 为
   `33a50e1bbd028ca06adcee3e18df0ea62f405ff72a6e982b318720c11bccf997`；
   不覆盖顶部历史 r007 pointer，而是在文件尾追加 `Authority pointer` 和
   r008/r009/R6 recovery 事件，声明 Registry/尾部 pointer 才是当前权威；
6. activation commit 是 R7 Task Base；随后只追加 Journal freeze event，
   冻结 Project、Repository、Assignee、Base、Policy ID/hash、Acceptance、
   Verification、Step/Issue 与禁止自动提交，再以完整 trailers 形成
   docs-only freeze commit；
7. 在任何 Go 测试前，静默核对 reconstruction base legacy channel SHA，
   以 Delete/Add 写入 synthetic test，运行无输出 source Gate；
8. 迁移其余 10 个批准产品文件，核对 11/11 R5 manifest；
9. R7 独立 `npm ci`，复跑全部 Go/race/vet、UI、双 base scope、lockfile
   Gate，生成 `FE-EVID-W01-020`；Evidence commit 也必须有完整 trailers；
10. R1–R6 和旧 0.6.0 worktree 保留；不删除历史或未提交 patch。

R6 `FE-EVID-W01-017` 保留为技术通过、治理阻断的事实输入，不能充当 R7
绿色 Evidence，也不能授权 S11。

## Mandatory commit trailers

从 r009 activation 起，每个 docs/evidence commit 至少包含：

```text
Task-ID: FE-W01-TRANSPORT-R1
Project-ID: goclaw-team-runtime
Task-Revision: 7
Work-Item: <stable-step-id>
Wave-ID: FE-W01
Wave-Revision: r009
Wave-Step: <stable-step-id>
Issue: <one line for every related issue>
Repository-ID: repo-goclaw-source-review
Policy-Bundle: 98bacd6013032cbaffd15095012ed6fc7cd274b62a78d3fdd738aeeadff94ebf
```

缺一即停止；不得以后续补偿 commit 冒充原 commit 合规。由于产品 commit
继续被 S08 阻断，本计划不授权创建产品 commit。

## 稳定 Step 与 Evidence

| Step | 动作 | Evidence | 状态 |
|---|---|---|---|
| `FE-W01-S12` | 隔离 R6，重建完整 tuple/trailer 的 R7 | 触发输入 `FE-EVID-W01-019`；R7 输出 `020` | ready-after-approval |
| `FE-W01-S01`–`S04`、`S06`、`S07`、`S09` | R7 独立迁移与重验 | `FE-EVID-W01-020` | blocked-by-S12 |
| `FE-W01-S08` | credential owner 撤销/轮换或从未有效证明 | `FE-EVID-W01-011` | blocked-external-owner |
| `FE-W01-S11` | inert fixture、下载完整性、syscall 零出站 | `FE-EVID-W01-016`、`018` | blocked-by-EVID020 |
| `FE-W01-S10` | 仓库外 synthetic Gateway/Vite/Playwright runtime | `FE-EVID-W01-015` | blocked-by-S11 |
| `FE-W01-S05` | Desktop/Mobile 登录、connected、刷新、退出 | `FE-EVID-W01-003`、`015` | blocked-by-S10 |

`FE-EVID-W01-020` 必须同时证明：

- activation/freeze/evidence commit trailers 完整；
- Task Base 是 activation commit，freeze parent 精确等于 Task Base；
- Repository-ID 与 policy hash 精确冻结，`AGENTS.md` hash 未漂移；
- R7 Journal 前 `26641` bytes SHA 精确为
  `33a50e1bbd028ca06adcee3e18df0ea62f405ff72a6e982b318720c11bccf997`，
  r009 authority pointer 只以尾部追加表达；
- R7 中批准的 `plan-r008.md` 存在且 SHA 精确为
  `dd25cb6397aeef4db1442ef79fea4e0a36fd3dcc2a11f5db87919f4904993392`；
- source-first、11/11 manifest、全部 deterministic/scope/lockfile Gate 通过。

`FE-EVID-W01-019` 属于 R6 independent governance review，不向 rev 6
追溯新增 S12；S12 只从获批的 r009/Task rev 7 开始，首个绿色输出是
`FE-EVID-W01-020`。`5160273...` 仅是内容/ancestry anchor，不追认 R5 或
更早提交符合 r009 新增的 Repository/policy hash 合同。

## r008 合同继承

以下合同原样继承 [`plan-r008`](plan-r008.md)：

- inert provider 的 ID/API/base URL/fixed public marker/model ID/model name；
- marker holder、`0600` config/sentinel、`env -i` 和生产 provider 排除；
- Gateway 启动前 `strace -f -e trace=connect`，Gateway/子进程零出站；
- 下载工件的 package-lock、package/node_modules/Chromium regular/symlink
  canonical manifest、owner/mode/link/setid 与只读 Gate；
- Browser/Vite/Gateway 精确 loopback allowlist、deny proxy 和进程归属；
- Desktop `1440×1000`、Mobile `390×844` 的登录、connected、刷新和退出；
- 禁 raw Trace/HAR/video/network dump、input mask、exact sentinel quiet
  scan、两阶段清理；
- 页面失败只保存脱敏 Evidence，并建立下一 plan revision 后才修复；
- `FE-W01-S08 / FE-EVID-W01-011` 继续阻断 W01 complete、产品 commit 和发布。

## 风险与失败关闭

| 信号 | 动作 |
|---|---|
| r009 activation 不直接继承 reconstruction base | 停止；不得创建 R7 |
| 任一 commit trailer 缺失/值漂移 | 停止并保留 Evidence；不得用补偿 commit 掩盖 |
| Repository/Policy hash/Task tuple 不完整 | 停止；新 revision |
| R7 manifest、测试、scope、lockfile 失败 | 停止 S11；记录脱敏 Evidence |
| inert/provider/network/sentinel/browser Gate 失败 | 按 r008 失败关闭，并先建新 plan 才修产品 |

## 入口门禁

- [x] `FE-EVID-W01-019` 已定位 R6 traceability 缺口并在 credential/runtime 前停止。
- [x] R6 产品 patch 未 staged/commit，R6 将保留。
- [x] Wave、Security 与文档 Reviewer 批准 r009。
- [x] Registry 与权威投影从 reconstruction base 原子激活 r009。
- [ ] R7 Task Base、完整 tuple、trailers 与 policy hash 冻结。
- [ ] `FE-EVID-W01-020` 通过独立复核。

## 退出门禁

- [ ] S12/R7 traceability 与全部 deterministic Gate 通过。
- [ ] S11/EVID018 inert config、下载完整性、Gateway syscall 零出站通过。
- [ ] S10/S05 Desktop/Mobile 页面流程和 Evidence015 通过。
- [ ] EVID015/016/018/019/020 通过独立代码、安全和文档复核。
- [ ] S08/EVID011 外部 owner 证明通过。

前四项通过但 S08 未完成时，页面修复可完成技术验证，W01 仍保持 active；
产品补丁不得提交或发布。
