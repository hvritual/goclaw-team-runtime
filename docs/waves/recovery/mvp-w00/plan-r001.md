---
schema: goclaw.wave/v1
wave_id: MVP-W00
track_id: MVP-RECOVERY-2026-07
title: Authoritative source and Git baseline recovery
revision: 1
plan_status: approved
wave_state: active
approved_by:
  - user-directive-2026-07-28
owner: Codex root agent
reviewers:
  - independent-code-review
  - independent-security-review
  - independent-documentation-review
depends_on: []
created_at: 2026-07-28
updated_at: 2026-07-28
steps:
  - MVP-W00-S01
  - MVP-W00-S02
  - MVP-W00-S03
  - MVP-W00-S04
  - MVP-W00-S05
allowed_change_scope:
  - docs/waves/**
  - docs/recovery/**
  - .tool-versions
  - scripts/recovery/**
product_code_changes_allowed: false
---

# MVP-W00 r001 — 权威源码与 Git 基线恢复

## 目标

从已校验的 `0.8.0-pilot.1` 源码归档建立新的、干净且可审计的 Git
权威基线，记录来源和校验值，重放确定性发布 Gate，为后续 MVP Wave
提供唯一 base commit。

## 权威输入

- 用户指令：2026-07-28 先执行源码恢复阶段以及 MVP。
- 源码归档：`goclaw-team-runtime-source-0.8.0-pilot.1.tar.gz`。
- 归档 SHA-256：
  `cf327169e7654d2284c98482e4d885085ed6068152f5ae9cbd103ea5ffd78c8f`。
- 发布记录：`docs/PILOT_3_PERSON_RELEASE_REPORT_CN.md`。
- 历史试点记录：`docs/waves/pilot-readiness/pilot-w00/journal.md`。
- 归档不含原始 `.git`，因此新历史只能声明为“归档导入历史”。

## 入口门禁

- [x] 四个发布归档通过 `sha256sum -c`。
- [x] 解压目录与源码归档通过 `tar --compare`。
- [x] 归档成员无绝对路径、`..` 穿越或 `.git`。
- [x] 原 `goclaw/` 脏工作树保持不变。
- [x] 用户已批准恢复阶段与 MVP。

## 范围

### 包含

- 建立新的本地 Git 仓库及来源导入提交。
- 保存归档、Git tree、工具链和 Gate 证据。
- 重放 Go、UI、Obsidian adapter、构建和归档安全 Gate。
- 只有全部恢复 Gate 通过后创建恢复标签。

### 不包含

- 修复产品 Bug 或修改运行时行为。
- 伪造、重建或猜测丢失的 7 月 27 日 Git commit。
- 拆分 `team_control` 与 `runner`。
- 使用真实 Codex OAuth、飞书凭据、device key 或业务数据。
- 宣称三人实机试点已经完成。

## 影响分析

| 影响面 | 当前契约 | 计划变化 | 兼容/迁移风险 |
|---|---|---|---|
| Git | 源码归档不含历史 | 新建归档导入历史 | 历史 SHA 不可解析 |
| Runtime | `0.8.0-pilot.1` | 不改变 | 无行为迁移 |
| UI/RPC | 发布候选合同 | 不改变 | 仅重放测试 |
| 凭据 | 不进入源码 | 不改变 | 扫描失败即停止 |
| 部署 | Linux Runner、跨平台控制端 | 重放构建 | 工具链差异可能暴露失败 |
| 文档 | 历史 Wave 指向旧 SHA | 增加恢复来源和新 base | 后续任务必须重新绑定 |

## 分步计划

| Step ID | 前置 | 计划动作 | 允许文件/模块 | 验证 | 状态 |
|---|---|---|---|---|---|
| `MVP-W00-S01` | 用户批准 | 校验归档并建立 import commit | Git metadata | SHA、tar compare、tree count | `complete` |
| `MVP-W00-S02` | S01 | 激活恢复 Wave，冻结工具链 | `docs/**`、`.tool-versions` | registry 唯一 active | `active` |
| `MVP-W00-S03` | S02 | 重放 Go/UI/插件确定性 Gate | 无产品改动 | test/race/vet/build | `planned` |
| `MVP-W00-S04` | S03 | 重放跨平台构建和归档扫描 | `scripts/recovery/**`、证据文档 | archive/checksum/secret scan | `planned` |
| `MVP-W00-S05` | S04 | 独立复核、冻结 base 和标签 | `docs/**`、Git metadata | reviews + clean tree | `planned` |

## 验证与证据计划

| Evidence ID | 类型 | 通过条件 | 保存位置 | 状态 |
|---|---|---|---|---|
| `MVP-EVID-001` | provenance | SHA、成员与解压树一致 | `docs/recovery/SOURCE_PROVENANCE.md` | `collecting` |
| `MVP-EVID-002` | Go | test、race、vet 全绿 | `docs/recovery/RECOVERY_GATE_REPORT.md` | `planned` |
| `MVP-EVID-003` | Web | UI test/build 全绿 | 同上 | `planned` |
| `MVP-EVID-004` | Obsidian | adapter test/build 全绿 | 同上 | `planned` |
| `MVP-EVID-005` | Release | 跨平台构建、归档与扫描全绿 | 同上 | `planned` |
| `MVP-EVID-006` | Review | 代码、安全、文档独立复核 | `docs/recovery/RECOVERY_REVIEW.md` | `planned` |

## 风险与回滚

| 风险 | 触发信号 | 缓解 | 回滚 |
|---|---|---|---|
| 工具链缺失 | `go`/Node/npm 不可用 | 冻结版本并补齐环境 | 保留 import tag，Wave blocked |
| 新环境暴露失败 | 任一 Gate 非零 | 不改产品代码，记录失败 | 回到 import commit |
| 凭据进入归档 | 扫描命中真实秘密 | 立即停止并隔离 | 不发布恢复标签 |
| 历史 SHA 不可解析 | 旧 Evidence 引用失败 | 创建 recovery 映射 | 不伪造原 SHA |
| 范围漂移 | 产品目录出现 diff | 立即停止 | 回退未提交恢复配置 |

## 退出门禁

- [ ] Git worktree 干净，恢复证据可由另一台机器重放。
- [ ] Go 全量测试、关键包 race 和 vet 通过。
- [ ] Web Console 和 Obsidian adapter 测试及构建通过。
- [ ] Linux Runner 和跨平台控制端构建通过。
- [ ] 归档危险成员、二进制和 credential-like material 扫描通过。
- [ ] 三类独立审查无未关闭的 P0/P1。
- [ ] `evidence-index.md` 已登记全部证据。
- [ ] 创建 `v0.8.0-pilot.1-recovered.1` 标签。

## 决策记录

| 日期 | Decision ID | 决策 | 原因与影响 |
|---|---|---|---|
| 2026-07-28 | `MVP-DEC-001` | 从归档建立新历史 | 原始 `.git` 不存在，不伪造历史 |
| 2026-07-28 | `MVP-DEC-002` | Recovery 独占 active Wave | 避免实机 Pilot 与基线恢复并行漂移 |
| 2026-07-28 | `MVP-DEC-003` | Recovery 不修产品缺陷 | 任一失败进入后续修复 revision/Wave |

## Plan revision

本计划获批后不可原地修改。范围、工具链、Gate、风险、停止条件或回滚发生
实质变化时，必须创建 `plan-r002.md` 并重新审批。
