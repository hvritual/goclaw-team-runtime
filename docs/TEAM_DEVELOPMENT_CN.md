# 十人团队开发闭环

本文面向约 10 人、多人多业务域、多人多电脑的研发团队，说明 GoClaw Team Runtime `0.7.0` 如何统一身份、项目、代码策略、任务、缺陷、证据、文档与组件复用。默认人工入口为 Team Web Console；文中遗留的 Obsidian 操作均可由 Web Console 对应工作区完成。

任何需要分阶段推进的更新都必须先进入
[`docs/waves`](waves/README.md)：先登记 Issue 和 Wave，批准带修订号的计划，
再冻结具体任务。当前这是仓库级人工/Agent/Reviewer 门禁，TeamControl 和
Gateway 尚不会在 freeze 时自动验证 Wave 计划修订或 SHA；不得把文档约束
误写为已经实现的服务端强制能力。

它不是让 10 台电脑各自维护一套状态，而是采用：

```text
                       ┌─ Obsidian Project Console
飞书群 / Obsidian ────┤
                       └─ 管理 CLI
                               │
                               ▼
                   Gateway + TeamControl
                     中央单写入控制面
            ┌──────────┼───────────┐
            │          │           │
       团队与 RBAC  Issue/Work  策略/文档/组件/关联
            │          │           │
            └──────────┼───────────┘
                       │ 任务租约
          ┌────────────┼────────────┐
          ▼            ▼            ▼
     成员电脑 A     成员电脑 B   … 成员电脑 J
     Local Runner   Local Runner    Local Runner
     Local Codex    Local Codex     Local Codex
```

中央 GoClaw 进程是 TeamControl 状态和 Runner 队列的唯一写入者。成员电脑只领取服务器授权的执行包、在本地隔离 worktree 中调用本机 Codex，并回传签名证据。

## 1. 三种身份与三类秘密

每位成员都应拥有相互独立的三种凭据：

| 凭据 | 用途 | 存放位置 | 是否共享 |
|---|---|---|---:|
| 个人 `GOCLAW_USER_TOKEN` | Gateway 第二层身份认证、项目 RBAC | 成员电脑的秘密管理或进程环境 | 否 |
| Runner device key | 工作站身份、EvidenceBundle HMAC 签名 | 成员电脑 `0600` 文件；控制面保存独立凭据副本 | 否 |
| 本地 Codex OAuth | 使用成员自己的 ChatGPT/Codex 订阅模型 | Codex CLI 自有凭据存储 | 否 |

此外，Gateway 自身仍有连接级 `gateway.websocket.auth_token`。连接 Token 不能替代个人 Token，个人 Token 也不能替代 Reviewer Token。Reviewer Token 继续用于 Harness、Ouroboros 和 Orchestrator Lite 的高风险审批。

Team 模式建议给描述性治理身份配置 `governance.reviewers.<key>.team_user_id`，例如把 `erin-final` 绑定到全局用户 `erin`。Gateway 只从个人 Token 取得 principal，再用该绑定查找 Reviewer Token 摘要与角色；客户端填写的 Reviewer ID 不决定身份，决策审计仍记录真实 principal。一个 Team 用户最多绑定一个 Reviewer key。

禁止把上述任何 Token、device key、Codex OAuth 文件放进 Git 仓库或同步 Vault。

## 2. 组织与授权模型

资源边界是：

```text
Team
└── Project
    ├── ProjectMember（角色、业务域、容量）
    ├── Repository
    ├── Issue
    ├── WorkItem
    ├── Assignment
    ├── Document
    ├── Component
    ├── PolicyBundle
    └── Artifact / CorrelationLink
```

团队角色：

- `owner`：团队所有者，可授予团队管理员或其他所有者。
- `admin`：团队管理员，可创建用户并管理普通成员。
- `member`：普通团队成员。

项目角色：

| 角色 | 主要权限 |
|---|---|
| `owner` | 项目内全部权限，包括成员、仓库和策略管理 |
| `maintainer` | 项目内全部资源操作 |
| `developer` | Issue、WorkItem、Artifact、Document、Component 的开发写操作；只能自指派并修改本人负责的 Issue/WorkItem |
| `reviewer` | 项目读取、Issue 状态评审、证据和文档登记 |
| `viewer` | 项目内只读 |

