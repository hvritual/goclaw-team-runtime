# Wave 决策日志

此日志只追加。决策被替代时新增记录并用 `supersedes` 指向旧 ID，不删除旧项。

| Decision ID | 日期 | 状态 | 决策 | 原因 | 影响 | Supersedes |
|---|---|---|---|---|---|---|
| `FE-DEC-001` | 2026-07-26 | active | 前台异常修复采用 Wave-first；计划文档必须先于产品代码变更 | 用户明确要求所有逐步更新计划落到文档 | 新增 Wave 注册表、模板、问题与证据索引，并写入工程门禁 |  |
| `FE-DEC-002` | 2026-07-26 | active | `FE-W00` 只做复现、基线和诊断，不修复产品代码 | 当前只有“多项功能异常”的聚合报告，缺少逐项证据 | W01 之前不得修改 React、Gateway 或 RPC 实现 |  |
| `FE-DEC-003` | 2026-07-26 | active | 继续以 Team Web Console 为默认控制面，Obsidian 保持可选适配器 | 前一版本已完成控制面迁移；前台异常不构成回退到 Obsidian 的证据 | 修复路线围绕 Web Console，知识与运行时权威边界不变 |  |
| `FE-DEC-004` | 2026-07-26 | active | Wave 顺序按共享基础设施 → 查询契约 → 命令流程 → UX 韧性 → 集成发布 | 会话、项目上下文和 RPC 客户端影响所有页面，应先稳定依赖层 | W01–W05 串行通过硬门禁，不按页面随机修补 |  |
| `FE-DEC-005` | 2026-07-26 | active | 在 0.7.0 审阅副本建立独立本地 Git 基线，不覆盖旧 0.6.0 脏工作树 | 旧 Git 树缺少 Team Web Console 增量；混合迁移会使源码、计划和证据失真 | source base 冻结为 `b288564`；旧工作树保持不变；当前仓库不配置 remote |  |
| `FE-DEC-006` | 2026-07-26 | active | W00 r002 在首批基础 Issue 证实后允许迁移到 W01；完整矩阵按责任 Wave 渐进复现 | Cloud Browser 拒绝 localhost，但确定性协议探针已能证明并回归基础 transport 故障 | `FE-ISSUE-001` 保持开启；W01 浏览器退出门禁不放宽；W00 仍禁止产品代码 | `FE-DEC-002` 的“W01 前不得修改”仅在 W00 r002 门禁完成后解除 |
| `FE-DEC-007` | 2026-07-26 | active | 拒绝执行 W00/W01 r002，改用 r003 保留稳定 Step 身份并扩大安全负例与范围 Gate | 独立 Reviewer 批准根因但发现 r002 重用 Step、缺真实 WS 101、CSRF/browser-session/RBAC、回滚和 untracked 检查 | r002 只保留历史；r003 通过复核和原子 Registry 切换前继续禁止产品代码 | `FE-DEC-006` 的迁移实现细节 |
| `FE-DEC-008` | 2026-07-26 | active | S04 发现的 `/dashboard/` baseline failure 不在 r003 顺手修复；先以 r004、Task revision 2 和新增 S06 授权 | `FE-EVID-W01-007` 证明 shell 301 自循环且相关路径不在 r003 allowlist；忽略会阻断所有页面，越权修改会破坏 Wave 门禁 | r003 S04 停止；r004 原子激活并从 activation commit 创建 R2 后才能迁移 patch、修改 shell；Browser Gate 不放宽 | `FE-DEC-007` 的 W01 执行范围 |
| `FE-DEC-009` | 2026-07-26 | active | 不跳过或等待十分钟 WeWork test；以 r005/S07 把它改为 synthetic、离线、确定性 constructor test | 全仓 Gate 被固定 sleep 阻断，且 test source 含 credential-shaped literals；提高 timeout 或排除 package 会掩盖安全与确定性问题 | r004 S04 停止；r005 仅新增 channel test path，不改生产 channel；历史值若曾有效必须外部轮换 | `FE-DEC-008` 的 W01 验收范围 |
| `FE-DEC-010` | 2026-07-26 | active | 不跳过 Memory Catalog 失败；以 r006/S09 修复默认 Markdown provenance kind 推断并补推断矩阵 | R3 全仓 Gate 稳定失败，且 Memory 路径不在 r005 allowlist；只排除 package 会掩盖错误来源元数据 | r005 S04 停止；r006 新增两个精确 Memory 路径，不做 schema/数据迁移；从新 Task Base 创建 R4 | `FE-DEC-009` 的 W01 验收范围 |
| `FE-DEC-011` | 2026-07-26 | active | 在 Cloud Browser 因安全策略拒绝 localhost 后，按用户明确授权建立 r007，使用仓库外本地 Playwright 完成真实浏览器回归 | 确定性协议 Gate 已通过但不能替代页面交互；继续等待 Cloud Browser 会永久阻断 S05，改用未授权浏览器或临时手测又会破坏证据边界 | r007 只新增 S10 本地测试设施和 S05 页面回归；固定 synthetic runtime、桌面/移动、脱敏与失败即另起 r008；S08 和发布阻塞不变 | `FE-DEC-010` 的 Browser 执行方式 |
| `FE-DEC-012` | 2026-07-26 | active | 真实 Gateway 启动强制构造 provider 时，不配置真实 provider 或修改产品代码；建立 r008/S11，使用固定公开 key marker、不可达 loopback inert provider 并强制 Gateway 零出站 | r007 要求真实 binary 与全部 provider 关闭不可同时满足，且 provider constructor 拒绝空 key；配置真实 provider 会引入凭据/外网，临时改产品会越权，自建 Gateway harness 又偏离真实启动路径 | r007 安全停止；r008 Task rev6 只改变 synthetic fixture 合同，产品 allowlist 不扩张；marker 进入 sentinel，任何 provider/model 调用或 Gateway 出站均失败；S08/发布阻塞不变 | `FE-DEC-011` 的 synthetic runtime 实现 |
| `FE-DEC-013` | 2026-07-26 | active | R6 freeze/commit traceability 不完整时不改写或继续该分支；建立 r009/S12，从 R5 trailers-present commit 重建完整 Repository/Policy hash/trailers 的 R7 | amend/rebase 会破坏失败证据，forward-only 补偿不能让原 commit 合规，忽略则违反 AGENTS；R6 产品尚未提交且 runtime 尚未创建，可安全失败关闭 | R6 只读保留；r009 Task rev7、Repository `repo-goclaw-source-review`、Policy SHA `98bacd...`；R7 重跑全部 Gate，产品范围、S08 和 Browser 合同不变 | `FE-DEC-012` 的 Task/commit 落地方式，不替代其 inert-provider 决策 |
| `FE-DEC-014` | 2026-07-26 | active | 当前容器和宿主级重试均拒绝 ptrace 时，不以 socket 轮询或其他未批准机制代替 S11 syscall Gate；将 EVID018 标记 environment-blocked，保持 r009 合同，等待可通过相同 `strace` 能力测试的受控本地环境 | 已授权本地 Playwright 仍不能证明 Gateway 子进程的短暂/失败 connect；降级会把未知出站冒充零出站 | 未创建 credential/runtime；若改变证明机制必须先批准新 plan revision | `FE-DEC-013` 的 S11 执行环境，不改变产品或 Browser 合同 |
| `PILOT-DEC-001` | 2026-07-27 | active | 将 `FE-W01` 明确置为 blocked，并激活独立 `PILOT-W00`；不把 Pilot Track 描述为 FE-W01 完成 | 用户新增三人试点与跨平台 Runner 授权，范围超过 FE-W01；外部 credential owner 和 ptrace 事实仍不能由代码解决 | 新 Wave 吸收尚未完成的前端工作并增加 Runner/Governance/Recovery；正式部署仍受外部 attestation 阻断 | `FE-DEC-014` 之后的执行 Track，不替代历史结论 |
| `PILOT-DEC-002` | 2026-07-27 | active | 三人试点的 Runner 统一运行在 Linux substrate：Linux 原生、Windows/WSL2、macOS/Lima；原生 Windows/macOS 执行失败关闭 | 三平台原生隔离的 ACL、进程树、文件系统、argv 和 sandbox 契约尚不等价；统一 bwrap 可被共同验证 | 发行 Linux amd64/arm64 Runner；其他 native binary 只用于控制命令，禁止冒充 Runner 支持 |  |
| `PILOT-DEC-003` | 2026-07-27 | active | 使用不可登录的 `planner-service` 作为 DevTask 创建者，并记录真实 `requested_by` | 现有严格策略在恰好三名真人时需要 creator、两名 reviewer 和独立 final approver，至少四个身份 | 三名真人不降级职责分离；service identity 无 Token、不能 review/accept/run |  |
| `PILOT-DEC-004` | 2026-07-27 | active | Pilot 使用停机冷备、restore-to-new-root 和一致性失败关闭，不声称热备或跨根事务 | TeamControl、Workstation、Orchestrator Lite 是三个单写存储，热 tar 可能截取不同逻辑时点 | maintenance lock、canonical manifest/hash、恢复后 consistency check；歧义时阻止 enqueue/accept |  |
| `PILOT-DEC-005` | 2026-07-27 | active | 项目聊天使用版本化、分段无歧义的会话键；只迁移可证明唯一的旧键 | 旧 `:`→`_` 文件名规范化可让不同 project/topic 组合碰撞 | agent 写入与 history 读取共用 key builder；模糊旧键失败关闭 | `plan-r002` |

