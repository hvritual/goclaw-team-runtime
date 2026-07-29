# TC-W02 当前源码责任矩阵

基线：
`d6a166ceb1f445e7098855841d20bf3903f0d3d5`
（tree `02698c5cc262b247ac37c53e80fa807ab5cb2c15`）。
本矩阵只依据源码和测试，不代表真实运行环境或真实数据已检查。

## 当前权威与存储

| 能力 | 当前类型/入口 | 当前存储与权威 | 已有保护 | 与目标差距 | 目标处置 |
|---|---|---|---|---|---|
| Policy layer | `teamcontrol.PolicyBundle`、`PutPolicyBundle`、`ResolvePolicy` | Team Control file state `Policies` | scope 授权、canonical JSON、bundle/resolved hash、项目/仓库/组件归属 | 无显式 global defaults；无 mandatory 终验层；resolved rule 没有逐条来源 | 保留 Team Control 为唯一权威，扩展成五层解析和 mandatory validation |
| Knowledge source registry | `teamcontrol.KnowledgeSource`、`knowledge.source.*` | Team Control file state `KnowledgeSources` | 项目复合 key、URI/revision/SHA、draft/approved/disabled、RBAC、原子写 | 只有 source metadata；无正文、owner、分类、可见性、有效期、关系、审批记录；approved 可由普通 registry transition 表达 | 迁移为 Team Control 管理的 KnowledgeRecord/Revision/Approval/Visibility 身份层；旧类型只作兼容读 |
| Context | `CompileContextInput`、`ContextBundle`、`context.compile/list` | Team Control file state `ContextBundles` | project/repository/member/budget 校验；只接受 approved SHA；canonical material/hash | 不含 task/attempt/lease、frozen task identity、knowledge state/expiry/visibility、citation、资源媒体类型、Skill checksum policy；未绑定 ExecutionPack | 升级为 immutable ContextManifest + content-addressed Bundle，并把 manifest hash 冻结进 ExecutionPack |
| Catalog record | `memory/catalog.Record` | 单独 SQLite `catalog_records` | 正文、version/checksum、provenance、approval、expiry、review、relations、confidence | 与 KnowledgeSource 平行成为第二知识权威；Team Control 不掌握其身份/审批状态 | 把 Catalog schema 能力收敛到 Team Control 知识域；SQLite 可作为迁移期 repository adapter，不再独立决定 active |
| Catalog index/search | `Service.Search`、可选 vector/QMD | SQLite 字段搜索及独立检索后端 | active 默认、expiry 过滤、project + optional `*` shared | 索引和正文边界未由 Team Control manifest 冻结；citation 不含 checksum | 索引变成可重建 projection；查询必须带 server-resolved scope 和 frozen context allowlist |
| Catalog circulation | `CirculationEvent`、`RecordUsage` | SQLite `catalog_events` | record/project/kind/actor/trace/metadata | 不绑定 task/attempt/runner/lease/context；自动注入没有强制写 circulation | 统一进入 Evidence/KnowledgeUsageEvent，绑定完整执行身份与结果关联 |
| Harness knowledge proposal | `knowledge.proposal.*` | Harness 自身 store/target path | pending/approved/rejected、human governance | 是第三条知识写入提案路径；批准语义不等于 active Catalog/Team Control knowledge | 后继迁移为统一 KnowledgeCandidate；Harness 仅作为来源 adapter |
| Agent catalog injection | `ContextBuilder.BuildCatalogContext`、`BuildApprovedContext` | 直接读 Catalog | active/expiry/共享过滤、转义成 quoted evidence | 绕过 Team Control ContextBundle；选择不冻结；使用记录不完整 | Team/Runner 执行只通过 ContextManifest + MCP；旧 personal mode 路径明确隔离或弃用 |
| Agent catalog tools | `search_project_memory`、`propose_project_memory` | 直接调用 Catalog service | context project scope 可防参数扩大；proposal 只 pending | 不验证 lease/ExecutionPack；创建者固定字符串；无签名 Runner Evidence | Team mode 替换为控制面 MCP；proposal 进入结构化反馈 ingestion |
| Runner queue | `workstation.ExecutionPack`、`Task`、`Lease` | Workstation file store | server-trusted enqueue、project/capability/assignee/lease、pack hash | 无 ContextManifest ID/hash、MCP audience、知识/Skill版本 | ExecutionPack 冻结 manifest ID/hash、MCP audience、resource budget snapshot |
| Runner Evidence | `workstation.EvidenceBundle` | 原子 evidence file + signed reference | task/project/runner/lease/attempt/pack/base 校验、HMAC、diff/checks | 无 Context/knowledge/citation/search event/feedback结构 | 扩展 ContextUsage + KnowledgeFeedback，仍保持签名、幂等和无秘密 |

