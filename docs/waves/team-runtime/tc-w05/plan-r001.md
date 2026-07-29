---
schema: goclaw.wave/v1
wave_id: TC-W05
track_id: TEAM-RUNTIME-2026-07
title: Lease-scoped MCP and signed Runner feedback
revision: 1
plan_status: draft
wave_state: proposed
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

## 权威输入

- `TC-W04` final ContextManifest/ExecutionPack Evidence；
- `TC-W02` MCP、citation、Evidence/feedback 合同；
- 当前 Workstation lease/signature/atomic Evidence 不变量。

## 范围与 non-goals

候选范围：

- 新/现有 MCP package；
- `teamcontrol/**`、`workstation/**`；
- `gateway/workstation.go`、`gateway/development.go`、Gateway auth；
- Runner CLI 和直接相关 tests/docs。

不包含 active mutation MCP、自动批准、UI/cutover、HA、外部同步或无关
Runner update。approved revision 必须冻结精确工具和文件。

## 分步计划

| Step | 内容 | 状态 |
|---|---|---|
| `TC-W05-S01` | activation/freeze 与 MCP audience threat model | proposed |
| `TC-W05-S02` | manifest/search/read/citation/policy explain read tools | proposed |
| `TC-W05-S03` | Evidence usage/citation/result association | proposed |
| `TC-W05-S04` | signed feedback verify/dedupe/candidate-only ingestion | proposed |
| `TC-W05-S05` | lease/replay/cross-project/secret/race Gate | proposed |
| `TC-W05-S06` | architecture/security/Runner-docs independent review | proposed |

## 验证与证据

- audience exact tuple 含 pack hash、lease generation/nonce、Context hash；
- lease expiry/cancel/requeue/attempt/runner mismatch；
- citation golden/fragment/checksum/authorization；
- typed opaque ref 与 transitive authorization/redaction；
- feedback signature/idempotency/concurrency/provenance aggregation；
- creator/runner/task executor 不能审批；
- 工具表没有 activate/overwrite/delete active。

## 风险与回滚

| 风险 | 缓解 | 回滚 |
|---|---|---|
| stale MCP session | 每次权威 tuple 重验和即时失效 | 关闭 MCP endpoint |
| feedback replay/suppression | signed atomic namespace + receipt | 停 ingestion，保留 Evidence |
| secret/cross-project 泄漏 | typed ref、allowlist、redaction | fail closed，不返回存在性 |

## 退出门禁

- Runner 只能访问 frozen Context 和其 lease/pack audience；
- 全部反馈只创建 candidate，独立人工审批不变；
- Evidence 保持 signed/secret-free/atomic；
- deterministic 与三路 independent review P0/P1=0；
- TC-W06 仍未激活。
