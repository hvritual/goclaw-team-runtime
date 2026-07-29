# 实现范围与剩余边界

版本：GoClaw Team Runtime `0.8.0-pilot.1`。

`0.8.0-pilot.1` 把现有团队闭环收敛为一个可检查的三人技术试点候选：
单中央写者、一个项目、恰好三名成员和三台 Linux substrate Runner。插件
源码仍保留作可选适配器，但默认发行、部署、审批和团队运维不依赖 Obsidian。

当前 active Wave 是
[`PILOT-W00`](waves/pilot-readiness/pilot-w00/plan-r003.md)。代码与
确定性测试可以形成候选，但前台真实浏览器、目标真机 bwrap/WSL2/Lima、
ChatGPT Workspace、飞书租户、Obsidian Desktop 和灾备介质仍必须在部署
现场补证。本文不会把交叉编译、模拟进程或页面存在写成“已上线”。

基线：`smallnest/goclaw` 的 `e05c79d`，在其上增加本方案代码。本文严格区分“领域模型/本地执行已实现”和“外部平台自动化已实现”，避免把可登记的关联对象误写成已经接通 GitHub、Jira 或 CI。

## 0.8.0-pilot.1 新增的试点 Gate

- `goclaw pilot check` 失败关闭验证：恰好三名 active 项目成员、三台不同
  active owner 且 online/busy 的 Runner、项目策略、唯一 active
  `PILOT-W00`、credential attestation，以及可实际解密并通过语义校验的
  age 冷备。
- 三台 Runner 都必须带 `goclaw-runtime-linux-v1` capability，metadata
  必须是 Linux amd64/arm64、`native-linux|wsl2|lima`、bwrap 和合法
  wrapper SHA-256；三台 wrapper 摘要必须完全一致。
- `runner doctor` 检查 substrate、guest-local 仓库、Codex
  `login status`、`CODEX_HOME`、路径归属、Git 配置、device key 和实际
  bwrap 启动；原生 Windows/macOS Runner 失败关闭。
- Team `dev create --wave-step STEP` 只把 Step 意图交给 Gateway。服务器
  从注册仓库的精确 base commit 解析唯一 active Registry、approved/active
  plan、revision、依赖、允许范围与 SHA-256，并固定
  `created_by=planner-service` 和真实 `requested_by`。
- `control.consistency.check` 检查 TeamControl、Workstation 与
  Orchestrator Lite 的跨存储绑定；critical finding 会阻止 enqueue 和
  accept。
- `pilot backup/verify-backup/restore` 提供维护锁、Gateway 停机检查、
  age 加密、manifest/digest/mode/type、事件链、Runner Evidence 和 Git
  bundle 的冷备语义验证；restore 只写入新的空目录。
- 项目会话改用版本化、逐段 base64url 的无碰撞 key；试点项目 ID 限定
  64 字符且禁止冒号。Web Console 增加授权项目选择、project/topic
  history、断线刷新和任务/Issue/WorkItem 操作入口。

完整部署步骤见
[`PILOT_3_PERSON_DEPLOYMENT_CN.md`](PILOT_3_PERSON_DEPLOYMENT_CN.md)。

## 已实现

- Codex app-server Provider：
  - 使用本地 Codex CLI 的 ChatGPT 登录态。
  - 不接收 OpenAI API Key。
  - 临时 Thread、只读 Sandbox、结构化输出。
  - Provider 单元测试使用协议模拟进程。
