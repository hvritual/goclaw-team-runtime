---
schema: goclaw.wave/v1
wave_id: TC-W03
track_id: TEAM-RUNTIME-2026-07
title: Knowledge authority and storage convergence
revision: 1
plan_status: draft
wave_state: proposed
owner: unassigned
reviewers: []
depends_on:
  - TC-W02
created_at: 2026-07-29
updated_at: 2026-07-29
allowed_change_scope:
  - candidate scope only; see TC-W02 migration-and-wave-roadmap.md
product_code_changes_allowed: false
---

# TC-W03 r001 — Proposed knowledge authority convergence

本文件只是后继路线占位，不是 implementation 授权。目标、候选路径、
non-goals、Gate 和回滚见
[`TC-W02 migration-and-wave-roadmap.md`](../tc-w02/migration-and-wave-roadmap.md)。

激活前必须基于 TC-W02 final Evidence 创建新的 approved revision，冻结
精确 Task；本 draft 不允许产品代码变更或真实数据迁移。

## 权威输入

- `TC-W02` final P0/P1=0 Evidence；
- `TC-ISSUE-002`；
- `TC-W02 target-contracts.md` Knowledge/authority/object/index 合同；
- `TC-W02 migration-and-wave-roadmap.md` inventory/shadow/cutover 合同。

## 范围与 non-goals

候选范围：

- `teamcontrol/**`；
- `memory/catalog/**`；
- `gateway/memory_catalog.go`、`gateway/team_control.go`、`gateway/team_guard.go`；
- `cli/commands/memory_catalog.go`；
- 独立 migration tool、直接相关 tests 和 docs。

不包含 MCP、Runner Evidence、UI 全量改造、真实数据迁移/切换、HA 或外部
同步。新 approved revision 必须把上述 glob 收敛为精确文件/contract。

## 分步计划

| Step | 内容 | 状态 |
|---|---|---|
| `TC-W03-S01` | 从 TC-W02 final exact commit 激活并冻结 Task | proposed |
| `TC-W03-S02` | KnowledgeRecord/Revision/Candidate/Approval/Audit schema 与授权 | proposed |
| `TC-W03-S03` | 单一 content store、可重建 index adapter、legacy shadow import | proposed |
| `TC-W03-S04` | migration/atomic/race/rollback deterministic Gate | proposed |
| `TC-W03-S05` | architecture/security/data-docs independent review | proposed |

## 验证与证据

- project/global authorization table；
- single-active CAS、creator/reviewer separation；
- checksum/object tamper、index drop/rebuild；
- `project_id="*"` inventory/shadow/quarantine/idempotency；
- atomic write、concurrency/race、backup/rollback；
- exact commit 三路 P0=0/P1=0。

Evidence ID 在 approved activation revision 创建，draft 不预先声称通过。

## 风险与回滚

| 风险 | 缓解 | 回滚 |
|---|---|---|
| 双 active/双写 | 单写者、CAS、compat flag 禁止新 active | 回到 shadow-off base |
| 错误 global 分类 | 客户端 `*` 拒绝、global 独立审批 | quarantine，不进入 Context |
| 正文或索引成为第二权威 | object checksum + Team Control state；index 可重建 | 丢弃 shadow/index |

## 退出门禁

- Team Control 是 knowledge lifecycle/approval 的唯一逻辑权威；
- 正文只有一个 content-addressed boundary，index 可从事件重建；
- 旧 Catalog active writer 可显式关闭，真实库仍未迁移；
- migration dry-run/rollback 和三路独立复核 P0/P1=0；
- TC-W04 仍未激活。
