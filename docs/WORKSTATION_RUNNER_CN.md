# 工作站 Runner 部署与运维

GoClaw Team Runtime `0.8.0-pilot.1` 允许每位成员在自己的电脑上使用本地 Codex OAuth 执行中央分配的项目任务。中央服务负责身份、授权、队列、租约和证据校验；工作站负责 Git worktree、Codex 执行、确定性验证和签名证据。

## 1. 执行边界

```text
中央 TeamControl + Gateway
  ├─ 校验个人 GOCLAW_USER_TOKEN 与项目 RBAC
  ├─ 保存 Runner 公共登记和 device key 凭据
  ├─ 保存 secret-free ExecutionPack
  ├─ 原子 claim、lease、heartbeat、complete/fail
  └─ 验证并保存签名 EvidenceBundle
                     │
                     │ TLS + Gateway/个人身份
                     ▼
成员电脑 Runner
  ├─ 本地仓库映射
  ├─ 本地 Codex OAuth
  ├─ revision/attempt 隔离 worktree
  ├─ diff、范围检查、冻结验证
  └─ device key 签名证据
```

Runner 不接收 Codex OAuth，也不会把它上传到中央服务。ExecutionPack 不包含运行时秘密；本地命令需要的秘密只能由工作站自己的受控环境解析。

## 2. 跨平台支持边界

Runner 不把三种宿主描述成同等级沙箱，而是使用显式 profile：

| 成员电脑 | `strict` | `codex-delegated` | 建议 |
|---|---|---|---|
| Linux amd64/arm64 | 支持 | 支持 | 高保证任务选 strict + bwrap |
| Windows amd64/arm64 | 原生拒绝；WSL2 guest 支持 | 原生支持 | 高风险任务用 strict WSL2 |
| macOS Intel/Apple Silicon | 原生拒绝；Lima guest 支持 | 原生支持 | 高风险任务用 strict Lima |

`strict` 是缺省值并保留 `goclaw.runner/linux-v1` 合同、受审 verifier
wrapper、固定 PATH、credential read-deny canary 和网络关闭。缺少边界时
失败关闭。

`codex-delegated` 是显式降级：GoClaw 仍执行 canonical repository/work
root、独立 worktree、allowed/denied path、Git 安全审计、最小环境、Codex
named permission canary、最终 diff/symlink/no-commit 拒绝和 Windows
进程树取消，但不提供 GoClaw 自身的 OS 进程/网络沙箱。它必须同时满足
Team Control 项目 policy allow 与 Runner 明确 capability，不能由错误或
缺失参数触发。不信任的仓库或高权限机器不得使用该 profile。

WSL2 必须关闭 interop、Windows PATH 和 automount，并把全部运行文件放在
发行版虚拟磁盘；`/mnt/*`、drvfs 和 9p 被拒。Lima 必须关闭 host mount，
virtiofs、9p、sshfs、fuse.lima 等共享文件系统被拒。模板分别位于
`deploy/wsl2/` 和 `deploy/lima/`。

## 3. 每台电脑的前置条件

- 安装与中央版本一致的 `goclaw` `0.8.0-pilot.1` Linux 包。
- 安装 Git 和 Codex CLI；Linux Production 另外安装 `bubblewrap`。
- 使用该成员自己的 ChatGPT/Codex 订阅执行 `codex login`。
- 使用专用操作系统用户；更高风险仓库使用专用 VM 或容器。
- 准备目标仓库的本地 checkout；其 Git 对象库中必须存在任务冻结的 `base_commit`。
- 能通过 HTTPS/WSS 访问中央 Gateway。
- 拥有该成员自己的 `GOCLAW_USER_TOKEN`。
- 该成员在目标项目具有 `owner`、`maintainer` 或 `developer` 角色；只读 `viewer` 和仅评审 `reviewer` 不能注册执行 Runner。
- 为 Runner 工作目录预留足够磁盘空间。

Linux 先从发布包安装受审 verifier wrapper。该文件必须由 root 持有且不可由 Runner 用户修改：

```bash
sudo install -d -o root -g root -m 0755 /usr/local/libexec/goclaw
sudo install -o root -g root -m 0755 \
  scripts/verify-sandbox-bwrap.sh \
  /usr/local/libexec/goclaw/verify-sandbox-bwrap.sh
sudo test "$(stat -c '%U:%G:%a' \
  /usr/local/libexec/goclaw/verify-sandbox-bwrap.sh)" = "root:root:755"
```

wrapper 固定使用 root 管理的 `/usr/bin/bwrap` 和固定安全 `PATH`，关闭网络与额外 namespace，遮蔽 host `/home`、`/root`、`/run`、`/tmp`，把宿主文件系统只读挂载，并只让任务 worktree 和该次验证的临时 HOME 可写。启动前会用 `--goclaw-doctor` 实际创建 namespace；仅文件存在但内核禁止 user namespace 仍会失败。

检查：

```bash
goclaw version
git --version
codex --version
codex login status
```

不要从管理员电脑复制 Codex OAuth 目录。10 位成员应各自登录，并各自承担订阅权限、使用限额和审计责任。专用用户/VM/容器只挂载授权仓库、Runner work root、device key 和该用户自己的 Codex OAuth；不要挂载整台电脑的 Home、SSH 密钥目录或其他项目。