- 中央 TeamControl：
  - 首个管理员一次性 `goclaw team bootstrap`，后续全部团队操作使用个人 `GOCLAW_USER_TOKEN`。
  - 个人 Token 只保存 SHA-256 摘要；团队、项目、仓库与项目成员均由服务器端解析。
  - User/Token 为全局对象；目标用户属于多个活动团队时，Token 签发/列出/撤销和全局用户状态变更要求操作者同时是所有这些团队的 active owner/admin。
  - 团队 `owner/admin/member` 与项目 `owner/maintainer/developer/reviewer/viewer` 两层 RBAC。
  - 项目成员业务域和容量点，团队页聚合活动、排队、阻塞任务与 Runner 在线状态。
  - Issue、WorkItem、Assignment 的结构化字段、依赖和状态机。
  - Issue 关闭需要 resolution 或关联修复证据；WorkItem 未完成依赖阻止开工。
  - 非本人 Assignment 要求 `project.manage`；已有 active owner 的 Issue/WorkItem 只允许 owner 或项目管理者迁移，同项目其他 developer 不能修改他人负责资源。
  - 一个 WorkItem 最多绑定一个开发任务；Issue 可被多个任务共享，只有全部关联任务都为 `done` 且全部 WorkItem 都为 `done` 时才 resolved。`cancelled` 永不算成功；取消分支只取消其 WorkItem，Issue 保持 open/verifying/blocked，等待重新指派或显式另行关闭。
  - Artifact 与 CorrelationLink 可关联 task、run、trace、diff/evidence、commit、PR、CI、release 和 regression case。
  - `dev link-pr --commit --url` 验证外部 commit 与已验收 Workstation patch 后，自动创建 commit/PR Artifact，并关联队列 Task、Repository、WorkItem 与 Issue。PR URL 仅做严格语法登记：必须是无凭据、无 query、无 fragment 的绝对 HTTP(S) 地址；没有 provider API，不验证远端 PR head、内容或状态。
  - Document Registry 提供类型、owner、revision、checksum 与 supersedes；Component Registry 提供仓库、路径、owner 和依赖。
  - PolicyBundle 按 team → project → repository → component 分层解析并生成稳定哈希。
- Workstation Runner：
  - 每台电脑独立注册 Runner，使用个人 Token 做控制面认证，并生成独立 `0600` device key。
  - 本地 Codex CLI 使用该电脑自己的 ChatGPT/Codex OAuth；控制面不接收 OAuth 文件。
  - 试点执行合同统一为原生 Linux、WSL2 或 Lima Linux guest；原生 Windows/macOS 仅提供控制 CLI。
  - `runner doctor` 失败关闭检查 Linux substrate、架构、guest-local 路径、Git 配置、`codex login status`、device key、`CODEX_HOME`、root-owned wrapper 和实际无网络 bwrap 启动。
  - 持久化排队任务、原子 claim、attempt、lease、heartbeat、幂等 complete/fail 与过期恢复。
  - `runner.cancel` 以持久 receipt 幂等取消 queued/failed 任务；leased、completed、cancelled 均拒绝。Task assignee 或项目管理者可取消，其他 developer 不可操作。
  - 同一 Runner 最多一个活动 lease；空闲时可通过 CLI/RPC 原子轮换 device key。
  - `goclaw runner update` 可由 owner 修改显示名，或在无活动 lease 时整体替换项目/能力并 disable/enable；Team 模式禁止 wildcard project 并逐项目校验 `work_item.write`。
  - Claim 强制 ExecutionPack 的非空 `assignee_id` 与 Runner owner 一致；空 assignee 的底层任务才允许任意项目/capability 匹配者领取。
  - secret-free ExecutionPack 固定项目、仓库、任务 revision、Issue/WorkItem、base commit、策略哈希、路径和验证 argv。
  - `dev.task.enqueue` 只从服务器端冻结任务构造 ExecutionPack；queue ID 和幂等键由 revision + execution bundle 在服务器派生，客户端无入队幂等参数；每个 WorkItem 必须恰有一个 active owner 且等于 task assignee，并校验 team/project/repository、base commit 和策略漂移，拒绝客户端伪造执行包。
  - revision/attempt 隔离 worktree、本地 Codex 执行；先跑完全部冻结验证，再从最终工作树收集 diff、范围和 no-commit 结果，最后生成 HMAC-SHA256 EvidenceBundle。
  - Codex、内部 Git 和 verifier wrapper 都从最小环境白名单启动；Codex 主进程通过 `CODEX_HOME` 使用本机订阅 OAuth，模型命令使用 named permission profile 对该目录 `deny`，并在模型调用前运行 read-deny canary。
  - GoClaw Token、SSH agent、Git askpass、Docker/Kubernetes 和云凭据路径等宿主能力变量永久剥离，显式 `--allow-env` 不能覆盖；allowlist 只交给 Codex，不进入内部 Git 或冻结 verifier。
  - `runner work` 默认失败关闭：必须提供绝对、受审且不可由 Runner 用户篡改的 `--verification-sandbox`，或仅在 Runner 整体已位于一次性隔离 VM/容器时显式使用互斥的 `--unsafe-host-verification`。Linux 包附带 bubblewrap wrapper，断网并遮蔽 host home/run/tmp，只让 worktree 与临时 HOME 可写。
  - 完成证据自动导入 Orchestrator Lite，重新校验 revision/bundle/base/head/diff/路径/冻结检查，重算范围与独立审查并运行 DoneGate；最终 `dev.task.accept` 再防漂移验收并关闭当前 WorkItem，共享 Issue 仅在所有 Task/WorkItem 都为 `done` 后 resolve。
  - 检测并拒绝 Codex 自动 commit；Runner 不 push、建 PR、等 CI、merge 或 release。
