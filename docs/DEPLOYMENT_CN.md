# GoClaw Team Runtime 0.7.0 部署手册

这份实现采用“中央单写入控制面、多端交互、本地工作站执行”的结构：

- 一台常驻机器运行 Gateway、TeamControl、Workstation 队列、Memory Catalog、Ouroboros、会话、Trace 与 Harness Registry。
- 10 名成员的电脑各自运行 Runner 和本地 Codex CLI，使用各自 ChatGPT/Codex OAuth。
- 所有成员通过同一 Team Web Console 登录；浏览器凭据换成 HttpOnly 短期会话。
- 飞书机器人按 `project_id + topic_id` 与 Web Console 共享项目会话。
- Markdown 知识位于 `harness.knowledge_root`，可使用普通文件目录或 Git 工作树；Obsidian 是可选编辑器，不是运行依赖。
- 知识目录只保存 Markdown，不保存 TeamControl、Runner 队列/租约、Token、device key、Codex OAuth、Catalog SQLite、Ouroboros/Harness/Development Runtime、会话 JSONL、锁或活动 Trace。

这样避免了多台电脑同时写控制面状态、活动会话、任务租约或 Harness 指针所产生的冲突。

## 部署层级

| 层级 | 适用场景 | 启用能力 | 仍需人工完成 |
|---|---|---|---|
| MVP | 先让 10 人使用同一项目、同一规则和同一看板 | 单节点 Gateway + Team Web Console、TeamControl、个人 Token、团队/项目/仓库/RBAC、Issue/Work/Assignment、飞书项目路由 | 本地启动开发、commit/PR/CI/release 及外部 Issue 对账 |
| Full Runtime | 让成员电脑领取冻结开发任务 | MVP + Workstation queue/lease/heartbeat、10 台 Runner、本地 Codex OAuth、签名 diff/evidence、Harness、Catalog、Ouroboros、Orchestrator Lite、外部 commit/accepted patch 校验与 commit/PR 自动关联 | 最终验收、应用 patch、commit/push、创建/批准/merge PR、CI/release |
| Production | 长期运行的单节点团队控制面 | Full Runtime + TLS/VPN/SSH 隧道、进程托管、个人秘密管理、备份恢复、监控告警和恢复演练 | HA、多 Leader、GitHub/Jira 双向同步 |

Production 在 `0.7.0` 中表示“加固后的单节点”，不表示高可用集群。TeamControl 与 Workstation 使用文件存储，只允许一个 GoClaw 进程写同一 root。

配置开关：

- MVP：`team_control.enabled=true`，`workstation.enabled=false`；先完成第 1～4、7～8 节。
- Full Runtime：再设 `workstation.enabled=true`，完成第 5～6 节并逐台注册 Runner。
- Production：在 Full Runtime 上完成第 9～12 节，尤其是 HTTPS、专用 OS 身份、备份与恢复演练。

## 1. 运行要求

- Go 1.25.5。
- Node.js 20 或更高版本。
- 中央对话 Provider 需要 Codex CLI，并由运行 GoClaw 的操作系统用户登录 ChatGPT。
- 每台启用 Runner 的成员电脑也需要 Codex CLI，并由该成员自己的操作系统账户完成独立登录。
- Linux Runner 需要 `bubblewrap`（通常为发行版软件包 `bubblewrap`），并使用发布包内的受审验证 wrapper。
- Production Runner 使用专用 OS 用户、VM 或容器，只挂载授权仓库、work root、device key 和该用户自己的 Codex OAuth。
- 现代浏览器；生产环境必须通过 HTTPS、VPN 或 SSH 隧道访问 Team Web Console。
- 可选：Obsidian Desktop 1.8 或更高版本。
- 自建飞书应用及机器人能力。

安装 Codex CLI 后登录：

```bash
npm install -g @openai/codex
codex login
codex doctor
```

登录时选择 “Sign in with ChatGPT”。示例模型 `codex/default` 让 Codex CLI 选择该 ChatGPT 工作区当前可用的默认订阅模型。本实现不会读取、复制或保存 Codex OAuth 文件；每次模型调用由本地 `codex app-server` 使用自己的登录状态完成。

## 2. 编译

在仓库根目录运行：

```bash
export PATH=/path/to/go1.25.5/bin:$PATH
chmod +x scripts/build-release.sh
./scripts/build-release.sh
```

输出：

- `dist/goclaw`
- `dist/goclaw-team-runtime-linux-amd64-0.7.0.tar.gz`
- `dist/goclaw-team-runtime-source-0.7.0.tar.gz`
- `dist/SHA256SUMS-0.7.0.txt`

只有确实保留 Obsidian 桌面入口时才额外构建适配器：

```bash
INCLUDE_OBSIDIAN_PLUGIN=1 ./scripts/build-release.sh
```

只需要生成经过同等凭据、路径和归档安全检查的源码包时，可以跳过
Go 测试与 Linux 二进制构建：

```bash
SOURCE_ONLY=1 ./scripts/build-release.sh
```

`SOURCE_ONLY=1` 不是完整发布验证；正式部署仍必须在 Go 1.25.5 环境运行
默认命令，完成 Go 测试和静态 Linux 二进制构建。

Linux Runtime 包还包含 `scripts/verify-sandbox-bwrap.sh`、`deploy/runner.env.example` 和 `deploy/systemd/goclaw-runner.service.example`。源码包按显式 allowlist 归档，排除 `gateway/goclaw_test`、构建产物和常见二进制垃圾；构建脚本在归档前后检查凭据特征、危险路径、symlink/hardlink 与异常成员类型。

只编译 GoClaw：

```bash
go test ./memory ./memory/catalog ./governance ./ouroboros ./orchestratorlite ./harness ./teamcontrol ./workstation ./providers ./gateway ./agent ./agent/tools ./config ./cli ./cli/commands ./internal/start
go build -trimpath -o dist/goclaw .
```

不要把旧 `dist/goclaw` 当成新版本。部署前先验证：

```bash
dist/goclaw version
dist/goclaw team --help
dist/goclaw runner --help
(cd dist && sha256sum -c SHA256SUMS-0.7.0.txt)
```

## 3. 准备受治理 Markdown 根目录

推荐目录：

