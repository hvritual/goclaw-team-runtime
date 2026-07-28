# Orchestrator Lite 开发执行手册

Orchestrator Lite 是本仓库内置的单机开发控制面。它不依赖尚未建设的 `go-orchestrator`，但保留了稳定执行开发任务所需的核心约束：

- 先把自然语言请求编译成结构化任务契约。
- 场景、容量、风险、成本四类评审全部通过且满足人数/每人类型上限后才能冻结。
- 每个任务使用独立 Git worktree 和分支。
- Codex CLI 复用当前操作系统用户的 ChatGPT 登录态，不接收 API Key。中央 Hand 每次运行使用独立 HOME/XDG/runtime/tmp 和最小环境，只通过 `CODEX_HOME` 读取该用户已有的订阅 OAuth。
- Go 代码独立执行路径、依赖、行数、测试、令牌预算和证据检查。
- 只有 Go DoneGate 能给出通过结论；Codex 的完成声明不构成验收。
- 与任务评审者分离的人类核对未被修改的证据包后才能验收，再由显式命令创建本地提交。

它适合单个常驻 GoClaw 控制面和多个项目并行使用。Full Runtime 可把冻结 revision 放进持久 Workstation 队列，由多台成员 Runner 执行并回流证据；这仍不是多 Leader 分布式调度器，也不提供 GitHub 自动推送或 PR 创建。

## 1. 数据流与权威边界

```text
自然语言需求
    │
    ▼
Task Contract
Goal / Alternatives / Falsifiers / Predictions / Kill Conditions
Plan / Milestone / WorkItem / EvidencePlan / DoneGate
    │
    ├── Scenario Review ─┐
    ├── Capacity Review  ├─ Reviewer 身份与职责分离
    ├── Risk Review      │
    └── Cost Review ─────┘
             │ 全部 approved
             ▼
冻结执行包 + Git Base Commit + SHA-256
             │
             ▼
中央 Hand，或唯一 revision 队列 ID → 成员 Runner
             │
             ▼
隔离 Worktree ── Codex exec --json --sandbox workspace-write
             │
             ▼
Go Policy + 确定性验证 + Falsifier/Prediction/Kill Check + 独立只读审查
             │
             ▼
本地 EvidencePackage
或签名 EvidenceBundle → 导入重验 → EvidencePackage + Go DoneGate
             │
             ▼
不同 Reviewer 最终验收
    ├─ 中央 Hand：显式本地 Git Commit
    └─ Workstation：外部 Commit/PR → link-pr 校验与关联
```

`tasks/<task-id>/events.jsonl` 是任务状态的唯一事实源。每条 `SessionEvent` 包含序号、前序哈希、事件数据和任务快照；读取时会验证完整哈希链。`task.json` 只是便于查看的投影，不参与状态判定。

哈希链能检测文件损坏和未同步重写，但它不是外部签名。拥有运行时目录写权限的攻击者仍可能重算整条链；生产环境应再增加签名、不可变备份和操作系统访问控制。

## 2. 运行目录

配置示例：

```json
{
  "development": {
    "enabled": true,
    "root": "/srv/goclaw-runtime/development",
    "worktree_root": "/srv/goclaw-runtime/worktrees",
    "repo_path": "/srv/repos/project-alpha",
    "codex_command": "codex",
    "codex_model": "",
    "run_timeout_seconds": 21600,
    "verify_timeout_seconds": 1800,
    "verification_sandbox": [
      "/usr/local/libexec/goclaw/verify-sandbox-bwrap.sh"
    ],
    "unsafe_host_verification": false,
    "max_repair_attempts": 2,
    "default_max_changed_files": 40,
    "default_max_changed_lines": 2000,
    "allow_dirty_repo": false,
    "independent_review": true,
    "gateway_allow_execution": false,
    "require_human_final_approval": true
  }
}
```