Gateway 先从个人 Token 解析 `user_id`，再由服务器根据已保存资源解析 `project_id` 和 RBAC。客户端传入的项目、仓库或资源 ID 不能自行扩大权限。

Assignment 和资源状态还受所有权约束：成员给自己建立 owner/contributor/reviewer Assignment 只需要对应资源写权限；把 Issue 或 WorkItem 指派给别人必须具备 `project.manage`（项目 `owner`/`maintainer`）。只要资源已有 active owner，同项目的其他 developer 就不能改变其状态；只有该 owner 或项目管理者可以操作。开发任务的 `freeze`、`revise/repair`、`enqueue`、`link-pr` 同样只允许 task assignee 或项目管理者；最终 `accept`、`cancel` 必须由具备 `project.manage` 的操作者完成，并额外通过 `task_accept`/`task_cancel` Governance 身份、理由和职责分离校验。

User 与个人 Token 是整个 TeamControl 存储中的全局对象，不属于某一个 Team。目标用户加入多个活动团队后，签发、列出、撤销其 Token 或改变其全局用户状态的操作者，必须同时是该用户**所有**活动团队的 active owner/admin；只管理其中一个团队不足以修改跨团队身份。普通成员当前没有 Token 自助管理入口。`team.user.create --team ...` 只授权创建全局用户，不会自动建立团队成员关系，仍须随后执行 `team member-add`。

TeamControl 启用后，Gateway 对方法采用 deny-by-default。旧的 process-global `config.*`、日志、渠道、会话、`browser.*` 和 `cron.*` 方法默认禁用；Harness、Memory Catalog 与 Ouroboros 则按它们实际绑定或解析出的项目执行 RBAC。项目路由和 Gateway 连接 Token 都不能绕过这一层。

TeamControl 的 RBAC 不替代治理层的职责分离。例如任务创建人、执行人、四类评审人和最终验收人是否可以重合，仍由 Governance、Orchestrator Lite 和 DoneGate 单独判断。

## 3. 业务域与容量

`ProjectMembership` 同时记录：

- `business_domains`：成员负责的业务板块，如 `billing`、`crm`、`data-platform`。
- `capacity_points`：一个迭代内可承诺的规划点数。
- `role`：权限，不等同于业务所有权。

`WorkItem` 记录 `business_domain` 与 `estimate_points`。Obsidian 团队页把活动任务点数与成员规划容量聚合为负载，但它只是当前中央状态的投影，不是独立排期器。

建议为 10 人建立明确而可重叠的职责表：

| 成员 | 示例项目角色 | 业务域 | 容量点 |
|---|---|---|---:|
| tech-lead | `owner` | architecture | 8 |
| maintainer | `maintainer` | platform | 10 |
| dev-01 ～ dev-06 | `developer` | 各自业务域 | 各 8～12 |
| qa-01 | `developer`（如需运行 Runner） | quality | 10 |
| release-01 | `developer`（如需运行 Runner） | release | 8 |

注册 Runner 需要 `work_item.write`，因此预期执行 Codex 任务的成员必须是项目 `owner`、`maintainer` 或 `developer`。只做评审的成员可使用 `reviewer`，只查看看板的成员使用 `viewer`。项目执行角色不替代 Governance Reviewer 身份与最终验收职责分离。

容量是规划信息，不会自动替代人类指派。出现超载、阻塞或无人负责时，由项目 owner/maintainer 调整 Assignment 或拆分 WorkItem。

## 4. Issue、WorkItem 与 Assignment

三者职责不同：

- `Issue` 表示 Bug、任务、改进或风险，保存严重度、优先级、复现、SLA、重复/回归关系和修复证据。
- `WorkItem` 是可执行增量，保存业务域、估算、依赖、组件、允许的验证命令和可执行说明。
- `Assignment` 把 Issue 或 WorkItem 以 `owner`、`contributor`、`reviewer` 角色绑定到具体项目成员。

Issue 状态机：

```text
new → triaged → assigned → in_progress → verifying → resolved → closed
         │            │          │             └──────────────→ reopened
         └────────────┴──────────┴→ blocked
```