## 新决策格式

追加新记录时必须说明：

- 决策状态：`proposed`、`active`、`rejected` 或 `superseded`；
- 触发来源；
- 至少一个被否决的替代方案；
- 对 Wave 顺序、范围、数据、权限、部署和回滚的影响；
- 相关 Issue、Evidence 和 Task。

不应记录为决策的内容：

- 尚未复现的根因猜测；
- 普通实现细节；
- 没有权威输入的产品目标；
- 为了让 Gate 通过而降低验收标准。
# 2026-07-28 — MVP-DEC-001：从校验归档恢复权威 Git 基线

- 决策：暂停 `PILOT-W00`，激活 `MVP-W00`，从
  `goclaw-team-runtime-source-0.8.0-pilot.1.tar.gz` 建立新的归档导入历史。
- 原因：归档不含原始 `.git`；任何后续 Task/Wave 都需要可解析、可复核的
  唯一 base commit。
- 边界：不伪造 7 月 27 日 commit，不修改运行时，不把历史实机 Gate
  声明为通过。
- 回滚：保留只读 import tag；恢复 Gate 失败时将 `MVP-W00` 置为
  `blocked`，不进入后继 MVP Wave。

## 2026-07-28 — MVP-DEC-001A：暂停旧 Pilot 激活状态

