---
schema: goclaw.wave/v1
wave_id: MVP-W00
track_id: MVP-RECOVERY-2026-07
title: Correct frozen prefix authority and finalize recovered baseline
revision: 5
supersedes: plan-r004.md
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
  - MVP-W00-S08A
  - MVP-W00-S08B
  - MVP-W00-S08C
  - MVP-W00-S08D
  - MVP-W00-S08E
allowed_change_scope:
  - docs/waves/**
  - docs/recovery/**
product_code_changes_allowed: false
---

# MVP-W00 r005 — 修正冻结前缀权威值并完成恢复基线

r004 已正确拆分 activation/freeze/implementation，并恢复四个 Journal
历史前缀；但其 Plan/Task 把 FE-W01 已冻结 26641-byte SHA 从权威值
`33a50e1bbd...` 误抄为 `33a50e8f3a...`。由于已批准 Plan 与 frozen Task
不可原地改写，r004 失败关闭，本 revision 只做 forward-only 修正与重验。

## 稳定 Issue

- `MVP-ISSUE-001`：Task/Wave/Policy 必须能从 exact base 解析；
- `MVP-ISSUE-002`：Journal 必须保留冻结历史字节，新事件只在 EOF 追加；
- `MVP-ISSUE-003`：冻结 acceptance 必须引用已有权威证据中的精确 SHA。

## SHA 权威链

FE-W01 Journal 前 26641 bytes 的 SHA-256 为：

`33a50e1bbd028ca06adcee3e18df0ea62f405ff72a6e982b318720c11bccf997`

该值必须同时由以下四项复算：

1. `frontend-stability/fe-w01/plan-r009.md`；
2. `frontend-stability/fe-w01/s12-r7-traceable-deterministic-revalidation.md`；
3. import tag 中 FE-W01 Journal 的前 26641 bytes；
4. 当前 Journal 的前 26641 bytes。

任何一项不一致均失败关闭；禁止修改历史 Journal 去匹配无来源常量。

## 顺序合同

### S08A — 只激活

第一个 commit 只能加入 r005 Plan、把 Registry 指向 r005、更新 Issue 投影并
放入 r005 Policy manifest。不得同时冻结 Task、修改 Journal、写 Gate 结论
或声明完成。

### S08B — 从 activation base 冻结

activation commit 成为新 Task 的 exact base。确认该 base 已包含 active
r005 Plan、Registry 和 Policy manifest 后，才在下一个 commit 创建 Task
freeze。

### S08C — 投影与追加完整性复算

- FE-W00 import 前缀：7574 bytes /
  `bc0d4ef39e3f76ec932451b695b7a2bd1a80f7d3cf00987ee7c3bb8c903f34aa`；
- FE-W01 import 前缀：44983 bytes /
  `2c7372d4869119380f5079f70865e5b7523329b83cc51ba2cdae81c6c9cbed85`；
- FE-W01 独立冻结前缀：26641 bytes /
  `33a50e1bbd028ca06adcee3e18df0ea62f405ff72a6e982b318720c11bccf997`；
- PILOT-W00 import 前缀：7145 bytes /
  `56c300e815e91170cbaffa145d02c1f9cec97bcf35632d649b2f81fa4f4c6d3e`；
- MVP-W00 首次提交前缀：1177 bytes /
  `3ef4b2fbfcf7d5926300c03c35943d190eaf07a2216e9f25bde44dbc1805e709`。

Registry、README、Gate report、Issue、Evidence 与 Review 当前投影同步到
r005；历史 Journal 顶部不作为 current authority。

### S08D/S08E — 重验、复核、最终发布

从 clean commit 重跑来源、Policy、Go、Web、Obsidian、archive negative 和
release 双构建。三路 reviewer 必须对 exact commit/tree 给出
P0=0/P1=0，写回结论后再从最终 clean commit 连续构建两次、核对
manifest/checksum，并创建 recovered tag。

## 验收

- [ ] activation base 含 active r005 Registry、Plan、Policy manifest；
- [ ] Task freeze 的 base commit/tree/policy 可从该 base 复算；
- [ ] activation、freeze、implementation 是三个顺序分离的 commit；
- [ ] 五个冻结前缀长度/SHA 全部与 S08C 精确一致；
- [ ] r004 错误 SHA 只作为失败历史，不再出现在 current acceptance；
- [ ] current projection 唯一指向 r005；
- [ ] 来源、Policy、Go、Web、Obsidian、archive、双构建全部通过；
- [ ] 三路 final review P0=0、P1=0；
- [ ] 最终 manifest commit/tree 等于 tag target，checksum 和 clean tree 通过。

## 回滚

- freeze 前失败只保留 activation，不声称 Task 已授权；
- 任一 SHA、Gate 或 review 失败则保持 Recovery blocked；
- 不 amend/rebase r004 或更早历史，不覆盖原始归档，不创建 tag。