## 4. 中央服务配置

中央 `config.json` 至少启用：

```json
{
  "gateway": {
    "websocket": {
      "enabled": true,
      "auth_token": "REPLACE_WITH_AT_LEAST_24_CHARACTERS"
    }
  },
  "team_control": {
    "enabled": true,
    "root": "/srv/goclaw-runtime/team-control"
  },
  "workstation": {
    "enabled": true,
    "root": "/srv/goclaw-runtime/workstation",
    "lease_duration_seconds": 120,
    "runner_offline_seconds": 300,
    "default_max_attempts": 3,
    "max_idempotency_receipts": 128
  }
}
```

要求：

- `workstation.enabled` 依赖 `team_control.enabled`。
- `lease_duration_seconds` 必须在 30～3600 秒之间。
- `runner_offline_seconds` 不得短于 lease，且最多 86400 秒。
- `default_max_attempts` 为 1～20。
- 两个 root 都必须在中央主机的非同步本地目录。

这些目录是单进程文件存储，不能由两个 GoClaw 实例或网络共享盘并发写入。

## 5. 首个管理员与个人 Token

仅在 TeamControl 尚无用户时执行一次：

```bash
goclaw team bootstrap \
  --user admin \
  --name "Team Admin" \
  --email admin@example.com \
  --label bootstrap \
  --token-file /secure/goclaw/admin.token
```

该命令：

- 创建第一个活动用户。
- 生成个人访问 Token。
- 只把明文写入新的 `0600` 文件。
- TeamControl 仅保留摘要，初始化后不能再次 bootstrap。

管理员后续命令需要：

```bash
export GOCLAW_USER_TOKEN="$(cat /secure/goclaw/admin.token)"
```

为成员签发独立 Token：

```bash
goclaw team token-issue \
  --user dev-01 \
  --label workstation-laptop \
  --expires 2027-07-25T00:00:00Z \
  --token-file /secure/goclaw/dev-01.token
```

`--expires` 可省略；生产环境建议配置到期时间、轮换流程和撤销审计。明文 Token 文件应通过安全渠道交给本人，而不是提交到 Vault 或团队群。

## 6. 工作站本地配置

成员电脑的 GoClaw 配置要指向中央 Gateway，并包含连接级 Gateway Token。个人身份通过环境变量单独发送：

```bash
export GOCLAW_GATEWAY_HTTP_URL="https://goclaw.example.com"
export GOCLAW_GATEWAY_TOKEN="$(cat /secure/goclaw/gateway.token)"
export GOCLAW_USER_TOKEN="$(cat /secure/goclaw/me.token)"
```

`GOCLAW_GATEWAY_HTTP_URL` 必须是绝对 `http`/`https` URL，不得包含用户名密码、query 或 fragment；CLI 会在其路径后追加 `/rpc`。远程工作站必须使用 `https://`、受控 VPN 或 SSH 隧道，不能把个人 Token 和 device key 注册流量放在公网明文 HTTP 上。

在 shell history、系统日志和进程管理器中避免直接写明文。生产环境应从操作系统 Keychain、Secret Service、systemd credential 或同等级秘密管理器注入。没有环境覆盖时，CLI 会回退到本地 GoClaw 配置中的 Gateway 地址和连接 Token。

为本地仓库准备稳定映射：

```text
repo-api=/Users/me/src/alpha-api
repo-web=/Users/me/src/alpha-web
```

左侧必须是中央 TeamControl 登记的 `repository_id`，右侧必须是该电脑上的绝对路径。中央保存的 `local_path` 不会自动转换为每台电脑的本地路径。

## 7. 注册 Runner

每台电脑只生成自己的 device key：

```bash
goclaw runner register \
  --id runner-dev-01-laptop \
  --name "Dev 01 Laptop" \
  --key-file /secure/goclaw/runner-dev-01.key \
  --project project-alpha \
  --capability codex \
  --capability go \
  --execution-profile strict \
  --verification-sandbox /usr/local/libexec/goclaw/verify-sandbox-bwrap.sh
```

行为：

- device key 使用安全随机数生成并写入新的 `0600` 文件。
- 注册请求通过个人 Token 绑定当前用户。
- Runner 只允许声明该用户有权访问的项目。
- 控制面登记 Runner 的 owner、项目、capability 和 key ID。
- Linux CLI 自动追加版本化 capability `goclaw-runtime-linux-v1`、架构和
  substrate capability；strict/delegated 还分别追加稳定的 profile
  capability，客户端不能用参数伪装成另一种 `GOOS`。
- `--verification-sandbox` 记录 `runner_goos`、`runner_goarch`、
  `host_profile`、`isolation_backend=bwrap` 和 wrapper 的
  `sandbox_sha256`。三人试点的 `pilot check` 会验证这些 metadata。
- device key 用于 EvidenceBundle 的 HMAC-SHA256；它不进入 Vault 或 Git。

device key 是 HMAC 共享秘密，不是非对称私钥：注册时必须通过 TLS/受控隧道发送给中央服务，中央以 `0600` 凭据文件保存用于验证；公开 Runner 记录只暴露 key ID。