- 状态：`active`。
- Supersedes：仅替代 `PILOT-DEC-001` 的“当前激活 `PILOT-W00`”状态，
  不替代其三人 Pilot 目标、Linux substrate 和外部门禁结论。
- 触发：用户批准先恢复源码和 MVP，且原发布归档不含 Git 历史。
- 否决方案：拒绝把旧 Pilot 继续保持 active，也不删除旧决策。
- 影响：`MVP-W00` 成为唯一 active；FE-W01/PILOT 仅 docs-only blocked。
- 关联：`MVP-W00-S01/S05A`、`MVP-EVID-001/006`、
  `Task-ID MVP-W00-RECOVERY-002`；Issue 为 `N/A — recovery bootstrap`。

## 2026-07-28 — MVP-DEC-002：Recovery 独占 active Wave

- 状态：`active`。
- 触发：恢复 base 未冻结时并行 Pilot/FE 修改会让 Task 与 Evidence 失去唯一
  commit。
- 否决方案：拒绝同时激活 Recovery、FE-W01 和 PILOT-W00。
- 影响：Registry 只能有一个 active record；后两者依赖 `MVP-W00` complete。
- 回滚：Recovery blocked 时所有后继 Wave 继续 blocked，不降级门禁。
- 关联：`MVP-W00 r002`、`MVP-EVID-006`、
  `Task-ID MVP-W00-RECOVERY-002`；Issue 为 `N/A — recovery bootstrap`。

## 2026-07-28 — MVP-DEC-003：Recovery 不修产品缺陷

- 状态：`active`。
- 触发：第一轮 Gate 和独立复核只发现恢复/发布/治理问题。
- 否决方案：拒绝在 Recovery 顺带修改 Gateway、Runner、TeamControl 或 UI
  运行时。
- 影响：r002 只扩展恢复脚本、工具链 package metadata 和文档；产品问题
  必须在 recovered base 上新建 Wave。
- 回滚：任一产品目录越界立即停止并回到最后一个 docs/config commit。
- 关联：`MVP-W00-S05B–S05E`、`MVP-EVID-005/006`、
  `Task-ID MVP-W00-RECOVERY-002`；Issue 为 `N/A — recovery bootstrap`。

## 2026-07-28 — MVP-DEC-004：一次性恢复 Bootstrap 例外

