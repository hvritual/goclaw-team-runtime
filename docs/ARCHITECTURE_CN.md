# 架构与数据权威

版本：GoClaw Team Runtime `0.7.0`。

`0.7.0` 的入口权威关系是：Team Web Console 为默认人工控制面，飞书为低权限消息入口，CLI 为运维/调试入口，Obsidian 为可选 Markdown 编辑适配器。任何入口都不保存 TeamControl、Workstation 或 Governance 的权威状态。

## 总体结构

```text
 Team Web Console ┐
 飞书机器人 ──────┼── WSS/HTTPS ── Gateway ── TeamControl
 管理 CLI ────────┘                  │          中央单写者
                                   │
        ┌──────────────────────────┼──────────────────────────┐
        │                          │                          │
 团队/项目/仓库/RBAC       Issue/Work/Assignment      Policy/Docs/Components
        │                          │                          │
        └──────────────────────────┼──────────────────────────┘
                                   │
                         Workstation durable queue
                         task / lease / evidence
            ┌──────────────────────┼──────────────────────┐
            ▼                      ▼                      ▼
       Runner A                Runner B              Runner J
       device key             device key            device key
       Local Codex OAuth      Local Codex OAuth     Local Codex OAuth
       Git worktree           Git worktree          Git worktree

 中央 GoClaw 同时托管：
 Sessions / Catalog / Ouroboros / Harness / Orchestrator Lite / Trace
                                   │
                                   ▼
                          Governed Markdown
                       filesystem 或 Git 工作树
```

TeamControl 与 Gateway 共同组成中央控制面：Gateway 负责连接认证、个人 Token 解析和 RPC；TeamControl 负责服务器端项目授权与业务状态。唯一例外是空状态上的首次 `goclaw team bootstrap`，它在中央主机本地直接创建首个管理员。Workstation 服务同样由中央进程单写入，成员 Runner 不能直接修改 TeamControl 文件或队列目录。

## 权威边界

| 数据 | 权威源 | 是否同步到多台电脑 | 写入者 |
|---|---|---:|---|
| 用户、团队、成员关系和个人 Token 摘要 | TeamControl | 否 | 中央 GoClaw；首个 admin bootstrap，后续受 RBAC 约束 |
| 项目、仓库、业务域、容量和 RBAC | TeamControl | 否 | 中央 GoClaw |
| Issue、WorkItem、Assignment | TeamControl | 否 | 中央 GoClaw |
| Artifact、CorrelationLink、Document、Component、PolicyBundle | TeamControl | 否 | 中央 GoClaw |
| Runner 公共登记、device key 凭据、任务、租约和签名证据 | Workstation Runtime | 否 | 中央 GoClaw；Runner 仅经 Gateway 协议提交 |
| 工作站 checkout、worktree、Codex 事件和本地证据 | 各成员电脑 | 否 | 对应 Runner |
| 目标、ADR、约束、需求、知识正文 | Governed Markdown Root | 视文件/Git策略 | 人类审批器 |
| 记忆目录、版本、来源、权威项、流通记录 | Memory Catalog SQLite | 否 | GoClaw；人工批准才进入 active |
| 内容发现索引 | builtin SQLite / QMD | 否，可重建 | GoClaw |
| 项目会话 | GoClaw Sessions JSONL | 否 | GoClaw |
| 规格访谈与状态 | Ouroboros `events.jsonl` | 否 | GoClaw 单写入服务 |
| Seed 与演化候选 | Ouroboros 内容寻址 JSON | 否 | GoClaw；人工批准才切换 active |
| Trace | Harness Runtime | 否 | GoClaw |
| Harness 版本 | Harness Registry | 否 | 实验提升流程 |
| Harness Candidate | Candidate 隔离目录 | 否 | 人类或开发代理 |
| 开发任务状态 | Orchestrator Lite `events.jsonl` | 否 | GoClaw 单写入服务 |
| 开发任务投影 | Orchestrator Lite `task.json` | 否 | 从 SessionEvent 派生 |
| 开发证据 | Task Run EvidencePackage | 否 | Codex Hand + Go Checker |
| 开发代码 | 中央或工作站独立 Git worktree/branch | 否 | Codex Hand/Runner 只改工作树；验收后由人类显式提交 |
| Team Web Console 看板 | Gateway 读取投影 | 否；刷新即可重建 | Web Console 只渲染 |
| UI 项目/Topic 设置 | 浏览器 React 内存 | 否 | 用户 |
| Gateway/Team Token | HttpOnly 短期会话建立后立即丢弃 | 否 | Gateway |
| 个人 `GOCLAW_USER_TOKEN` | 各成员秘密管理器 | 否 | 对应成员 |
| Runner device key | 对应成员电脑的 `0600` 文件 | 否 | Runner 注册流程 |
| Reviewer Token | Web Console 页面内存 / CLI 环境 | 否 | 各审批人 |
| ChatGPT 凭据 | Codex CLI 自有存储 | 否 | Codex CLI |