如果首次注册在网络响应前后状态不明，保留已生成的 key，并用相同参数重试：

```bash
goclaw runner register \
  --id runner-dev-01-laptop \
  --name "Dev 01 Laptop" \
  --key-file /secure/goclaw/runner-dev-01.key \
  --project project-alpha \
  --capability codex \
  --capability go \
  --execution-profile strict \
  --verification-sandbox /usr/local/libexec/goclaw/verify-sandbox-bwrap.sh \
  --reuse-key
```

不要删除 key 后用同一个 Runner ID 静默生成新 key；应先完成明确的禁用/轮换流程。

先停止工作循环并确认 Runner 没有活动 lease，再轮换 device key：

```bash
goclaw runner rotate-key \
  --id runner-dev-01-laptop \
  --new-key-file /secure/goclaw/runner-dev-01.next.key
```

控制面拒绝正在持有活动 lease 的 Runner 轮换。成功后用新 key 文件重启 Runner，并安全归档/销毁旧 key。网络结果不明确时保留候选 key，并加 `--reuse-key` 原参数重试。

Runner owner 可以维护登记信息：

```bash
# 显示名可单独更新
goclaw runner update \
  --id runner-dev-01-laptop \
  --name "Dev 01 Build Laptop"

# project/capability 是整体替换；Runner 必须先处于空闲
goclaw runner update \
  --id runner-dev-01-laptop \
  --project project-alpha \
  --project project-shared \
  --capability codex \
  --capability go

goclaw runner update --id runner-dev-01-laptop --disable
goclaw runner update --id runner-dev-01-laptop --enable
```

`--id` 必填；`--name`、`--project`、`--capability`、`--disable`、`--enable` 至少提供一项，`--disable` 与 `--enable` 互斥。`--project` 和 `--capability` 会整体替换旧集合，不是追加；Team 模式禁止项目 `*`，新增的每个项目都要求 Runner owner 具有 `work_item.write`。有活动 lease 时，项目、capability 和启停状态都不能修改；显示名只是 cosmetic，可单独修改。enable 后状态先变成 `offline`，下次成功 ping 才变成 online。owner 和 device key 不能通过此命令修改。

## 8. 先执行 Runner Doctor

注册和每次服务启动前，用与 `runner work` 相同的本地参数执行：

```bash
goclaw runner doctor \
  --key-file /secure/goclaw/runner-dev-01.key \
  --work-root /home/alice/.local/share/goclaw/workstation-work \
  --repo repo-api=/home/alice/src/alpha-api \
  --codex-command /usr/local/bin/codex \
  --execution-profile strict \
  --verification-sandbox /usr/local/libexec/goclaw/verify-sandbox-bwrap.sh \
  --json
```

原生 Windows/macOS 的 delegated doctor 使用
`--execution-profile codex-delegated`，并且不传两个 verifier isolation
参数。JSON 的 `ready` 必须为 `true`；delegated 的
`verification-isolation` 固定为 `warn`，明确记录其降级姿态。Doctor 检查：

- GOOS/GOARCH、选定 execution profile、substrate 和版本化 runtime contract；
- WSL interop、Windows PATH、`/mnt/*` 与 Linux guest 共享挂载；
- Git、Codex、`codex login status`、该成员的 `CODEX_HOME`、device key 和 work root；
- 每个仓库是否为 guest-local Git worktree，以及 local Git config 是否包含
  hook、filter、fsmonitor、include、credential helper 或外部 diff/merge；
- device key、`CODEX_HOME`、repo/work root 是否属于 Runner uid，路径父级
  是否只有 root/Runner 可控；wrapper 是否 root-owned、绝对、不可被
  group/other 写，并能真正启动无网络 bwrap。

默认输出适合人工阅读；`--json` 是部署 Gate 的机器接口。任何 `fail` 都以
非零状态退出。`runner work` 也会先运行同一套检查，因此不能绕过预检。

## 9. 启动工作循环

单仓库：

```bash
goclaw runner work \
  --id runner-dev-01-laptop \
  --key-file /secure/goclaw/runner-dev-01.key \
  --work-root /home/alice/.local/share/goclaw/workstation-work \
  --repo repo-api=/home/alice/src/alpha-api \
  --project project-alpha \
  --codex-command /usr/local/bin/codex \
  --execution-profile strict \
  --verification-sandbox /usr/local/libexec/goclaw/verify-sandbox-bwrap.sh \
  --heartbeat 30s \
  --poll 5s
```

多仓库：

```bash
goclaw runner work \
  --id runner-dev-01-laptop \
  --key-file /secure/goclaw/runner-dev-01.key \
  --work-root /home/alice/.local/share/goclaw/workstation-work \
  --repo repo-api=/home/alice/src/alpha-api \
  --repo repo-web=/home/alice/src/alpha-web \
  --project project-alpha \
  --codex-command /usr/local/bin/codex \
  --execution-profile strict \
  --verification-sandbox /usr/local/libexec/goclaw/verify-sandbox-bwrap.sh
```