- 状态：`active`。
- 触发：待恢复归档内没有 `.git`，无法在新仓库中先提交 Wave plan 再创建
  根提交。
- 决策：用户指令先行授权后，允许且仅允许把校验归档逐字导入为根提交；
  第二个提交立即激活 docs-only Recovery Wave。
- 否决方案：拒绝伪造一个早于导入的历史 commit，也拒绝在根提交中混入恢复
  文档或产品修复。
- 影响：例外仅覆盖 `e4783a4` import；后续所有变更恢复正常 Wave-first。
- 回滚：保留 import tag 和 BLOCK 证据，不重写根提交。
- 关联：`MVP-W00-S01`、`MVP-EVID-001`、
  `Task-ID RECOVERY-IMPORT-001`；Issue 为 `N/A — recovery bootstrap`。

## 2026-07-28 — MVP-DEC-005：不改写 r002，前向重建可追溯验收

- 状态：`active`。
- 触发：r002 第二轮 code/security PASS，但 docs/governance 因 current
  projection、approval attribution 和 frozen Task tuple 三项 P1 BLOCK。
- 否决方案：拒绝 amend/rebase 历史提交伪装当时合规；拒绝扩大 bootstrap
  例外；拒绝在无 Task tuple 下直接打 tag。
- 决策：登记 `MVP-ISSUE-001`，创建 r003，只授权文档治理与完整重验；从
  exact base 冻结 Repository/assignee/Issue/policy/acceptance/verification。
- 影响：r002 发布实现保留为候选证据，不再修改；r003 最终 Gate 通过前
  `MVP-W00` 保持唯一 active，FE/PILOT 继续 blocked。
- 回滚：任一 P1 保持未关闭即停止 tag；不回滚或覆盖原始归档/import tag。
- 关联：`MVP-W00-S06A–S06D`、`MVP-EVID-006`、
  `Task-ID MVP-W00-RECOVERY-003`、`Issue MVP-ISSUE-001`、
  `Policy-Bundle 1d725fc514890338a8d6e7ad338287474674a709b650bacf329ec0ff2f07e0d2`。

## 2026-07-28 — MVP-DEC-006：从可解析 activation base 冻结并恢复 Journal 前缀

- 状态：`active`。
- 触发：r003 final code review 发现 Task base 不含当时 active
  Plan/Registry/Policy；docs review 发现四个 Journal 的历史前缀被改写。
- 否决方案：拒绝 amend/rebase 历史；拒绝只在新 Task 文本中声称 base
  可解析；拒绝用新的 current pointer 覆盖 Journal 顶部历史文字。
- 决策：r004 严格拆分 activation、freeze、implementation；activation
  base 必须已经包含 active r004 Plan/Registry/Policy。四个 Journal 恢复
  各自冻结前缀，后续事件只在 EOF 追加，current authority 由 Registry、
  current Plan 和 README 表达。
- 影响：Recovery 仍是唯一 active、docs-only Wave；全量 Gate 与三路复核
  重跑前禁止 recovered tag，FE/PILOT 继续 blocked。
- 回滚：发现任何前缀或 tuple 不一致即新增 superseding revision，保留
  r003/r004 失败证据，不覆盖原归档和 import tag。
- 关联：`MVP-W00-S07A–S07E`、`MVP-EVID-006`、
  `Task-ID MVP-W00-RECOVERY-004`、`MVP-ISSUE-001/002`、
  `Policy-Bundle 4479549a338b7c6859defb738f93ce37e518c6bddb9ee3d9a58598f1b3f7a1f4`。

## 2026-07-28 — MVP-DEC-007：错误的冻结 SHA 不以改写历史满足

- 状态：`active`。
- 触发：r004 S07C 实算发现 Plan/Task 把 FE-W01 的 26641-byte SHA 从
  权威值 `33a50e1bbd...` 误抄为 `33a50e8f3a...`。
- 否决方案：拒绝修改已批准 r004 Plan/Task；拒绝改造 Journal 字节去匹配
  无来源的错误哈希；拒绝跳过该 acceptance。
- 决策：保留 r004 失败证据，登记 `MVP-ISSUE-003`，以前向 revision
  引用 r009 Plan、R7 Evidence、import tree 与本次实算的同一正确 SHA，
  再重新执行 activation、freeze、Gate 和复核。
