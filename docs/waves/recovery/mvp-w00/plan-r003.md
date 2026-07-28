---
schema: goclaw.wave/v1
wave_id: MVP-W00
track_id: MVP-RECOVERY-2026-07
title: Traceable recovery governance and final release
revision: 3
supersedes: plan-r002.md
plan_status: approved
wave_state: active
approved_by:
  - user-directive-2026-07-28
owner: Codex root agent
reviewers:
  - recovery_code_review
  - recovery_security_review
  - recovery_docs_review
depends_on: []
created_at: 2026-07-28
updated_at: 2026-07-28
steps:
  - MVP-W00-S06A
  - MVP-W00-S06B
  - MVP-W00-S06C
  - MVP-W00-S06D
allowed_change_scope:
  - docs/waves/**
  - docs/recovery/**
product_code_changes_allowed: false
---

# MVP-W00 r003 — 可追溯治理与最终恢复发布

本修订由 r002 第二轮治理复核的三个 P1 触发，不把 `BLOCK` reviewer 记作
批准者。r002 的发布实现和确定性验证保持为候选证据，但只有在本 revision
下建立完整 frozen Task、修正文档投影、重新执行全部 Gate 并通过三路复核
后，才能进入最终 recovered tag。

## 稳定 Issue

`MVP-ISSUE-001`：Recovery current projection、approval attribution 与 frozen
Task tuple 不符合仓库级治理合同。

## 目标

- Registry、当前 Plan、总 README 和 Track 对 revision/权限给出唯一投影；
- reviewer finding/trigger 与 user approval 分离，禁止越权归因；
- 从 exact repository/base/policy bundle 冻结恢复 Task；
- 旧的非合规候选不重写历史，只在合规 Task 下完整重验；
- 最终 release manifest 的 commit/tree 与 tag target 完全相同。

## 非目标

- 不再修改发布脚本、package manifest 或任何运行时代码；
- 不签名 tag/checksum，不补 SBOM/WORM/CI digest；
- 不处理 credential owner、浏览器、Codex OAuth、飞书或三台实机 Gate；
- 不 amend/rebase r001/r002 的历史提交来伪装当时已具备 frozen tuple。

## 冻结 Task

权威 Task 文件：
[`task-freeze-r003.md`](task-freeze-r003.md)。它必须包含：

- Project、Repository、assignee、base commit/tree；
- `MVP-ISSUE-001`、Wave revision/Step；
- policy bundle ID/hash；
- acceptance criteria 与确定性 verification；
- final reviewer 和回滚。

## 分步计划

| Step ID | 前置 | 动作 | 验证 |
|---|---|---|---|
| `MVP-W00-S06A` | r002 review BLOCK | 登记 Issue、冻结 Task/policy、激活 r003 | tuple/hash/Registry 可复算 |
| `MVP-W00-S06B` | S06A | 新建 FE/PILOT projection revisions，修正 README、ID 和 approval attribution | Registry/Plan/Track 一致 |
| `MVP-W00-S06C` | S06B | 写入 exact review locator，完整重跑来源、Go、Web、Obsidian、archive Gate 和三路复核 | P0/P1=0 |
| `MVP-W00-S06D` | S06C | 从最终 clean commit 连续构建两次并核对 manifest；再创建 recovered tag | commit/tree/tag/checksum 一致 |

## 验收

- [ ] `MVP-ISSUE-001` 至少达到 `verified`；
- [ ] r003 frozen Task tuple 和 policy hash 可复算；
- [ ] 当前 revisions 的 `approved_by` 只含真实批准者；
- [ ] 总 README、Registry、Plan、Track 无 current revision/权限冲突；
- [ ] Evidence/Wave ID 规则同时覆盖 FE、PILOT、MVP；
- [ ] 第二轮/最终复核对象绑定 exact commit/tree；
- [ ] 来源、Go 全包、race、vet、Web、Obsidian、archive negative 与 release
      双构建全部通过；
- [ ] 最终 manifest commit/tree 等于 recovered tag target；
- [ ] `git diff --check` 与 `git status --short` 均为空；
- [ ] 三路独立复核 P0=0、P1=0。

## 回滚

- 任一 P1 未关闭则保持 `MVP-W00 active` 或转 `blocked`，不创建 tag；
- 最终制品不一致时保留 checksum 证据，移走未发布候选后从同 commit 重建；
- 原始 2026-07-27 归档、import tag 和历史 review 记录始终只读保留。