WorkItem 状态机：

```text
pending → ready → in_progress → verifying → done
              └──────────────→ blocked
```

有未完成依赖的 WorkItem 不能进入 `ready` 或 `in_progress`。Issue 进入 `resolved`/`closed` 时必须有 resolution，或已经关联 commit、PR、CI、release 等修复证据。

一个稳定的拆解原则是：一个 WorkItem 只产生一个可独立验证的增量，并同时指定项目、仓库、业务域、负责人、基线提交、策略哈希、允许路径和验证命令。系统进一步强制**一个 WorkItem 最多绑定一个开发任务**；需要拆成两个独立任务时，必须先拆成两个 WorkItem。Issue 是聚合对象，可以由多个 WorkItem/开发任务共享。

冻结 revision 入队前，每个关联 WorkItem 必须**恰有一个**状态为 active、角色为 owner 的 Assignment，而且该 owner 必须等于冻结任务的 `assignee_id`。零个 owner、多个 owner 或 owner 与 assignee 不一致都会被服务器拒绝；这让“谁负责这个增量”和“哪台成员 Runner 可以执行”成为同一条可审计约束。

共享 Issue 不随某一个任务过早结束。只有全部关联开发任务均为 `done`，且全部 WorkItem 均为 `done` 时，最后一次成功验收才把 Issue 从 `verifying` 聚合为 `resolved`。`cancelled` 永不算作成功终态：取消一个开发任务会取消该任务绑定的 WorkItem，但不会自动 resolve 或终态 cancel Issue；Issue 保持 `new`/`triaged`/`assigned`/`verifying`，或从执行态回到 `blocked`，等待重新拆解、指派或由有权人员给出明确 resolution 后另行关闭。

## 5. 增量代码与任务的完整关联

TeamControl 提供 Artifact Registry 和 CorrelationLink，可表达下列链路：

```text
Issue
  → WorkItem
  → Runner Task / Run
  → Diff + Evidence + Trace
  → Commit
  → Pull Request
  → CI
  → Release
  → Regression Case
```

执行包原生携带：

- `task_id`、`task_revision`、`project_id`、`correlation_id`
- `issue_ids`、`work_item_ids`
- `repository_id`、`base_commit`、`branch`
- `harness_version`、`policy_pack_version`、`policy_bundle_hash`
- `allowed_paths`、`denied_paths`、冻结的验证 argv

Runner 自动生成并签名 diff、变更文件、验证结果、Codex 事件与 EvidenceBundle。验收后的外部 commit/PR 可通过 `dev link-pr` 做内容与谱系校验并自动登记 Artifact/CorrelationLink；CI、release、regression case 仍由人工或外部适配器登记。

冻结的 Orchestrator Lite 任务使用：

```bash
goclaw dev enqueue TASK_ID \
  --priority 10 \
  --capability codex \
  --max-attempts 3
```

Gateway 从服务器端冻结任务重新构造 ExecutionPack，拒绝客户端伪造执行包，并校验 assignee、项目成员、Issue/WorkItem、唯一 active owner、仓库、base commit 与 PolicyBundle hash。一个不可变 revision 的队列 ID 固定为 `<TASK_ID>-r<REVISION>`；入队幂等键由服务器按 task ID、revision 和 execution bundle hash 派生。`goclaw dev enqueue` 不接收 `--idempotency-key`，客户端不能换 ID 或幂等键把同一 revision 重复排成多个任务。

Runner 完成后不是直接宣告开发完成。Gateway 先验证 device-key HMAC 与 ExecutionPack，再把 EvidenceBundle 导入对应 Orchestrator Lite 任务；Orchestrator Lite 重新绑定 revision 与 execution bundle、检查 base/head、diff SHA 和路径集合，重算范围策略与冻结验证结果，按冻结 DoneGate 执行独立模型审查，并生成自己的 EvidencePackage/DoneGate。门通过且要求人工验收时，任务进入 `awaiting_acceptance`，WorkItem/Issue 进入 `verifying`；门失败则进入修复、阻塞或失败路径。