- 影响：r004 不进入 S07D/S07E，不创建 tag。
- 回滚：后继 revision 仍不一致则保持 Recovery blocked。
- 关联：`MVP-W00-S07C`、`MVP-EVID-006`、
  `Task-ID MVP-W00-RECOVERY-004`、`MVP-ISSUE-003`。

## 2026-07-28 — MVP-DEC-008：Recovery 完成后只激活 FE-W01 浏览器门禁

- 状态：`active`。
- 触发：`v0.8.0-pilot.1-recovered.1` 的来源、Gate、三路复核、双构建和
  tag/manifest 对齐全部完成；用户要求先完成恢复再进入 MVP。
- 否决方案：拒绝同时激活 PILOT-W00/FE-W02；拒绝在未完成 ptrace、
  credential owner 和真实浏览器 Gate 时直接启动三台 Runner。
- 决策：用终态 `MVP-W00 r006` 关闭 Recovery，只激活 `FE-W01 r012`。
  Browser plugin 优先；明确允许在 plugin localhost/runtime 失败后使用
  仓库外 Playwright fallback，但不降低 syscall/owner Gate。
- 影响：current release 更新为 `0.8.0-pilot.1-recovered.1`；FE-W01 仅在
  已复现范围内允许产品变更，实际 freeze 前仍禁止 runtime/代码改动。
- 回滚：Browser/ptrace/owner 任一失败则 FE-W01 进入 blocked Evidence，
  PILOT-W00 保持 blocked。
- 关联：`MVP-W00-S09`、`FE-W01-S13A–S13E`、
  `FE-ISSUE-003`–`010`、`MVP-EVID-005/006`。

## 2026-07-28 — TR-DEC-001：用双应用路线替代未完成前端路线

- 状态：`active`。
- 触发：用户明确要求按对话历史自动规划并完整实现 Team Control 与 Runner，
  每个 Wave 完成后推送 GitHub。
- 决策：保留 FE-W01–FE-W05 历史和失败 Evidence，但状态改为
  `superseded`；激活只依赖已完成 `MVP-W00` 的 `TR-W00`，后继严格按
  `TC-W01 → RN-W01 → INT-W01 → REL-W01` 顺序推进。
- 安全边界：GitHub 是源码/规划/Evidence 权威源，但凭据不进入仓库；授权
  被撤销或过期时停止推送并保留本地 commit，不尝试绕过权限。
- 否决方案：不伪造 FE-W01 complete，不同时激活多个 Wave，不把所有范围
  塞入一个不可复核的提交。
- 关联：`TR-W00-S01`、`TR-ISSUE-001`、`TR-EVID-W00-001`。

## 2026-07-29 — TR-DEC-002：提前合并后只做前向验收修复

- 状态：`active`。
- 触发：PR #1 在其声明的三路 P0=0/P1=0 Gate 前被合并；随后 code、
  security、docs 独立复核均返回 BLOCK。
- 决策：不改写 merge commit `3a75c737...`，创建 TR-W00 r002 和独立
  acceptance-fix 分支；TR-W00 继续 active，TC-W01 不得提前激活。
- Codex 边界：采用 named permission profile 对真实 `CODEX_HOME` 执行
  OS sandbox read-deny，并以不调用模型的负向 canary 失败关闭；不接受
  prompt-only 隔离。
- 状态机：补充受治理的 `active -> superseded`，使 TR-DEC-001 的路线
  替代可由规则解释，同时明确 superseded 不等于 complete。
- 回滚：任一 P1 未关闭则保留 r002/偏差 Evidence，停止后继 Wave；
  不 force-push、不删除 PR #1、不伪造验收先于合并。
- 关联：`TR-W00-S06`–`S08`、`TR-ISSUE-002`、
  `TR-EVID-W00-002`、`Task-ID TR-W00-ACCEPTANCE-002`。

## 2026-07-29 — TR-DEC-003：TR-W00 验收后单独激活 TC-W01

- 状态：`active`。
- 触发：exact `60465b59...` 的 code/security/docs final review 均
  P0=0/P1=0，`TR-EVID-W00-003` 收录确定性 Gate 与目标主机边界。
- 决策：将 `TR-W00 r002` 标记 complete，只激活其直接后继
  `TC-W01 r002`；TC 先冻结中央 Registry、预算账本和 deterministic
  Context Bundle，不提前实现 Runner 更新、MCP 或 installer。