- Better Harness：
  - 不可变版本和原子活动指针。
  - 隔离 Candidate 与 Change Manifest。
  - Candidate 变更范围校验与 Protected Paths 防篡改。
  - Optimization、Golden、Holdout 分组。
  - 基线与 Candidate 逐用例对照；基线通过而 Candidate 失败即拒绝。
  - 显式 Trace 的令牌与延迟增量门槛。
  - 命令型 Eval 默认禁用执行。
  - 身份化人工批准、批准/提升职责分离、提升 Compare-And-Swap、认证回滚。
- 认知与决策治理：
  - Reviewer Token 摘要认证、角色校验、理由、最强反方论点和证据引用。
  - 可选 `reviewers.<key>.team_user_id` 把个人 Token principal 绑定到描述性 Reviewer 策略 key；绑定大小写不敏感且唯一，审计身份仍记录 principal。
  - Seed/高风险 Seed/演化/Harness 法定人数；重复审批和自我审批防护。
  - 任务评审人数、每人评审类型上限、最终验收分离。
  - 需求评估分歧、阈值灰区和同模型相关性升级。
  - 利益相关方 Claim/Conflict 保留与逐条解决。
  - 评估争议人工裁决；不能覆盖机械门，也不等于任务验收。
  - 取消、停止条件、Harness 回滚均进入认证决策记录。
- Trace：
  - 项目、话题、渠道、会话、Harness 版本、上下文文件、工具输入、工具结果、工具错误、输出和耗时。
  - 人类反馈。
- Go 原生 Ouroboros：
  - 多视角苏格拉底式访谈、Greenfield/Brownfield 加权歧义度、维度下限、分歧/灰区/相关性和连续 readiness 门。
  - 严格模型 JSON、一次有界格式修复、上下文和输出预算。
  - 内容寻址的不可变 Seed、父哈希谱系、备选方案、证伪条件、未行动成本、预演失败、参考类预测、停止条件和人工批准状态。
  - 带前向 SHA-256 的 append-only SessionEvent；读取时验证事件链与 Seed 内容哈希。
  - 只有 approved active Seed 能单向编译为 Orchestrator Lite 任务；任务仍需四审。
  - EvidencePackage 机械门、语义门、盲化角色评审、关键发现否决、分差检测和 Go 多数决。
  - 通过/失败/取消/回滚/无反馈结果、按任务结果替代和项目参考类。
  - 只生成后继 Seed 候选的 Reflect/Wonder 演化、最近评估窗口、连续通过要求、累计模型预算、代数硬上限、本体相似度和 period-2 振荡检测。
  - 飞书仅开放访谈/回答/读取/重评/结晶工具；审批、编译和执行不暴露给聊天工具。
- Orchestrator Lite：
  - Goal、Plan、Milestone、WorkItem、Capability、EvidencePlan、Scope、Risk、Cost 与 DoneGate 结构化契约。
  - Scenario、Capacity、Risk、Cost 四类人工评审；全部批准后才能冻结。
  - 固定 Git base commit 和执行包 SHA-256。
  - 每任务隔离 worktree/branch，单任务运行锁和陈旧锁显式恢复。
  - Codex `exec --json` 开发 Hand，复用 ChatGPT 登录态并支持 Thread resume；每次 Hand 使用隔离 HOME/XDG/runtime/tmp 和最小环境，只通过 `CODEX_HOME` 读取订阅 OAuth。
  - 中央单用户 Hand 在执行前后都校验 `HEAD == BaseCommit`；检测到 Codex 自动 commit 即失败。
  - 中央冻结 verifier 默认要求 `verification_sandbox` argv wrapper；首项必须是绝对、普通、可执行且不可被 group/other 写入的文件。`unsafe_host_verification` 与其互斥，只供整体已位于一次性隔离 VM/容器的中央 Hand 使用。
  - 路径 allow/deny、符号链接边界、依赖清单、文件数、总行数和令牌预算检查。
  - 结构化 argv 验证、只读独立模型审查、EvidencePackage 和 Go-only DoneGate。
  - 可导入 Gateway 已验证的 Workstation EvidenceBundle；不信任远端 DoneGate 结论，并对相同 Bundle SHA 幂等。
  - 已验收的 Workstation 任务可绑定外部 commit/PR：验证中央受管 repo 可见 commit、frozen-base 祖先关系、规范化 diff 精确相等和完整 trailers；相同 commit/URL 幂等。
  - 修复次数上限、人类最终验收防篡改、本地提交和完整事件审计。
  - ChangeIntent 修订会递增 Revision 并重置四类评审。
  - Team `dev create` 要求稳定 ID，`dev list` 要求项目；create/review/freeze/accept/cancel/link-pr 已定义收敛式重试语义，revise/repair 通过必填 `expected_revision` 做 CAS。CLI 与 Obsidian 自动读取当前 revision 后发送。
  - Team `dev create` 还要求 `--wave-step`；Gateway 从注册仓库的精确 base 解析并冻结 active Wave、plan revision/path/hash、Registry hash 和 Step，忽略客户端自报的其他 Wave 权威字段。
  - revise/repair 拒绝仍 queued/leased 的旧 revision；queued/failed 可先由 `runner.cancel` 撤销。修订会把 in_progress/verifying WorkItem 先退回 blocked，新 revision 必须重新四审、冻结和入队。