最终由同时具备项目 `project.manage` 和 `task_accept` Governance 角色的人调用 `dev.task.accept`。验收会再次校验证据未漂移；成功后把当前任务的关联 WorkItem 置为 `done`。只有共享 Issue 的全部关联任务与 WorkItem 都为 `done`，Issue 才进入 `resolved` 并写入聚合 resolution；只要存在 pending、active 或 cancelled 分支，就保持可追踪的 open/verifying/blocked 状态。签名 Evidence、Task、WorkItem、Issue 与 Repository 会自动关联。

开发者随后在正常 Git 工作流中把 accepted Workstation patch 应用到从 frozen base 派生的分支，提交并在托管平台创建 PR。中央受管仓库的 `local_path` 能解析该 commit 后执行：

```bash
goclaw dev link-pr TASK_ID \
  --commit <COMMIT_SHA> \
  --url https://git.example.com/product/alpha-api/pulls/123
```

服务会确认任务已验收且证据来自 Workstation，commit 存在于受管 `local_path`、继承 frozen base，`base..commit` 的 binary diff 与 accepted patch 精确一致（比较时只忽略 Git `index ...` 元数据行），并校验所有必需 trailers。通过后记录 Orchestrator Lite 的 commit/PR 身份，自动创建 TeamControl commit/PR Artifact，并建立队列 Task→commit、Task→PR、commit→PR、commit→Repository、commit→WorkItem、PR→Issue 的 CorrelationLink。相同 commit/URL 重试是幂等的，已绑定为其他值时拒绝覆盖。

`link-pr` 路径的提交信息必须保留以下 trailers：

```text
Task-ID: task-123
Project-ID: project-alpha
Task-Revision: 2
Repository-ID: repo-api
Correlation-ID: corr-123
Policy-Bundle: <POLICY_BUNDLE_HASH>
Work-Item: work-456
Issue: issue-789
Wave-ID: FE-W03
Wave-Revision: r001
Wave-Step: FE-W03-S04
```

`Repository-ID`、`Correlation-ID` 和 `Policy-Bundle` 在冻结任务对应字段非空时必需；所有 WorkItem 和 Issue 各需要一行。`link-pr` 不会 fetch、push、创建、批准或合并 PR，也不会调用 GitHub/GitLab 等 provider API。它只校验中央本地可见的 Git commit；PR URL 只接受无用户名密码、无 query、无 fragment 的绝对 HTTP(S) 地址并登记，不能证明远端 PR head、内容或状态。因此开发者/CI 必须先 push、创建 PR，再由 CI 或管理员把 commit fetch 到中央 `local_path`。

`Wave-ID`、`Wave-Revision` 和 `Wave-Step` 是仓库治理关联。当前
`dev link-pr` 不验证这三个扩展 trailer；Reviewer 必须按 Wave 注册表、冻结
计划和 journal 进行人工核对，直到单独的治理 Wave 实现服务端强制。

当前仍没有 GitHub/GitLab/Jira 双向适配器。commit/PR 的**校验与本地关联登记**已经实现，但远端对象创建、状态回写、webhook 对账、CI/release/regression 登记仍需人工或另行部署适配器；不能把 `external_issue_id` 或一次 `link-pr` 当成已完成平台同步。

### 修订、修复与队列取消

团队模式不在中央进程直接运行 `repair` Hand。`goclaw dev repair TASK_ID --reason ...` 与带 replacement 的 `goclaw dev revise` 都会创建新 revision：

1. CLI 或 Obsidian 先读取任务当前 revision，并在 `dev.task.revise` 中发送 `expected_revision`；原始 RPC 调用方必须自己提供该字段。
2. Gateway 用 Compare-And-Swap 拒绝过期 revision，避免同一已构造请求被重复应用。
3. 当前 revision 对应队列若仍为 `queued`，先执行 `goclaw runner cancel <TASK_ID-rREVISION> --reason ...`；`failed` 也可取消。`leased` 表示仍有活动执行，取消和 revise 都会拒绝，必须等待 complete/fail 或租约恢复。`completed`/`cancelled` 队列已不活动，无需再取消。
4. revise/repair 会先把 `in_progress` 或 `verifying` WorkItem 迁回 `blocked`，再把任务 revision 加一并清空四审、冻结和旧证据。
5. 新 revision 必须重新完成 Scenario/Capacity/Risk/Cost 四审和 freeze，随后才能 enqueue；入队才把 `blocked` WorkItem 重新推进到 `in_progress`。