`verification_sandbox` 是中央单用户 Hand 执行冻结验证命令时使用的 argv 前缀；首项必须是绝对、普通、可执行且不可被 group/other 写入的 wrapper。GoClaw 会追加 `WORKTREE SANDBOX_HOME -- COMMAND...`。Linux 可使用发布包内的 `scripts/verify-sandbox-bwrap.sh`，安装为 root-owned `0755` 后引用。`verification_sandbox` 与 `unsafe_host_verification` 互斥；默认必须配置前者。只有整个中央 Runner/Hand 已位于一次性隔离 VM/容器时，才可将数组留空并显式设 `unsafe_host_verification=true`。

运行时结构：

```text
development/
├── tasks/
│   └── task-.../
│       ├── events.jsonl
│       ├── task.json
│       └── runs/
│           └── run-.../
│               ├── task-snapshot.json
│               ├── repository-before.json
│               ├── repository-after.json
│               ├── codex-events.jsonl
│               ├── codex-result.json
│               ├── codex-final.md
│               ├── diff.patch
│               ├── policy.json
│               ├── verification.json
│               ├── independent-review.json
│               ├── evidence.json
│               └── donegate.json
├── locks/
└── worktrees/                  # 默认位置，可由 worktree_root 改写
```

`development.root`、`worktree_root`、Harness、Sessions 和活动 Trace 必须位于本机非同步目录。不要放入 Obsidian Vault；Vault 只承载人类知识和 Markdown 投影。

Reviewer 身份、角色和法定人数由顶层 `governance` 配置控制，见 [`GOVERNANCE_CLOSED_LOOP_CN.md`](GOVERNANCE_CLOSED_LOOP_CN.md)。

## 3. Codex 订阅模型

以运行 GoClaw 的同一个操作系统用户安装并登录：

```bash
npm install -g @openai/codex
codex login
codex exec --json --sandbox read-only "只输出 OK"
```

`codex_model` 留空时不覆盖 Codex CLI 的默认模型，因此使用该 ChatGPT 工作区当前授权的默认订阅模型。开发 Hand 使用：

```text
codex --ask-for-approval never exec --json --sandbox workspace-write -
```

独立审查使用 `read-only` Sandbox 和 JSON Schema。恢复运行使用 Codex 返回的 Thread ID：

```text
codex ... exec --json --sandbox workspace-write resume <thread-id> -
```

GoClaw 不读取、复制或保存 Codex OAuth 文件。`systemd` 的 `User` 和 `HOME` 必须对应执行过 `codex login` 的用户。执行时 GoClaw 不把服务进程的完整环境传给 Codex：每次运行都会创建隔离的 HOME、XDG 与临时目录，子进程环境只保留执行所需的最小工具链变量，并通过 `CODEX_HOME` 指向既有登录目录。GoClaw Token、SSH agent、Docker/Kubernetes 和云凭据路径不进入中央 Hand。

## 4. 创建任务契约

完整模板见 `deploy/dev-task.example.json`。推荐显式填写：

- `goal.objective`、`non_goals`、`success_tests`
- `plan.milestones[].work_items[]`
- `evidence_plan.commands[]`，每条命令必须是 argv 数组
- `scope.allowed_paths`、行数和文件数上限
- `risk`、`cost`
- `done_gate`

创建：

```bash
goclaw dev init
TASK_ID=task-project-alpha-orders-001
goclaw dev create \
  --id "$TASK_ID" \
  --repository-id repo-api \
  --assignee user-alice \
  --spec deploy/dev-task.example.json \
  --json
goclaw dev list --project project-alpha --json
```

也可以用精简参数：

```bash
TASK_ID=task-project-alpha-orders-001
goclaw dev create \
  --id "$TASK_ID" \
  --project project-alpha \
  --repository-id repo-api \
  --assignee user-alice \
  --title "修复订单幂等" \
  --request "重复请求不得创建第二个订单" \
  --base main \
  --allow-path 'internal/orders/**' \
  --max-files 12 \
  --max-lines 500 \
  --verify '["go","test","./internal/orders/..."]' \
  --verify '["git","diff","--check"]' \
  --json
```