`runner work` 没有 `--capability` 参数；capability 在 `runner register` 时声明。
注册、doctor 和 work 的 profile 必须一致。strict 启动默认失败关闭：必须
选择 `--verification-sandbox`，或在整个 Runner 已位于一次性隔离 VM/容器
时显式选择 `--unsafe-host-verification`；两者互斥。delegated 不接受这两个
参数，避免制造“已经隔离”的错误印象。

可选参数：

| 参数 | 默认值 | 说明 |
|---|---:|---|
| `--codex-command` | `codex` | 本地 Codex CLI；systemd 模板使用预先解析的绝对路径 |
| `--codex-model` | `default` | 使用本地账号默认订阅模型 |
| `--execution-profile` | `strict` | `strict` 或显式降级的 `codex-delegated` |
| `--allow-env` | 空 | 仅向 Codex 显式放行确有必要且不属于宿主能力边界的敏感变量，可重复 |
| `--verification-sandbox` | 必选之一 | 绝对 verifier wrapper argv 前缀；Runner 追加 `WORKTREE SANDBOX_HOME -- COMMAND...` |
| `--unsafe-host-verification` | false | 仅在一次性隔离 VM/容器内显式允许 verifier 直跑；与 `--verification-sandbox` 互斥 |
| `--timeout` | `21600` | Codex 和每条验证命令的最大秒数 |
| `--heartbeat` | `30s` | 租约心跳，最小 5 秒 |
| `--poll` | `5s` | 空队列轮询，最小 1 秒 |
| `--once` | false | 最多处理一个任务；空队列时立即返回 |
| `--project` | 必填 | 领取的项目队列 |

Strict 的 Codex、内部 Git 和 wrapper 使用固定安全 PATH；delegated 为了支持
本机工具链使用宿主 PATH，但每个解析后的 executable 必须是普通文件，Windows
还会检查 ACL。两个 profile 都从最小环境白名单启动，使用隔离
HOME/XDG/TMP；Codex 主进程通过 `CODEX_HOME` 使用该成员已有的本机订阅
OAuth，模型命令的 named permission profile 对真实目录设置 `deny`，并在
调用前运行对应 OS 的 read-deny canary。上下文取消、超时或 lease 心跳失败
会终止 Unix 进程组或 Windows 进程树。控制面 Token、Runner/Codex Token、
SSH agent/Git askpass、Docker、Kubernetes、云凭据、Kerberos 和 Vault
宿主能力变量始终剥离；`--allow-env` 不能放行。allowlist 永不进入内部 Git
或冻结 verifier。

## 10. Claim、lease 与 heartbeat

循环行为：

1. `runner.ping` 更新 Runner 在线状态。
2. `runner.claim` 原子领取一个项目和 capability 都兼容的排队任务；ExecutionPack 有 `assignee_id` 时还必须等于 Runner owner。
3. 中央创建带 attempt、claimed/heartbeat/expiry 时间的 lease。
4. Runner 执行期间按 `--heartbeat` 调用 `runner.heartbeat` 延长 lease。
5. 心跳失败会取消本地执行，避免无租约的工作继续冒充有效结果。
6. 成功时调用 `runner.complete`；失败时调用 `runner.fail`。
7. 中央验证 lease、执行包哈希、device key 签名和幂等键后保存结果。

同一个 Runner ID 同时只能持有一个活动 lease，避免两个工作循环用同一设备身份并行领取多个任务。不要启动重复的 Runner 进程。

建议让 heartbeat 小于 lease 的三分之一。例如 lease 为 120 秒时使用 30 秒。不要把 heartbeat 设置得接近 lease，否则短暂网络抖动就可能使任务过期。

中央进程周期性恢复过期 lease：

- 未超过 `max_attempts` 的任务重新入队。
- 达到上限的任务进入失败状态。
- 超过 `runner_offline_seconds` 没有心跳的 Runner 标记为离线。

租约恢复防止一台电脑断电后永久占用任务，但不是跨 Leader 共识；仍只能运行一个中央写入进程。

## 11. 从冻结开发任务入队

推荐入口不是由客户端自造 ExecutionPack，而是把完成四审并冻结的 Orchestrator Lite 任务交给服务器编译：

```bash
goclaw dev enqueue TASK_ID \
  --priority 10 \
  --capability codex \
  --capability go \
  --execution-profile strict \
  --max-attempts 3
```

`dev.task.enqueue` 会在服务器端重新读取冻结任务，检查
team/project/repository、活动 assignee、Issue/WorkItem、WorkItem 状态、
base commit 和当前 PolicyBundle hash，然后构造可信 ExecutionPack。每个
WorkItem 必须恰有一个 active owner Assignment，且 owner 必须等于冻结任务
assignee。客户端提供的 `execution_pack` 不会被信任。缺少
`runner.execution_profiles` policy 时只有 strict 可入队；delegated 要求
resolved policy 显式列出 `codex-delegated`，Gateway 还会追加对应 required
capability。可选的 `runner.target_version`、`runner.target_release_id` 会
同样转成 required capability，未达到目标的 Runner 不会领取；
`runner.rollout_paused=true` 同时阻止新 enqueue 与新 claim。工作循环心跳
只上报非秘密的当前 version/release/profile，项目列表据此显示 rollout
状态。