不要把 `dev cancel` 与 `runner cancel` 混淆：前者是具备 `project.manage + task_cancel` 身份的人取消整个开发任务；后者只撤销尚未执行的 queued/failed 队列项，并保留开发任务供 revise/repair。

### 写操作的重试契约

| 操作 | 安全重试语义 |
|---|---|
| `dev create` | Team 模式必须先分配稳定 `--id`；完全相同的创建请求返回已有任务且不重复事件，同 ID 不同请求冲突 |
| `dev review` | 相同 kind、decision、Reviewer、理由、反方论点和证据引用返回当前任务且不重复事件；字段变化视为新的评审决定 |
| `dev freeze` | 已冻结时返回同一 revision 和 bundle hash，不重复冻结事件 |
| `dev enqueue` | queue ID 与幂等键由服务器从 revision + bundle 派生；同 revision 重试收敛到同一队列项 |
| `dev accept` | 已验收任务返回 done，并继续以幂等状态迁移补齐 WorkItem/Issue 聚合，不重复任务验收 |
| `dev cancel` | 已取消任务返回 cancelled，并收敛队列和资源状态，不重复任务取消 |
| `dev link-pr` | 相同 commit/URL 返回原绑定并幂等补齐 Artifact/Link；不同 commit 或 URL 拒绝覆盖 |
| `dev revise` / Team `dev repair` | `expected_revision` 是 CAS；同一 RPC 参数再次送达会因 revision 已前进而冲突，不会再加一。CLI/Obsidian 会先读取后发送；遇到响应不明时应先 `dev show` 核对 revision，再决定是否发起新的修订 |

## 6. 统一代码风格与策略层级

策略按固定顺序解析：

```text
team → project → repository → component
```

每一层是带版本和 SHA-256 的 `PolicyBundle`。后层只覆盖同名规则，解析结果也生成稳定哈希。任务冻结时应把该哈希写入执行包，Runner 证据再原样回传，避免“开始执行后规则悄悄变化”。

适合进入策略包的内容包括：

- 格式化、静态检查、测试和构建命令。
- 允许/禁止路径、生成代码边界和依赖政策。
- API 兼容性、错误处理、日志和安全基线。
- 必须关联的文档、回归用例和证据类型。
- 共享组件优先、废弃周期和 owner 要求。

`policy.status` 当前能展示生效哈希和策略层。它不是外部仓库的持续合规扫描器；真实合规仍以冻结执行包、确定性验证、EvidencePackage 和 CI 证据为准。

## 7. 组件复用

Component Registry 保存：

- 项目与仓库归属。
- 组件类型和根路径。
- owner、依赖组件和元数据。
- 与 WorkItem、Issue、文档、构建和发布的关联。

创建任务前先查询组件目录，再决定扩展现有组件还是新建组件。新共享组件至少应有：

1. 明确 owner。
2. 稳定的兼容性契约。
3. 测试与使用示例。
4. 版本和废弃策略。
5. 对应 ADR/API/Runbook 文档。

Registry 记录“可发现与可追踪”，不会自动判断两段代码在语义上是否重复，也不会自动把复制代码重构为共享库。

## 8. 统一文档目录

Vault 可以同步 Markdown，但中央 Document Registry 才保存项目内文档身份、状态、owner、revision、校验和和替代关系。建议目录：

```text
ObsidianVault/
├── 00-index/
│   └── project-alpha/
├── 01-goals/
│   └── project-alpha/
├── 02-decisions/
│   └── project-alpha/        # ADR
├── 03-constraints/
│   └── project-alpha/
├── 04-requirements/
│   └── project-alpha/        # PRD
├── 05-knowledge/
│   └── project-alpha/        # Design / API / knowledge
├── 06-test-plans/
│   └── project-alpha/
├── 07-runbooks/
│   └── project-alpha/
├── 08-reviews/
│   ├── inbox/
│   ├── approved/
│   └── rejected/
└── 09-releases/
    └── project-alpha/
```