```text
TeamKnowledge/
├── 00-index/
│   └── project-alpha/
├── 01-goals/
│   └── project-alpha/
├── 02-decisions/
│   └── project-alpha/
├── 03-constraints/
│   └── project-alpha/
├── 04-requirements/
│   └── project-alpha/
├── 05-knowledge/
│   └── project-alpha/
├── 06-test-plans/
│   └── project-alpha/
├── 07-runbooks/
│   └── project-alpha/
├── 08-reviews/
│   ├── inbox/
│   ├── approved/
│   └── rejected/
├── 09-releases/
│   └── project-alpha/
└── .obsidian/
```

目标、决策、约束、需求和知识的受控 Markdown 权威层是 `01` 到 `05`；Test Plan、Runbook 和 Release 进入各自目录并在 Document Registry 登记。多项目时在各类目录下使用稳定 `project_id` 子目录。知识写入治理提供：

- `search_project_knowledge`：只读全文检索。
- `read_project_knowledge`：只读单个 Markdown。
- `propose_knowledge_change`：生成提案，不修改目标。
- 只有人类批准 `knowledge.proposal.approve` 后才原子写入目标。

批准时会比较创建提案时的 SHA-256。`knowledge_backend=git` 时还会比较创建提案时的 HEAD revision；任一变化都会失败并要求重新生成提案。

Memory Catalog 会把这些 Markdown 编目为 `pending` 候选，并维护 Work/Expression/Manifestation/Item 身份、权威词、来源、有效期和版本。编目审批不修改原 Markdown；知识提案审批负责修改 Markdown。两条审批链职责不同，完整说明见 [`LIBRARY_MEMORY_CN.md`](LIBRARY_MEMORY_CN.md)。

## 4. 配置 GoClaw

复制示例：

```bash
mkdir -p ~/.goclaw
cp deploy/config.codex-obsidian.example.json ~/.goclaw/config.json
chmod 600 ~/.goclaw/config.json
```

至少替换以下值：

- `workspace.path`
- `models.providers.codex.runtime.workingDir`
- `channels.feishu.app_id`
- `channels.feishu.app_secret`
- `gateway.websocket.auth_token`
- `governance.reviewers.*.token_sha256`
- `governance.reviewers.*.team_user_id`（Team 模式建议配置；每个 Team 用户最多绑定一个 Reviewer key）
- `harness.root`
- `harness.project_id`
- `harness.knowledge_root`
- `harness.knowledge_backend`（`filesystem` 或 `git`）
- `ouroboros.root`
- `development.root`
- `development.worktree_root`
- `development.repo_path`
- `development.verification_sandbox`（Linux 填 root-owned `0755` wrapper 的绝对路径 argv 数组）
- `development.unsafe_host_verification=false`
- `team_control.root`
- `workstation.root`
- `memory.catalog.database_path`
- `memory.catalog.source_root`
- `memory.catalog.source_paths`
- 所有工具路径

生成网关 Token：

```bash
openssl rand -base64 48
```

为每个审批身份生成不同的 Reviewer Token，并把 SHA-256 摘要写入配置：

```bash
REVIEWER_TOKEN="$(openssl rand -hex 32)"
printf '%s' "$REVIEWER_TOKEN" | sha256sum
```

示例中的全零摘要会被配置校验直接拒绝，是故意设置的失败关闭占位符；不同身份也不能复用同一摘要。示例同时用 `team_user_id` 把 `erin-final` 等描述性 Reviewer key 绑定到个人 Token 解析出的 TeamControl principal；该绑定大小写不敏感且必须唯一。Gateway 仍会单独验证 Reviewer Token 和角色，最终审计身份记录真实 principal，而不是信任客户端填写的 Reviewer ID。原始 Token 不进入配置：Obsidian 保存到 SecretStorage；CLI 通过 `GOCLAW_REVIEWER_TOKEN` 或受保护的 RPC 参数传入。角色、法定人数和职责分离说明见 [`GOVERNANCE_CLOSED_LOOP_CN.md`](GOVERNANCE_CLOSED_LOOP_CN.md)。

`team_control.root`、`workstation.root`、`memory.catalog.database_path`、builtin memory database、`harness.root`、`ouroboros.root` 和 `development.root` 必须是中央主机的非同步目录。不要把它们放入知识根目录，也不要由两个 GoClaw 进程或共享盘并发写入。

中央单用户 Hand 的冻结 verifier 也默认失败关闭。Linux 在启动 GoClaw 前安装发布包 wrapper：

```bash
sudo install -d -o root -g root -m 0755 /usr/local/libexec/goclaw
sudo install -o root -g root -m 0755 \
  scripts/verify-sandbox-bwrap.sh \
  /usr/local/libexec/goclaw/verify-sandbox-bwrap.sh
```

示例配置使用 `"verification_sandbox": ["/usr/local/libexec/goclaw/verify-sandbox-bwrap.sh"]` 和 `"unsafe_host_verification": false`。前者是 argv 数组，首项必须是绝对、普通、可执行且不可被 group/other 写入的 wrapper；两项互斥。只有整个中央 Hand 已在一次性隔离 VM/容器中，才可清空数组并把危险开关显式设为 `true`。Team 模式仍不能用该配置绕过 `dev.task.run/repair/resume` 禁令。

启用 `team_control.enabled` 后，Gateway 自动进入 deny-by-default Team 模式，没有“继续开放旧全局 RPC”的兼容开关。旧的 process-global 配置、日志、渠道、会话、Browser 和 Cron RPC 会被拒绝；Harness、Memory Catalog 与 Ouroboros 使用项目 RBAC。上线前应把依赖旧 RPC 的运维脚本改为中央主机本地操作或显式项目资源接口。

### 多项目飞书路由

路由从精确到宽泛依次匹配：

```json
{
  "harness": {
    "project_id": "default",
    "routes": {
      "feishu:account-id:chat-id": "project-alpha",
      "feishu:chat-id": "project-alpha",
      "feishu": "default"
    }
  }
}
```

Obsidian 插件会在每条消息里显式发送 `project_id` 和 `topic_id`。飞书通常按群 `chat_id` 路由。启用 Harness 后，会话键固定为 `project:<project_id>:<topic_id>`，因此两个入口共享同一段历史。