冻结 revision 的队列 ID 由服务器固定为 `<DEV_TASK_ID>-r<REVISION>`，幂等键由服务器按开发任务 ID、revision 和 execution bundle hash 派生；`goclaw dev enqueue` 不提供客户端 `--idempotency-key`。客户端不能通过换 ID 或 key 把同一不可变 revision 重复入队。入队会把 WorkItem 迁移到执行状态，登记 Task，并连接 WorkItem、Issue 与 Repository。冻结开发任务会把 `assignee_id` 放进 ExecutionPack，Claim 强制 Runner owner 与其一致；只有绕过 `dev.task.enqueue` 创建的无 assignee 底层任务，才可能由任意项目/capability 匹配 Runner 领取。业务域和容量仍不参与自动排程优化。

每个 WorkItem 只能绑定一个开发任务；若另一个任务包含相同 WorkItem ID，enqueue 会拒绝。Issue 可以跨多个任务共享，不能据某一个队列项的结果提前终结。

如果冻结后发现范围需要修订，而该 revision 尚未开始执行，可先取消其队列项：

```bash
WORKSTATION_TASK_ID="${TASK_ID}-r${TASK_REVISION}"
goclaw runner cancel "$WORKSTATION_TASK_ID" \
  --reason "创建新的受审修订"
goclaw dev repair "$TASK_ID" \
  --reason "DoneGate/评审发现需要修改任务契约"
```

`runner cancel` 只接受 `queued` 或 `failed`；`leased` 有活动执行时明确拒绝，不能作为抢占。任务 assignee 或项目管理者可以取消 queued/failed 项；同项目其他 developer 不能取消。`completed`/`cancelled` 已是终态，也不能再次作为新的取消请求。命令为本次调用内的网络重试生成同一个 key，也可显式传 `--idempotency-key` 供跨进程重试；相同 key 与相同请求返回原结果，同 key 改 reason 会冲突。

`dev revise`/Team `dev repair` 也会检查旧 revision 队列：仍为 queued 或 leased 时拒绝。修订请求必须带 `expected_revision`；CLI 和 Obsidian 会先读取当前 revision 后发送，原始 RPC 客户端必须自行传入。修订会把 `in_progress`/`verifying` WorkItem 先退到 `blocked`，清空旧四审和冻结状态；新 revision 重新四审、freeze、enqueue 后，WorkItem 才再次进入 `in_progress`。

## 12. 本地执行与证据

领取任务后，Runner 会：

1. 验证 ExecutionPack SHA-256。
2. 用 `repository_id` 查找本地仓库映射。
3. 在 checkout 前拒绝 local Git hook/filter/fsmonitor/include/credential/external
   driver 配置，并检查 frozen tree 中所有 `.gitattributes`；Git 命令始终覆盖
   hooks、fsmonitor、credential helper、file protocol 和 submodule recurse。
4. 验证冻结 `base_commit` 确实存在且解析为同一 SHA。
5. 在 `work_root/<task>/<revision-attempt>/` 创建独立分支和 worktree。
6. 运行 `codex exec --json`，使用本机 OAuth。
7. 收集 Codex JSONL 和 stderr。
8. 在证据目录创建 `0700` 的一次性 verifier HOME/XDG/TMP；通过 `--verification-sandbox` wrapper 执行冻结 argv。Linux wrapper 清空环境、关闭网络、遮蔽 host home/run/tmp，把宿主根只读挂载，仅让 worktree 与临时 HOME 可写。
9. 再次审计 Git 配置，防止 Codex 或 unsafe host verifier 在执行中植入宿主执行入口。
10. 在验证全部结束后收集最终 changed files、binary diff 和 diff SHA-256。
11. 对最终工作树执行 allowed/denied path、scope policy 和 no-automatic-commit 检查。
12. 生成并用 device key 签名 EvidenceBundle。

证据包含 task/project/lease/attempt、执行包哈希、base/head、分支、diff、
检查结果、Artifact、Trace ID、Harness 版本、策略哈希、冻结
`execution_profile` 和实际 `runner_goos`/`runner_goarch`/`host_profile`。

Runner 故意不做：

- `git commit`
- `git push`
- 创建或合并 Pull Request
- 等待或批准 CI
- 发布或回滚 release
- 删除执行 worktree

如果 Codex 在执行过程中创建了 commit，`no-automatic-commit` 检查会失败。人类必须先查看 diff 和证据，再按团队流程提交、推送和评审。

`--allow-env` 只影响 Codex，且不能越过永久禁止的宿主能力变量。Runner 内部 Git 使用最小环境；冻结 verifier 必须进入 wrapper，不再直接继承 host。Linux bubblewrap 基线阻断 verifier 网络和宿主可写面，但 Codex Hand 本身仍依赖 Codex 的 workspace sandbox；Production 继续使用最小权限专用用户，并可用 VM/容器增加整机纵深隔离。

读取签名证据或下载经过 diff SHA-256 校验的补丁：