- 否决方案：不并行激活 RN/INT/REL，不把 reviewer P2 冒充已解决，也不因
  PR #1 已合并而跳过新的 Task Freeze。
- 安全影响：Context Compiler 在控制面只编译已批准的元数据与 hash，
  不读取 Vault 正文，不接收 Codex OAuth；Budget event 必须幂等并防溢出。
- 回滚：TC schema/RBAC/hash 任一 P1 未关闭则 TC 保持 active，RN 不激活；
  TR 的历史 acceptance 与 PR 偏差只读保留。
- 关联：`TR-EVID-W00-003`、`TC-ISSUE-001`、
  `TC-EVID-W01-001`、后续 `Task-ID TC-W01-CONTROL-003`。

## 2026-07-29 — TC-DEC-001：r002 失败证据前向升级 r003

- 状态：`active`。
- 触发：TC r002 exact `6aab01f...` 三路独立复核均 BLOCK。
- 决策：不 rewrite 已推送 commit；以 r003 修复项目复合身份、Context target
  identity、显式 secret-safe schema、CRUD、UI 五状态和 traceability。
- 否决方案：不以关键词扫描代替显式 schema，不把裸 ID conflict 当作安全
  隔离，不把静态源码正则当作 UI 行为测试。
- 影响：TC 保持唯一 active；RN/INT/REL 不激活；legacy unsafe state
  失败关闭并要求管理员显式迁移。
- 回滚：migration/schema Gate 或任一 P1 失败时保持 TC active，不覆盖
  r002 Evidence。
- 关联：`TC-EVID-W01-002`、`TC-ISSUE-001`、后续
  `Task-ID TC-W01-ACCEPTANCE-004`。

## 2026-07-29 — TC-DEC-002：r003 路径与 RBAC provenance 升级 r004

- 状态：`active`。
- 触发：r003 exact `e879b0e2...` 三路 review 均 BLOCK；除路径 traversal
  和 superscript device 外，docs review 发现 r002 的
  `teamcontrol/service.go` 修改从未被 Plan/Task scope 授权。
- 决策：不改写 r002/r003；创建 r004，显式纳入 `service.go`、路径校验和
  回归文件。activation 推送后再以远端 exact SHA 冻结 Task，冻结后以等价
  RBAC helper 前向建立 provenance。
- 安全边界：raw/decoded path 使用同一平台中立 lexical boundary；
  Registry error 不回显不受信任内容且 URI 限长。
- 回滚：任一 P1 未关闭，TC 保持 active，RN 不激活；不 force-push、不把
  历史 failed Evidence 改为 passed。
- 关联：`TC-EVID-W01-010`、`TC-ISSUE-001`、`TC-W01 r004`。

## 2026-07-29 — TC-DEC-003：以 TC-W02 替代未验收 RN 与宽泛 INT 路线

- 状态：`active`。
- 触发：用户明确要求 Team Control 成为全局规则、项目规则和项目知识的
  权威控制面，并要求本阶段只做源码盘点与合同规划。
- 决策：RN-W01 在已有实现但未完成 independent acceptance 的状态下转为
  `superseded`，不标记 complete；INT-W01 在未激活、未冻结、未实现时转为
  `superseded`；只激活 docs-only `TC-W02 r001`。
- 否决方案：拒绝把 RN 实现提交冒充已验收完成；拒绝直接激活 INT-W01；
  拒绝同时激活多个 Wave。
- 影响：替代路线为
  `TC-W02 → TC-W03 → TC-W04 → TC-W05 → TC-W06 → REL-W01`；
  TC-W03–W06 先保持 proposed/product-code-false。
- 回滚：撤销尚未合并的规划提交并恢复 RN-W01 active；不触碰 RN 实现、
  Evidence 或产品数据。
- 关联：`TC-ISSUE-002`、`INT-ISSUE-001`、`RN-ISSUE-001`、
  `TC-EVID-W02-001`、`TC-W02-S01–S04`。

## 2026-07-29 — TC-DEC-004：可覆盖默认值与不可覆盖强制约束分层

- 状态：`active`。
- 触发：用户继承默认要求
  `global defaults → team → project → repository → component`，同时要求
  global mandatory constraints 最终校验且下层不得覆盖。
- 决策：global defaults 作为第一层可被项目覆盖；global mandatory 不参加
  普通 merge，而在所有覆盖后验证 resolved result，违反时失败关闭且不生成
  executable Context。
