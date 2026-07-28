---
schema: goclaw.wave/v1
wave_id: FE-W02
track_id: FE-STABILITY-2026-07
title: Read model and loader contracts
revision: 1
plan_status: draft
wave_state: planned
depends_on:
  - FE-W01
created_at: 2026-07-26
updated_at: 2026-07-26
allowed_change_scope:
  - all Team Web Console loader paths
  - typed RPC response adapters
  - query handlers required by confirmed issues
  - directly related tests and documentation
product_code_changes_allowed: true
---

# FE-W02 — 全页面读取模型与 RPC 契约

## 目标

让九个页面的所有读取 loader 对真实 Gateway 返回、项目权限、可选字段、
空数据、拒绝、超时和部分失败有冻结且可测试的行为，为 W03 命令状态机提供
稳定前置。

## 入口门禁

- [ ] `FE-W01` 为 `complete`。
- [ ] client/context/session 契约已冻结。
- [ ] W00 的查询异常均已拆分并绑定本 Wave。
- [ ] 每个查询有真实脱敏 fixture 和 Gateway handler/RBAC 对照。
- [ ] 是否允许局部降级已有明确决策，不能由实现者临时猜测。

## 查询契约清单

| RPC | 消费页面 |
|---|---|
| `work.items` | 总览、团队 |
| `issue.list` | 总览、团队 |
| `runner.list` | 总览、团队 |
| `policy.status` | 总览、团队 |
| `dev.tasks` | 总览、审批、开发、进度 |
| `harness.traces` | 总览、进度、Harness |
| `knowledge.proposals` | 总览、审批 |
| `ouroboros.sessions` | 规格、审批 |
| `memory.catalog.status` | 记忆 |
| `memory.catalog.list` | 记忆、审批 |
| `memory.catalog.search` | 记忆 |
| `harness.experiments` | 审批、Harness |
| `team.members` | 团队 |
| `docs.summary` | 团队 |
| `components.summary` | 团队 |
| `harness.status` | Harness |

Spec、Approvals、Development 和 Harness 虽然有命令，也必须先在本 Wave
完成 loader 契约验证，不能全部推迟到 W03。

## 分步计划

| Step ID | 前置 | 计划动作 | 验证 | 状态 |
|---|---|---|---|---|
| `FE-W02-S01` | W01 | 保存 16 个查询的真实 JSON、权限和项目归属基线 | schema/fixture diff | `planned` |
| `FE-W02-S02` | S01 | 对齐 TypeScript 类型、collection envelope、enum 与可选字段 | compile + contract tests | `planned` |
| `FE-W02-S03` | S02 | 验证并修复项目切换、竞态、取消与旧数据保留问题 | A→B 快速切换和慢响应 | `planned` |
| `FE-W02-S04` | S02 | 按冻结决策处理多路查询的整页失败或局部降级 | 单路 401/403/500/timeout | `planned` |
| `FE-W02-S05` | S03–S04 | 逐页验证 success/empty/denied/error/optional/large | 页面矩阵 | `planned` |
| `FE-W02-S06` | S05 | 独立复核统计、状态色、时间、来源和项目语义 | 语义断言与截图 | `planned` |

## 契约不变量

- UI 不自行发明后端不存在的终态、优先级或权限。
- 未授权与无数据是不同状态。
- 共享项目 `*` 数据必须有明确来源标记。
- 一个项目的慢响应不能覆盖后来选择的项目。
- 聚合指标必须由同一项目、同一快照语义的数据生成。
- Harness/Knowledge 的服务绑定项目与左侧选择器冲突时必须显式拒绝或解释。

## 验证与证据计划

| Evidence ID | 类型 | 通过条件 | 状态 |
|---|---|---|---|
| `FE-EVID-W02-001` | RPC fixtures | 16 个查询有真实合法/空/拒绝 fixture | `planned` |
| `FE-EVID-W02-002` | contract tests | 类型、枚举、envelope、optional 字段通过 | `planned` |
| `FE-EVID-W02-003` | browser | 九页 loader 状态矩阵通过 | `planned` |
| `FE-EVID-W02-004` | isolation | 快速切项目与未授权项目无旧数据泄露 | `planned` |

## 风险与回滚

| 风险 | 触发信号 | 回滚 |
|---|---|---|
| 为适配 UI 修改权威后端语义 | 其他客户端或状态机回归 | 回滚 handler，改用显式前端 adapter |
| 局部降级掩盖权限错误 | 403 被展示为空卡片 | 回滚降级并恢复 denied 状态 |
| enum “兼容”吞掉未知状态 | 未知值被错误映射为成功 | 失败关闭并记录新契约 Issue |

## 退出门禁

- [ ] 16 个查询契约全部有 fixture、权限和项目归属。
- [ ] 九页 success/empty/denied/error/optional/large 状态通过。
- [ ] 快速切项目、超时和竞态不展示旧项目数据。
- [ ] W00 enum、ambiguity、Promise.all 等观察均有证据结论。
- [ ] W03 可依赖读取后的权威状态决定可见命令。
