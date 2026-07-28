---
schema: goclaw.wave/v1
wave_id: MVP-W00
track_id: MVP-RECOVERY-2026-07
title: Base-resolvable task freeze and append-only evidence repair
revision: 4
supersedes: plan-r003.md
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
  - MVP-W00-S07A
  - MVP-W00-S07B
  - MVP-W00-S07C
  - MVP-W00-S07D
  - MVP-W00-S07E
allowed_change_scope:
  - docs/waves/**
  - docs/recovery/**
product_code_changes_allowed: false
---

# MVP-W00 r004 — 可从 base 解析的 Task 与追加式证据修复

r003 final review 关闭了此前 projection/approval/tuple 内容缺口，但发现两个
新的 P1：

1. r003 Task base 上尚不存在 r003 Plan/Registry/Policy，freeze 无法从 base
   复算，形成 self-authorizing binding；
2. Recovery 期间改写了多个 Journal 的历史字节，FE-W01 冻结前缀 SHA
   `33a50e...` 不再成立。

本 revision 只做 forward-only 治理修复，不修改或重写已有 commit。

## 稳定 Issue

- `MVP-ISSUE-001`：Task/Wave/Policy 必须能从 exact base 解析；
- `MVP-ISSUE-002`：Journal 必须恢复冻结历史字节，新事件只允许 EOF 追加。

## 顺序合同

### S07A — 只激活，不冻结、不实施

第一个 commit 只能加入 r004 Plan、把 Registry 指向 r004、登记 Issue 和放入
r004 Policy manifest。它是 activation/current-projection commit，不得包含
Journal 修复、Task freeze、测试结论或完成声明。

### S07B — 从 activation base 冻结

activation commit 成为新 Task 的 exact base。只有确认该 base 已包含 active
r004 Plan、Registry 和 Policy manifest 后，才能在下一个 commit 创建 Task
freeze；产品/治理修复从再下一个 commit 开始。

### S07C — 恢复 Journal 历史完整性

- FE-W01 前 26641 bytes 必须恢复 SHA-256
  `33a50e8f3a613b1de95cb76a63ae40e9514a16e9ee9bfa4fbae8c17ffbf44555`；
- FE-W00/FE-W01/PILOT/MVP Journal 恢复各自已提交历史前缀；
- Recovery 新事件按实际时间顺序只追加到 EOF；
- current authority 由 Registry/Track/Plan 表达，不能改写历史 Journal 顶部。

### S07D/S07E — 重验、复核、最终发布

在可解析 Task 下重跑全部 Gate和三路复核。复核写回后，从最终 clean commit
连续构建两次，核对 manifest commit/tree/checksum，再创建 recovered tag。

## 验收

- [ ] activation base 含 active r004 Registry、Plan、Policy manifest；
- [ ] Task freeze 的 base commit/tree/policy 可从该 base 复算；
- [ ] activation、freeze、implementation 是三个顺序分离的 commit；
- [ ] FE-W01 冻结前缀精确恢复 `33a50e...`；
- [ ] 四个 Journal 的新 Recovery 记录只位于 EOF；
- [ ] `RECOVERY_GATE_REPORT` 当前元数据指向 r004；
- [ ] 来源、Policy、Go、Web、Obsidian、archive、双构建全部通过；
- [ ] 三路 final review P0=0、P1=0；
- [ ] 最终 manifest commit/tree 等于 tag target，checksum 和 clean tree 通过。

## 回滚

- freeze 前失败只保留 activation，不声称 Task 已授权；
- Journal 修复失败则保持 `MVP-ISSUE-002 fixing` 并停止；
- 不 amend/rebase r003 或更早历史，不覆盖原始归档，不创建 tag。