文档从 `draft` 进入 `active`；新版本通过 `supersedes` 替代旧版本，旧版本变为 `superseded`。Document Registry 保存 URI 和校验和，不代替 Git/Vault 的文件版本历史。

分步更新计划统一放在仓库
[`docs/waves`](waves/README.md)，最少包含 registry、修订化 plan、append-only
journal、Issue register、decision log 和 evidence index。Wave 文档负责说明
“为什么、按什么顺序、什么证据允许前进”；TeamControl/Orchestrator Lite
仍分别是任务状态、执行与 DoneGate 的权威来源，不能用 Markdown 手工状态
替代中央状态机。

## 9. Obsidian 的角色

Obsidian Project Console 是投影与交互窗口：

- “团队”：成员负载、任务、Bug、Runner/租约、策略、文档和组件概览。
- “聊天”：项目话题会话。
- “记忆”：受治理项目知识。
- “规格/审批/开发”：Ouroboros 与 Orchestrator Lite 的人工控制面。
- “进度/Harness”：Trace 和 Harness 状态。

团队页用当前 `project_id` 调用七个只读 RPC：

| 模块 | RPC |
|---|---|
| 成员负载 | `team.members` |
| 项目任务 | `work.items` |
| Bug/Issue | `issue.list` |
| Runner/lease | `runner.list` |
| 策略层 | `policy.status` |
| 文档目录 | `docs.summary` |
| 组件目录 | `components.summary` |

服务器先校验项目成员和读权限；任何模块失败只影响该模块，不会把其他模块一起清空。

Vault Sync、Syncthing 或 Git 只负责 Markdown 副本。不得通过 Vault 同步：

- TeamControl 状态或 Runner 队列。
- JSONL、SQLite、锁、租约、Trace 和 worktree。
- `GOCLAW_USER_TOKEN`、Reviewer Token、device key 或 Codex OAuth。

如果两台电脑同时打开同一 Vault，它们看到的是中央状态的两个投影，不是两个控制面。

## 10. 飞书项目路由

飞书消息按下列优先级映射项目：

```text
channel:account_id:chat_id
channel:chat_id
channel
default project
```

示例：

```json
{
  "harness": {
    "project_id": "default",
    "routes": {
      "feishu:tenant-bot:oc_alpha": "project-alpha",
      "feishu:oc_beta": "project-beta",
      "feishu": "default"
    }
  }
}
```

路由决定消息进入哪个 `project_id`，不是授权凭据。高风险审批和团队管理仍应使用绑定个人身份的 CLI 或 Obsidian Reviewer 凭据。启用 Harness 后，会话键为 `project:<project_id>:<topic_id>`，因此飞书与 Obsidian 可以在同一项目话题下共享历史。

## 11. 十人团队首次落地

`0.7.0` 的团队管理 CLI：

| 命令 | 必填参数 | 用途 |
|---|---|---|
| `goclaw team bootstrap` | `--user --name --token-file` | 本地创建首个管理员；可选 `--root --email --label` |
| `goclaw team create` | `--id --name` | 创建团队，调用者成为 owner |
| `goclaw team user-create` | `--team --id --name` | 创建用户 |
| `goclaw team member-add` | `--team --user` | 授予 `owner/admin/member` 团队角色 |
| `goclaw team token-issue` | `--user --token-file` | 签发个人 Token；可选 RFC3339 `--expires` |
| `goclaw team project-create` | `--team --id --key --name` | 创建项目 |
| `goclaw team project-member-add` | `--project --user` | 设置项目角色，可重复 `--domain` 并设置 `--capacity` |
| `goclaw team repository-create` | `--project --id --name` | 登记 remote/local path 与默认分支 |

Issue、WorkItem、Assignment、Artifact、Document、Component 和 Policy 的完整操作目前通过项目授权的 Gateway RPC；CLI 只覆盖首次团队/项目落地和工作站生命周期。

第一个管理员在中央主机执行：

```bash
goclaw team bootstrap \
  --user admin \
  --name "Team Admin" \
  --email admin@example.com \
  --token-file /secure/goclaw/admin.token

export GOCLAW_USER_TOKEN="$(cat /secure/goclaw/admin.token)"
```

