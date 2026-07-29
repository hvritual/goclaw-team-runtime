---
schema: goclaw.wave/v1
wave_id: TC-W02
track_id: TEAM-RUNTIME-2026-07
title: Team Control knowledge authority replan and contract baseline
revision: 1
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
supersedes_route:
  - RN-W01
  - INT-W01
created_at: 2026-07-29
updated_at: 2026-07-29
steps:
  - TC-W02-S01
  - TC-W02-S02
  - TC-W02-S03
  - TC-W02-S04
allowed_change_scope:
  - docs/waves/**
product_code_changes_allowed: false
---

# TC-W02 r001 — Team Control 知识权威重规划与合同基线

## 目标

基于精确源码基线
`d6a166ceb1f445e7098855841d20bf3903f0d3d5`
（tree `02698c5cc262b247ac37c53e80fa807ab5cb2c15`）重新定义 Team
Control，使其成为全局规则、项目规则、知识身份、版本、审批状态和
Context Manifest 的唯一权威控制面。

本 revision 只做 discovery、合同冻结准备和后继 Wave 路线设计。它不修改
Go、TypeScript、运行时配置，不迁移真实数据，不访问正在运行的 Team
Control、Runner 或真实知识库。

## 权威输入

- 用户指令：`user-directive-2026-07-29-team-control-authority-replan`。
- 仓库政策：[`AGENTS.md`](../../../../AGENTS.md)。
- 当前 Registry：[`wave-registry.json`](../../wave-registry.json)。
- 被替代路线：
  [`RN-W01 r004`](../rn-w01/plan-r004.md) 与
  [`INT-W01 r001`](../int-w01/plan-r001.md)。
- 已完成前置：
  [`TC-W01 r004`](../tc-w01/plan-r004.md) 与
  [`TC-EVID-W01-020`](../../evidence-index.md)。
- 当前源码责任证据：
  [`current-state-responsibility-matrix.md`](current-state-responsibility-matrix.md)。
- 目标合同：
  [`target-contracts.md`](target-contracts.md)。
- 迁移与后继路线：
  [`migration-and-wave-roadmap.md`](migration-and-wave-roadmap.md)。
- 关联 Issue：`TC-ISSUE-002`、`INT-ISSUE-001`、`RN-ISSUE-001`。
- 关联决策：`TC-DEC-003`–`TC-DEC-006`。

## 基线与规划事实

1. 当前工作树在预检时干净，未发现并行未提交修改。
2. 本地 `main` 比 `origin/main` 领先 28 个提交；精确基线同时存在于远端
   `origin/agent/rn-w01-lifecycle-001`。因此本计划描述“当前源码候选”，
   不把它称为已发布主线。
3. RN-W01 的产品实现提交已存在，但 r004 的独立验收尚未完成；路线改变时
   RN-W01 必须是 `superseded`，不得伪装成 `complete`。
4. Team Control `KnowledgeSource` 只保存项目级 URI/revision/SHA/status；
   Memory Catalog SQLite 同时保存正文、版本、审批、有效期、关系和检索
   索引。两者目前是相互独立的知识真相源。
5. Agent 运行时可通过 Memory Catalog 直接构造 prompt 上下文；现有
   `ContextBundle` 没有任务、成员、Skill/知识选择策略、正文资源身份或
   ExecutionPack 绑定，也没有 Runner MCP 合同。
6. 当前 Policy 解析顺序为
   `team → project → repository → component`，只支持后层覆盖，不区分
   global defaults 与最终不可覆盖的 global mandatory constraints。

这些是静态源码和测试合同事实，不是对真实运行环境或真实数据的迁移证明。

## 入口门禁

- [x] 用户已明确授权本次纯文档重规划。
- [x] `TC-W01` 已 `complete`，`TC-EVID-W01-020` 记录三路 P0/P1=0。
- [x] 精确 base、tree、分支、工作树和 Registry 唯一 active 状态已记录。
- [x] RN-W01/INT-W01 当前 Plan 与 RN journal 已只读核对。
- [x] 当前类型、存储、RPC、UI、CLI、Agent、Runner 和测试已只读盘点。
- [x] 范围与 non-goals 已冻结为 docs-only。
- [ ] `TC-W02 r001` 的确定性文档 Gate 通过并登记 Evidence。
- [ ] 独立 architecture/security/docs review 均 P0=0/P1=0。

## 范围

### 包含

- 当前源码到目标权威边界的逐项责任矩阵。
- 全局规则、项目覆盖和不可覆盖强制约束的解析合同。
- Knowledge、Context Manifest/Bundle、MCP、Evidence 和候选审批合同。
- `project_id="*"` 到显式 global scope 的兼容迁移设计。
- 唯一正文/索引存储边界、双写禁止、备份恢复和回滚设计。
- Gateway RPC、Team Web Console、CLI、Obsidian projection 和现有测试的
  影响面。
- 后继 implementation Waves、依赖、候选允许路径、确定性 Gate 和独立
  复核要求。

### 不包含

- 修改任何 Go、TypeScript、JavaScript 构建产物或运行时配置。
- 实现 MCP Server/Client、Context Compiler v2 或 Runner 反馈。
- 迁移、删除、重写或读取真实知识数据。
- 自动批准、批量激活或降低 `memory_approve` 职责分离。
- HA、Leader 选举、外部 Git/Jira 同步或无关 Runner 更新功能。
- 把本地规划任务描述为已由 Team runtime freeze/enqueue。

## 核心不变量

1. Team Control 管理身份、scope、授权、策略解析、知识身份/版本/状态和
   Context Manifest；客户端参数只能收窄，不能扩大服务器解析范围。
2. 知识正文和检索索引只有一个权威存储边界；索引是可重建 projection，
   不是第二权威源。
3. 规则顺序是
   `global defaults → team → project → repository → component`；项目可覆盖
   global defaults；合并后再执行 global mandatory constraints，冲突时
   失败关闭。
4. 只有 active、未过期、项目可见且 checksum 匹配的知识版本可进入
   Context Bundle。
5. Runner 只读取 lease/ExecutionPack 冻结的 Context；MCP 无激活、覆盖、
   删除 active 知识的工具。
6. Runner 反馈只能创建 candidate；必须由与创建者、Runner、任务执行者
   独立的 `memory_approve` 人工角色批准。
7. Context、MCP、Evidence 和审计数据不包含 Token、device key、OAuth、
   secret URI 或未批准正文。
8. 后继迁移明确替代前，保留现有 checksum、secret-free bundle、单写者和
   原子写入约束。

## 问题与事实

| Issue ID | 当前事实 | 状态 | 本 Wave 责任 |
|---|---|---|---|
| `TC-ISSUE-002` | Team Control Registry 与 Memory Catalog 都表达批准知识，Context 与 Agent 注入路径未统一 | `unverified`（静态合同差距） | 建立可审查的责任矩阵和目标合同，不宣称运行时缺陷已复现 |
| `INT-ISSUE-001` | Runner/Codex ExecutionPack 没有统一 MCP/Context 合同 | `planned` | 由 TC-W02 拆解并替代原宽泛 INT-W01 路线 |
| `RN-ISSUE-001` | RN 产品实现存在但未完成独立验收 | `deferred` | 保留实现与证据，不继续版本管理范围；后继仅消费已验证的 lease/Evidence 边界 |

## 分步计划

| Step ID | 前置 | 计划动作 | 允许路径 | 验证 | 状态 |
|---|---|---|---|---|---|
| `TC-W02-S01` | 用户指令、clean base | 盘点源码与现有计划，记录冲突和双权威事实 | `docs/waves/**` | 文件/符号/RPC/测试逐项可追溯 | `complete` |
| `TC-W02-S02` | S01 | 定义 Policy、Knowledge、Context、MCP、Evidence 合同和冲突矩阵 | `docs/waves/team-runtime/tc-w02/**` | 合同清单、状态机、授权负例、secret 禁止清单完整 | `complete` |
| `TC-W02-S03` | S02 | 前向更新 Registry、journal、decision、issue、evidence 和后继路线 | `docs/waves/**` | JSON、唯一 active、依赖、链接和 append-only 检查 | `active` |
| `TC-W02-S04` | S03 deterministic PASS | 独立 architecture/security/docs review；修正文档 P0/P1 | `docs/waves/**` | 三路 P0=0/P1=0；结果进入 Evidence index | `planned` |

S04 通过后停止。TC-W02 保持唯一 active，直到用户另行批准其完成及某个
implementation Wave 的新 approved revision、activation 和 Task Freeze。

## 验证与证据计划

| Evidence ID | 类型 | 通过条件 | 保存位置 | 状态 |
|---|---|---|---|---|
| `TC-EVID-W02-001` | current-state inventory | 每个当前类型/API/存储/测试都有源码位置和目标处置 | `current-state-responsibility-matrix.md` | `collecting` |
| `TC-EVID-W02-002` | deterministic docs Gate | JSON 可解析、唯一 active、依赖/链接/状态/范围检查通过，产品代码 diff 为空 | `s04-deterministic-verification.md` | `planned` |
| `TC-EVID-W02-003` | independent review | architecture/security/docs 均 P0=0/P1=0 | `s04-independent-review.md` | `planned` |

## 确定性验证

最低 Gate：

```bash
node -e 'JSON.parse(require("fs").readFileSync("docs/waves/wave-registry.json","utf8"))'
node scripts/validate-wave-docs.mjs
git diff --check
git diff --name-only -- . ':!docs/waves/**'
git status --short
```

若仓库没有 `scripts/validate-wave-docs.mjs`，本 Wave 只允许在
`docs/waves/**` 内新增一个确定性验证脚本，且它只能读取仓库文件、不得访问
网络或运行时。验证至少检查：

- Registry 恰好一个 `status=active`，且等于 `active_wave`；
- 每个 `document` 存在，Wave ID/revision/state 与 Registry 相容；
- active TC-W02 `product_code_changes_allowed=false`；
- RN-W01 与 INT-W01 的 superseded 原因、替代关系和未完成状态可追溯；
- proposed implementation Waves 不允许产品代码变更；
- Markdown 相对链接存在；
- 必填合同章节和冲突矩阵存在；
- 本次 diff 仅位于 `docs/waves/**`。

## 风险与回滚

| 风险 | 触发信号 | 缓解 | 回滚 |
|---|---|---|---|
| 误把 Catalog 或 Team Control 当作可直接删除的旧源 | 计划出现数据删除、无清单覆盖或不可逆 cutover | 采用 inventory → shadow verify → read cutover → retirement；禁止无清单删除 | 撤销未合并规划提交，Registry 恢复 RN-W01 active；不触碰产品数据 |
| 降低职责分离 | Runner/创建者可 approve，或自动晋升 | 状态机和 MCP 明确无 promote 工具；独立角色矩阵 Gate | 停止，TC-W02 保持 active，不激活实现 |
| secret 进入 Bundle/Evidence | schema 出现 token、key、OAuth、secret URI 字段 | 显式 allowlist、结构化扫描、拒绝自由 metadata 承载秘密 | 停止，拒绝相关合同 revision |
| global scope 泄漏项目数据 | `*` 继续作为普通项目字符串或客户端可请求 shared | 显式 `scope_kind=global`；服务器计算 visibility；迁移逐条分类 | 失败关闭，保留旧数据只读且不进入新 Context |
| 新索引成为第二权威 | 索引可写状态/正文且无法从权威重建 | 索引只消费 immutable revision event，可从对象正文重建 | 丢弃并重建索引，不改权威记录 |
| 当前源码与发布主线混淆 | 把 ahead-of-origin 候选称为 release | 固定 base/tree 和 remote feature ref；独立 release Wave 再证明 | 回到已发布 release，不倒灌真实数据 |

## 停止条件

发现以下任一项立即停止并报告：

- 方案需要删除现有知识或在未清单化前覆盖正文；
- 改变 `memory_approve` 独立审批、允许自动晋升或执行者自批；
- Context/MCP/Evidence 需要携带凭据或 secret URI；
- Runner、MCP 或客户端可扩大项目范围或直接写 active；
- Registry、base、计划 hash 与本 revision 记录不一致；
- 相关文档出现并行未合并修改；
- 确定性 Gate 或任一独立 review 有未关闭 P0/P1。

## 退出门禁

- [ ] 当前责任矩阵覆盖 Team Control、Catalog、Agent、Runner、Gateway、
  Web Console、CLI、Obsidian projection 和测试。
- [ ] Policy/Knowledge/Context/MCP/Evidence 合同与状态机完整。
- [ ] global defaults、global mandatory 和下层覆盖冲突矩阵完整。
- [ ] 唯一正文/索引权威边界及 `project_id="*"` 迁移无第二真相源。
- [ ] 后继 Waves 的依赖、候选路径、非目标、Gate 与回滚完整。
- [ ] Registry 可解析且恰好一个 active Wave。
- [ ] RN-W01 superseded 原因、替代 Wave 和未完成状态可追溯。
- [ ] INT-W01 不再作为可直接激活的宽泛 implementation Wave。
- [ ] deterministic Gate 通过并索引。
- [ ] architecture/security/docs 独立复核 P0=0/P1=0。
- [ ] 明确记录“产品代码尚未修改”。

满足这些门禁只允许将 TC-W02 置为“可关闭”；不得在同一任务激活或实现
TC-W03 及后继 Wave。

## Plan revision

r001 获批后不可原地改变目标、合同、scope、状态机、迁移、Gate 或后继
路线。任何实质变化必须创建 `plan-r002.md`，在 journal 前向追加替代原因
并重新独立复核。