路由只决定项目上下文，不授予 TeamControl 权限。飞书聊天不能替代个人 `GOCLAW_USER_TOKEN`、项目 RBAC 或 Reviewer Token。

### 初始化 TeamControl

MVP 和 Full Runtime 都先在中央主机建立首个管理员。以下命令要由计划运行中央 GoClaw 的同一操作系统用户执行；使用 `/srv` 时由系统管理员预先把父目录归属该用户。`bootstrap` 是唯一不经过 Gateway 的 team 命令，只能在空状态上执行一次：

```bash
install -d -m 0700 /srv/goclaw-runtime/team-control
install -d -m 0700 /srv/goclaw-secrets

goclaw team bootstrap \
  --root /srv/goclaw-runtime/team-control \
  --user admin \
  --name "Team Admin" \
  --email admin@example.com \
  --token-file /srv/goclaw-secrets/admin.token

export GOCLAW_USER_TOKEN="$(cat /srv/goclaw-secrets/admin.token)"
```

`--root` 必须与配置中的 `team_control.root` 一致。Token 明文只写入新建的 `0600` 文件，TeamControl 保存摘要。

然后在另一终端启动中央服务：

```bash
goclaw gateway run
```

确认 Gateway 可用后，为中央控制面准备受管 checkout，再创建团队、项目和仓库：

```bash
install -d -m 0750 /srv/goclaw-repositories
git clone \
  https://git.example.com/product/alpha-api.git \
  /srv/goclaw-repositories/alpha-api

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
  --local-path /srv/goclaw-repositories/alpha-api \
  --branch main
```

`--local-path` 是中央 Orchestrator Lite 冻结 base、导入证据和 `dev link-pr` 验证 Git commit 的受管仓库，不是成员电脑 `--repo ID=/path` 的映射。Production 应让中央服务用户拥有只读 Git 对象访问和必要的受控 fetch 能力；`link-pr` 自身绝不会 fetch。

为成员创建账户时，顺序必须是创建用户、加入团队、签发 Token、加入项目：

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
  --label workstation \
  --expires 2027-07-25T00:00:00Z \
  --token-file /srv/goclaw-secrets/dev-01.token

goclaw team project-member-add \
  --project project-alpha \
  --user dev-01 \
  --role developer \
  --domain billing \
  --capacity 10
```

重复到 10 人全部建立独立身份。团队/项目角色、业务域、容量与完整闭环见 [`TEAM_DEVELOPMENT_CN.md`](TEAM_DEVELOPMENT_CN.md)。

User 与个人 Token 是整个 TeamControl root 的全局对象。若同一用户属于多个活动团队，签发、列出、撤销其 Token 或修改全局用户状态的操作者，必须同时是该用户所有活动团队的 active owner/admin；只管理当前团队不够。新增用户仍按 `user-create → member-add → token-issue`，普通成员当前不能自助签发或撤销个人 Token。

项目内的 Assignment 也不是普通标签：developer 可以自指派，但把 Issue/WorkItem 指派给别人需要 `project.manage`。资源已有 active owner 后，同项目其他 developer 不能迁移其状态。开发任务的 freeze/revise/repair/enqueue/link-pr 只允许 task assignee 或项目管理者；最终 accept/cancel 还必须由项目管理者提供对应 Governance Reviewer Token。

### 注册 10 台工作站 Runner

每位成员先在自己的电脑上：

```bash
codex login
codex doctor
export GOCLAW_GATEWAY_HTTP_URL="https://goclaw.example.com"
export GOCLAW_GATEWAY_TOKEN="$(cat /secure/goclaw/gateway.token)"
export GOCLAW_USER_TOKEN="$(cat /secure/goclaw/me.token)"
```

然后生成本机独有 device key 并注册：

```bash
goclaw runner register \
  --id runner-dev-01-laptop \
  --name "Dev 01 Laptop" \
  --key-file /secure/goclaw/runner-dev-01.key \
  --project project-alpha \
  --capability codex \
  --capability go
```

device key 是工作站与中央验证端共同持有的 HMAC 共享秘密，中央凭据文件泄露者可伪造相应 Runner 证据；它不提供 TPM attestation、公钥设备证书或不可抵赖签名。高保证环境应把 TPM/硬件密钥身份列为后续强化项，并在当前版本严格限制中央凭据目录访问。

Linux 工作站先安装 `bubblewrap`，再把发布包中的 wrapper 安装为 root-owned `0755`；Runner 用户不得拥有其父目录或文件的写权限：

```bash
sudo install -d -o root -g root -m 0755 /usr/local/libexec/goclaw
sudo install -o root -g root -m 0755 \
  scripts/verify-sandbox-bwrap.sh \
  /usr/local/libexec/goclaw/verify-sandbox-bwrap.sh
```

启动工作循环：

```bash
goclaw runner work \
  --id runner-dev-01-laptop \
  --key-file /secure/goclaw/runner-dev-01.key \
  --work-root /local/goclaw-worktrees \
  --repo repo-api=/local/checkouts/alpha-api \
  --project project-alpha \
  --verification-sandbox /usr/local/libexec/goclaw/verify-sandbox-bwrap.sh \
  --heartbeat 30s \
  --poll 5s
```

`--repo ID=/path` 可重复，左侧必须匹配中央登记的 repository ID。Runner 每轮 ping/claim，执行期间维持 lease heartbeat；heartbeat 失败会取消本地 Codex。建议 heartbeat 不超过中央 lease 的三分之一。`runner work` 默认失败关闭：必须提供 `--verification-sandbox`，或仅在 Runner 整体已经运行于一次性隔离 VM/容器时改用 `--unsafe-host-verification`；两者互斥。

空闲时可更新 Runner 登记：

```bash
goclaw runner update \
  --id runner-dev-01-laptop \
  --name "Dev 01 Build Laptop" \
  --project project-alpha \
  --capability codex \
  --capability go