```bash
goclaw runner list --project project-alpha
goclaw runner evidence WORKSTATION_TASK_ID
install -d -m 0700 /secure/goclaw/review
goclaw runner patch WORKSTATION_TASK_ID \
  --output /secure/goclaw/review/WORKSTATION_TASK_ID.patch
```

对于 `dev enqueue` 生成的任务，`WORKSTATION_TASK_ID` 是 `<DEV_TASK_ID>-r<REVISION>`；`dev link-pr` 的参数仍使用原始 `DEV_TASK_ID`。

`runner.complete` 成功保存证据后，Gateway 会把完成证据导入 ExecutionPack 中绑定的 Orchestrator Lite 任务，而不是信任 Runner 自报的 DoneGate。导入会重新校验 task revision、execution bundle、base/head、no-commit、diff SHA、changed paths、冻结检查和范围策略，必要时重新执行独立模型审查，再由 Go DoneGate 决定：

- 通过且要求人工验收：开发任务进入 `awaiting_acceptance`，TeamControl WorkItem/Issue 进入 `verifying`。
- 未通过：开发任务进入 repair pending、blocked 或 failed；TeamControl 资源进入相应失败路径。

最终验收通过 Obsidian 开发页或 `dev.task.accept` RPC 完成，操作者必须同时具备项目 `project.manage` 和 `task_accept` Governance 角色。它会再次验证导入证据和 diff 未漂移；成功后把当前任务的关联 WorkItem 置为 `done`。共享 Issue 只有在全部关联开发任务及 WorkItem 都是 `done` 时才会自动 `resolved`；任何 `cancelled` 分支都不算成功，Issue 保持 open/verifying/blocked，等待重新分配或明确另行关闭。这仍不会 commit、push 或创建 PR；签名 patch 由负责人在原工作站或受控评审环境进入既有 Git 流程。

负责人应从 frozen base 派生正常开发分支，应用 accepted patch，并使用任务要求的 Task/Project/Revision/Repository/Correlation/Policy/WorkItem/Issue trailers 提交和创建 PR。待中央受管 Repository `local_path` 已能解析该 commit（例如 push 后由 CI/管理员 fetch），执行：

```bash
goclaw dev link-pr TASK_ID \
  --commit <COMMIT_SHA> \
  --url <ABSOLUTE_HTTP_OR_HTTPS_PR_URL>
```

服务会验证 commit 继承 frozen base、累计 binary diff 与 accepted patch 精确一致（只忽略 Git `index` 元数据行）且 trailers 完整，然后自动登记 commit/PR Artifact 与 Task/Repository/WorkItem/Issue CorrelationLink。PR URL 只能是无凭据、无 query、无 fragment 的绝对 HTTP(S) 地址；服务不调用 provider API，不能证明远端 PR head、内容或状态。它不会 fetch、push、创建、批准或合并 PR，也不会等待 CI；完整 trailer 列表见 [`TEAM_DEVELOPMENT_CN.md`](TEAM_DEVELOPMENT_CN.md)。

## 13. 守护进程与升级

生产工作站应使用操作系统服务管理器运行 `goclaw runner work`，并满足：

- 进程用户就是执行过 `codex login` 的用户。
- 在服务环境中把 `CODEX_HOME` 明确设为该用户执行 `codex login` 后的绝对目录；Codex 主进程把它作为订阅 OAuth 来源，模型命令由 named permission profile 和每次执行前 canary 拒绝读取该目录。
- Linux 将发布包中的 `scripts/verify-sandbox-bwrap.sh` 安装到 `/usr/local/libexec/goclaw/verify-sandbox-bwrap.sh`，保持 `root:root 0755`，并在 systemd `ExecStart` 中传 `--verification-sandbox`。
- systemd unit 的 `ExecStartPre` 运行 `runner doctor`，`KillMode=control-group`
  配合 Runner 自身的 Unix 进程组取消，避免停机留下孙进程。
- 该用户不应同时持有管理员 SSH key、其他项目 checkout 或无关生产凭据。
- `GOCLAW_USER_TOKEN` 通过秘密管理器注入。
- 只开放本地 checkout、work root、device key 和必要工具。
- 日志不得输出 Token 或 device key。
- 崩溃后可以重启，但不要并行启动两个相同 Runner ID 的实例。
- 所有子进程使用最小环境；`--allow-env` 仅给 Codex，且永远不能放行控制面 Token、SSH/Docker/Kube/云凭据路径等宿主能力。

Linux 安装示例：

```bash
sudo install -d -o root -g root -m 0755 /usr/local/libexec/goclaw
sudo install -o root -g root -m 0755 \
  scripts/verify-sandbox-bwrap.sh \
  /usr/local/libexec/goclaw/verify-sandbox-bwrap.sh
install -d -m 0700 "$HOME/.config/goclaw"
install -m 0600 deploy/runner.env.example "$HOME/.config/goclaw/runner.env"
install -d -m 0700 "$HOME/.config/systemd/user"
install -m 0644 deploy/systemd/goclaw-runner.service.example \
  "$HOME/.config/systemd/user/goclaw-runner.service"
systemctl --user daemon-reload
systemctl --user enable --now goclaw-runner
```

