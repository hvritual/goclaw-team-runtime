# TC-W02 Journal

本文件只追加。当前状态以 Registry 和
[`plan-r001.md`](plan-r001.md) 为准。

## 2026-07-29 — 用户改变路线并激活 docs-only 重规划

- 用户明确要求 Team Control 成为全局规则、项目规则和项目知识的权威控制
  面，Runner 通过项目隔离 MCP 只读消费冻结 Context；
- 当前 base 固定为
  `d6a166ceb1f445e7098855841d20bf3903f0d3d5` /
  tree `02698c5cc262b247ac37c53e80fa807ab5cb2c15`；
- 预检工作树干净，Registry 原先恰好只有 RN-W01 active；
- 本地 main 比 origin/main 领先 28 个提交，但 base 已存在于远端
  `origin/agent/rn-w01-lifecycle-001`；本 Wave 只描述当前候选源码；
- RN-W01 已有产品实现但尚未完成独立验收，因此由 `active` 前向转为
  `superseded`，不标记 complete、不删除实现或 Evidence；
- INT-W01 r001 的宽泛集成路线由 TC-W02 合同基线和
  `TC-W03 → TC-W04 → TC-W05 → TC-W06` 后继路线替代；原 Plan 保持只读；
- TC-W02 r001 只允许 `docs/waves/**`，产品代码变更明确为 false。

## 2026-07-29 — S01 current-state inventory

- Team Control `KnowledgeSource` 是 project-scoped metadata registry，
  `PolicyBundle` 支持 team/project/repository/component 后层覆盖，
  `ContextBundle` 只绑定 policy/budget/手选 KnowledgeSource/Skill；
- Memory Catalog SQLite 独立持有正文、版本、审批、有效期、关系、citation
  和 circulation event，并把 `project_id="*"` 当共享知识；
- Agent 可直接调用 Catalog 自动注入 prompt，ExecutionPack/EvidenceBundle
  不携带 Context Bundle、知识版本、citation 或检索事件；
- Gateway 对两套知识 API 分别做项目授权；Web Console 同时展示 Catalog
  候选和 Harness knowledge proposal，CLI 仍可直接打开本地 Catalog；
- 以上差距作为规划输入登记，不访问真实运行时，不将其冒充已复现产品缺陷。

## 状态事件

| Seq | 时间 | Actor | From | To | 原因 | Evidence |
|---:|---|---|---|---|---|---|
| 1 | 2026-07-29 | user directive / Codex root | `proposed` | `active` | 用户明确改变路线；激活 discovery/contract-only TC-W02 | `TC-EVID-W02-001` collecting |

## 进度事件

| 时间 | Step ID | 状态变化 | 实际结果/阻塞 | 下一动作 |
|---|---|---|---|---|
| 2026-07-29 | `TC-W02-S01` | `planned → complete` | 静态源码责任盘点完成；未访问真实环境 | 编写目标合同 |
| 2026-07-29 | `TC-W02-S02` | `planned → complete` | Policy/Knowledge/Context/MCP/Evidence 与迁移路线形成 r001 | 更新治理索引 |
| 2026-07-29 | `TC-W02-S03` | `planned → active` | 等待 deterministic Gate | 运行 docs Gate |

## 2026-07-29 — r001 exact candidate review BLOCK

- reviewed exact commit：
  `e9a4cc04cb6113fe98685ac0db809e33739c9277`；
  tree `dd574e6c65b859f13ef8dcb6ce1859cc28570340`；
- architecture P0=0/P1=4/P2=1，security P0=0/P1=5/P2=2，
  docs/governance P0=0/P1=2/P2=2；三路均 BLOCK；
- r001 candidate 保留为 `TC-EVID-W02-003` failed，不 amend/rebase；
- P1 聚焦 ExecutionPack-bound MCP audience、citation schema、global mandatory/
  global knowledge authority、transitive ref authorization、monotonic restore、
  W04 scope、RPC 逐项映射和 non-vacuous deterministic Gate；
- docs reviewer 证明 r001 manual freeze 的 plan/policy 不存在于其 base；
  r001 commit 只按用户直接授权保留为 governance activation candidate，不再
  声称是 runtime/manual frozen implementation Task；
- 激活 r002 后，必须从包含 r002 Plan/Registry/Policy 的 exact commit 冻结
  remediation Task。