goclaw runner update --id runner-dev-01-laptop --disable
goclaw runner update --id runner-dev-01-laptop --enable
```

`--project`/`--capability` 整体替换旧集合；`--disable`/`--enable` 互斥。显示名可在执行中修改，但项目、能力和启停状态属于执行配置，有活动 lease 时会被拒绝。Team 模式禁止项目 `*`，并逐项目校验 Runner owner 的 `work_item.write`。

冻结开发任务的 `assignee_id` 会强制匹配 Runner owner；没有 assignee 的底层任务才只按项目与 capability 匹配。`business_domain` 和容量用于冻结前规划、校验与看板，不参与自动排程优化。

Runner 使用本机 Codex OAuth，在 revision/attempt 独立 worktree 中生成 diff 和签名 EvidenceBundle。它不会自动 commit、push、创建 PR、等待 CI、merge 或发布。完整命令、安全边界与故障恢复见 [`WORKSTATION_RUNNER_CN.md`](WORKSTATION_RUNNER_CN.md)。

Codex、内部 Git 和 verifier wrapper 都从最小环境白名单启动。Codex 每次运行使用独立 HOME/XDG/runtime/tmp，仅通过 `CODEX_HOME` 读取本机订阅 OAuth。GoClaw Token、SSH agent、Git askpass、Docker/Kubernetes 和云凭据路径等宿主能力变量永久剥离，`--allow-env` 不能覆盖；它只把其他显式变量交给 Codex，不会交给内部 Git 或冻结 verifier。Linux bubblewrap wrapper 断网、遮蔽 host home/run/tmp，并只让 worktree 与临时 HOME 可写。不要用日常管理员账户常驻 Runner；VM/容器也只挂载当前授权仓库、work root、device key 和 Codex OAuth，不得挂载整个 Home。

冻结任务完成四审和 freeze 后，由服务器构造可信 ExecutionPack：

```bash
goclaw dev enqueue TASK_ID \
  --priority 10 \
  --capability codex \
  --max-attempts 3
```

该桥接会校验 assignee、Issue/WorkItem、项目、仓库、base commit 和当前策略 hash；每个 WorkItem 必须恰有一个 active owner 且等于 task assignee，并且不能已绑定到另一个开发任务。队列 ID 固定为 `<TASK_ID>-r<REVISION>`，入队幂等键由服务器按 revision + execution bundle 派生；`dev enqueue` 没有客户端幂等 flag，也不能覆盖 ExecutionPack。每个 Runner 同时最多持有一个活动 lease。

### 初始化项目记忆

首次迁移建议保持 `memory.catalog.auto_ingest=false`，手动扫描并抽样：

```bash
goclaw memory catalog ingest /absolute/path/to/ObsidianVault \
  --source-root /absolute/path/to/ObsidianVault \
  --project project-alpha \
  --actor initial-migration

goclaw memory catalog status --project project-alpha
goclaw memory catalog list --project project-alpha --status pending
```

导入只生成待审批候选。完成审核后再考虑启用自动扫描；多个 `source_paths` 必须配置共同的 `source_root`。审批命令、Frontmatter、权威控制、备份和多机迁移见 [`LIBRARY_MEMORY_CN.md`](LIBRARY_MEMORY_CN.md)。

### 初始化 Go 原生 Ouroboros

```bash
goclaw ouroboros init
goclaw ouroboros list --project project-alpha
```

飞书可发起访谈、回答问题和结晶 Seed，但不能批准、编译或执行。Team Web Console 的“规格”页负责访谈进度，“审批”页负责认知分歧、Seed 与演化候选；完整流程见 [`OUROBOROS_GO_CN.md`](OUROBOROS_GO_CN.md)。

## 5. 初始化 Harness

```bash
goclaw harness init
goclaw harness status
goclaw harness evals
```

运行目录结构：

```text
~/.goclaw/harness/
├── active.json
├── versions/
├── candidates/
├── experiments/
├── evals/
│   ├── cases/
│   └── fixtures/
├── reports/
├── traces/
└── knowledge/proposals/
```

Harness 版本不可变；`active.json` 是原子切换指针。提升实验前必须同时满足：

1. Golden 与 Holdout 门槛。
2. 无关键用例失败。
3. 人类明确批准。
4. 提升者没有批准同一个 Candidate。
5. 当前版本仍等于实验的基准版本。

示例：

```bash
goclaw harness experiment create \
  --candidate v0.2.0 \
  --target components/instructions.md \
  --root-cause "完成声明缺少验证证据" \
  --summary "完成前强制输出可复核证据" \
  --fix-tags evidence_completeness

# 编辑命令输出中的 candidate_path，再执行：
goclaw harness experiment validate EXPERIMENT_ID
goclaw harness experiment approve EXPERIMENT_ID --reviewer alice
goclaw harness experiment promote EXPERIMENT_ID --reviewer alice

# 一步回滚到前一版本：
goclaw harness rollback --reviewer alice
```

命令型 Eval 默认不会执行。只有本机 CLI 显式传入 `--execute` 才允许运行用例中的命令；Gateway 即使收到 `execute=true` 也会拒绝。

当前验证会同时运行基线与 Candidate，拒绝基线已通过但 Candidate 失败的用例，并比较显式 Trace 中的令牌和延迟指标。Candidate 只能修改 Change Manifest 声明的组件，且不能触碰 Protected Paths 或降低评测策略。

## 6. 初始化开发执行链

开发运行时必须位于本机非同步目录，目标项目必须是 Git 仓库：

```bash
goclaw dev init
TASK_ID=task-project-alpha-orders-001
goclaw dev create \
  --id "$TASK_ID" \
  --repository-id repo-api \
  --assignee dev-01 \
  --spec deploy/dev-task.example.json \
  --json
goclaw dev list --project project-alpha --json
```

Team 模式必须在首次调用前生成并持久化稳定 `TASK_ID`；完全相同的 create 可按该 ID 重试，同 ID 不同请求会冲突。`dev list` 必须指定项目。模板里的 WorkItem ID 必须对应中央已登记的唯一 WorkItem；一个 WorkItem 不能再绑定其他开发任务，但同一 Issue 可以被多个任务共享。然后完成四类人工评审：

```bash
goclaw dev review "$TASK_ID" --kind scenario --decision approved --reviewer alice
goclaw dev review "$TASK_ID" --kind capacity --decision approved --reviewer alice
goclaw dev review "$TASK_ID" --kind risk --decision approved --reviewer alice
goclaw dev review "$TASK_ID" --kind cost --decision approved --reviewer alice
goclaw dev freeze "$TASK_ID" --reviewer alice
```

冻结后有两条显式执行路径：

```bash
# 单机 Orchestrator Lite Hand
goclaw dev run "$TASK_ID" --reviewer runner