- 通用 Markdown 知识治理：
  - 受控全文检索和读取。
  - 完整内容提案。
  - SHA-256 冲突检测；Git 后端额外冻结并验证 HEAD revision。
  - Inbox/Approved/Rejected Markdown 投影。
- 图书信息管理式项目记忆：
  - Work/Expression/Manifestation/Item 四层身份、内容校验和与同源版本替代。
  - title/abstract/subject/language/collection 描述元数据和受控 Facet。
  - 人、组织、项目、系统、主题、地点、设备的首选名称、别名、重定向与人工合并。
  - Source/Agent/Activity/Trace/Revision/SHA-256 来源追踪和可引用检索结果。
  - pending/active/rejected/superseded/withdrawn 生命周期、有效期、失效时间和复核周期。
  - 显式 supersedes/contradicts/derived_from/supports/related_to 关系及未解决冲突统计。
  - 项目作用域、共享项目只读引用、跨项目覆盖拒绝和提示注入隔离。
  - Agent 仅能检索 approved 记录或创建 pending 候选；人工 Reviewer 才能批准、拒绝、续期或撤回。
  - Markdown 扫描、稳定相对路径身份、`markdown`/`git+markdown` 来源、幂等导入、同名馆藏项区分与符号链接拒绝。
  - 本地无 Key 哈希嵌入和 SQLite 余弦回退，修复无 sqlite-vec 环境下的 builtin 记忆。
  - CLI、Gateway JSON-RPC、Obsidian“记忆”页和 Harness Trace 记忆 ID 串联。
- 跨渠道项目会话：
  - `project_id + topic_id` 作为统一会话边界。
  - 飞书按路由表映射项目。
- Gateway：
  - Harness、Trace、实验、反馈与知识提案 JSON-RPC。
  - WebSocket Token 子协议认证。
  - Team/Runner RPC 使用个人 Token 身份和服务器端项目 RBAC；`dev.task.enqueue`、`runner.key.rotate` 等操作不能只靠客户端 project ID。
  - freeze/revise/repair/enqueue/link-pr 仅 task assignee 或项目管理者；accept/cancel 必须具备 `project.manage` 并分别通过 `task_accept`/`task_cancel` Governance 角色。
  - `dev.task.link-pr` 要求项目 `artifact.write`；验证成功后由 Gateway 自动登记 TeamControl commit/PR Artifact 和 CorrelationLink。
  - Team 模式 deny-by-default：旧 process-global 配置、日志、渠道、会话、Browser 和 Cron RPC 默认禁用；Harness、Memory Catalog 与 Ouroboros 按绑定项目、请求项目或资源所属项目做 RBAC。
  - 决策方法使用独立 Reviewer 身份和角色；不能退回到连接 Session ID。
  - 命令型 Harness Eval 和长开发任务默认拒绝远程执行。
  - Team `dev.task.run/repair/resume` 无条件禁用，即使打开 `development.gateway_allow_execution` 也不放行；该开关只适用于未启用 TeamControl 的单用户模式。团队唯一执行路径是 `dev enqueue` → Workstation 持久队列。
  - 项目聊天事件携带明确 `project_id/topic_id`，历史记录和浏览器投影按授权项目会话隔离；旧的无作用域事件不能作为跨项目广播依据。
  - `control.consistency.check` 提供跨 TeamControl/Workstation/Development 的只读一致性报告，critical finding 会阻止 enqueue/accept。

