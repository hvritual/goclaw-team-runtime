---
schema: goclaw.wave/v1
wave_id: XX-W00
track_id: replace-me
title: Replace me
revision: 1
plan_status: draft
wave_state: proposed
approved_by:
owner: unassigned
reviewers: []
depends_on: []
created_at: YYYY-MM-DD
updated_at: YYYY-MM-DD
allowed_change_scope: []
product_code_changes_allowed: false
---

# XX-W00 — Wave title

`plan_status` 表示计划文档是否已获批，使用 `draft`、`approved` 或
`superseded`；`wave_state` 表示执行状态，必须与 registry 一致。计划获批
不等于 Wave 已激活，只有 registry 中唯一的 `active` Wave 可以执行。

## 目标

用可观察结果描述本 Wave 完成后发生的变化。不要把实现手段写成目标。

## 权威输入

- 用户确认的目标或约束：
- 关联 Issue：
- 关联决策：
- 前置 Wave 证据：
- 当前代码/运行时事实：

无法提供来源的内容必须标为“假设”或“待确认”。

## 入口门禁

- [ ] 前置 Wave 已 `complete`。
- [ ] 每个问题都有稳定 Issue ID、复现步骤、期望与实际结果。
- [ ] 允许修改范围和非目标已冻结。
- [ ] RPC、权限、数据迁移和兼容影响已分析。
- [ ] 验证环境、命令、夹具和证据保存位置已确定。
- [ ] 回滚路径可执行。

## 范围

### 包含

- 

### 不包含

- 

## 问题与事实

| Issue ID | 表面症状 | 当前状态 | 证据 | 本 Wave 责任 |
|---|---|---|---|---|
| `XX-ISSUE-001` |  | `reported` |  |  |

允许状态：`reported`、`unverified`、`reproduced`、`root-caused`、
`fixing`、`fixed`、`verified`、`deferred`、`not-a-bug`。

## 影响分析

| 影响面 | 当前契约 | 计划变化 | 兼容/迁移风险 |
|---|---|---|---|
| UI |  |  |  |
| RPC/API |  |  |  |
| 权限 |  |  |  |
| 数据 |  |  |  |
| 部署 |  |  |  |
| 文档 |  |  |  |

## 分步计划

| Step ID | 前置 | 计划动作 | 允许文件/模块 | 验证 | 状态 |
|---|---|---|---|---|---|
| `XX-W00-S01` |  |  |  |  | `planned` |

任何 Step 的范围变化都必须先更新本表，再修改代码。

## 验证与证据计划

| Evidence ID | 类型 | 通过条件 | 保存位置 | 状态 |
|---|---|---|---|---|
| `EVID-XX-W00-001` | deterministic test |  |  | `planned` |

证据正文保存在测试报告、Trace 或 EvidencePackage；本文件只登记稳定引用。

## 风险与回滚

| 风险 | 触发信号 | 缓解 | 回滚 |
|---|---|---|---|
|  |  |  |  |

## 退出门禁

- [ ] 所有纳入范围的 Issue 已为 `verified`、`deferred` 或 `not-a-bug`，并有理由。
- [ ] 确定性测试通过。
- [ ] 权限和跨项目隔离回归通过。
- [ ] loading/empty/denied/error/disconnected 状态已验证。
- [ ] 文档、RPC 契约和部署说明与实现一致。
- [ ] `evidence-index.md` 已登记证据。
- [ ] 回滚演练或可执行性检查通过。
- [ ] 未解决风险已进入后续 Wave，不以口头说明隐藏。

## 决策记录

| 日期 | Decision ID | 决策 | 原因与影响 |
|---|---|---|---|
|  |  |  |  |

## Plan revision

本文件获批后不可原地修改。目标、范围、Step、验收、风险、回滚、基线或
验证方式发生实质变化时，复制为下一 `plan-rNNN.md`，更新 revision，
并在 `journal.md` 记录 supersedes、理由、影响和重新审批证据。