# Full Runtime：服务器构造可信 ExecutionPack 并交给成员工作站
goclaw dev enqueue "$TASK_ID" \
  --priority 10 \
  --capability codex \
  --max-attempts 3
```

不要对同一 revision 同时使用两条路径。`dev enqueue` 要求任务已包含 team/project/repository、active assignee、WorkItem、base commit 与 PolicyBundle hash；每个 WorkItem 必须恰有一个 active owner 且等于 assignee。服务器用 `<TASK_ID>-r<REVISION>` 作为唯一队列身份，并从 revision + bundle 派生幂等键，拒绝策略漂移、重复变体和客户端自造 ExecutionPack。

查看事件、证据和状态：

```bash
goclaw dev events "$TASK_ID" --json
goclaw dev show "$TASK_ID" --json
```

单机 Hand 路径中，DoneGate 失败时使用 `repair`；通过后必须人工验收，之后才能创建本地提交：

```bash
goclaw dev repair "$TASK_ID" --reviewer runner
goclaw dev accept "$TASK_ID" --reviewer alice --comment "已核对证据"
goclaw dev commit "$TASK_ID" --reviewer alice --message "fix: accepted task"
```

`goclaw dev commit` 是最终验收后的显式本地命令，不是 Runner 自动 commit。工作站路径应先读取签名 Evidence/patch、完成人工评审，再由既有 Git/PR 流程提交。

Full Runtime 工作站路径不同：`runner.complete` 后，Gateway 自动把证据导入 Orchestrator Lite，重新校验 revision、execution bundle、base/head、no-commit、diff SHA、路径、冻结检查和范围策略，必要时重做独立模型审查并运行 DoneGate。通过后任务进入 `awaiting_acceptance`，WorkItem/Issue 进入 `verifying`。可在 Obsidian 开发页验收，或由同时具备项目 `project.manage` 和 `task_accept` Governance 角色的人调用 Team RPC。

若 Full Runtime 需要 repair/revise，CLI 与 Obsidian 会先读取当前任务并发送 `expected_revision`。旧 revision 仍为 queued 时先取消队列项；failed 也可以取消，leased 会拒绝取消和修订：

```bash
WORKSTATION_TASK_ID="${TASK_ID}-r${TASK_REVISION}"
goclaw runner cancel "$WORKSTATION_TASK_ID" \
  --reason "创建新的受审修订"
goclaw dev repair "$TASK_ID" \
  --reason "DoneGate 失败，需要调整任务契约"
```

repair 会先把 `in_progress`/`verifying` WorkItem 退回 `blocked`，再创建 `review_pending` 的新 revision。必须重新四审、freeze、enqueue，WorkItem 才重新进入执行。原始 RPC 调用方必须显式传 `expected_revision`；响应不明时先 `dev show`，不要盲目再次读取最新 revision 后新建一轮修订。

例如准备一个仅当前用户可读、位于 Vault 之外的参数文件：

```json
{
  "id": "TASK_ID",
  "reviewer_token": "REPLACE_FROM_SECRET_MANAGER",
  "rationale": "已核对导入证据、diff、冻结验证、策略和独立审查",
  "counterargument": "本地验证仍不能覆盖全部生产流量与外部依赖",
  "evidence_refs": ["workstation:evidence", "orchestrator:donegate"]
}
```

```bash
chmod 600 /secure/goclaw/dev-task-accept.json
goclaw team rpc dev.task.accept \
  --params /secure/goclaw/dev-task-accept.json
```

Gateway 从 `GOCLAW_USER_TOKEN` 取得 principal，并用 `team_user_id`（若配置）找到描述性 Reviewer 策略；请求中的 Reviewer ID 不被信任。调用者必须同时具备 `project.manage` 和 `task_accept` Governance 角色。`dev.task.accept` 会再次检查导入证据和 diff 未漂移，成功后把当前任务的关联 WorkItem 置为 `done`。共享 Issue 只有在全部关联任务都为 `done` 且全部关联 WorkItem 都为 `done` 时才 `resolved`；`cancelled` 永不算成功。取消分支让 Issue 保持 open/verifying/blocked，等待重新指派或由有权人员给出明确 resolution 后另行关闭。参数文件含 Reviewer Token，用完应安全删除或交由临时秘密注入机制；不得放进 Vault、Git 或聊天。

控制面不会替工作站 commit，且 `goclaw dev commit` 会拒绝由 Workstation 导入的证据。验收后由负责人下载并核对签名 patch：

```bash
WORKSTATION_TASK_ID="${TASK_ID}-r${TASK_REVISION}"
goclaw runner evidence "$WORKSTATION_TASK_ID"
goclaw runner patch "$WORKSTATION_TASK_ID" \
  --output "/secure/goclaw/review/$TASK_ID.patch"

git -C /local/checkouts/alpha-api switch \
  -c "goclaw/$TASK_ID-r$TASK_REVISION" \
  "$FROZEN_BASE"
git -C /local/checkouts/alpha-api apply \
  --check "/secure/goclaw/review/$TASK_ID.patch"
git -C /local/checkouts/alpha-api apply \
  "/secure/goclaw/review/$TASK_ID.patch"
git -C /local/checkouts/alpha-api add -A
git -C /local/checkouts/alpha-api commit
```

提交信息必须包含完整 trailers；应在编辑器中按任务实际值填写：

```text
Task-ID: task-123
Project-ID: project-alpha
Task-Revision: 2
Repository-ID: repo-api
Correlation-ID: corr-123
Policy-Bundle: <POLICY_BUNDLE_HASH>
Work-Item: work-456
Issue: issue-789
```

`Repository-ID`、`Correlation-ID`、`Policy-Bundle` 仅在任务对应字段非空时要求；所有关联 WorkItem 和 Issue 都必须各有一行。开发者随后按既有流程 push 并创建 PR。`dev link-pr` 不会 fetch，因此在调用前由 CI 或管理员让中央受管 checkout 能解析该 commit，例如：

```bash
git -C /srv/goclaw-repositories/alpha-api fetch \
  origin "goclaw/$TASK_ID-r$TASK_REVISION"