## 团队控制闭环

```text
Team
  → Project
    → ProjectMember(role, business_domains, capacity_points)
    → Repository
      → Issue
        → WorkItem
          → Assignment
            → ExecutionPack
              → Runner lease
                → Diff / Evidence / Trace
                  → Commit → PR → CI → Release → Regression
```

冻结任务通过服务器端 `dev.task.enqueue` 编译为 ExecutionPack，客户端不能提供受信执行包。每个冻结 revision 使用唯一的 `<TASK_ID>-r<REVISION>` 队列身份，幂等键由服务器按 revision 与 execution bundle 派生，CLI 不接收入队幂等 flag；每个关联 WorkItem 必须恰有一个 active owner，且等于 task assignee。一个 WorkItem 只能属于一个开发任务，Issue 则可以聚合多个任务。入队 Task 会自动连接 WorkItem、Issue 与 Repository。

Runner 完成时，Gateway 验证并登记签名 Evidence，再导入 Orchestrator Lite。后者不信任 Runner 的结论，而是重新绑定 revision/execution bundle，复核 base/head、no-commit、diff SHA、路径、范围和冻结检查，必要时重做独立审查并运行 Go DoneGate。通过后进入 `awaiting_acceptance`；最终 `dev.task.accept` 再校验证据未漂移，把当前 WorkItem 置为 done。共享 Issue 只有在所有关联任务都为 done，且所有关联 WorkItem 都为 done 时才 resolved；cancelled 永不算成功。取消分支让 Issue 保持 open/verifying/blocked，等待重新指派或显式另行关闭。

验收后 commit/PR 由正常 Git/托管平台流程产生，再通过 `goclaw dev link-pr TASK_ID --commit <SHA> --url <PR_URL>` 回链。中央服务在受管 Repository `local_path` 中解析 commit，验证 frozen base 是其祖先、累计 diff 与 accepted Workstation patch 一致（忽略 Git `index` 元数据行）且 trailers 完整，然后自动创建 commit/PR Artifact，以及 Task/Repository/WorkItem/Issue 的 CorrelationLink。PR URL 只接受无凭据、无 query、无 fragment 的绝对 HTTP(S) 地址并登记；没有 provider API，不能证明远端 PR head、内容或状态。独立 Run/Trace、CI、release 与 regression case Artifact 仍需人工或外部适配器登记。模型和本地 link 能力存在不等于已接入 GitHub/Jira 双向同步。

Issue 与 WorkItem 有各自的显式状态机；依赖未完成时 WorkItem 不能开工，Issue 没有 resolution 或关联修复证据时不能关闭。Assignment 将 owner、contributor、reviewer 与具体项目成员绑定。非本人指派需要 `project.manage`；已有 active owner 时，同项目其他 developer 也不能修改该 Issue/WorkItem。开发任务的 freeze/revise/enqueue/link-pr 限于 assignee 或项目管理者，accept/cancel 再叠加 `project.manage` 和 Governance 角色。团队控制详见 [`TEAM_DEVELOPMENT_CN.md`](TEAM_DEVELOPMENT_CN.md)。