精简形式会补齐默认 Goal、Plan、CapabilityManifest、Risk 和 DoneGate。冻结前至少要有一个确定性验证命令。

Team 模式必须提供可由客户端稳定重建的 `--id`，且 `dev list` 必须指定 `--project`。同 ID、同一规范化创建请求的重试返回原任务且不增加事件；同 ID 不同请求会冲突。单机模式仍可让服务生成 ID，但生产团队不应依赖响应中的随机 ID 来恢复一次结果不明的 create。

## 5. 四类评审与冻结

四类评审都必须由人类显式记录：

| 评审 | 核对内容 |
|---|---|
| `scenario` | 目标、非目标、成功条件、边界场景是否完整 |
| `capacity` | 文件/行数、依赖、计算和令牌预算是否合理 |
| `risk` | 禁止动作、数据安全、回滚和升级条件 |
| `cost` | 最大修复次数、输入/输出令牌与人工成本 |

```bash
TASK_ID=task-REPLACE

GOCLAW_REVIEWER_TOKEN="$CAROL_TOKEN" goclaw dev review "$TASK_ID" \
  --kind scenario --decision approved --reviewer carol-scenario-risk \
  --comment "场景边界已核对" --counterargument "仍可能遗漏罕见并发序列"
GOCLAW_REVIEWER_TOKEN="$DAVE_TOKEN" goclaw dev review "$TASK_ID" \
  --kind capacity --decision approved --reviewer dave-capacity-cost \
  --comment "范围和预算可执行" --counterargument "全仓测试可能超过估算"
GOCLAW_REVIEWER_TOKEN="$CAROL_TOKEN" goclaw dev review "$TASK_ID" \
  --kind risk --decision approved --reviewer carol-scenario-risk \
  --comment "回滚和禁止动作明确" --counterargument "第三方故障仍可能扩大影响"
GOCLAW_REVIEWER_TOKEN="$DAVE_TOKEN" goclaw dev review "$TASK_ID" \
  --kind cost --decision approved --reviewer dave-capacity-cost \
  --comment "修复次数和令牌上限合理" --counterargument "复杂故障可能需要重做规格"
goclaw dev freeze "$TASK_ID" --reviewer freezer
```

冻结会：

1. 再验证四类批准和验证命令。
2. 默认拒绝有未提交修改的源仓库。
3. 解析并固定 `base_ref` 对应的 commit。
4. 计算完整执行包 SHA-256。

冻结后发现新需求时，不应直接修改任务投影。先导出 `goclaw dev show` 的任务 JSON，修改可变契约字段，然后使用：

```bash
goclaw dev revise "$TASK_ID" \
  --spec replacement-task.json \
  --reason "ChangeIntent: 新增并发重复请求场景" \
  --reviewer alice
```

Revision 会递增，旧冻结信息和运行结果失效，四类评审全部回到 `pending`。

Team 模式的 `dev revise` 和 `dev repair` 在发起 RPC 前会先 `dev.task.get`，自动读取并发送 `expected_revision`；Obsidian 修订按钮也发送当前投影的 revision。Gateway 用它做 Compare-And-Swap，过期请求不会重复递增 revision。直接调用 `dev.task.revise` 时必须显式提供 `expected_revision`。如果响应结果不明，不要无条件再次发起一轮“读取当前值后修订”；应先 `goclaw dev show "$TASK_ID"` 判断 revision 是否已前进。

## 6. 执行、证据和修复

```bash
goclaw dev run "$TASK_ID" --reviewer runner
goclaw dev show "$TASK_ID" --json
goclaw dev events "$TASK_ID" --json
```

执行期间：