COMMIT_SHA="$(git -C /local/checkouts/alpha-api rev-parse HEAD)"
goclaw dev link-pr "$TASK_ID" \
  --commit "$COMMIT_SHA" \
  --url "$PR_URL"
```

Team 模式的 `link-pr` 使用个人 Token principal，要求项目 `artifact.write`，并只允许 task assignee 或项目管理者。服务会验证 commit 位于受管 `local_path`、继承 frozen base，`base..commit` 的 binary diff 与 accepted Workstation patch 精确相同（只忽略 Git `index ...` 元数据行），且 trailers 完整。成功后自动创建 TeamControl commit/PR Artifact，并建立队列 Task/Repository/WorkItem/Issue CorrelationLink；CI、release 与 regression Artifact 仍需人工或适配器登记。

该命令只校验本地 Git commit，并把 PR URL 作为严格语法字段登记：URL 必须是无用户名密码、无 query、无 fragment 的绝对 HTTP(S) 地址。它不会调用 provider API，也不会 fetch、push、创建、批准或 merge PR，因此不能证明远端 PR head、内容、状态或 commit 归属。相同 commit/URL 重试幂等；若任务已绑定不同 commit 或 URL 则拒绝覆盖。

`gateway_allow_execution` 只适用于未启用 TeamControl 的单用户部署。Team 模式的 `dev.task.run/repair/resume` 无条件禁用，即使显式设为 `true` 也不会放行；团队唯一执行路径是 `dev enqueue` → Workstation 持久队列。完整任务契约、状态机、恢复和证据说明见 [`ORCHESTRATOR_LITE_CN.md`](ORCHESTRATOR_LITE_CN.md)。

## 7. 接入飞书

在飞书开放平台创建企业自建应用：

1. 启用机器人。
2. 为机器人配置接收消息及发送消息所需权限。
3. 添加消息接收事件。
4. 选择长连接接收事件；本实现不要求公网 Webhook。
5. 发布应用并将机器人加入目标群。

启动 GoClaw 后，私聊默认使用 `pairing`：

```bash
goclaw pairing list feishu
goclaw pairing approve feishu PAIRING_CODE
```

生产环境建议：

- `group_policy` 使用 `whitelist`。
- 保持 `dm_policy` 为 `pairing` 或 `allowlist`。
- 不要把 `app_secret` 提交到 Git。

## 8. 启用 Team Web Console

Web Console 已嵌入 `goclaw`，无需单独安装。默认地址：

```text
https://goclaw.example.com/dashboard/
```

首次进入时输入 Gateway Token 与个人 Team Token。Gateway 验证后只发放
HttpOnly、SameSite=Strict 短期 Cookie；Token 不进入浏览器持久存储。高风险审批时，
从右上角身份菜单输入 Reviewer Token，它只存在于当前页面内存。

生产反向代理必须把 `/dashboard/`、`/assets/`、`/auth/session`、`/rpc` 和
`/ws` 指向同一个 Gateway Origin。验证步骤见
[`TEAM_WEB_CONSOLE_CN.md`](TEAM_WEB_CONSOLE_CN.md)。

### 可选：安装 Obsidian 适配器

从源码安装：

```bash
./scripts/install-obsidian-plugin.sh /absolute/path/to/ObsidianVault
```

或者将这四个文件复制到：

```text
ObsidianVault/.obsidian/plugins/goclaw-project-console/
├── manifest.json
├── main.js
├── styles.css
└── versions.json
```

重启 Obsidian，在“设置 → 第三方插件”启用 “GoClaw Project Console”。

插件设置：

- Gateway：本机使用 `ws://127.0.0.1:28789/ws`。
- Project ID：与配置中的项目一致。
- Topic ID：不同讨论可使用 `inbox`、`architecture`、`release` 等。
- Gateway Token：写入 Obsidian SecretStorage，不进入 `data.json`。
- Team User Token：每位成员自己的个人 Token，写入独立 SecretStorage 项，用于 TeamControl 项目 RBAC。
- Reviewer ID：非 Team 模式与 `governance.reviewers` 的键一致；Team 模式不信任该客户端字段，而是从 Team User Token principal 解析身份，并按 map key 或 `team_user_id` 找 Reviewer 策略。
- Reviewer Token：写入独立 SecretStorage 项，不进入 `data.json`，也不能用 Gateway Token 替代。

侧边栏提供：

- 聊天：接收增量、工具与最终事件。
- 记忆：Catalog 状态、已批准检索、候选审批、到期续期/撤回和来源校验和。
- 规格：Ouroboros 访谈、歧义度、Seed 结晶、评估与演化。
- 审批：需求/评估分歧、利益相关方冲突、Seed/演化候选、知识提案、Harness 实验和开发任务审批。
- 开发：四类评审、冻结、DoneGate 状态、修复和最终验收。
- 进度：项目 Trace、耗时和错误状态。
- Harness：当前版本、门槛、组件、最近实验和认证回滚。
- 团队：成员负载、项目任务、Bug、Runner/租约、策略层、文档和组件概览；这是中央状态的只读投影。

插件不会伪造 Obsidian Sync 的远端状态，底部只报告 Vault 已就绪；具体同步健康度应在 Obsidian Sync 面板确认。

Team User Token、Gateway Token 和 Reviewer Token 都不能写入 Vault。Vault 同步只承载 Markdown，不承载 TeamControl 队列、lease 或秘密。

## 9. 远程连接与 TLS

不要把未加密的 `ws://` 暴露到局域网或公网。示例 `deploy/Caddyfile.example` 将 `/ws` 反向代理为 `wss://`。

Team Web Console 入口：

```text
https://goclaw.example.com/dashboard/
```

浏览器先通过 `/auth/session` 换取 HttpOnly Cookie，随后同源 `/rpc` 请求附带 CSRF Header，`/ws` 复用同一 Cookie。服务器拒绝跨站 Origin。可选 Obsidian 适配器仍使用 WebSocket 子协议携带 Token；旧原生客户端可使用 `Authorization: Bearer`。

启用 Gateway 认证后，HTTP `/rpc` 同样强制：

```text
Authorization: Bearer <gateway-token>
```

