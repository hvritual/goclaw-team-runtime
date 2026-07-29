---
schema: goclaw.wave/v1
wave_id: RN-W01
track_id: TEAM-RUNTIME-2026-07
title: Runner lifecycle with strict and Codex-delegated execution profiles
revision: 2
plan_status: approved
wave_state: active
approved_by:
  - user-directive-2026-07-29-independent-agents-and-auto-continue
  - user-decision-1A-plus-2A
owner: Codex root agent
reviewers:
  - independent_code_reviewer
  - independent_security_reviewer
  - independent_documentation_reviewer
depends_on:
  - TC-W01
supersedes:
  - docs/waves/team-runtime/rn-w01/plan-r001.md
created_at: 2026-07-29
updated_at: 2026-07-29
steps:
  - RN-W01-S01
  - RN-W01-S02
  - RN-W01-S03
  - RN-W01-S04
  - RN-W01-S05
  - RN-W01-S06
  - RN-W01-S07
  - RN-W01-S08
allowed_change_scope:
  - docs/waves/**
  - docs/**
  - teamcontrol/**
  - workstation/**
  - gateway/workstation.go
  - gateway/workstation_test.go
  - gateway/team_runtime_test.go
  - cli/runner.go
  - cli/runner_test.go
  - config/**
  - deploy/**
  - scripts/**
product_code_changes_allowed: true
---

# RN-W01 r002 — Runner 生命周期与双安全执行 Profile

## 定位

RN-W01 不重新实现编排器。它把现有 queue/lease/local Codex 闭环推进为可由
Team Control 管理、可跨平台试点、可更新回滚的 Runner 产品面，并保持现有
RPC/CLI 的默认行为。

## 决策：1A + 2A

### `strict` profile

- 保留现有 Linux/WSL2/Lima、受审 verifier sandbox、最小环境、
  Codex credential read-deny canary 与网络关闭；
- 用于高保证任务，并继续作为未显式选择时的默认；
- 缺少 OS sandbox 或 canary 时失败关闭。

### `codex-delegated` profile

- GoClaw 自身负责目录限定：仓库 root canonicalization、独立 worktree、
  allowed/denied path、safe join、Git hooks/config/TMP 清理、diff 后验检查；
- 不承诺 OS 级网络隔离，也不把该模式描述成 security sandbox；
- 超出工作目录的读写限制交给受支持的 Codex named permission profile；
- 必须由 Team Control 项目策略显式允许、Runner doctor 明确报告 posture，
  并进入 Evidence；不能静默从 `strict` 降级；
- Windows/macOS 原生试点只允许本 profile，直到对应 OS sandbox 后端通过
  独立验收。

## Team Control 职责

- 管理项目允许的 execution profile、最低 Runner contract/version、
  release pin、rollout ring、暂停/回滚开关；
- 保存 Runner posture/capability/当前与目标版本的非秘密投影；
- 在 enqueue/claim 前验证项目 policy、Runner capability 与 profile；
- 预算、成员、项目、Registry、Context Compile 继续由 Team Control
  单写；Runner 不接收 Team Token 或 Codex OAuth；
- release artifact 必须绑定 platform/arch、size、SHA-256 与 immutable
  release ID。远程获取若未实现完整 SSRF/redirect/DNS policy 则失败关闭。

## 兼容策略

- 现有字段与命令保持；新增字段均可选，缺省解析为 `strict`；
- capability 协商采用稳定字符串，不支持请求 profile 的旧 Runner 不领取；
- state migration 可重复、原子、失败不覆盖旧 state；
- 更新过程为 stage → verify → atomic activate → health confirm；失败恢复前一
  已验证版本。运行中不得原地覆盖 binary。

## Steps

| Step | 内容 | 状态 |
|---|---|---|
| `RN-W01-S01` | 激活 r002、登记风险、生成 Policy manifest | `active` |
| `RN-W01-S02` | 冻结 exact Task 与兼容矩阵 | `planned` |
| `RN-W01-S03` | execution profile 合同、migration 与 Team policy | `planned` |
| `RN-W01-S04` | 跨平台 doctor、目录限定和 capability 协商 | `planned` |
| `RN-W01-S05` | release stage/verify/atomic activate/rollback | `planned` |
| `RN-W01-S06` | 多项目公平 claim、并发与 lifecycle Evidence | `planned` |
| `RN-W01-S07` | CLI/deploy/Windows/macOS/Linux 操作文档 | `planned` |
| `RN-W01-S08` | 全量 Gate 与三路 exact final review | `planned` |

## Acceptance

- [ ] activation 与 Task Freeze 的远端 exact base/tree、Policy hash 可复核；
- [ ] 旧配置与 API 不变时继续选择 `strict`，无静默降级；
- [ ] `codex-delegated` 只有项目允许且 Runner 明确声明时才可 claim；
- [ ] 两个 profile 都做 canonical directory boundary 与 post-diff 拒绝；
- [ ] Windows/macOS/Linux doctor 准确区分可运行 profile 与阻断原因；
- [ ] release artifact identity、SHA-256、平台和架构在落盘前后复验；
- [ ] 更新/回滚原子、可恢复，运行中和不兼容版本失败关闭；
- [ ] 多项目并发无跨项目领取、饥饿或 Evidence 串线；
- [ ] Token、device key、Codex OAuth 不进入 Git、日志、Vault 或 artifact；
- [ ] 全仓 test/vet、关键 race、跨平台 build 与 CLI/deploy smoke 通过；
- [ ] final exact commit 的 code/security/docs 均 P0=0/P1=0。

## 风险与回滚

- `codex-delegated` 是显式降级，不提供恶意代码的 OS 级遏制；高风险任务
  必须使用 `strict`；
- 远程 release fetch 未满足网络策略时只允许 operator 提供已下载的本地
  artifact，不自行联网；
- 任何 profile、版本或 policy 不匹配均拒绝 claim；
- 任一 P1 未关闭：RN-W01 保持 active，INT-W01 不激活。