Windows 成员使用 [`../deploy/wsl2/README_CN.md`](../deploy/wsl2/README_CN.md)
准备 WSL2；macOS 成员使用
[`../deploy/lima/README_CN.md`](../deploy/lima/README_CN.md) 准备 Lima。发布
脚本同时生成 linux/amd64 和 linux/arm64 Runner 包，并对 Windows/macOS
控制 CLI 做交叉编译检查。

### 13.1 受控本地 release 切换

Team Control 先创建并批准 immutable Runner release；新记录必须包含正数
`size_bytes`。Runner 不自动下载 `uri`，operator 用可信渠道把对应平台产物
放到本机绝对路径，再执行：

```bash
goclaw runner release stage-from-control RELEASE_ID \
  --project project-alpha \
  --work-root /home/alice/.local/share/goclaw/workstation-work \
  --artifact /absolute/staging/goclaw-runner
```

该命令从 Team Control 读取 approved release，并在写入前后核对
ID、version、OS、arch、release protocol `1`、size 和 SHA-256。legacy
零大小记录只能读取，不能 stage。URI 不会被 fetch，从而避免未实现完整
SSRF/redirect/DNS policy 时产生隐式网络面。

停止 Runner 并确认没有 active lease，然后原子选择新版本：

```bash
goclaw runner release activate RELEASE_ID \
  --work-root /home/alice/.local/share/goclaw/workstation-work
goclaw runner release status \
  --work-root /home/alice/.local/share/goclaw/workstation-work
goclaw runner release path \
  --work-root /home/alice/.local/share/goclaw/workstation-work
```

service manager 应只启动 `release path` 返回且刚刚重新校验过的 binary。完成
doctor 和 `--once` smoke 后，停止工作循环并确认健康：

```bash
goclaw runner release confirm RELEASE_ID \
  --work-root /home/alice/.local/share/goclaw/workstation-work
```

smoke 失败时保持停止状态并回滚到前一个已确认版本：

```bash
goclaw runner release rollback \
  --work-root /home/alice/.local/share/goclaw/workstation-work
```

运行中存在 `.runner-process.lock` 时，activate/confirm/rollback 全部拒绝。
release mutation 也使用独占锁；staged 目录和状态用同文件系统
temp + rename 原子发布。陈旧锁不会被自动猜测清理，必须由 operator 在确认
没有 Runner/release 进程后处理。本 Wave 不负责 service 自动重启、远程下载、
平台代码签名或 installer。

## 14. 验收清单

每台电脑至少执行：

```bash
codex login status
goclaw runner doctor \
  --key-file /secure/goclaw/runner-dev-01.key \
  --work-root /home/alice/.local/share/goclaw/workstation-work \
  --repo repo-api=/home/alice/src/alpha-api \
  --codex-command /usr/local/bin/codex \
  --verification-sandbox /usr/local/libexec/goclaw/verify-sandbox-bwrap.sh \
  --json
goclaw runner work \
  --id runner-dev-01-laptop \
  --key-file /secure/goclaw/runner-dev-01.key \
  --work-root /home/alice/.local/share/goclaw/workstation-work \
  --repo repo-api=/home/alice/src/alpha-api \
  --project project-alpha \
  --codex-command /usr/local/bin/codex \
  --verification-sandbox /usr/local/libexec/goclaw/verify-sandbox-bwrap.sh \
  --once
```

然后由管理员确认：

1. `runner.list`/Obsidian 团队页显示正确 owner、项目、capability 和 last heartbeat。
2. 未授权项目无法注册或领取。
3. repo ID 无映射时执行失败且有签名失败证据。
4. base commit 不匹配时失败，不会退回到当前 HEAD。
5. 超出 allowed path 的变更失败。
6. 冻结验证命令失败时任务失败。
7. 断网后心跳失败会取消执行，lease 最终被中央恢复。
8. 完成任务只留下 worktree、diff 和证据，不产生 commit/push/PR。
9. 缺少两种 verifier 选项时 runner work 失败；两者同时提供也失败。Linux wrapper 路径必须是绝对、可执行且不可被 group/other 写，`--goclaw-doctor` 必须实际通过。
10. 验证 wrapper 内无网络、host home/run/tmp 被遮蔽、宿主根只读，只有 worktree 和临时 HOME 可写；`--allow-env` 不进入内部 Git/verifier。
11. Codex 只看到最小环境、隔离 HOME/XDG 与指定 `CODEX_HOME`；GoClaw Token、SSH agent、Docker/Kube/云凭据路径即使加入 `--allow-env` 也不可见。
12. 有活动 lease 时，`runner update --project/--capability/--disable` 被拒绝；空闲时 update/disable/enable 正常。
13. queued/failed 队列可由 assignee/项目管理者用 `runner cancel` 取消；leased、completed、cancelled 均拒绝，重复同 key 请求返回同一结果。
14. 旧 revision queued/leased 时 revise/repair 被拒绝；取消 queued 项后，repair 把 verifying/in_progress WorkItem 退到 blocked，并要求新 revision 重新四审、freeze、enqueue。
15. 完成证据自动进入 Orchestrator Lite 重验和 DoneGate；门通过后仍须具备 `project.manage + task_accept` 的操作者验收。当前 WorkItem 变为 done，共享 Issue 只在所有 Task/WorkItem 都 done 后 resolved。
16. 取消其中一个共享 Issue 任务时，仅其 WorkItem 进入 cancelled，Issue 保持 open/verifying/blocked，必须重新安排或明确另行关闭。
17. 应用 accepted patch 形成 commit/PR 后，`dev link-pr` 只在中央 `local_path` 可见 commit 时成功；篡改 patch、缺 trailer 或非 frozen-base 后代均被拒绝。URL 带凭据/query/fragment 会拒绝；成功只登记 URL，不代表远端 PR/head/status 已验证。
18. Windows/macOS 原生 strict 失败；delegated 只有 policy allow、Runner
    capability 与任务 profile 同时匹配才可领取。Doctor 明确给出无 OS
    process/network isolation 的 warning；Windows 的可写 ACL/reparse point
    或 macOS 的不安全 owner/mode 会失败。