健康检查 `/health` 保持无敏感信息、可供本机探针读取；不要通过反向代理公开其他未使用端点。

`goclaw team` 与 `goclaw runner` CLI 支持：

```bash
export GOCLAW_GATEWAY_HTTP_URL="https://goclaw.example.com"
export GOCLAW_GATEWAY_TOKEN="$(cat /secure/goclaw/gateway.token)"
export GOCLAW_USER_TOKEN="$(cat /secure/goclaw/me.token)"
```

CLI 会向该 base URL 的 `/rpc` 发请求。URL 必须是绝对 `http`/`https` URL，不能包含凭据、query 或 fragment。远程 10 台工作站不得把 HTTP RPC 直接暴露到公网，因为 Gateway Bearer、个人 Token 和首次注册的 device key 都会经过该连接。Production 推荐直接使用受信 HTTPS；也可在受控 VPN 内访问或通过 SSH 本地转发连接中央本地端口。

无论采用哪种方式，都要验证连接端到端加密和服务端身份，不能仅依赖“内网地址看起来可信”。

## 10. systemd

修改：

```text
deploy/systemd/goclaw.service
```

重点：

- `User` 必须是执行过 `codex login` 的用户。
- `HOME` 必须指向该用户的 Home，否则 Codex 看不到登录状态。
- `ReadWritePaths` 只开放 GoClaw 状态、Workspace、Vault、Ouroboros/Harness/Development Runtime、目标仓库和 worktree 根目录；不要开放整个 Home 或文件系统根目录。
- 中央 `team_control.root` 与 `workstation.root` 只允许该服务用户访问。
- 每台成员电脑的 Runner 必须由执行过本地 `codex login` 的对应成员账户运行，不能集中使用管理员 OAuth。
- Runner 环境文件显式设置 `CODEX_HOME` 为该成员执行 `codex login` 后的绝对目录；每次执行另建隔离 HOME/XDG。
- Runner 服务使用专用低权限 OS 用户，并强制通过 root-owned `0755` 的 bubblewrap wrapper 执行冻结 verifier；不要在普通宿主上使用 `--unsafe-host-verification`。

安装：

```bash
sudo cp dist/goclaw /opt/goclaw/goclaw
sudo cp deploy/systemd/goclaw.service /etc/systemd/system/goclaw.service
sudo systemctl daemon-reload
sudo systemctl enable --now goclaw
sudo journalctl -u goclaw -f
```

每台 Linux Runner 另行安装 wrapper、环境文件和用户级 systemd unit：

```bash
sudo install -d -o root -g root -m 0755 /usr/local/libexec/goclaw
sudo install -o root -g root -m 0755 \
  scripts/verify-sandbox-bwrap.sh \
  /usr/local/libexec/goclaw/verify-sandbox-bwrap.sh
install -d -m 0700 "$HOME/.config/goclaw" "$HOME/.config/systemd/user"
install -m 0600 deploy/runner.env.example "$HOME/.config/goclaw/runner.env"
install -m 0644 deploy/systemd/goclaw-runner.service.example \
  "$HOME/.config/systemd/user/goclaw-runner.service"
systemctl --user daemon-reload
systemctl --user enable --now goclaw-runner
```

启动前编辑 `runner.env` 的 Token、Runner、仓库、项目和 `CODEX_HOME`。unit 从 `GOCLAW_VERIFICATION_SANDBOX` 传入必需的 wrapper；两种 verifier 选项不能同时出现。

## 11. 验收

```bash
curl http://127.0.0.1:28789/health
goclaw team --help
goclaw runner --help
goclaw memory catalog status --project project-alpha
goclaw harness status
goclaw harness traces
goclaw ouroboros list --project project-alpha
goclaw dev list --project project-alpha --json
```

端到端检查：

