# TC-W02 迁移、兼容与后继 Wave 路线

## 兼容迁移原则

1. 先 inventory，后 shadow import，再双读比对，最后单读切换；禁止无清单
   双写 active。
2. 任何旧记录在 scope、checksum、审批或正文身份不明确时进入 quarantine/
   pending，不进入 Runner Context。
3. 原始数据库、Markdown、KnowledgeSource 和 Evidence 只读保留到迁移验收
   与备份恢复演练通过。
4. 迁移器是单写者，使用 idempotency key、原子 checkpoint 和可重放
   manifest。
5. 索引可重建；迁移完成不要求保留旧索引作为权威。

## `project_id="*"` 到 explicit global scope

迁移表：

| 旧记录 | 新 scope | 处理 |
|---|---|---|
| `project_id="*"` 且来源/owner 明确、批准有效 | `scope_kind=global` + fixed global authority ID | 生成 candidate，独立 global `memory_approve` 后才能 active |
| `project_id="*"` 但实际只适用部分项目 | 对每个明确项目创建 project-scoped candidate | 不保留隐式 shared |
| `project_id="*"` 且来源、owner、checksum 或可见性不明确 | quarantined/pending | 不索引、不编译 |
| 普通 project record | `scope_kind=project` + 原 project ID | 验证 membership/source/checksum 后 shadow import |
| 客户端继续传 `"*"` | 拒绝或 compatibility read-only mapping | 客户端不能创建 global scope |

global visibility 由服务器按 policy 和 project entitlement 计算；Runner 不能
通过 `include_shared=true` 自行扩大范围。

## 旧类型与 API 兼容

### `KnowledgeSource`

- 读取 adapter 将旧记录投影为 source descriptor；
- `draft` → candidate/pending；
- `approved` 不能直接映射 active，必须与 Catalog/正文 checksum 和审批
  decision 对账；缺任一项进入 pending/quarantine；
- `disabled` → withdrawn（若曾 active）或 rejected（若从未 active），需保留
  原状态 provenance；
- 旧 delete API 在迁移期只允许删除从未批准、未被 Context 引用的 draft；
  新域不硬删除已审计 revision。

### Memory Catalog

- `Record.ID` 映射为 legacy alias；WorkID 可作为 durable knowledge identity
  候选，ManifestationID/Version 映射 revision；
- Content 写入唯一 object store并实算 checksum；
- provenance、relations、evidence、validity、review、confidence 原样规范化；
- `pending/active/rejected/superseded/withdrawn/quarantined` 映射到统一状态；
  active 仍需 decision 和 scope 校验；
- `catalog_events` 映射 audit/usage event；缺 task/attempt/lease 的历史 event
  标记 `legacy_unbound=true`，不用于证明 Runner 效果；
- 旧 SQLite 在 read cutover 后只读归档，不再接受 active mutation。

### 当前 Context Bundle

- v1 Bundle 可读验证，但不授予 Runner MCP；
- v1 只含 source ref，不能自动升级为 v2 executable Context；
- 迁移器生成 v2 candidate manifest 时必须重新解析 policy、scope、expiry、
  visibility、body checksum、task/member/Skill identity；
- v1 ID/hash 永不重写；v2 使用新 schema/compiler namespace。

### Harness proposal、CLI、Obsidian

- Harness proposal 转换成 KnowledgeCandidate source adapter；
- `goclaw memory catalog` 的直接 SQLite mutation 在 Team mode 进入
  read/export-only；候选/审批必须经 Team Control Gateway；
- Obsidian/Web Console 只做 projection 和受控 command，不在 Vault 保存
  active state、Token、reviewer token 或 device secret。

## 迁移阶段

| 阶段 | 写权威 | 读路径 | Gate | 回滚 |
|---|---|---|---|---|
| inventory | 旧系统不变 | 旧路径 | 记录数/checksum/scope/status/source 全清单 | 无产品变更 |
| shadow import | 旧系统仍是唯一写者 | 旧路径；新域不可执行 | 每条映射、对象 checksum、关系和审批对账 | 丢弃 shadow state |
| dual-read compare | 旧系统仍写；禁止双写 active | 用户仍读旧；验证器对比新 | 查询集合、citation、expiry/visibility 无未解释差异 | 关闭 shadow read |
| read cutover | Team Control 新域唯一写者 | Team Control + MCP；旧库只读 | Context/MCP/Evidence/backup/restore Gate | 切回旧 read snapshot，保留新 audit |
| retirement | Team Control | 新路径 | observation window、无旧 writer、恢复演练 | 继续保留只读归档 |

## 后继 Wave 路线

