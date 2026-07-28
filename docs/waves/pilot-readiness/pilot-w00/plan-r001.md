---
schema: goclaw.wave/v1
wave_id: PILOT-W00
track_id: PILOT-READINESS-2026-07
title: Three-person controlled pilot
revision: 1
plan_status: approved
wave_state: active
approved_by:
  - user-directive-2026-07-27
  - runner_xplat_audit
  - pilot_control_audit
  - pilot_frontend_audit
owner: Codex root agent
reviewers:
  - runner_xplat_audit
  - pilot_control_audit
  - pilot_frontend_audit
depends_on:
  - FE-W00
created_at: 2026-07-27
updated_at: 2026-07-27
steps:
  - PILOT-W00-S01
  - PILOT-W00-S02
  - PILOT-W00-S03
  - PILOT-W00-S04
  - PILOT-W00-S05
  - PILOT-W00-S06
  - PILOT-W00-S07
allowed_change_scope:
  - AGENTS.md
  - README.md
  - CHANGELOG.md
  - CHANGE-HANDOFF.md
  - Makefile
  - .goreleaser.yaml
  - config.json.example
  - config/**
  - cli/**
  - gateway/**
  - internal/start/**
  - orchestratorlite/**
  - workstation/**
  - teamcontrol/**
  - ouroboros/**
  - session/**
  - ui/package.json
  - ui/package-lock.json
  - ui/src/team/**
  - ui/tests/**
  - scripts/**
  - deploy/**
  - docs/**
product_code_changes_allowed: true
---

# PILOT-W00 r001 — 三人受控试点

## 目标

把可追溯 R7 源码推进为 `0.8.0-pilot.1` 候选：一个中央控制面、一个项目、
三名真人和三台独立工作站可完成“需求/Issue → 冻结任务 → 本机 Codex →
隔离验证 → Evidence → 人工验收”的闭环，并能在 Web Console 中共同查看
项目聊天、任务、审批、Runner 和进度。

“可试点”表示通过本计划的确定性 Gate，并在目标 Linux/WSL2/Lima 环境
完成部署前预检；不表示高可用、原生三平台隔离或正式生产发布。

## 权威输入

- 用户在 2026-07-27 要求补充 Runner 跨平台需求并推进到 3 人试点；
- 用户此前冻结：Web Console 为默认控制面，Obsidian 为可选适配器；
- 每名成员使用自己的 ChatGPT 订阅与本机 Codex OAuth，禁止复制 OAuth；
- `FE-W01 r009` 的 R7 确定性证据与未解决外部阻断；
- 2026-07-27 三路只读审计：Runner、控制面、前端；
- 当前源码事实：单机文件存储、Linux bwrap、项目级 RBAC、签名 Evidence、
  Orchestrator Lite、Better Harness 与 Ouroboros 已存在。

## 入口门禁

- [x] 用户已明确授权实现三人试点和跨平台 Runner 要求。
- [x] 三路只读审计完成，P0 风险和最小范围已记录。
- [x] `FE-W00` 已 `complete`。
- [x] `FE-W01` 因外部凭据责任人与 ptrace 环境保持 `blocked`，未伪造完成。
- [x] 本计划冻结 Linux 统一执行底座，不承诺原生 Windows/macOS Runner。
- [x] 源码来自 R7 handoff 的独立目录，不覆盖旧 0.6 脏工作树。
- [ ] Go 1.25.5、Node 24、Browser 与目标平台环境可用性分别收集证据。

## 范围

### 包含

- Runner 平台识别、`runner doctor`、注册 metadata 和失败关闭支持矩阵；
- Linux/WSL2/Lima 路径、文件系统、interop、bwrap 与 Codex 登录预检；
- Unix 进程组取消、隔离临时目录、受控 Git 与恶意 hook/filter 防护；
- linux/amd64 与 linux/arm64 Runner 包；跨平台控制 CLI 构建检查；
- Task 的 machine-readable Wave binding、freeze/enqueue 双重 Gate 与 trailers；
- Team 模式禁用 raw `runner.enqueue`，统一经冻结 DevTask 进入队列；
- 不可登录的 `planner-service` 创建者和真实 `requested_by`，使三名真人在
  不降低职责分离的前提下完成两人任务评审 + 第三人最终验收；
- linked WorkItem/Issue 终态防绕过与 Ouroboros direct compile 防绕过；
- project selector、scope 清空、401 同步、共享 chat history、seq 去重、
  reconnect/focus/visibility 刷新和最小 Issue/WorkItem/Assignment 操作；
- 三 Runner 并发、租约恢复、跨项目/错 owner、幂等与 Evidence 验收；
- 冷备、manifest/hash 校验、恢复演练和一致性预检；
- 浏览器桌面/移动与三会话 E2E；环境不支持的 Gate 必须明确失败关闭。

### 不包含

- 原生 Windows Runner 的 ACL、Job Object、Windows Sandbox/AppContainer；
- 原生 macOS Runner 的 Seatbelt、Virtualization.framework 或 launchd 执行；
- 自动 commit、push、PR、merge 或 CI provider；
- 多控制面节点、数据库事务、消息队列、HA、自动升级和 TPM 密钥；
- 把 Obsidian Vault 当运行时数据库或同步 Runner worktree/Codex OAuth；
- 解决外部凭据所有者证明或绕过 ptrace syscall Gate。

## 问题与事实

| Issue ID | 表面症状 | 状态 | 本 Wave 责任 |
|---|---|---|---|
| `PILOT-ISSUE-001` | 文档声称跨平台，但只有 Linux bwrap 与 linux/amd64 Runner 包 | `root-caused` | 统一 Linux substrate、doctor、双架构包与明确失败关闭 |
| `PILOT-ISSUE-002` | Runner 取消只杀直接进程，TMP 与宿主 Git hook/filter 可逃逸 | `root-caused` | 进程组、隔离 TMP、受控 Git 与负例测试 |
| `PILOT-ISSUE-003` | Wave 只存在于文档，freeze/enqueue 没有运行时硬门禁 | `root-caused` | WaveBinding、Git-base 校验、hash/scope/step Gate |
| `PILOT-ISSUE-004` | 三个文件存储跨步骤崩溃后仅能人工判断 | `root-caused` | 一致性检查、失败关闭与显式修复/冷备 runbook |
| `PILOT-ISSUE-005` | 没有三根 + Git base 的一致冷备/恢复验证 | `root-caused` | maintenance lock、manifest/hash、restore-to-new-root |
| `PILOT-ISSUE-006` | 项目切换可短暂展示旧数据；401 不退出；聊天无历史 | `root-caused` | scope reset、auth event、history 与 seq Gate |
| `PILOT-ISSUE-007` | Team 页面只读且其他成员更新不会自动出现 | `root-caused` | 最小 mutation + visibility-aware refresh |
| `PILOT-ISSUE-008` | linked WorkItem/Issue 可绕过 DevTask DoneGate 直接终态 | `root-caused` | Gateway 终态 guard |
| `PILOT-ISSUE-009` | 严格治理下三名真人无法同时满足 creator/reviewer/final 分离 | `root-caused` | `planner-service` + `requested_by` |
| `PILOT-ISSUE-010` | manager raw enqueue 与 Ouroboros direct compile 可绕过冻结链 | `root-caused` | Team 模式禁用或统一 task factory |
| `PILOT-ISSUE-011` | 历史 credential-shaped material 缺 owner closure | `root-caused` | preflight 保持外部 blocker，不伪造代码修复 |
| `PILOT-ISSUE-012` | 当前环境 ptrace 不可用 | `root-caused` | 保留失败证据；Browser QA 不冒充 syscall 零出站 |

## 影响分析

| 影响面 | 当前契约 | 计划变化 | 兼容/迁移风险 |
|---|---|---|---|
| UI | 自由文本 project、内存 chat、读取为主 | 授权项目选择、历史恢复、最小操作、轮询刷新 | 新旧 RPC 投影需兼容 |
| RPC | Team 方法项目授权，但存在 direct bypass | 新增 `chat.history`、pilot/consistency check；收紧终态和 enqueue | 旧 Team 自动化可能被拒绝并需改入口 |
| Task | 四审、freeze、Evidence，无 Wave 字段 | WaveBinding 和 `requested_by` 进入 immutable hash/trailers | 旧任务只读兼容，不允许重新 freeze 为试点任务 |
| 权限 | 角色与 owner 校验 | service creator 不可登录；terminal guard | 需验证三真人职责分离 |
| 数据 | 多个单写文件根 | 只读一致性报告、冷备/恢复到新根 | 不提供跨根事务或热备 |
| 部署 | Linux amd64 systemd | Linux amd64/arm64、WSL2 systemd、Lima guest | host 共享目录和 native execution 被拒 |
| 安全 | verifier bwrap，宿主 Git/TMP 有缺口 | 固定 bwrap/PATH、路径/挂载/配置审计、进程组 | Git LFS/filter 在试点禁用 |

## 分步计划

| Step ID | 前置 | 计划动作 | 允许文件/模块 | 验证 | 状态 |
|---|---|---|---|---|---|
| `PILOT-W00-S01` | 入口门禁 | 登记计划、Issue、决策、基线与工具能力 | `docs/**` | JSON/frontmatter/link 检查 | `active` |
| `PILOT-W00-S02` | S01 | Runner doctor、Linux substrate、进程/TMP/Git 加固、发行资产 | `workstation/**`、`cli/**`、`scripts/**`、`deploy/**` | 单测、race、shell、交叉编译 |
| `PILOT-W00-S03` | S01 | Wave runtime Gate、planner-service、bypass/终态 guard | `orchestratorlite/**`、`gateway/**`、`teamcontrol/**`、`ouroboros/**` | table-driven + Gateway RBAC tests |
| `PILOT-W00-S04` | S03 | 一致性 check、冷备/恢复与失败关闭 | `gateway/**`、`cli/**`、`scripts/**`、`docs/**` | roundtrip/tamper/maintenance tests |
| `PILOT-W00-S05` | S01 | UI scope/auth/history/refresh/团队操作 | `ui/src/team/**`、`ui/tests/**`、`gateway/**`、`session/**` | Node/UI build + RPC tests |
| `PILOT-W00-S06` | S02–S05 | 三用户/三 Runner 并发、lease 恢复、完整治理闭环 | tests、fixtures、docs | Go race + deterministic 3-runner E2E |
| `PILOT-W00-S07` | S06 | Browser、平台 smoke、打包、部署与回滚文档 | tests、scripts、deploy、docs | Desktop/Mobile/3 context + package manifest | `planned` |

任何 Step 的范围、平台承诺、权限或验收变化都先创建 `plan-r002.md`。

## 验证与证据计划

| Evidence ID | 类型 | 通过条件 | 状态 |
|---|---|---|---|
| `PILOT-EVID-001` | baseline | R7 source manifest、工具版本、strace/Browser 能力准确记录 | `collecting` |
| `PILOT-EVID-002` | Runner tests | 平台、路径、wrapper、TMP、Git、process group 负例与 race 通过 | `planned` |
| `PILOT-EVID-003` | governance tests | Wave/bypass/终态/planner-service 正负例通过 | `planned` |
| `PILOT-EVID-004` | recovery tests | 冷备 roundtrip、tamper/运行中/缺件拒绝 | `planned` |
| `PILOT-EVID-005` | UI tests | scope/auth/history/seq/刷新/mutation 确定性测试通过 | `planned` |
| `PILOT-EVID-006` | 3-runner E2E | 三 owner 并发、签名 Evidence、accept、掉线恢复无串领 | `planned` |
| `PILOT-EVID-007` | browser | Desktop/Mobile 与三个独立 browser context 关键流程通过 | `planned` |
| `PILOT-EVID-008` | release | linux 双架构包、控制 CLI 交叉编译、SHA256/source bundle | `planned` |
| `PILOT-EVID-009` | external attestation | 历史 credential 已撤销/轮换/从未有效 | `blocked-external-owner` |

## 风险与回滚

| 风险 | 触发信号 | 缓解 | 回滚 |
|---|---|---|---|
| 旧任务没有 WaveBinding | freeze/enqueue 缺字段 | 旧任务只读；试点任务必须新建 | 关闭试点 Gate 配置仅用于退出试点，不迁旧任务 |
| 三存储出现歧义 | consistency finding 为 critical | 阻止 enqueue/accept，停机冷备后人工修复 | 恢复到新根并重新 check |
| WSL/Lima 暴露 host | interop、`/mnt/*`、共享 mount | doctor 和 executor 双重拒绝 | 停 Runner，重建 guest |
| 恶意 Git 配置执行 | hooks/filter/fsmonitor/credential/include | 固定 config 环境并拒绝仓库 | 移除 worktree，隔离原仓库 |
| UI 错 scope | project 切换后旧实体仍可见 | identity-keyed state reset | 回滚 UI bundle |
| 外部凭据证明缺失 | preflight 无 attestation | 禁止部署候选 | 只保留源码包 |
| ptrace 仍不可用 | `/bin/true` capability probe 失败 | Browser 功能 QA 与 syscall Gate 分开标记 | 不声称零出站验证通过 |

## 退出门禁

- [ ] `PILOT-ISSUE-001`–`010` 均为 `verified` 或有明确 deferred 风险。
- [ ] Go 全仓、race、vet、UI tests/build 和 scope Gate 通过。
- [ ] linux/amd64 与 linux/arm64 包内容及架构校验通过。
- [ ] 三个独立 owner 的任务并发闭环、lease 恢复与 Evidence 验证通过。
- [ ] Desktop/Mobile 和三个独立浏览器会话的项目隔离与共享更新通过。
- [ ] 冷备、篡改拒绝和 restore-to-new-root 演练通过。
- [ ] 部署前 `pilot check` 对 Token、Runner、Wave、policy、backup、外部
  credential attestation 和平台基线失败关闭。
- [ ] `evidence-index.md` 与 Journal 已登记最终证据和环境限制。
- [ ] 未通过 `PILOT-EVID-009` 时只能交付“源码候选”，不得声明已经部署试点。

## 决策记录

| 日期 | Decision ID | 决策 | 原因与影响 |
|---|---|---|---|
| 2026-07-27 | `PILOT-DEC-001` | 激活独立 Pilot Track，`FE-W01` 保持 blocked | 不伪造外部凭据/ptrace完成，同时允许用户新授权范围推进 |
| 2026-07-27 | `PILOT-DEC-002` | 三平台统一 Linux substrate | 原生 Windows/macOS 尚无同等隔离，试点必须失败关闭 |
| 2026-07-27 | `PILOT-DEC-003` | 使用不可登录 `planner-service` + `requested_by` | 三真人保持严格 reviewer/final 职责分离，不降低治理策略 |
| 2026-07-27 | `PILOT-DEC-004` | 试点采用停机冷备和一致性失败关闭 | 文件存储无跨根事务，热备会制造不一致快照 |

## Plan revision

本 r001 由用户实现指令与三路独立只读审计批准。获批后不原地改变目标、
范围、平台承诺、Step、验收或风险；实质变化创建下一 revision。