- 否决方案：拒绝把 mandatory 放在最前面后被下层覆盖；拒绝把所有 global
  值都设成不可覆盖，从而消灭项目差异。
- 影响：ResolvedPolicy 必须带逐 rule provenance、ordered bundle refs、
  mandatory refs、canonical hash 和 validation result。
- 回滚：resolver revision 未通过冲突矩阵和 golden hash Gate 时保留现有
  四层 resolver，不激活 TC-W04。
- 关联：`TC-W02 target-contracts.md`、`TC-W04`。

## 2026-07-29 — TC-DEC-005：一个逻辑知识权威、一个正文边界、可重建索引

- 状态：`active`。
- 触发：静态源码证明 Team Control KnowledgeSource 与 SQLite Memory
  Catalog 独立表达批准知识，Agent 还能绕过 Context Bundle 注入 Catalog。
- 决策：Team Control 成为 Knowledge identity/revision/approval/visibility
  的逻辑权威；正文进入一个 content-addressed object store；检索索引只是
  可丢弃 projection。旧 Catalog/Markdown/Harness/Obsidian 都是 adapter。
- 否决方案：拒绝长期双写两个 active store；拒绝把正文直接塞入 Team
  Control registry state；拒绝把索引当作不可重建权威。
- 影响：`project_id="*"` 迁移为 explicit global scope，所有 legacy active
  必须先对账 checksum/scope/decision；歧义记录进入 pending/quarantine。
- 回滚：shadow import 可整体丢弃；read cutover 前旧系统仍是唯一 writer；
  不删除旧库。
- 关联：`TC-ISSUE-002`、`TC-W03`、`TC-EVID-W02-001`。

## 2026-07-29 — TC-DEC-006：Runner 只读 MCP，反馈只创建候选

- 状态：`active`。
- 触发：用户要求 Runner 的使用证据、矛盾、陈旧和修订建议形成闭环，但
  不能直接修改 active 知识。
- 决策：MCP audience 绑定 project/task/revision/attempt/runner/lease/
  Context hash；只提供 manifest/search/read/citation/policy explain。
  Evidence 先验签、去重、scope 校验，再创建 candidate；独立
  `memory_approve` 人工审批后才能 active。
- 否决方案：拒绝 MCP 提供 promote/delete active；拒绝任务完成时自动学习
  并晋升；拒绝客户端自报 project/role。
- 影响：ExecutionPack 冻结 Context hash，Evidence 新增知识使用/citation/
  feedback，但继续 secret-free 与签名；TC-W05 独立实现和复核。
- 回滚：feedback ingestion 可停用且保留 Evidence；任何 P1 保持 TC-W05
  active/blocked，不切换 MCP read path。
- 关联：`TC-W05`、`TC-W02 target-contracts.md`。

## 2026-07-29 — TC-DEC-007：客户端仅为投影，知识迁移分阶段且切换独立授权

- 状态：`active`。
- 触发：目标要求 Console、CLI 与 Obsidian 展示 effective policy、知识版本、
  citation usage、候选和审批，同时不能成为 active state 或项目授权来源；
  旧 KnowledgeSource/Catalog 又必须可回滚地迁移。
- 决策：Web/CLI/Obsidian 只读取服务器端 project/global entitlement 派生的
  projection，并通过受控 command 提交候选/审批；迁移严格分 inventory、
  non-executable shadow import、dual-read compare、read cutover 和 retirement。
  TC-W03 只建立并验证 shadow authority/content/index，不 disable 旧 writer；
  runtime old-writer disable/read cutover 只允许在 TC-W06 的新 approved
  revision、synthetic Gate 通过且另有 operator-owned Task 时执行。
- 否决方案：拒绝客户端参数扩大 scope 或直接写 active；拒绝 TC-W03 在无
  真实迁移授权时宣称成为 runtime 唯一 writer；拒绝回退到未 replay 的旧
  snapshot。
- 影响：TC-W06 必须覆盖 loading/empty/denied/error/conflict/stale、
  server-side authorization、migration comparison 和 monotonic recovery；
  本规划阶段不迁移真实数据。
- 回滚：停止 mutation，验证 signed monotonic checkpoint，replay/reconcile
  cutover 后 authority events；缺失或倒退时只能 non-executable recovery。
- 关联：`TC-W03`、`TC-W06`、`TC-ISSUE-002`、
  `TC-W02 migration-and-wave-roadmap.md`。
