---
schema: goclaw.wave/v1
wave_id: FE-W03
track_id: FE-STABILITY-2026-07
title: Command workflow state machines
revision: 1
plan_status: draft
wave_state: planned
depends_on:
  - FE-W02
created_at: 2026-07-26
updated_at: 2026-07-26
allowed_change_scope:
  - Chat, Spec, Approvals, Development and Harness command flows
  - confirmed mutation RPC contract fixes
  - directly related tests and documentation
product_code_changes_allowed: true
---

# FE-W03 — 命令流程、治理与状态机

## 目标

逐条验证并修复 Team Web Console 的 23 个不同命令 RPC，使按钮可见性、
参数、角色、幂等/CAS、忙碌态、失败恢复和刷新后的权威状态与 Gateway
状态机一致。

## 入口门禁

- [ ] `FE-W02` 为 `complete`。
- [ ] 所有 loader 可稳定返回命令所依赖的权威状态。
- [ ] 每个 mutation Issue 已绑定一次性项目、角色、状态夹具和回滚。
- [ ] 高风险命令已有独立 reviewer 与职责分离测试身份。
- [ ] 禁止在真实生产项目运行破坏性验收。

## 命令清单

| 领域 | 命令 |
|---|---|
| 对话 | `agent`；事件 `chat.event` |
| Ouroboros | `session.start`、`answer`、`reassess`、`crystallize`、`compile`、`evaluate`、`evolve` |
| Knowledge | proposal approve、reject |
| Memory | candidate approve、reject |
| Seed | approve、reject |
| Development | review、freeze、accept、revise、enqueue |
| Harness | experiment approve、promote、reject、rollback |

表中的简称在证据里必须记录完整 RPC method。

## 分步计划

| Step ID | 前置 | 计划动作 | 验证 | 状态 |
|---|---|---|---|---|
| `FE-W03-S01` | W02 | 建立每个命令的允许状态、拒绝状态、角色和参数矩阵 | handler/状态机对照 | `planned` |
| `FE-W03-S02` | S01 | 验证 Chat 发送与 `chat.event` 顺序、重复、断线和 Topic/项目隔离 | 事件序列测试 | `planned` |
| `FE-W03-S03` | S01 | 验证 Ouroboros 七步状态机、超时和任务关联 | 全状态 fixture | `planned` |
| `FE-W03-S04` | S01 | 验证 Knowledge、Memory、Seed 和四审的治理输入与职责分离 | role/quorum/CAS tests | `planned` |
| `FE-W03-S05` | S01 | 验证 freeze、enqueue、revise、accept 的 revision、幂等和 DoneGate | disposable task lifecycle | `planned` |
| `FE-W03-S06` | S01 | 验证 Harness approve/promote/reject/rollback 及刷新 | experiment lifecycle | `planned` |
| `FE-W03-S07` | S02–S06 | 修复已确认问题并统一 busy、重复点击、成功刷新和错误恢复 | browser + RPC tests | `planned` |
| `FE-W03-S08` | S07 | 独立复核所有 mutation 没有绕过项目权限或治理 | security regression | `planned` |

## 命令不变量

- UI 隐藏按钮不能替代服务端拒绝。
- 重复点击不能创建重复 revision、审批、入队或提升。
- `expected_revision`、execution bundle 和证据引用必须来自当前权威状态。
- 任务创建者/assignee 不能绕过独立评审和最终验收。
- Reviewer Token 缺失或角色错误必须失败关闭。
- 命令成功后必须从服务端刷新，不以本地乐观状态宣告完成。
- timeout 不等于服务端未执行；重试前必须查询权威状态。

## 验证与证据计划

| Evidence ID | 类型 | 通过条件 | 状态 |
|---|---|---|---|
| `FE-EVID-W03-001` | command matrix | 23 个命令有允许/拒绝/重复/超时结论 | `planned` |
| `FE-EVID-W03-002` | role tests | 五类项目角色和 Governance 角色矩阵通过 | `planned` |
| `FE-EVID-W03-003` | lifecycle E2E | Spec→审批→任务→队列→证据→验收通过 | `planned` |
| `FE-EVID-W03-004` | security | 跨项目、职责分离、CSRF 和 stale revision 拒绝 | `planned` |

## 风险与回滚

| 风险 | 触发信号 | 回滚 |
|---|---|---|
| 重复 mutation | 两条审批/任务/lease | 停止 Wave，恢复一次性项目并回滚触发变更 |
| 超时后盲重试 | 服务端已执行而 UI 再提交 | 回滚重试逻辑，强制先查询状态 |
| UI 修复绕过治理 | 无 Reviewer/错误角色成功 | 立即按 S0/S1 处理并回滚 |

## 退出门禁

- [ ] 23 个命令的允许、拒绝、重复、超时和刷新通过。
- [ ] `chat.event` 顺序、重复和项目/Topic 隔离通过。
- [ ] Spec、审批、开发和 Harness 状态机通过。
- [ ] 五类项目角色和 Governance 职责分离通过。
- [ ] 所有命令结果由刷新后的权威状态确认。
- [ ] 没有自动提交、push、PR、merge 或发布范围扩张。