后继 Wave 当前均为 `proposed`，`product_code_changes_allowed=false`。下面的
路径是候选边界，不是实现授权。每个 Wave 在执行前必须新建 approved plan
revision、成为唯一 active、从其 activation exact commit 冻结 Task。

### TC-W03 — Knowledge authority and storage convergence

- 依赖：TC-W02 complete。
- 候选路径：
  `teamcontrol/**`、`memory/catalog/**`、`gateway/memory_catalog.go`、
  `gateway/team_control.go`、`cli/commands/memory_catalog.go`、migration tools、
  直接相关 tests/docs。
- 目标：统一 KnowledgeRecord/Revision/Candidate/Approval/Audit；建立唯一
  object store 和可重建 index boundary；实现 `*` shadow migration。
- 非目标：MCP、Runner Evidence、UI 大改、真实数据 cutover。
- 确定性 Gate：schema/state table tests、project/global authorization
  matrix、candidate approval separation、checksum/object/index rebuild、
  legacy migration idempotency/rollback、race/atomic write。
- 独立复核：architecture + security + data migration/docs，P0/P1=0。
- 退出：Team Control 是逻辑权威；旧 Catalog active mutation 可被
  compatibility flag 禁止；尚不迁移真实数据。

### TC-W04 — Policy resolver and Context Compiler v2

- 依赖：TC-W03 complete。
- 候选路径：
  `teamcontrol/**`、`gateway/team_control.go`、`orchestratorlite/**`、
  `gateway/development.go`、直接相关 tests/docs。
- 目标：global defaults/mandatory、rule provenance、canonical resolved
  policy、ContextManifest v2、frozen task/member/knowledge/Skill/budget。
- 非目标：MCP server、Runner feedback、UI 全功能。
- 确定性 Gate：冲突矩阵 table tests、canonical golden vectors、hash
  determinism、secret field negative schema、expiry/visibility/checksum、
  ExecutionPack manifest hash server binding。
- 独立复核：architecture + security/crypto-boundary + docs，P0/P1=0。
- 退出：相同输入稳定 manifest；任何 mandatory/secret/scope 错误失败关闭。

### TC-W05 — Lease-scoped MCP and signed Runner feedback

- 依赖：TC-W04 complete。
- 候选路径：
  新/现有 MCP package、`teamcontrol/**`、`workstation/**`、
  `gateway/workstation.go`、`gateway/development.go`、Runner CLI、直接相关
  tests/docs。
- 目标：manifest/search/read/citation/policy explain；lease audience；
  Evidence usage/citation/feedback；candidate-only ingestion。
- 非目标：自动激活、通用写 MCP、HA、外部同步。
- 确定性 Gate：lease expiry/cancel/requeue/attempt matrix、cross-project/
  tamper/replay/secret negative tests、citation golden vectors、signed feedback
  dedupe、creator/runner/reviewer separation、race。
- 独立复核：architecture + security + Runner/docs，P0/P1=0。
- 退出：Runner 只能读取 frozen Context；所有反馈只产生 candidate。

### TC-W06 — Console, CLI, operations and controlled cutover

- 依赖：TC-W05 complete。
- 候选路径：
  `ui/src/team/**`、`ui/tests/**`、Gateway projections、Team CLI、Obsidian
  adapter、config/deploy/docs、migration/backup/restore tools 和 tests。
- 目标：规则/知识/Context/usage/candidate UI；显式七态；Team CLI；
  inventory/shadow/compare/read-cutover；冷备/恢复/回滚。
- 非目标：HA、Leader、真实 Pilot release、外部 Git/Jira sync。
- 确定性 Gate：UI loading/empty/denied/error/conflict/stale/disconnected，
  RPC auth，migration dry-run/roundtrip/rollback，backup manifest tamper，
  credential scan，旧 writer disabled。
- 独立复核：UX/accessibility + security/operations + docs/data，P0/P1=0。
- 退出：受控 synthetic migration/cutover/restore 通过；真实数据仍需单独
  operator Task 和备份批准。

### REL-W01 — release route

`REL-W01 r002` 只在 TC-W06 complete 后才可激活。它消费上述已验证合同做
发行与 Pilot Gate，不在 release Wave 补做知识迁移或安全语义。

## 每阶段共同冻结要求

每个 implementation Task 必须记录：

- Task-ID、Project-ID、Repository-ID、assignee、base commit/tree、branch；
- Wave-ID、approved revision、step、Issue、policy bundle hash；
- exact file scope、acceptance、deterministic commands、rollback；
- creator/assignee 之外的 final reviewer；
- 无真实 secret/数据的 synthetic fixtures；
- 前一 Wave exact Evidence refs。

Creator 或 assignee 不能做自己变更的 final acceptance。确定性检查必须先于
模型审查；P0/P1 未关闭不得进入下一 Wave。