在另一终端或服务管理器中启动中央 Gateway：

```bash
goclaw gateway run
```

后续 team 命令均经 Gateway + 个人 Token：

```bash
goclaw team create \
  --id team-product \
  --name "Product Engineering"

goclaw team project-create \
  --team team-product \
  --id project-alpha \
  --key ALPHA \
  --name "Project Alpha"

goclaw team repository-create \
  --project project-alpha \
  --id repo-api \
  --name "Alpha API" \
  --remote https://git.example.com/product/alpha-api.git \
  --branch main
```

为每位成员创建用户、签发一次性明文 Token、加入团队和项目：

```bash
goclaw team user-create \
  --team team-product \
  --id dev-01 \
  --name "Developer 01" \
  --email dev01@example.com

goclaw team member-add \
  --team team-product \
  --user dev-01 \
  --role member

goclaw team token-issue \
  --user dev-01 \
  --label laptop \
  --token-file /secure/goclaw/dev-01.token

goclaw team project-member-add \
  --project project-alpha \
  --user dev-01 \
  --role developer \
  --domain billing \
  --capacity 10
```

`user-create → member-add → token-issue` 的顺序不能颠倒：目标用户至少要有一个 active team membership，且签发人必须同时管理其所有活动团队。重复以上步骤直到 10 人全部完成。Token 文件创建使用排他写入和 `0600` 权限；把明文安全交给对应成员后，不要集中共享。`bootstrap --root` 如有指定，必须与 Gateway 配置的 `team_control.root` 完全一致。

每位成员随后在自己的电脑上完成 Runner 注册和 `codex login`，具体命令见 [`WORKSTATION_RUNNER_CN.md`](WORKSTATION_RUNNER_CN.md)。

## 12. 日常闭环

一次标准开发增量：

1. 在 Wave Issue register 记录报告；尚未复现时只能标记 `reported` 或 `unverified`。
2. 建立或修订 Wave plan，写清依赖、允许范围、步骤、验证、风险、回滚和 Exit Gate；在 registry 激活后才能进入其授权步骤。
3. 在中央控制面登记已复现的 Issue；Bug 必须包含复现、预期、实际和严重度，并关联 `Wave-ID` 与计划修订。
4. 拆成一个或多个 WorkItem，绑定业务域、组件、依赖、估算和验证命令。
5. 用 Assignment 指定 owner/contributor/reviewer，并检查成员容量；非本人指派由项目管理者执行，每个准备入队的 WorkItem 保持且仅保持一个 active owner。
6. 为开发增量预分配稳定 `TASK_ID`，执行 `goclaw dev create --id "$TASK_ID" ...`；同一 WorkItem 不得出现在第二个开发任务中，项目列表始终用 `goclaw dev list --project PROJECT_ID`。
7. 解析 Team → Project → Repository → Component 策略，完成四审并冻结 base commit、策略哈希以及 Wave/Issue/Step 关联。
8. 用 `goclaw dev enqueue` 让服务器从冻结任务构造 secret-free ExecutionPack；该 revision 使用服务器派生的唯一队列 ID/幂等键，只有 Runner owner 与任务 assignee 相同且项目/capability 匹配时才能领取租约。
9. 本地 Codex 在独立 worktree 修改代码；Runner 维持租约心跳并执行冻结验证。
10. Runner 回传签名 diff/EvidenceBundle；Gateway 导入 Orchestrator Lite，重新验证并运行 DoneGate，人类评审其 EvidencePackage。
11. 若需 repair/revise，先更新 Wave journal；实质改变范围或门禁时先创建新 plan revision。随后取消仍 queued/failed 的旧任务 revision；leased 必须等待。CLI/Web Console 用 `expected_revision` 做 CAS，WorkItem 先 blocked，新 revision 重新四审/freeze/enqueue。
12. 具备 `project.manage + task_accept` 的最终 Reviewer 调用 `dev.task.accept`，让当前 WorkItem 进入 done；共享 Issue 只在全部关联 Task=Done 且 WorkItem=Done 后自动 resolved，任何 cancelled 分支都要求重新安排或明确另行关闭。
13. 开发者从 frozen base 应用 accepted patch，提交并 push/create PR；CI 或管理员让中央 `local_path` 可见 commit，再由 assignee/项目管理者执行 `goclaw dev link-pr TASK_ID --commit <SHA> --url <PR_URL>` 自动登记 commit/PR Artifact 与关联。
14. 外部平台完成人工/既有策略的 PR 审批与 merge；登记 CI、release 和 regression case，连接剩余 CorrelationLink。
15. 把测试、审查、发布与回滚演练结果写入 Wave evidence index 和 journal；没有证据不得推进 Wave 状态。
16. 发布验收后关闭已 resolved 的 Issue；复发时通过 `regression_of` 或 `reopened` 保留历史。
17. 同步更新 ADR、API、Test Plan、Runbook 与 Component Registry。