19. local Git 配置包含 hook/filter/fsmonitor/include/credential/external
    diff/merge，或 frozen `.gitattributes` 启用相应 driver 时，在 checkout
    前失败。
20. 超时、Ctrl-C 或 lease 心跳失败后，Codex/verifier 的孙进程也被清理；
    隔离 TMP 不回退到宿主 `/tmp`。
21. release stage 拒绝 URI/相对路径、错误 size/hash/platform/arch/protocol
    和非 approved 中央记录；运行中拒绝 activate。新版本未 confirm 时可以
    回滚，但不能再次回到未确认版本。

## 15. 故障排查

### `GOCLAW_USER_TOKEN is required`

在当前进程环境注入成员自己的 Token。不要使用管理员 Token 代跑所有工作站。

### `repository is not registered on this workstation`

检查 `--repo ID=/absolute/path` 左侧 ID 是否与 ExecutionPack 的 `repository_id` 完全一致。

### `frozen base commit` 不存在

在本地 checkout 安全获取对应 Git 对象，再重试。Runner 不应擅自把基线改成最新分支。

### `device_key_path must not be readable or writable by group or others`

在 Linux、WSL2 或 Lima guest 内执行：

```bash
chmod 600 /secure/goclaw/runner-dev-01.key
```

### lease 持续过期

检查中央和工作站时钟、网络、Gateway TLS、`--heartbeat` 与中央 lease 配置。先停止重复的 Runner 实例，再恢复任务。

### Codex 要求重新登录

使用运行 Runner 的同一 Linux guest 用户执行 `codex login` 和
`codex login status`。不要把 OAuth 文件复制到中央服务。

## 16. 当前边界

- 队列、Runner 和 lease 已持久化，但文件存储只保证单进程内并发和原子替换，不支持多中央进程。
- device key 是中央与工作站都持有的 HMAC 共享秘密，不是 TPM attestation、硬件证明、公钥设备证书或不可抵赖签名；中央凭据泄露者能够伪造该 Runner 的证据。
- `runner_goos`、`host_profile` 和 `sandbox_sha256` 是受控 CLI 生成并由
  试点流程比对的 metadata，不是远程硬件证明；绕过 CLI 的恶意 owner
  仍可能伪造。因此三人试点要求管理员核对 doctor 输出和同版 wrapper hash。
- 没有 GitHub/GitLab/Jira 双向同步。
- 没有自动 commit、push、PR、CI wait、merge 或远程 release 下载；当前
  release 功能只做经中央 identity pin 的本地 stage 和原子选择/回滚。
- `dev link-pr` 已实现外部 commit 的本地校验与 PR URL 登记，但不执行 fetch、push、PR 创建/批准/merge，不调用 provider API，也不验证远端 PR head/内容/状态。
- Codex 使用 worktree sandbox、最小环境和 run 隔离 HOME/XDG；Linux frozen verifier 额外进入无网络 bubblewrap wrapper。wrapper 不是整个 Runner 的 microVM，专用用户与最小挂载仍由部署者负责。
- strict 的 `--verification-sandbox` 是默认必需安全边界；
  `--unsafe-host-verification` 只允许已经处于一次性隔离 VM/容器的 Runner
  显式使用。codex-delegated 是项目批准的降级 profile，不提供 GoClaw OS
  级进程/网络隔离，不能用 Doctor warning 冒充 sandbox 证明。
- 原生 Windows/macOS delegated 已通过交叉生产构建，但目标平台 ACL、
  process-tree cancel、Codex permission CLI 和真实 OAuth smoke 仍必须在
  试点设备补现场证据；高保证试点继续使用 WSL2/Lima strict。
- Runner 不负责最终验收；签名证据证明来源和完整性，不自动证明方案正确。
- 冻结任务的 assignee 会匹配 Runner owner，但 `business_domain` 和容量仍不参与自动排程优化；这些字段用于计划、校验与看板。
- Team 模式无条件拒绝远程 `dev.task.run/repair/resume`；`development.gateway_allow_execution` 仅单用户模式有效。团队唯一执行路径是 `dev enqueue` + Workstation 持久队列。

团队对象、策略、组件和日常闭环见 [`TEAM_DEVELOPMENT_CN.md`](TEAM_DEVELOPMENT_CN.md)。