### Team 模式 RPC 能力矩阵

| RPC 家族 | Team 模式行为 |
|---|---|
| `health` | 连接认证后可用，不读取团队资源 |
| `chat`、`agent`、`agent.wait` | 要求显式项目并校验 `project.read` |
| `team.*`、`project.*`、`repository.*`、`issue.*`、`work.*`、`assignment.*`、`artifact.*`、`correlation.*`、`document.*`、`component.*`、`policy.*`、`runner.*`、`dev.*` | 各 handler 从资源或参数解析项目并执行对应 RBAC；高风险决策再叠加 Reviewer 认证 |
| `harness.*`、`knowledge.*` | 请求项目必须匹配 `harness.project_id`；按 `artifact.read/write` 或 `document.read/write` |
| `memory.*` | 按请求项目或 Catalog 记录所属项目执行 `document.read/write`；共享记录要求具体项目 |
| `ouroboros.*` | 按请求项目或 Session/Seed 所属项目执行 `project.read`、`work_item.write` 或 `artifact.write` |
| 旧 process-global 配置、日志、渠道、会话、`browser.*`、`cron.*` | 默认拒绝 |
| 未列入授权策略的方法 | 默认拒绝，不因持有 Gateway Token 自动放行 |

- Team Web Console：
  - HttpOnly/SameSite 浏览器会话、CSRF、同源 WebSocket 与安全响应头。
  - 总览、对话、规格、记忆、审批、开发、团队、进度和 Harness 九个工作区。
  - 个人 Team Token 与 Gateway Token 不进入浏览器持久存储；Reviewer Token 仅在页面内存。
  - 其余能力与原 Obsidian 控制台保持同一 Gateway RPC 和项目 RBAC。
  - 上述条目表示试点候选代码面，不等于目标浏览器、真实 OAuth 或三人现场运行验收；实际可用性以 `PILOT-W00` evidence 和现场 Gate 为准。
- 可选 Obsidian 适配器：
  - 实时聊天。
  - 项目记忆统计、检索、候选审批、到期续期/撤回和来源展示。
  - Ouroboros 规格访谈、歧义度、Seed 结晶、评估与演化。
  - Seed/演化候选、需求/评估分歧、利益冲突、知识、Harness 与开发任务审批。
  - 开发任务状态、DoneGate、修复和最终验收面板。
  - Trace 进度列表。
  - Harness 状态与认证回滚。
  - 团队页只读展示成员负载、项目任务、Bug、Runner/租约、策略、文档与组件；七类 RPC 独立失败降级。
  - Gateway Token、Team User Token 与 Reviewer Token 分别使用 SecretStorage。
  - 自动重连。
- 部署：
  - Codex/飞书/Harness/Development 示例配置和完整开发任务模板。
  - 三人 Governance 配置片段、Linux/WSL2/Lima Runner 模板和三人试点部署手册。
  - age 加密冷备、实际解密验证、恢复到新目录和 credential attestation Gate。
  - 本地构建和 Obsidian 安装脚本。
  - systemd 与 Caddy 示例。

## 这是 0.8.0-pilot.1 技术试点候选，不是假定已完成的企业平台

当前明确边界：

- Team 开发任务已由 Gateway 在创建、冻结、入队和验收路径重验 active
  Wave、计划修订、Step、Registry/plan SHA 与精确 base；这依赖目标仓库在
  该 base 内包含合法 Registry/plan，并不等于外部 Git 托管平台已提供
  branch protection 或签名提交。