修订采用 revision CAS。CLI 与 Obsidian 先读取任务并发送 `expected_revision`；Gateway 拒绝过期请求。若旧 revision 仍 queued，必须先用 `runner cancel` 撤销；leased 不能取消，revise 也会拒绝。repair/revise 会先把 in_progress/verifying WorkItem 退到 blocked，再创建需要重新四审、冻结和入队的新 revision。队列取消只改变 queued/failed Workstation Task，不等于取消整个开发任务。

TeamControl 启用即进入 Gateway deny-by-default 模式：旧的 process-global 配置、日志、渠道、会话、Browser、Cron RPC 被拒绝，未知方法也不会自动放行；聊天/Agent、Harness、Memory Catalog、Ouroboros 和团队对象分别按请求项目、绑定项目或资源所属项目做 RBAC。全局 User/Token 的变更还要求操作者同时管理目标用户参加的所有活动团队。

## 策略、组件与文档

策略解析顺序固定为：

```text
team → project → repository → component
```

每层 PolicyBundle 均带版本和 SHA-256；后层覆盖同名规则。冻结任务把解析后的 policy hash 带入 ExecutionPack，Runner 再把它带回签名证据。

Component Registry 记录仓库、根路径、owner、依赖和元数据，用于“复用优先”和影响分析；它不会自动检测语义重复。Document Registry 记录 PRD、ADR、Design、API、Test Plan、Runbook 等文档的 URI、owner、revision、checksum 和 supersedes 关系；正文仍由 Vault/Git 保存。

## 图书信息管理式记忆闭环

```text
Vault / Agent / Gateway
  → pending 编目候选
  → 来源、校验和、项目、类型、Facet、权威项和关系校验
  → memory_approve 人工评审
  → active（有效期内才可自动进入上下文）
  → 使用/引用反馈 + 定期复核
  → 续期 / 新 Manifestation 替代 / 撤回
```

Catalog 是可信度和生命周期控制面；builtin/QMD 只负责内容发现。Work/Expression 保持概念身份，Manifestation 表示具体版本，Item 对应稳定的 Vault 相对路径来源。完整模型、迁移与备份见 [`LIBRARY_MEMORY_CN.md`](LIBRARY_MEMORY_CN.md)。

## Better Harness 闭环

```text
运行 → 完整 Trace（工具输入、结果、错误）→ 人类反馈/根因与变更清单
  → Candidate 隔离副本 → 基线与 Candidate 的 Optimization/Golden/Holdout Eval
  → 独立人类批准 → 不同角色原子提升 → 新 Trace
                                  └→ 认证回滚
```

实现刻意把“生成 Candidate”与“上线 Candidate”分开。Candidate 只能改 Change Manifest 声明的组件，不能改 Protected Paths 或评测门槛；上线前会拒绝逐用例基线回归、Golden/Holdout 退化，以及显式 Trace 的令牌或延迟超限。自动化不能绕过 Eval、人类批准或活动版本 Compare-And-Swap。

## Orchestrator Lite 闭环

```text
需求编译
  → Goal / Plan / Milestone / WorkItem / EvidencePlan / DoneGate
  → Scenario / Capacity / Risk / Cost 四审（人数和每人类型上限）
  → 固定 Git base commit 与执行包哈希
  → 独立 worktree 中运行 Codex exec
  → Go Scope/Policy/Verification + 独立只读审查
  → EvidencePackage + Go DoneGate
  → 与任务评审者分离的最终验收
  → 本地 Git commit
```

Codex 是受控执行 Hand，不是状态机或最终裁决者。所有模型输出都要进入 Go 侧检查；DoneGate 结论只由 Go 代码计算。新需求必须通过 ChangeIntent 递增 Revision，并重新完成四类评审。

