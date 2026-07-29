---
schema: goclaw.wave/v1
wave_id: TC-W05
track_id: TEAM-RUNTIME-2026-07
title: Lease-scoped MCP and signed Runner feedback
revision: 1
plan_status: draft
wave_state: proposed
approved_by:
owner: unassigned
reviewers: []
depends_on:
  - TC-W04
created_at: 2026-07-29
updated_at: 2026-07-29
allowed_change_scope:
  - candidate scope only; see TC-W02 migration-and-wave-roadmap.md
product_code_changes_allowed: false
---

# TC-W05 r001 — Proposed MCP and feedback loop

本文件只是路线占位。目标、候选路径、non-goals、Gate 和回滚见
[`TC-W02 migration-and-wave-roadmap.md`](../tc-w02/migration-and-wave-roadmap.md)。

激活前必须新建 approved revision 和 exact Task Freeze；本 draft 不允许
实现 MCP、修改 Runner 或自动批准知识。

## 目标

提供绑定 exact ExecutionPack/lease/attempt/runner/Context 的只读 MCP，并将
签名 Runner 使用证据和反馈收敛为去重后的知识候选，绝不直接修改 active。

## 权威输入

- `TC-W04` final ContextManifest/ExecutionPack Evidence；
- `TC-W02` MCP、citation、Evidence/feedback 合同；
- 当前 Workstation lease/signature/atomic Evidence 不变量。

## 入口门禁

- [ ] TC-W04 已 `complete`，final Evidence 已索引且 P0/P1=0。
- [ ] Context/ExecutionPack identity 和 citation golden contract 已冻结。
- [ ] 新 approved revision 已冻结 MCP tools、Runner/Workstation 精确路径。
- [ ] lease/replay/cross-project threat model 与 fixtures 已确定。
- [ ] endpoint disable 与 ingestion stop 回滚可执行。

## 范围

### 包含

候选范围：

- 新/现有 MCP package；
- `teamcontrol/**`、`workstation/**`；
- `gateway/workstation.go`、`gateway/development.go`、Gateway auth；
- Runner CLI 和直接相关 tests/docs。

### 不包含

不包含 active mutation MCP、自动批准、UI/cutover、HA、外部同步或无关
Runner update。approved revision 必须冻结精确工具和文件。

## 问题与事实

| Issue ID | 表面症状 | 当前状态 | 证据 | 本 Wave 责任 |
|---|---|---|---|---|
| `TC-ISSUE-002` | Runner 缺少 frozen Context 读取和 citation/feedback 闭环 | `unverified` contract gap | TC-W02 matrix/contracts | 实现 lease-scoped read 与 candidate-only signed ingestion |

## 影响分析

| 影响面 | 当前契约 | 计划变化 | 兼容/迁移风险 |
|---|---|---|---|
| UI | 无完整 usage/candidate projection | 本 Wave 只产稳定事件/RPC contract | UI 提前假定字段 |
| RPC/API | 无 Team Control MCP | allowlisted read tools + candidate submit | tool surface 扩权 |
| 权限 | lease/runner 检查分散 | 每次 exact audience 与传递授权 | stale session 越权 |
| 数据 | Evidence 未完整记录引用效果 | signed usage/result/feedback events | replay 或重复候选 |
| 部署 | 无 MCP endpoint kill switch | explicit endpoint/ingestion disable | 半启用状态 |
| 文档 | Runner lifecycle 与知识闭环分散 | threat model/tool/state contract | 客户端误用 active API |

## 分步计划

| Step ID | 前置 | 计划动作 | 允许文件/模块 | 验证 | 状态 |
|---|---|---|---|---|---|
| `TC-W05-S01` | TC-W04 complete | activation/freeze 与 MCP audience threat model | docs/waves | binding/threat review | `proposed` |
| `TC-W05-S02` | S01 | manifest/search/read/citation/policy explain read tools | approved exact scope | tool/auth tests | `proposed` |
| `TC-W05-S03` | S02 | Evidence usage/citation/result association | approved exact scope | signed association tests | `proposed` |
| `TC-W05-S04` | S03 | signed feedback verify/dedupe/candidate-only ingestion | approved exact scope | replay/concurrency tests | `proposed` |
| `TC-W05-S05` | S04 | lease/replay/cross-project/secret/race Gate | tests | deterministic suite | `proposed` |
| `TC-W05-S06` | S05 | architecture/security/Runner-docs independent review | evidence/docs | P0/P1=0 | `proposed` |

## 验证与证据计划

| Evidence ID | 类型 | 通过条件 | 保存位置 | 状态 |
|---|---|---|---|---|
| `TC-EVID-W05-001` | MCP auth deterministic | exact audience、expiry/cancel/requeue/mismatch、citation/ref/redaction 通过 | 待 approved revision 冻结 | `planned` |
| `TC-EVID-W05-002` | feedback deterministic | signature/idempotency/concurrency/provenance 与 candidate-only 通过 | 待 approved revision 冻结 | `planned` |
| `TC-EVID-W05-003` | independent review | 工具表无 active mutation；exact candidate 三路 P0/P1=0 | `docs/waves/evidence-index.md` | `planned` |

## 风险与回滚

| 风险 | 触发信号 | 缓解 | 回滚 |
|---|---|---|---|
| stale MCP session | revoked lease 仍可读 | 每次权威 tuple 重验和即时失效 | 关闭 MCP endpoint |
| feedback replay/suppression | receipt/provenance 不一致 | signed atomic namespace + receipt | 停 ingestion，保留 Evidence |
| secret/cross-project 泄漏 | negative test 返回存在性/URI | typed ref、allowlist、redaction | fail closed，不返回存在性 |

## 退出门禁

- [ ] Runner 只能访问 frozen Context 和其 lease/pack audience。
- [ ] 全部反馈只创建 candidate，独立人工审批不变。
- [ ] Evidence 保持 signed/secret-free/atomic。
- [ ] deterministic 与三路 independent review P0/P1=0。
- [ ] Evidence 已登记，endpoint/ingestion 回滚可执行。
- [ ] TC-W06 仍未激活。

## 决策记录

| 日期 | Decision ID | 决策 | 原因与影响 |
|---|---|---|---|
| 2026-07-29 | `TC-DEC-005` | MCP 只读且逐次授权；Runner 反馈只能创建 candidate | 保持 active 人工审批和服务器端项目隔离 |

## Plan revision

本 r001 是 proposed draft。激活必须创建新 approved revision，冻结 TC-W04
Evidence、exact tool/file scope、threat model、verification 与 reviewer；
本文件不授权 MCP/Runner 产品变更。