- 每个任务只有一个运行锁。
- Codex 只能在任务 worktree 的 `workspace-write` Sandbox 中修改。
- 提示词禁止提交、推送、创建 PR 和修改 Git 配置。
- GoClaw 在 Hand 前后都要求 worktree `HEAD` 等于冻结的 `BaseCommit`；Codex 自动创建 commit 会使本次执行失败。
- Go 重新收集相对冻结 commit 的全部变更。
- 默认拒绝 `.env`、credential、secret、运行时自身和 `.git` 路径。
- 检查 allowed/denied glob、符号链接越界、依赖清单、文件数和总增删行数。
- 验证命令直接以结构化 argv 执行，不经过 shell 拼接。
- 确定性检查通过后，另起只读 Codex Thread 做独立审查。

单机 Hand 的 DoneGate 失败时：

```bash
goclaw dev repair "$TASK_ID" --reviewer runner
```

修复次数不能超过冻结的 `cost.max_repair_attempts`。进程异常退出后可以恢复：

```bash
goclaw dev resume "$TASK_ID" --reviewer runner
```

只有确认旧进程已经停止、锁确实陈旧时，才允许：

```bash
goclaw dev resume "$TASK_ID" --reviewer runner --force
```

`--force` 会删除任务运行锁；它不是并发抢占机制。

Team 模式的 `repair` 不会在中央直接重跑 Hand，而是创建一个必须重新评审的新 revision：

```bash
# 仅当旧队列仍为 queued/failed 时需要先取消；leased 会拒绝取消。
goclaw runner cancel "${TASK_ID}-r${TASK_REVISION}" \
  --reason "改为新的受审修订"
goclaw dev repair "$TASK_ID" \
  --reason "DoneGate 失败，需要调整任务契约"
```

旧 revision 队列仍为 `queued` 或 `leased` 时，Gateway 拒绝 revise/repair；`leased` 必须等待 Runner complete/fail 或租约恢复。修订会先把 TeamControl 中 `in_progress`/`verifying` 的 WorkItem 退回 `blocked`，然后 revision 加一、四审清零。只有重新完成四审、freeze 和 enqueue，WorkItem 才重新进入执行。一个 WorkItem 只能绑定一个开发任务；Issue 可以由多个任务共享。

Full Runtime 也可将冻结任务入队：

```bash
goclaw dev enqueue "$TASK_ID" \
  --priority 10 \
  --capability codex \
  --max-attempts 3
```

每个冻结 revision 只有一个 `<TASK_ID>-r<REVISION>` 队列身份；关联 WorkItem 必须恰有一个 active owner 且等于 task assignee。Runner 完成后，Gateway 把已验证 HMAC 的 EvidenceBundle 导入本服务。导入不会信任远端 DoneGate，而会重验 revision/execution bundle、base/head、no-commit、diff SHA、changed paths 和冻结 checks，重算 scope/falsifier/prediction/kill checks，必要时重新运行独立模型审查，再生成本地 EvidencePackage 与 Go DoneGate。相同 Bundle SHA 的重复导入是幂等的。

## 7. 最终验收和本地提交

DoneGate 通过后，默认状态为 `awaiting_acceptance`。验收命令会重新验证：

- `evidence.json` 的 SHA-256 未改变。
- worktree 的 diff 和变更文件集合未改变。
- 上一次 DoneGate 确实通过。

```bash
goclaw dev accept "$TASK_ID" \
  --reviewer erin-final \
  --comment "已核对 diff、测试、策略和独立审查" \
  --counterargument "测试环境不能覆盖全部生产流量形态"
```

单机 Hand 的验收不会自动提交。确认 Git 用户信息已配置后：

```bash
git config --global user.name "Alice"
git config --global user.email "alice@example.com"

goclaw dev commit "$TASK_ID" \
  --reviewer alice \
  --message "fix: make order creation idempotent"
```

提交只会暂存任务相对冻结基线的变更文件，且在提交前再次验证 worktree 未改变。当前版本不会 push，也不会创建 Pull Request。