## Go 原生 Ouroboros 闭环

```text
多视角访谈 → Go 歧义评分/分歧/灰区/相关性门 → 冲突解决
  → 不可变 Seed（备选、证伪、参考类、预测、停止条件）
  → 身份认证与法定人数审批 → 单向任务编译
  → Orchestrator Lite 执行与 EvidencePackage
  → 机械门 → 语义门 → 盲化角色多数决/关键发现否决
  → 分歧时人工仅裁决证据 → 记录结果与参考类
  → 后继 Seed 候选 → 连续通过后收敛或人工批准下一代
```

Ouroboros 不能直接调用开发 Hand。它只把人工批准的 active Seed 编译为新的 Orchestrator Lite `CreateRequest`；生成的任务仍从四类评审开始。演化也只写不可变候选，不能修改 active Seed、Harness、Vault、任务或仓库。完整认知控制见 [`GOVERNANCE_CLOSED_LOOP_CN.md`](GOVERNANCE_CLOSED_LOOP_CN.md)，阈值、CLI 和故障恢复见 [`OUROBOROS_GO_CN.md`](OUROBOROS_GO_CN.md)。

## 为什么只有一个中央写入进程

同步工具能解决 Markdown 文件复制，但不能天然解决以下并发对象：

- TeamControl `teamcontrol.json` 的 revision 和原子快照。
- Workstation 队列、幂等 receipt、任务 claim、lease 和 heartbeat。
- 正在增长的 JSONL。
- 活动 Harness 指针。
- 同一个项目话题中的对话顺序。
- Catalog SQLite、审批状态和流通事件。
- 两台控制面同时审批同一个知识修改。
- 同一 ADR 被不同设备基于旧版本覆盖。

因此默认部署只有一个 GoClaw 中央进程。TeamControl 和 Workstation 的 mutex 只提供进程内并发安全，原子文件替换只保证单进程落盘完整性，不提供跨进程锁或多 Leader 共识。多台电脑只是本地 Runner、UI 和 Vault 副本。知识批准再通过 SHA-256 乐观锁处理同步延迟。

## 安全模型

