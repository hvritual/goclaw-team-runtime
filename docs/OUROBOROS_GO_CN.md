# Go 原生 Ouroboros 使用与部署

本实现把 [Q00/ouroboros](https://github.com/Q00/ouroboros) 的核心思路原生融入 GoClaw，而不是启动一个 Python sidecar。它与现有 Better Harness、Orchestrator Lite、飞书和 Obsidian 共用同一个 Go 进程与 ChatGPT 订阅模型入口。

## 1. 能力边界

已实现的闭环：

```text
需求
  → 苏格拉底式访谈
  → 多评估器 + Go 计算歧义度、分差、灰区与模型相关性
  → 利益相关方冲突逐条解决
  → 连续两次满足门槛
  → 生成带备选/证伪/参考类/预测/停止条件的不可变 Seed 候选
  → Reviewer 身份、职责分离与法定人数批准
  → 单向编译为 Orchestrator Lite 任务
  → 四类人工评审
  → 隔离 worktree 执行
  → Go DoneGate + EvidencePackage
  → 语义评估 + 盲化评审 + 关键发现否决/分歧升级
  → 结果和参考类反馈
  → 生成下一代 Seed 候选
  → 收敛，或人工批准后进入下一代
```

模型负责提问、结构化候选和语义审查。Go Core 负责状态转换、阈值、哈希、权限边界、机械检查和多数决。模型不能直接：

- 批准 Seed 或演化候选。
- 创建并执行开发任务。
- 修改 active Seed、Harness、知识库或仓库。
- 把聊天记忆当作验收证据。

人工批准表示“允许进入下一阶段”，不表示“实现已经正确”。正确性仍由冻结的验收命令、EvidencePackage、DoneGate 和独立审查决定。

## 2. 与上游设计的对应

| 概念 | GoClaw 实现 |
|---|---|
| interview → seed → execute → evaluate → evolve | `ouroboros.Service` 状态机 |
| `ambiguity = 1 - weighted clarity` | Go 中的确定性评分函数 |
| Greenfield 权重 | goal 0.40 / constraint 0.30 / success 0.30 |
| Brownfield 权重 | goal 0.35 / constraint 0.25 / success 0.25 / context 0.15 |
| 维度下限 | goal 0.75 / constraint 0.65 / success 0.70 / context 0.60 |
| Ready 稳定性 | 默认连续两次满足 `ambiguity <= 0.20` 且所有下限通过 |
| 本体相似度 | 0.5 名称重叠 + 0.3 类型一致 + 0.2 完全一致 |
| 收敛阈值 | 默认 0.95 |
| 演化上限 | 默认 30 代，并检测 period-2 本体振荡 |

这是独立的 Go 原生适配，并非把上游 Python 包嵌入运行时。归属和许可证见根目录 `THIRD_PARTY_NOTICES.md`。

## 3. 权威数据

运行目录必须位于单写入主机的本地磁盘，不要放入同步 Vault：

```text
<ouroboros.root>/
├── sessions/
│   └── <session-id>/
│       ├── events.jsonl   # 带前向哈希的权威事件链
│       └── session.json   # 可重建投影，不是独立权威
└── seeds/
    └── <sha256>.json      # 内容寻址、不可覆盖的 Seed
```

每次读取会验证完整事件链。每个 Seed 会验证文件名、内嵌哈希与内容哈希。不要手工编辑这些文件；备份时应停止写入或对整个运行目录做一致性快照。

Obsidian Vault 仍是目标、ADR、约束、需求和知识的权威 Markdown 层。Ouroboros 运行状态不进入 Vault，因此多台电脑同步不会产生 JSONL 或 active 指针冲突。

创建会话时，Go Core 只采集仓库顶层名称及白名单构建清单（例如
`go.mod`、`package.json`、`pyproject.toml`），并限制单文件、总量和模型上下文大小。
隐藏项、疑似凭据文件、符号链接和任意源码不会被自动读取。该快照是供访谈参考的
**不可信上下文**，不能替代固定 Revision、验收命令、diff、EvidencePackage 或
DoneGate 证据。

## 4. 配置

从 `deploy/config.codex-obsidian.example.json` 复制完整示例。核心配置：

```json
{
  "ouroboros": {
    "enabled": true,
    "root": "/srv/goclaw-runtime/ouroboros",
    "model": "",
    "evaluation_models": [],
    "assessment_models": [],
    "ambiguity_threshold": 0.2,
    "convergence_threshold": 0.95,
    "required_ready_streak": 2,
    "max_generations": 30,
    "assessment_reviewers": 2,
    "assessment_max_spread": 0.15,
    "assessment_gray_zone": 0.03,
    "consensus_reviewers": 3,
    "critical_finding_veto": true,
    "consensus_max_spread": 0.25,
    "evaluation_history_window": 5,
    "required_passing_evaluations": 2,
    "max_session_model_calls": 120,
    "max_session_model_tokens": 2000000,
    "max_questions_per_round": 5,
    "max_context_bytes": 131072,
    "max_output_tokens": 12000
  }
}
```

`model` 留空时复用 GoClaw 当前主 Provider。示例中的主 Provider 是 `codex-app-server`，因此模型调用使用运行 GoClaw 的操作系统用户已有的 `codex login` / ChatGPT 工作区订阅，不读取或复制 OAuth 文件。

`assessment_models` 或 `evaluation_models` 留空时，多个角色使用同一配置 Provider 的独立调用。系统会保存实际模型身份；多个评估器仍是同一模型时会明确升级人工，不会把角色提示伪装成独立共识。不同模型 ID 也只表示声明的模型不同，不证明供应商或训练数据独立。

同时启用：

- `development.enabled=true`：批准的 Seed 才能编译为受控开发任务。
- `development.gateway_allow_execution=false`：控制未启用 TeamControl 的单用户 Gateway Hand。Team 模式的 `dev.task.run/repair/resume` 无条件禁用，团队只通过 `dev enqueue` → Workstation 持久队列执行。
- `development.require_human_final_approval=true`：DoneGate 后仍需人工最终验收。
- `gateway.websocket.enable_auth=true`：Obsidian 审批控制面必须认证。
- `governance.enabled=true` 与 `require_authenticated_reviewers=true`：决策使用独立 Reviewer Token；完整配置和角色见 [`GOVERNANCE_CLOSED_LOOP_CN.md`](GOVERNANCE_CLOSED_LOOP_CN.md)。

## 5. 启动与健康检查

```bash
cp deploy/config.codex-obsidian.example.json ~/.goclaw/config.json
chmod 600 ~/.goclaw/config.json

codex login
goclaw ouroboros init
goclaw start
```

启动日志应同时出现：

```text
Go-native Ouroboros store ready
Go-native Ouroboros model and channel tools ready
Orchestrator Lite ready
```

如果 `codex` 不在 systemd 的 `PATH` 中，请在 unit 的 `Environment=PATH=...` 中加入 Codex CLI 所在目录。GoClaw 不会把 ChatGPT 凭据写进配置。

## 6. CLI 完整流程

守护进程运行时，应通过 Obsidian/Gateway 进行写操作。若要使用下列本地 CLI
修改同一个 Ouroboros 数据目录，请先停止守护进程；当前实现只有进程内互斥，
不支持多个进程同时写同一事件链。

示例治理配置启用 Reviewer 认证和反方论点。执行审批命令前应设置对应身份的
`GOCLAW_REVIEWER_TOKEN`，并在所有批准/接受/提升/回滚命令中加入
`--counterargument` 与必要的 `--evidence-ref`。完整示例见
[`GOVERNANCE_CLOSED_LOOP_CN.md`](GOVERNANCE_CLOSED_LOOP_CN.md)。

### 6.1 访谈与 Seed

```bash
goclaw ouroboros start \
  --project project-alpha \
  --title "实现项目级审计导出" \
  --request "为现有 Go 项目增加可验证的审计导出" \
  --repo /absolute/path/to/repository \
  --base HEAD \
  --brownfield

goclaw ouroboros list --project project-alpha
goclaw ouroboros show <session-id>

goclaw ouroboros answer <session-id> \
  --question <question-id> \
  --answer "项目负责人确认的明确答案"

# 或批量回答
goclaw ouroboros answer <session-id> \
  --answers deploy/ouroboros-answers.example.json

# 没有新增答案但需要第二次稳定性评估时
goclaw ouroboros reassess <session-id>

goclaw ouroboros crystallize <session-id>
goclaw ouroboros seed <seed-sha256>
goclaw ouroboros events <session-id>
```

`crystallize` 只生成候选，状态是 `awaiting_seed_approval`：

```bash
goclaw ouroboros approve-seed <session-id> \
  --reviewer alice \
  --comment "目标、范围、测试命令与预算已复核"

# 或拒绝并回到澄清
goclaw ouroboros reject-seed <session-id> \
  --reviewer alice \
  --comment "缺少回滚验收条件"
```

### 6.2 编译、执行与评估

```bash
goclaw ouroboros compile <session-id> --reviewer alice
goclaw dev show <task-id>

goclaw dev review <task-id> --kind scenario --decision approved --reviewer carol-scenario-risk
goclaw dev review <task-id> --kind capacity --decision approved --reviewer dave-capacity-cost
goclaw dev review <task-id> --kind risk --decision approved --reviewer carol-scenario-risk
goclaw dev review <task-id> --kind cost --decision approved --reviewer dave-capacity-cost

goclaw dev freeze <task-id> --reviewer alice
goclaw dev run <task-id> --reviewer runner
goclaw dev show <task-id>

goclaw ouroboros evaluate <session-id> --task <task-id> --reviewer alice
```

`evaluate` 依次执行：

1. Go 机械门：任务/Revision、DoneGate 哈希、范围策略、验证结果、独立只读审查和 diff 证据。
2. 语义门：逐项核对不可变 Seed。
3. 默认三个盲化角色评审，省略执行器的最终自述。
4. Go Core 多数决、关键发现否决、分差和模型相关性检查。

机械门失败时不会调用后续模型评审，避免用模型文字掩盖缺失证据。

如果返回 `human_decision_required`，必须先裁决证据争议；裁决不是任务验收：

```bash
goclaw ouroboros resolve-evaluation <session-id> \
  --evaluation <evaluation-id> \
  --accept \
  --reviewer alice-spec \
  --comment "diff、测试日志和证伪结果支持验收条件" \
  --counterargument "评审模型仍可能存在相关盲点" \
  --evidence-ref "artifact:test-log"
```

### 6.3 演化

```bash
goclaw ouroboros evolve <session-id>

goclaw ouroboros approve-evolution <session-id> \
  --reviewer alice \
  --comment "变化由评估证据支持"

# 或保留当前 active Seed
goclaw ouroboros reject-evolution <session-id> \
  --reviewer alice \
  --comment "候选扩大了未授权范围"
```

候选生成后不可修改 active Seed。只有 `approve-evolution` 会切换 active Seed；之后必须重新 `compile`，新一代任务重新完成四审、冻结、执行和验收。

## 7. 飞书

启用后，Agent 会获得以下低权限工具：

- `ouroboros_start`
- `ouroboros_answer`
- `ouroboros_get`
- `ouroboros_reassess`
- `ouroboros_crystallize`

内置 `ouroboros-spec` 技能会引导 Agent 只使用 Go Core 返回的问题。典型对话：

```text
用户：为 project-alpha 开始一个 Ouroboros 规格访谈，我要给现有 API 增加幂等写入。
机器人：返回会话 ID、歧义度、阻塞问题和 readiness streak。
用户：q1 的答案是……；q2 的验收命令是……
机器人：记录明确答案并重新评估。
用户：生成 Seed。
机器人：仅在 seed_ready 后结晶，并提示到 Obsidian/CLI 审批。
```

飞书工具集中不存在 Seed 审批、任务编译、执行、最终验收、演化批准或 Harness 提升工具。不要要求机器人在聊天里绕过这些边界。

飞书和 Obsidian 使用相同的 `project_id + topic_id` 时共享项目会话上下文；Ouroboros 规格会话本身以返回的 `session_id` 精确寻址。

## 8. Obsidian

0.7.0 Team Web Console 的“规格”页：

- 创建 Brownfield/Greenfield 访谈。
- 显示歧义度、阈值和 readiness streak。
- 回答阻塞问题、重新评估、结晶 Seed。
- 编译已批准 Seed、读取开发证据并评估、生成演化候选。

“审批”页新增：

- 需求评估分歧、利益相关方冲突和实现评估争议裁决。
- Seed 批准/拒绝。
- 演化候选批准/拒绝。
- 原有知识提案、Harness 实验和开发四审。

“开发”页仍负责冻结、执行、修复和最终验收。这个拆分是有意的：规格审批不会直接触发代码执行。

## 9. 恢复与排障

### 模型返回格式错误

运行时只允许严格 JSON Schema 形状，并自动进行一次有界格式修复。第二次仍失败时记录 `interview.failed` 或返回操作错误，不会降级为猜测。

### 事件链或 Seed 哈希错误

停止服务并从可信备份恢复整个 `ouroboros.root`。不要编辑 JSONL 或重新计算单个文件来掩盖篡改。

### `task has no EvidencePackage`

任务尚未完成受控执行和 DoneGate。先用中央 Hand，或让 Workstation Runner 回传证据并由 Orchestrator Lite 导入重验，再运行 `ouroboros evaluate`。

### 多台电脑

只运行一个 GoClaw Leader。多台 Obsidian 连接同一个 WSS Gateway。不要通过 Obsidian Sync、Syncthing 或 Git 同步 `ouroboros.root`、development runtime、worktree、锁或活动事件链。

## 10. 当前生产边界

- 单节点、进程内互斥；模型操作会串行化，尚无分布式队列、租约或 HA Leader。
- Gateway Token 认证连接；Team 模式再用个人 Token principal 与 Session/Seed 所属项目执行 RBAC，审批、裁决、取消、提升和回滚还需 Reviewer 角色 Token。无团队授权策略的旧全局 RPC 会失败关闭；Reviewer Token 仍不是密码学个人签名。
- 事件和 Seed 有本地哈希完整性校验，但尚未写入外部签名/WORM 存储。
- 多角色评审默认可能仍由同一个 ChatGPT 订阅模型完成；现在会升级人工而不是把它计为独立共识。
- 尚无自动 PR、CI 等待、合并策略或远程执行；这些保持在人工/外部流水线边界。
- 本体相似度是结构启发式，不是业务等价证明；振荡或缺失业务决策会升级给人类。

这些边界是显式限制，不应被解释为生产级分布式编排已经完成。