Workstation 导入路径通过 Obsidian 或 `dev.task.accept` Gateway RPC 验收。Gateway 从个人 Token principal 解析身份，要求其具备项目 `project.manage`，再验证 Reviewer Token/`task_accept` 角色；验收会重新检查导入证据和 diff 未改变，随后把当前任务的 TeamControl WorkItem 置为 done。共享 Issue 只有在全部关联任务都为 `done`，且全部关联 WorkItem 都为 `done` 时才聚合为 resolved；`cancelled` 永不算成功。取消分支会让 Issue 保持 open/verifying/blocked，等待重新指派或由有权人员给出明确 resolution 后另行关闭。控制面会拒绝对导入证据执行 `goclaw dev commit`，负责人应在原工作站或受控 checkout 中核对签名 patch 后进入正常 Git/PR 流程。

提交并创建 PR 后，通过 Team Gateway 回链：

```bash
goclaw dev link-pr "$TASK_ID" \
  --commit "$COMMIT_SHA" \
  --url "$PR_URL"
```

只有已验收的 Workstation 导入证据可使用 `--commit` 路径。服务在任务冻结时绑定的受管 `repo_path`/TeamControl `local_path` 中解析 commit，要求它是 frozen base 的后代，`base..commit` 的 binary diff 与 accepted `diff.patch` 精确相同（规范化时仅删除 Git `index ...` 元数据行），并检查 commit message 包含完整 Task/Project/Revision、可选 Repository/Correlation/Policy，以及全部 WorkItem/Issue trailers。验证通过后写入 task commit/PR 身份和事件，并由 Gateway 自动创建 TeamControl commit/PR Artifact 与 Task/Repository/WorkItem/Issue CorrelationLink。

`link-pr` 不会 fetch commit、push 分支、创建/批准/merge PR 或等待 CI；调用前必须由开发者、CI 或管理员让中央 `local_path` 已能看到该 commit。PR URL 只接受无用户名密码、无 query、无 fragment 的绝对 HTTP(S) 地址并登记；当前没有 provider API，因此不能声称远端 PR head、内容、状态或 commit 归属已经验证。

## 8. Obsidian 控制面

启用 `development` 后，Obsidian 插件增加：

- “审批”：显示四类评审、冻结按钮和最终验收按钮。
- “开发”：显示任务状态、分支、范围上限、DoneGate 结果和修复入口。

推荐安全配置：

```json
{
  "development": {
    "gateway_allow_execution": false
  }
}
```

这样 Obsidian 可查看、使用个人 Token + Reviewer Token 评审、冻结和最终验收。修订/修复时插件自动读取任务投影中的 revision，并在 RPC 中发送 `expected_revision`。`development.gateway_allow_execution` 仅适用于未启用 TeamControl 的单用户部署；Team 模式下 `dev.task.run/repair/resume` 无条件禁用，即使把该值设为 `true` 也不会放行。团队唯一执行路径是 `dev enqueue` → Workstation 持久队列。TeamControl 启用时，`dev.*` 按任务所属项目做 RBAC，旧 process-global RPC 默认失败关闭；这仍只适合受信网络中的单中央控制面。

未启用 `development` 时，插件的其他审批功能仍可使用；开发页会显示该 RPC 不可用。

## 9. 状态机

```text
review_pending ──四审通过──> ready_to_freeze ──freeze──> frozen
      │                              │
      └──任一 rejected──> blocked    └──run──> running → checking
                                                        │
                           DoneGate fail ────────────────┤
                           ▼                            ▼
                     repair_pending              awaiting_acceptance
                           │                            │
                           ├──本地 repair/resume         └──accept──> done
                           ├──Team repair/revise
                           │       └──WorkItem blocked → revision+1 → 四审/freeze/enqueue
                           └──预算耗尽──> failed                        ├─中央 Hand commit
                                                                       └─外部 commit/PR → link-pr
```