## 2026-07-29 — r002 activation

- r002 完整继承 r001 目标和 docs-only boundary，只修复三路 findings；
- `TC-W02-S05` active；S06 必须等待本 activation commit 后冻结 exact Task；
- r001 approved plan 和 failed Evidence 保持不变。

## 2026-07-29 — r002 exact Task Freeze

- activation commit：
  `d2df5fd59fffb2d38838364614974ebce62bbda1`；
- activation tree：
  `af6d487039df7086daa4224d341bfdbb9422772d`；
- Task：`TC-W02-REPLAN-001` r002；
- branch：`codex/tc-w02-replan-r001`；
- Policy manifest SHA-256：
  `04b76061b2dd61fe995568e1c8646f5eb5d7684cd0f3fcb12c2f7eee4033536d`；
- base 已包含 r002 Plan/Registry/Policy 和 r001 failed Evidence；
- freeze 后只允许 `docs/waves/**` 内关闭 r001 review findings、运行
  base→candidate Gate 和重做三路 review。

## 2026-07-29 — r002 S07 review remediation and deterministic PASS

- MCP audience 加入 exact ExecutionPack hash、lease generation/nonce；
- citation v1 冻结 canonical identity、UTF-8 byte range、checksum 和 resolve；
- global mandatory/global knowledge 使用独立 global authority 和审批角色；
- resource refs 改为 typed opaque ID，并逐次做传递授权与脱敏；
- backup/restore 加入 signed monotonic epoch、完整 event replay 和
  non-executable recovery；
- Knowledge successor activation 冻结 single-active atomic CAS；
- feedback idempotency 绑定签名 envelope，canonical hash 使用
  domain-separated RFC 8785 JCS 与 golden vectors；
- 当前 RPC/CLI/Agent tool 已逐项映射，W04 candidate scope 纳入最小
  Workstation binding，TC-W03–W06 draft 补齐模板结构；
- validator 改为强制 frozen base/candidate commit、clean worktree、非空
  diff、exact links/scope/journal 检查；
- exact base `d2df5fd...` 到 candidate `df47a8b...` 的 Policy 3/3、
  validator、diff/check、product-code-empty 全部 PASS；
- `TC-EVID-W02-004` passed；下一步对包含 Evidence 的 final candidate 重跑
  Gate 与 architecture/security/docs review。

## 2026-07-29 — r002 S08 security re-review correction

- security re-review 关闭既有 finding 后仍发现 P1：Runner-visible
  ContextBundle 内联正文会在 lease/audience 被撤销后绕过 MCP 逐次授权；
- 前向修正 Runner wire contract 为 Manifest + typed opaque refs only；
- 只允许受控编译器内部的 `CompilerMaterialEnvelope` 临时持有正文；它不进入
  ExecutionPack、不返回 Runner、不成为 Runner 可访问 Evidence，编译后丢弃；
- 该 P1 必须经 exact candidate deterministic Gate 和独立复审关闭。

## 2026-07-29 — r002 S08 architecture re-review corrections

- architecture re-review 报告 P0=0/P1=2/P2=1；
- CitationV1 wire query 改为唯一 lexical 顺序，并冻结 Whole golden
  representation；
- read-cutover rollback 不再允许直接切回 stale snapshot；必须 quiesce
  mutation、验证 monotonic checkpoint 并 replay/reconcile 全部 authority
  event，失败时只能 non-executable recovery；
- global knowledge 迁移审批角色统一为 `global_memory_approve`；
- 上述 findings 必须经 exact candidate deterministic Gate 和独立复审关闭。

## 2026-07-29 — r002 S08 documentation re-review correction

- documentation re-review 报告 P0=0/P1=0/P2=1；因 r002 要求文档 P2 关闭，
  结果仍为 BLOCK；
- TC-W03–TC-W06 proposed drafts 补齐强制 Wave template 的 `approved_by`、
  目标、入口门禁、范围、问题事实、影响分析、完整 Step/Evidence/Risk 表、
  退出清单、决策记录和 Plan revision；
- validator 同步检查完整 mandatory template surface，避免只检查子集形成
  假阳性；
- 这些 draft 仍是 `proposed`、`product_code_changes_allowed=false`，不会
  因模板补全获得实现授权。