## 当前知识模型逐项映射

| 目标字段 | `KnowledgeSource` | Memory Catalog `Record` | 当前缺口/冲突 |
|---|---|---|---|
| global/project scope | 只有 `ProjectID` | `ProjectID`，`"*"` 表 shared | 缺显式 scope kind；`*` 可与普通项目字符串混淆 |
| identity | `ID` | `ID`、Work/Expression/Manifestation/Item IDs | 两套独立 identity |
| revision | `Revision` 字符串 | `Version` + manifestation identity | 语义和唯一性规则不同 |
| checksum | `SHA256` | `Checksum`；provenance 可有 SourceSHA256 | citation/context 未统一绑定 checksum |
| owner | 无，仅 creator/updater | `CreatedBy`、ReviewedBy；无明确 owner | 需要显式 accountable owner |
| category | `Name` + metadata | `Kind`、collection、subjects、facets | Registry metadata 自由度过高 |
| visibility | 隐含项目 | project + optional shared search | 缺显式 global/project visibility policy |
| confidence | metadata 可模拟 | `Confidence` | 需纳入 schema 和审批可见信息 |
| validity/expiry/review | 无 | ValidFrom/Until、ReviewAt、ExpiresAt | Context Compiler 只看 registry approved，不看 expiry |
| supersedes/conflict | 无 | Relations | 需要一致的图约束与跨项目拒绝 |
| evidence refs | 无 | `EvidenceRefs` | 未绑定签名 EvidenceBundle |
| approval | `Status=approved`，无 decision | pending/active/rejected 等 + Decision | 同名“批准”语义不同，职责分离只在 Catalog 实现 |
| body | URI 指向外部 | SQLite 内 `Content` | 两个独立 truth source；URI 可能另指第三份正文 |

## Policy 当前行为

`teamcontrol/policy.go` 当前对最新 enabled layer 排序：

1. `team`
2. `project`
3. `repository`
4. `component`

同层再按 priority/name/version/ID 排序，后应用的键覆盖前值。当前
`ResolvedPolicy` 只保留 bundle IDs/hashes 和最终 rules，没有：

- `global_default` 层；
- `global_mandatory` 层；
- 每个 rule 的 winning source/overridden sources；
- mandatory violation 列表或 fail-closed outcome；
- canonical schema/version 标识。

## Runner 生命周期与反馈当前行为

| 阶段 | 当前合同 | 目标增量 |
|---|---|---|
| enqueue | Gateway 从 frozen dev task 构造 server-trusted ExecutionPack | 同时由服务器解析并冻结 ContextManifest；客户端不可传入扩大资源 |
| claim | Runner/project/capability/profile/assignee 匹配后创建 lease | MCP credential/audience 由 lease 派生，只能访问该 task/attempt/context |
| execute | LocalExecutor 校验 pack hash/base/path/verification | MCP 读取 manifest/search/read/citation；任何结果记录稳定 citation |
| evidence | 签名 task/project/runner/lease/attempt/diff/check | 加入 ContextManifest hash、knowledge version/checksum、retrieval/citation/result link 和 feedback |
| complete/fail | 验证签名与 lease 后原子保存 evidence | 先验签、去重、授权和 scope，再创建 candidates；不在完成事务中激活知识 |
| retry/requeue | 新 attempt/lease，保留事件 | 每 attempt 独立 usage/feedback；不得复用过期 lease MCP audience |