1. 导入一个普通目录或 Git 工作树中的 Markdown，确认记录只进入 `pending`，且聊天不能把它当作 approved memory。
2. 在 Team Web Console“记忆/审批”页批准该记录，再询问相关问题，确认 Trace 带有对应 memory ID、来源 URI 和 Git revision。
3. 修改同一 Markdown 并重新导入，确认产生同一 Work 的新版本；批准后旧版变为 `superseded`。
4. 创建已过期 context，确认不会自动进入提示词；跨项目替代请求应失败。
5. 从 Team Web Console 发送“读取当前项目的架构决策”，确认 GoClaw 调用只读知识工具。
6. 在飞书继续追问，确认回答能引用前一入口的项目会话。
7. 要求修改 ADR，确认 `08-reviews/inbox/` 出现提案，目标 ADR 未直接改变。
8. 在 Team Web Console 审批，确认目标 Markdown 更新且提案移动到 `approved`。
9. 手动改动目标后再批准旧提案，确认收到同步冲突。
10. 用错误 Reviewer Token 审批，确认失败；用无对应角色的身份审批，确认失败。
11. 创建并验证 Harness 实验；未批准时提升应失败；同一批准人提升也应失败。
12. 提升后新会话 Trace 使用新版本；使用 `harness_rollback` 身份和完整理由回滚。
13. 发起 Ouroboros 访谈；同一订阅模型承担多个评估器时，确认出现人工分歧裁决，而不是伪装为独立共识。
14. 确认开放的利益相关方冲突和阻塞问题会阻止 readiness；Seed 未批准时不能编译。
15. 高风险 Seed 只批准一次时仍保持 pending，达到双人法定人数后才激活。
16. 批准 Seed 后编译任务，确认任务仍处于四类评审阶段；单人超过评审类型上限时应失败。
17. 运行任务，确认变更只出现在独立 worktree，且证据目录包含 diff、Policy、Verification、Falsifier、Prediction、Kill Check、独立审查和 DoneGate。
18. 在 DoneGate 后手工改动文件，确认最终验收被拒绝；参与过任务评审的人也不能做最终验收。
19. 制造同模型评估争议，确认裁决前不能演化且不进入结果分母；裁决后仍需独立任务验收。
20. 记录同一任务先通过后回滚，确认参考类只计一个最新样本，审计历史保留两条并带 `supersedes_id`。
21. 生成后继候选，确认批准前 active Seed 未变化；连续通过数不足时不能宣告收敛。
22. 分别用管理员、developer、reviewer 和 viewer 的个人 Token 请求项目资源，确认服务器端 RBAC 拒绝越权操作。
23. Team 模式请求 `config.get`、日志、渠道、会话、Browser 或 Cron 等 process-global RPC，确认失败关闭；Harness、Memory 与 Ouroboros 使用无权限项目时也应拒绝。
24. 让一个测试用户同时加入两个团队；只管理其中一个团队的管理员不能签发/列出/撤销其 Token 或改变全局状态，两边共同管理员才可操作。
25. 查看 Team Web Console“团队”页，确认 10 名成员的业务域、容量、负载与 Runner 在线状态来自同一项目，切换未授权项目应失败。
26. 注册一台测试 Runner，确认 device key 只存在于本地 `0600` 文件和中央凭据目录，知识目录中没有任何副本。
27. 空闲时验证 `runner update` 的项目/能力整体替换和 disable/enable；持有活动 lease 时这些执行配置更新必须失败。
28. 用稳定 task ID 重试同一 create，确认不重复事件；同 ID 不同请求冲突，`dev list` 缺 `--project` 时拒绝。
29. 入队一个冻结测试任务，确认零个/多个 active WorkItem owner、owner≠assignee、或 WorkItem 已绑定另一开发任务都被拒绝；合法 revision 的队列 ID/幂等键由 revision + bundle 派生，只有项目/capability/owner 匹配的 Runner 能 claim。
30. 缺少 `--verification-sandbox` 与 `--unsafe-host-verification` 时确认 `runner work` 失败，两者同时提供也失败；wrapper 非绝对、不可执行或可被 group/other 写时也拒绝。验证 bubblewrap 内无网络、host home/run/tmp 被遮蔽，只有 worktree 与临时 HOME 可写。
31. 验证 Codex 只看到最小环境、隔离 HOME/XDG 与指定 `CODEX_HOME`；普通 `--allow-env` 测试变量只对 Codex 可见，内部 Git/verifier 不可见；GoClaw Token、SSH agent、Docker/Kube/云凭据路径即使加入 allowlist 也不可见。
32. 中断测试 Runner，确认 lease 过期后按尝试次数重新排队或失败，且 Runner 最终显示离线。
33. 确认 runner cancel 只接受 queued/failed，leased 明确拒绝；旧 revision queued/leased 时 revise 同样拒绝。
34. 从 in_progress/verifying 发起 repair，确认 WorkItem 先 blocked，新 revision 重新四审/freeze/enqueue 后才回到 in_progress，且过期 `expected_revision` 不会重复加 revision。
35. 完成测试任务，确认签名 EvidenceBundle 包含 task/revision/lease、base、diff、验证和策略哈希，并自动导入 Orchestrator Lite 重验和 DoneGate。
36. 用两个任务共享 Issue：验收第一个时 Issue 不 resolved；取消另一个只取消其 WorkItem，Issue 保持 open/verifying/blocked。只有取消分支被重新指派并最终使全部 Task/WorkItem 都为 done，或有权人员明确另行关闭时，Issue 才结束。
37. 确认 freeze/revise/enqueue/link-pr 的 assignee/manager 边界，以及 accept/cancel 的 `project.manage + Governance role`；其他 developer 不能改本人未负责资源。Team 模式调用 run/repair/resume 时，无论 `gateway_allow_execution` 为何都应拒绝。
38. 确认 Runner 没有自动 commit、push、建 PR 或 merge，控制面也拒绝提交 Workstation 导入证据；在正常 Git 流程应用 accepted patch、提交并创建测试 PR。
39. 在中央 `local_path` 不可见 commit 时确认 `dev link-pr` 失败；fetch 后分别用非 frozen-base 后代、篡改 diff、缺 trailer，以及带凭据/query/fragment 的 URL 验证拒绝，再用合法 commit/URL 确认自动生成 commit/PR Artifact 及 Task/Repository/WorkItem/Issue CorrelationLink。
40. 确认 `dev link-pr` 没有 provider API，不会 fetch/push、创建/批准/merge PR 或读取远端 PR/head/CI 状态；CI/release/regression 仍由人工或适配器登记。
41. 在真实 Obsidian Desktop 验收团队页的宽/窄窗口、空态、拒绝态和单模块失败态；云浏览器的本地地址访问曾被安全策略阻止，不能替代该项。

## 12. 备份与恢复

备份：

- Obsidian Vault：由现有同步/版本控制体系负责。
- `team_control.root`：用户、团队、项目、仓库、RBAC、Issue/Work/Assignment、Artifact、Document、Component 和 Policy；必须与其他状态一致备份。
- `workstation.root`：Runner 凭据、任务、lease、幂等记录和签名 Evidence；不得通过 Vault Sync 同步。
- `memory.catalog.database_path`：保存审批状态、版本、权威项、来源和流通记录；使用 SQLite 一致性备份，不通过 Vault Sync 同步。
- builtin memory database：保存本地发现索引，可按保留策略备份或重建。
- `~/.goclaw/ouroboros` 或配置的 `ouroboros.root`：保留规格事件链与不可变 Seed。
- `~/.goclaw/harness`：保留版本、实验、报告和 Trace。
- `~/.goclaw/development` 或配置的 `development.root`：保留事件链与 EvidencePackage。
- `~/.goclaw/sessions`：保留跨渠道项目会话。
- `~/.goclaw/config.json`：进入秘密管理系统，不进入源码库。

恢复顺序：

1. 恢复 Vault。
2. 恢复 Catalog 的一致性备份；只恢复 Vault 会丢失审批、权威与使用历史。
3. 恢复 TeamControl 与 Workstation 的同一时间点备份；先保持所有 Runner 停止，检查 task/lease 状态。
4. 恢复 Ouroboros、Harness、Development Runtime 与 Sessions；恢复 worktree 前先核对对应 Git commit。
5. 恢复配置、Gateway Token、各成员个人 Token 和各审批人的 Reviewer Token；原始 Token 不应从配置摘要反推或在成员之间共享。
6. 中央对话 Provider 和每台成员 Runner 分别以原用户恢复 Codex 登录，或重新执行 `codex login`。
7. 只启动一个中央 GoClaw 进程，先恢复/检查过期 lease，再逐台启动 Runner。
8. 运行全部验收命令。