Team 模式下，`freeze`、`revise/repair`、`enqueue`、`link-pr` 只允许 task assignee 或项目管理者。`accept` 和 `cancel` 必须具备 `project.manage`，并分别叠加 `task_accept`/`task_cancel` Governance 角色。`cancel` 保留全部事件和证据，不删除 worktree 或审计记录：

```bash
goclaw dev cancel "$TASK_ID" \
  --reviewer erin-final \
  --reason "需求撤销" \
  --comment "产品负责人已撤销该 Revision" \
  --counterargument "已完成的工作可能仍可用于下一版"
```

开发任务取消会取消其绑定 WorkItem，但共享 Issue 不进入 `cancelled`；没有其他活动任务时，执行中的 Issue 回到 `blocked`。队列层的 `runner cancel` 是另一操作，只能取消 queued/failed，leased 明确拒绝。旧 revision queued/leased 时 revise 会拒绝，避免旧执行与新契约并行。

主要写操作的重试边界如下：

| 操作 | 重试行为 |
|---|---|
| create | 稳定 ID + 完全相同请求返回已有任务；同 ID 不同请求冲突 |
| review | 完全相同的评审载荷不重复事件；载荷变化是新评审决定 |
| freeze | 已冻结时返回原 revision/bundle，不重复事件 |
| enqueue | 服务端按 revision + bundle 派生唯一 queue ID/幂等键；无客户端幂等 flag |
| accept / cancel | Team Gateway 对已完成结果做收敛式重试，不重复任务终态事件 |
| link-pr | 相同 commit/URL 幂等；不同绑定冲突 |
| revise / Team repair | `expected_revision` CAS 阻止同一请求重复递增；结果不明时先 show 再决定 |

## 10. 故障排查

### `codex executable not found`

确认 `codex` 位于服务用户的 `PATH`，或将 `development.codex_command` 设置为绝对路径。

### Codex 要求重新登录

停止服务，以 `systemd User` 对应用户执行 `codex login`，再恢复服务。不要把其他用户的 OAuth 文件复制到服务账户。

### `repository has uncommitted changes`

冻结基线必须可复现。提交或暂存源仓库改动后重试。只有明确接受不可复现风险时才设置 `allow_dirty_repo=true`。

### `denied path changed` / `path outside approved scope`

查看 `<last_evidence>/policy.json` 和 `diff.patch`。若确属新范围，使用 ChangeIntent 修订并重新完成四类评审；不要直接放宽已冻结任务。

### `worktree changed after DoneGate`

验收前发生了额外修改。重新运行修复/验证以生成新的 EvidencePackage。

### 验证命令超时

调整 `verify_timeout_seconds`，或把过大的全仓测试拆成冻结的分层命令。不要把验证改成不检查真实行为的空命令。

## 11. 当前边界

已闭合的是单中央控制面、中央 Hand 或成员 Workstation Runner 的开发控制和证据链。生产级系统仍需：

- Workstation 已提供持久队列、租约、幂等消费和多 Runner，但仍需事务数据库/外部队列、Leader 选举和多控制面 HA。
- Team 模式已为 `dev.*`、Harness、Memory 和 Ouroboros 提供项目 RBAC，并禁用无策略的旧全局 RPC；仍需 Reviewer 个人证书/硬件密钥签名。
- 事件链外部签名、WORM 备份和集中审计。
- 外部 commit/PR 的本地验证和关联已实现；仍需 GitHub/GitLab fetch/push、PR 创建与远端状态、CI 状态和合并策略适配器。
- TeamControl、Workstation 和 Orchestrator Lite 的当前闭环以一个中央 GoClaw 写入进程为前提；不能让多实例同时写同一 root。
- 容器或微虚机级网络/进程隔离。
- Trace 失败自动聚类、根因候选和基于统计置信度的在线评测。

这些边界不会影响当前本地链路的使用，但不能把 Orchestrator Lite 描述为已经具备高可用和无人值守生产发布能力。