## 13. 当前真实边界

- TeamControl 与 Workstation 都是文件存储；进程内有锁和原子替换，但仅支持一个 GoClaw 进程安全写入。多个进程或共享盘同时写同一目录不受支持。
- 没有多 Leader 共识、数据库级事务集群或自动故障转移。
- 没有 GitHub/GitLab/Jira 双向同步；外部对象的创建、状态回写和 webhook 对账尚未实现。
- Runner 不会自动 commit、push、创建/合并 PR 或发布；检测到 Codex 自动 commit 会把执行判为失败。
- `dev link-pr` 只验证中央 `local_path` 已可见的 commit 与 accepted patch，并登记本地 Artifact/CorrelationLink；不会 fetch、push、创建、批准、merge PR 或等待 CI。
- Team 模式的 `dev.task.run/repair/resume` 无条件禁用，即使 `development.gateway_allow_execution=true` 也不能调用；该开关仅适用于未启用 TeamControl 的单用户模式。团队只能通过 `dev enqueue` → Workstation 持久队列执行。
- 每个 Runner 同时只持有一个活动 lease；device key 仅在 Runner 空闲时允许轮换。
- 冻结任务的 `assignee_id` 会强制匹配 Runner owner；`business_domain` 和容量当前用于计划、校验与看板，不参与自动排程优化。
- Codex 主进程只继承最小工具链环境，并通过显式 `CODEX_HOME` 使用本机订阅 OAuth；模型命令的 named permission profile 对真实目录设置 `deny`，每次模型调用前还必须通过 read-deny canary。GoClaw/Reviewer/Runner/Codex Token、SSH agent、Docker/Kubernetes socket/context、云凭据路径等宿主能力变量永久拒绝，`--allow-env` 也不能放行。
- 冻结 verifier 不再直接运行于宿主环境。`runner work` 必须指定绝对、受审、不可由 Runner 用户篡改的 `--verification-sandbox`；Linux 基线 wrapper 使用 bubblewrap 断网、遮蔽 host home/run/tmp，并只给 worktree 与临时 HOME 写权限。只有整个 Runner 已运行在一次性隔离 VM/容器时，才能显式使用与前者互斥的 `--unsafe-host-verification`。
- Production 必须使用专用用户并安装 root-owned `0755` verifier wrapper；VM/容器仍是 Codex Hand 和整体 Runner 的纵深隔离。device key 是控制面与工作站共享的 HMAC 秘密，不是 TPM attestation、公钥设备证书或不可抵赖签名；中央凭据泄露者能够伪造该 Runner 的证据。
- 组件目录不会自动发现重复实现；文档目录不会自动证明内容正确。
- Wave 门禁目前由仓库政策与 Reviewer 执行；服务端尚不校验 active Wave、plan revision、Step 或计划文件 SHA。要实现失败关闭的运行时门禁，必须另立治理 Wave 并迁移现有任务契约。
- Obsidian 团队页是读取投影，不是队列或秘密存储。
- 当前环境中的浏览器视觉 QA 因安全策略拒绝本地地址访问而未完成；插件已通过 TypeScript 构建与单元测试，但发布前仍应在真实 Obsidian Desktop 做一次人工视觉和交互验收。

部署分层、备份和验收见 [`DEPLOYMENT_CN.md`](DEPLOYMENT_CN.md)。
