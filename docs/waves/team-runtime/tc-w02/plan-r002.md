---
schema: goclaw.wave/v1
wave_id: TC-W02
track_id: TEAM-RUNTIME-2026-07
title: Team Control knowledge authority contract review remediation
revision: 2
plan_status: approved
wave_state: active
approved_by:
  - user-directive-2026-07-29-team-control-authority-replan
owner: Codex root agent
reviewers:
  - independent_architecture_reviewer
  - independent_security_reviewer
  - independent_documentation_reviewer
depends_on:
  - TC-W01
supersedes:
  - docs/waves/team-runtime/tc-w02/plan-r001.md
created_at: 2026-07-29
updated_at: 2026-07-29
steps:
  - TC-W02-S05
  - TC-W02-S06
  - TC-W02-S07
  - TC-W02-S08
allowed_change_scope:
  - docs/waves/**
product_code_changes_allowed: false
---

# TC-W02 r002 — 合同审查前向修复

r001 exact candidate
`e9a4cc04cb6113fe98685ac0db809e33739c9277`
（tree `dd574e6c65b859f13ef8dcb6ce1859cc28570340`）的三路独立审查均
BLOCK：

- architecture：P0=0/P1=4/P2=1；
- security：P0=0/P1=5/P2=2；
- docs/governance：P0=0/P1=2/P2=2。

r002 完整继承 r001 的目标、权威输入、范围、non-goals、核心不变量、
风险、停止条件和后继路线，只授权关闭上述 review findings。r001 及其
exact Evidence 不改写。

## Review findings 与必须修复

| Finding | 级别 | r002 必须结果 |
|---|---:|---|
| MCP audience 缺 ExecutionPack identity | P1（architecture/security） | 加入 exact pack hash、lease generation/nonce；每次调用重验并定义失效 |
| stable citation 未定义 | P1 | 定义 canonical citation schema、hash/ID、fragment/range 和 resolve 等价 |
| TC-W04 Gate 超出候选路径 | P1 | 将最小 Workstation 类型/测试纳入 W04 candidate scope，或把 Gate 移到 W05 |
| 当前 RPC 不是逐项映射 | P1 | 每个 policy/knowledge/context/catalog/harness RPC 都有目标处置 |
| global mandatory 可被弱化 | P1 | 单独 global authority/role、不可变 revision、审批/disable 治理、set epoch/hash |
| global knowledge 审批 scope 歧义 | P1 | global approval permission 独立；客户端 `"*"` 一律拒绝；compat 只由服务器派生 |
| 引用资源缺传递授权/脱敏 | P1 | typed opaque refs、每次 dereference 授权、redaction、body 只经 MCP |
| restore/rollback 可倒退权威 | P1 | monotonic signed checkpoint、event replay、非 executable recovery |
| deterministic Gate 未绑定 diff | P1（docs） | validator 必须接受 frozen base/candidate；禁止提交后 0-file vacuous PASS |
| r001 manual freeze 自授权 | P1（docs） | r001 记录为 direct-user-authorized governance commit；r002 activation 后再冻结 Task |
| single-active revision、replay、canonical hash | P2 | 一并定义原子事务、签名 idempotency、domain-separated canonical hashes |
| proposed plan 结构不足、r001 command path 错 | P2 | r002 使用正确 validator；TC-W03–W06 drafts 补齐模板结构 |

## 分步计划

| Step | 前置 | 动作 | 验证 | 状态 |
|---|---|---|---|---|
| `TC-W02-S05` | r001 exact review BLOCK | 激活 r002，登记失败 Evidence 和 policy manifest | Registry 唯一 active；r001 不变 | `active` |
| `TC-W02-S06` | r002 activation exact commit | 从 activation commit/tree 冻结 r002 docs-only Task | base 可解析 r002 Plan/Registry/Policy | `planned` |
| `TC-W02-S07` | S06 | 前向修复合同、API 矩阵、路线和 validator；补 proposed Plan 结构 | base→candidate deterministic Gate | `planned` |
| `TC-W02-S08` | S07 PASS | 对新的 exact candidate 重跑 architecture/security/docs review | 三路 P0=0/P1=0 | `planned` |

## r002 确定性验证

validator 的规范调用：

```bash
node docs/waves/validate-wave-docs.mjs \
  --base <frozen-base-commit> \
  --candidate <candidate-commit>
git diff --check <frozen-base-commit>...<candidate-commit>
git diff --name-only <frozen-base-commit>...<candidate-commit>
```

validator 必须实际检查 base→candidate 的全部 tracked files；candidate 必须
是 commit，工作树必须干净。Journal append-only、changed Markdown links
和 `docs/waves/**` scope 都基于该 diff，不接受 0-file 自证。

## Additional acceptance

- [ ] r001 三路 findings 原文、exact candidate 和 BLOCK 结果已索引；
- [ ] r002 Task base 已包含 r002 Plan、Registry 和 Policy manifest；
- [ ] architecture/security 的全部 P1 和选定 P2 已前向关闭；
- [ ] docs/governance 的全部 P1/P2 已前向关闭；
- [ ] validator 对 r002 base→candidate 返回非零 changed files 并 PASS；
- [ ] Policy manifest、Registry、links、append-only 和产品代码空 diff 通过；
- [ ] final architecture/security/docs review P0=0/P1=0；
- [ ] 产品代码、运行时配置和真实数据仍未修改或访问。

## 回滚

- r002 activation 或 freeze 不一致：停止，不修改 r001 Evidence；
- remediation 失败：回到 r002 activation commit，保留 failed candidate；
- 任一 P1 未关闭：TC-W02 保持 active，不激活 TC-W03；
- 只撤销尚未合并的 docs commits，不触碰产品数据或 RN 实现。