- 从大量 Trace 自动聚类失败并生成根因报告。
- 自动调用开发 Agent 修改 Candidate 文件。
- 基于统计显著性的在线 A/B 与自动回滚。
- 控制面 Leader 选举、Leader 租约、共识和自动故障转移。
- TeamControl 与 Workstation 使用文件存储：进程内有锁、写入使用 fsync + 原子替换，但同一 root 只支持一个 GoClaw 进程写入，没有跨进程锁、外部数据库事务或水平扩容。
- 当前任务级互斥和幂等收敛不构成跨 TeamControl、Workstation、Orchestrator Lite 的数据库事务；必须维持单中央写者并对三类状态做一致性备份。
- Workstation 已有持久任务、Runner、lease/heartbeat 和过期恢复，但 Orchestrator Lite 仍是单实例控制面；这不是分布式调度共识。
- 没有 GitHub/GitLab/Jira 双向同步。外部 commit/PR 的本地内容校验与关联登记已实现，但不会 fetch/push、创建/批准/merge PR、读取远端 PR/CI 状态或回写外部对象。
- Runner 不自动 commit、push、创建 Pull Request、等待 CI、执行 merge、发布或回滚；自动 commit 会被证据门拒绝。
- 冻结开发任务的 assignee 会约束 Runner owner；`business_domain` 和容量仍不参与自动排程优化，只用于计划、校验与看板。
- Runner 的 frozen verifier 必须经 `--verification-sandbox` wrapper 执行；Linux 基线 bubblewrap wrapper 禁网、遮蔽 host home/run/tmp 并限制可写面。`--unsafe-host-verification` 只适合 Runner 本身已在一次性隔离 VM/容器中；当前没有系统强制的 microVM 网络隔离。
- 当前构建与确定性测试不能证明目标真机的 bwrap、WSL2 或 Lima 配置；
  每一种实际 substrate 都必须用同参数 Doctor 和受控任务补现场证据。
- Runner device key 是控制面和工作站共享的 HMAC 秘密，不是 TPM attestation、公钥设备证书或不可抵赖签名；中央凭据泄露者可以伪造相应 Runner 证据。
- Catalog 的多语种神经语义检索；当前使用本地字段加权词法检索，builtin 另有无 Key 哈希嵌入，QMD 可作为可选内容发现索引。
- 自动主题词推荐、自动权威消歧、RDF/SPARQL 互操作与 MARC/RDA 完整交换。
- Catalog 字段级加密、自动脱敏和外部签名/WORM 审计；当前目录权限为 `0700`、数据库为 `0600`，内容仍是明文 SQLite。
- 外部 Markdown/Git 托管平台的远端同步健康 API；GoClaw 只验证本地知识根和 Git revision。
- Team 模式已禁用无项目策略的旧 process-global RPC，并给 Harness、Memory Catalog、Ouroboros 等已支持域增加项目 RBAC；这不等于所有历史 Gateway 功能都已改造成多租户资源，未列入授权策略的方法会失败关闭。
- Ouroboros 模型调用的分布式并发、任务队列和租约；当前为单节点进程内串行状态机。
- 密码学签名的 Seed/审批身份与外部 WORM 事件存储；当前为本地 SHA-256 完整性校验。
- 真正的多供应商评审；当前会检测同模型相关性并升级人工，但不同模型 ID 仍不证明供应商或训练数据独立。
- 可选 Obsidian 适配器仍需在真实 Obsidian Desktop 做兼容性验收；它不再阻断核心 Runtime 发布。
- 没有真实 ChatGPT Workspace/Codex OAuth 时，不能宣称本地 Runner 或中央
  模型调用已打通；OAuth 不由 GoClaw 分发或跨成员同步。
- 没有真实飞书 App 凭据、回调、白名单/pairing 与路由验收时，不能宣称
  飞书机器人已打通。
- age 归档代码存在不等于灾备可用；必须用现场 identity、介质、恢复目录和
  恢复时限完成一次实际演练。

在生产阶段，应优先补：

1. Reviewer 共享秘密与 Runner HMAC device key 升级为个人证书、TPM/硬件密钥或公钥签名，并持续把需要保留的历史 RPC 改造成显式项目资源。
2. 把 TeamControl/Workstation 文件存储迁移到支持事务、租约和 HA 的数据库/队列，并增加孤儿 worktree 回收。
3. Trace/Evidence 中敏感字段的脱敏与保留周期。
4. 外部秘密管理。
5. SessionEvent 与 Harness Registry 的外部签名、WORM 备份和恢复演练。
6. Git 托管平台、Jira、CI 与 PR Policy 双向适配器。
7. Runner device key 吊销审计、中央凭据访问审计，以及 TPM/硬件密钥或公钥设备身份。
8. 自动诊断只能生成 Candidate，仍不得绕过 Eval、职责分离与人工批准。

三人试点从
[`PILOT_3_PERSON_DEPLOYMENT_CN.md`](PILOT_3_PERSON_DEPLOYMENT_CN.md)
开始；团队使用方法见
[`TEAM_DEVELOPMENT_CN.md`](TEAM_DEVELOPMENT_CN.md)，工作站命令与租约
语义见 [`WORKSTATION_RUNNER_CN.md`](WORKSTATION_RUNNER_CN.md)。