- 对话 Provider 的 Codex app-server 每次调用使用临时 Thread、`read-only` Sandbox 和结构化输出 Schema。
- 开发 Hand 使用独立 Git worktree、`codex exec --json --sandbox workspace-write`、禁用交互式批准，并保存原始 JSONL；独立审查仍为 `read-only`。
- 中央单用户 Hand 在执行前后都要求 `HEAD` 等于冻结 `BaseCommit`，Codex 自动 commit 会失败；每次 Hand 使用隔离 HOME/XDG/runtime/tmp 和最小环境，只通过 `CODEX_HOME` 读取本机订阅 OAuth。冻结 verifier 默认还要求 `development.verification_sandbox` argv wrapper；它与仅供一次性隔离 VM/容器使用的 `unsafe_host_verification` 互斥。
- Codex 在 Provider 层只负责给出 GoClaw 下一步决策；真正的工具执行仍由 GoClaw Registry 控制。
- 开发 Hand 不能提交或推送；Go 侧重新执行冻结的 argv 验证、范围和预算检查。
- Codex 登录态不进入 GoClaw 配置。
- 个人 `GOCLAW_USER_TOKEN` 与 Gateway 连接 Token 分离；项目与资源授权由服务器端 TeamControl 解析。
- 每台 Runner 有独立 device key；控制面保存 HMAC 验证所需的共享凭据，公共 Runner 投影只显示 key ID。device key 不替代个人 Token，也不是 TPM attestation、公钥设备证书或不可抵赖签名；中央凭据泄露可伪造相应 Runner 证据。
- Runner 只执行内容寻址、无秘密的 ExecutionPack，并把 task、lease、attempt、diff、验证和 policy hash 写入签名 EvidenceBundle。
- Runner 先完成全部冻结验证，再对最终工作树计算 changed files/diff、scope policy 与 no-commit，避免验证命令改动文件后证据过时。
- Runner 如果检测到 Codex 自动创建 commit 会失败；commit、push、创建/批准/merge PR、CI 和 release 仍是外部人工或既有平台流程。验收后的 `dev link-pr` 只校验本地可见 commit 并登记关联，不会 fetch 或改变远端平台。
- Runner 的 Codex 子进程从最小环境白名单构造，每次运行使用独立 HOME/XDG/runtime/tmp，只通过 `CODEX_HOME` 读取本机订阅 OAuth。GoClaw Token、SSH agent、Git askpass、Docker/Kubernetes 和云凭据路径等宿主能力变量永久剥离，`--allow-env` 不能覆盖；内部 Git 与冻结 verifier 也不接收该 allowlist。
- `runner work` 默认失败关闭，必须提供绝对、受审且不可由 Runner 用户篡改的 `--verification-sandbox`。Linux 发布包的 bubblewrap wrapper 断网、遮蔽 host home/run/tmp，只让任务 worktree 与临时 HOME 可写；它应安装为 root-owned `0755`。只有整个 Runner 已位于一次性隔离 VM/容器时，才可使用与 wrapper 互斥的 `--unsafe-host-verification`。
- 受控知识目录应加入通用文件工具 `denied_paths`。
- 受控知识只能由三个专用工具读取、检索或创建提案。
- Agent 记忆工具只能检索 active 记录或创建 pending 候选；渠道绑定的项目不能被工具参数覆盖。
- Catalog 正文按不可信引用注入，排除 pending、旧版、撤回、未生效和过期内容；Catalog 目录/数据库分别使用 `0700`/`0600`。
- 通用文件工具使用路径边界和符号链接解析，避免字符串前缀与链接逃逸。
- Gateway Token 放入 WebSocket 子协议 Header，避免出现在 Obsidian 持久配置和普通 URL 日志；HTTP `/rpc` 使用 Bearer Token。
- Reviewer Token 与 Gateway Token 分离；可选 `governance.reviewers.<key>.team_user_id` 把个人 Token 得到的 Team principal 映射到描述性 Reviewer 策略 key，最终审计身份仍是 principal。角色、理由、最强反方论点、证据引用、法定人数和职责分离由 Go Core 校验。
- Harness 提升需要 Eval、独立批准者、不同提升者和 Compare-And-Swap；回滚也需要认证决策。
- Seed 与演化候选的激活需要显式人工批准；高风险 Seed 默认双人批准；聊天渠道没有对应特权工具。
- Ouroboros Event 与 Seed 每次读取都做 SHA-256 完整性校验。
- Gateway 触发长开发任务和命令型 Harness Eval 默认关闭；本地 CLI 是默认执行入口。
- Team 模式的中央 `dev.task.run/repair/resume` 无条件禁用，即使开启 `development.gateway_allow_execution` 也不会放行；该开关仅用于未启用 TeamControl 的单用户部署。团队唯一执行路径是服务器 enqueue、成员本地 Runner 执行。

## Obsidian 与飞书

Obsidian 是 Gateway 状态的投影和人工交互窗口，不是状态数据库。Vault Sync 不承载队列、租约、Token、device key、Codex OAuth 或任何活动 JSONL/SQLite。

飞书路由按 `channel:account:chat → channel:chat → channel → default` 解析 `project_id`。路由只决定项目上下文，不赋予团队管理权限；高风险变更仍需要个人 Token、项目 RBAC 和对应 Reviewer 身份。

## 会话关联

启用 Harness 后，会话键为：

```text
project:<project_id>:<topic_id>
```

渠道不是会话键的一部分。Obsidian、飞书和其他入口只要解析到同一 `project_id + topic_id`，就共享上下文。

该设计也意味着同一项目话题内的参与者能看到共享历史。需要私密讨论时，应使用不同 `topic_id` 或独立项目。