## Gateway、UI、CLI 与 projection

| 表面 | 当前入口 | 状态处理 | 规划影响 |
|---|---|---|---|
| Gateway Team Control | `policy.*`、`knowledge.source.*`、`context.*`、`control.summary` | handler 调 Team Control RBAC | 变为唯一写入/审批/编译 API；旧 knowledge.source 只兼容 |
| Gateway Catalog | `memory.catalog.*`、`memory.authority.*` | Team guard 先按 requested/resolved project 授权 | API 迁移到统一 knowledge 域；旧 RPC 做显式版本化 compatibility adapter |
| Gateway Harness proposal | `knowledge.proposal.*` | Harness 固定项目 + document RBAC | 迁移为 candidate source adapter，不能直接成为 active |
| Team Web Memory | active/pending/search/stats | loading/empty/error；review_due 显示 | 增加 denied/conflict/stale/checksum mismatch/global visibility/citation usage |
| Team Web Approvals | Catalog candidate + Harness proposal +其他审批混合 | loading/empty/error/action error | 合并为统一 candidate queue；显示来源、original revision、签名 evidence 和 separation check |
| Team Web Team page | policy status + control summary | 有 compliant/drift 和 control summary loading/empty/denied/error/disconnected helper | 展示 global defaults、project override、mandatory result、rule provenance、manifest |
| `goclaw team` CLI | 经 Gateway token 调 policy/context/control RPC | authenticated Team operation | 保留为受控操作面，扩展 inspect/resolve/compile；客户端不能扩大 scope |
| `goclaw memory catalog` CLI | 直接打开本地 SQLite，能 ingest/approve/reject/withdraw | local governance config | Team mode 禁止作为 active 写入口；迁移后只允许 export/import candidate 或经 Gateway |
| Obsidian plugin | 直连 Catalog RPC 展示/审批 | project params + Gateway guard | 继续作为 projection；不能保存 token/active truth 到 Vault |

## 现有测试责任

| 测试 | 已证明 | 尚未证明 |
|---|---|---|
| `teamcontrol/policy_test.go` | 四层覆盖、hash 稳定、stored hash fail-closed | global default/mandatory、rule provenance、mandatory conflict |
| `teamcontrol/controlplane_test.go` | Registry project composite identity、approved SHA Context、hash determinism、legacy key migration | active expiry/visibility、task/lease binding、正文 identity |
| `memory/catalog/service_test.go` | candidate→active、supersede、expiry、prompt escaping、cross-project relations、`*` shared | Team Control authority、Context allowlist、签名 Evidence feedback |
| `gateway/team_runtime_test.go` | knowledge/source/context RPC project scope 与 server-trusted enqueue | 统一 knowledge contract、MCP lease audience、Context hash in ExecutionPack |
| `gateway/team_guard_test.go` | Catalog RPC project authorization | global visibility server resolution、MCP authorization |
| `gateway/memory_catalog_test.go` | proposal/approval 分离 | reviewer 与 Runner/creator independence across unified domain |
| `workstation/service_test.go` | lease、并发 claim、签名 evidence、profile、expiry | knowledge usage/feedback、citation/checksum、candidate ingestion |
| UI tests | control summary projection 和基本状态 | rule provenance、conflict/stale/checksum/denied、candidate evidence |

## 无未说明第二权威源的结论

目标态只允许：

- Team Control database：身份、scope、策略、知识 revision/approval/visibility、
  ContextManifest、candidate/decision/audit 的逻辑权威；
- content object store：由 Team Control revision 记录的 immutable body
  blobs，key 是 checksum；它不自行决定 active；
- retrieval index：从 active revision event 构建的可丢弃 projection；
- Evidence store：签名执行事实，不能直接修改知识；
- Git/Markdown/Harness/Obsidian：来源或 projection，不是 active truth。

任何无法由 Team Control manifest + immutable body checksum 重建或验证的
正文/索引都不得进入 Runner Context。
