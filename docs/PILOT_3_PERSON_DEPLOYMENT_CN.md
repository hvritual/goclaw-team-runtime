# 三人技术试点部署与使用

版本：GoClaw Team Runtime `0.8.0-pilot.1`。

状态：技术试点候选，不是生产放行声明。本文只描述当前代码已提供的命令与
失败关闭边界。没有完成文末的真机、凭据、TLS、备份恢复和浏览器验收时，
不得把系统标记为“已上线”。

## 1. 试点拓扑与固定约束

试点只有一个中央写者、一个项目和三名自然人：

```text
                  HTTPS / WSS
        ┌──────────────┼──────────────┐
        │              │              │
      Alice           Bob           Carol
   Runner + Codex  Runner + Codex  Runner + Codex
        │              │              │
        └──────────────┼──────────────┘
                       ▼
             Caddy → 单实例 Gateway
                       │
       TeamControl / Workstation / Development
       Harness / Ouroboros / Session / Catalog
```

固定试点合同：

- 恰好 `3` 个 active 项目成员、`3` 个个人 Team Token、`3` 个不同 owner
  的 Runner；每人只持有自己的 Token、Reviewer Token、device key 和
  Codex OAuth。
- 恰好 `1` 个项目。示例使用 `pilot-alpha`。
- 项目 ID 必须匹配
  `^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`：总长不超过 64，首字符为字母或
  数字，禁止冒号。TeamControl 的通用 ID 虽更宽，三人试点仍采用这个更窄
  的项目会话边界。
- 中央状态目录只允许一个 GoClaw 进程写入，不放在 NFS、SMB、Obsidian
  Sync、iCloud、OneDrive 或其他多写共享盘。
- Runner 执行底座统一为 Linux：原生 Linux、WSL2 Linux guest 或 Lima
  Linux guest。原生 Windows/macOS 只可运行控制 CLI，不能运行
  `runner work`。
- Team 模式不允许中央 Gateway 直接执行开发任务。唯一执行路径是
  `dev create → 四审 → freeze → enqueue → runner work → accept`。
- 自动 commit、push、建 PR、等待 CI、merge、release 均不在试点自动化
  范围内。

其中“恰好 3 枚 active 个人 Team Token”是操作审计门，不是
`goclaw pilot check` 当前会自动统计的项目门。放行前应由 Carol 分别调用
`team.token.list`（例如
`goclaw team rpc team.token.list --params /secure/goclaw/token-list-alice.json`，
其中参数文件是 `{"user_id":"alice"}`；Bob、Carol 同理），
确认 Alice、Bob、Carol 各只有一枚未撤销、未过期的个人 Token；不能从
“3 个 active 成员”推断 Token 数量。

三人职责固定为：

| 成员 | 项目职责 | 任务治理职责 |
|---|---|---|
| Alice | developer，业务域与容量 owner | `scenario` + `capacity`；Harness approve |
| Bob | developer，业务域与容量 owner | `risk` + `cost`；Harness promote/rollback |
| Carol | project owner，试点运维与放行 | 不参加四审；独立 `task_accept` / `task_cancel` |

Carol 的 Runner 保持在线以验证第三个独立设备身份和故障恢复，但试点任务不
应同时由 Carol 实现并由 Carol 最终验收。当前运行时的职责分离确保 Carol
不能参加四审后再 final；“final 不验收自己的实现”仍是本试点的操作规程。

## 2. 前置条件

中央 Linux 主机：

- 受支持的 Linux amd64/arm64、Git、Caddy、`age`；
- 一个专用非 root 服务用户，例如 `goclaw`;
- 中央受管 Git checkout，且其冻结基线中包含
  `docs/waves/wave-registry.json` 和该 Registry 指向的 active Wave plan；
- DNS 与受信 TLS 证书；
- 如需 Web Console 对话或 Ouroboros 模型调用，中央服务用户还需要一个
  单独、获许可的 ChatGPT Workspace/Codex 登录。它不替代三位成员各自的
  Runner 登录。

每个 Runner 的 Linux 执行环境：

- Git、Codex CLI、`bubblewrap` 和对应架构的 Linux `goclaw`；
- guest-local 仓库、work root、device key 与 `CODEX_HOME`；
- 能以本人账号成功执行 `codex login status`；
- 管理员安装且 Runner 用户不可修改的 bubblewrap wrapper。

构建或解包后先核对：

```bash
sha256sum -c SHA256SUMS-0.8.0-pilot.1.txt
goclaw version
```

构建脚本的 Linux Runtime 目标包名如下；只有实际构建并核对 checksum 后才
能视为可用产物：

```text
goclaw-team-runtime-linux-amd64-0.8.0-pilot.1.tar.gz
goclaw-team-runtime-linux-arm64-0.8.0-pilot.1.tar.gz
```

Windows/macOS 控制 CLI 仅做编译检查，不是 Runner 发布合同。只生成源码包可用：

```bash
SOURCE_ONLY=1 ./scripts/build-release.sh
```

## 3. 生成中央配置

以 [`deploy/config.codex-obsidian.example.json`](../deploy/config.codex-obsidian.example.json)
为完整配置底稿，用
[`deploy/governance.pilot-3.fragment.json`](../deploy/governance.pilot-3.fragment.json)
**整体替换**其中的 `governance` 对象。不要把两个 `reviewers` map 递归
合并，否则底稿中的示例身份会残留。

治理片段中的 `aa…`、`bb…`、`cc…` 是三个互不相同、非零且格式合法的
已知占位摘要，不是秘密，也不能用于运行。先在受保护目录为每个人生成不同
Reviewer Token：

先创建只有中央服务用户可访问的暂存目录，再进入该用户的 login shell。以下
秘密生成命令都在 `goclaw` 用户 shell 中执行；交付个人秘密后应按组织的
密钥托管流程移走或销毁中央暂存副本。

```bash
sudo install -d -o goclaw -g goclaw -m 0700 /srv/goclaw-secrets
sudo -iu goclaw
umask 077

openssl rand -hex 32 | tr -d '\n' > /srv/goclaw-secrets/alice.reviewer.token
openssl rand -hex 32 | tr -d '\n' > /srv/goclaw-secrets/bob.reviewer.token
openssl rand -hex 32 | tr -d '\n' > /srv/goclaw-secrets/carol.reviewer.token
openssl rand -hex 32 | tr -d '\n' > /srv/goclaw-secrets/gateway.token

sha256sum /srv/goclaw-secrets/alice.reviewer.token
sha256sum /srv/goclaw-secrets/bob.reviewer.token
sha256sum /srv/goclaw-secrets/carol.reviewer.token

exit
```

把三个结果的首列分别替换到 `alice`、`bob`、`carol` 的
`token_sha256`；把 `gateway.token` 的原文完整写入
`gateway.websocket.auth_token`。只有完成替换后，才把最终配置以
`goclaw:goclaw 0600` 安装到中央服务用户的：

```text
/home/goclaw/.goclaw/config.json
```

同时至少完成这些替换：

- `gateway.host=127.0.0.1`、`gateway.port=8080`；
- `gateway.websocket.host=127.0.0.1`、`port=28789`、
  `enable_auth=true`，`auth_token` 使用至少 32 个随机字节；
- `team_control.enabled=true`、`workstation.enabled=true`；
- `harness.enabled=true`、`harness.project_id=pilot-alpha`；
- `ouroboros.enabled=true`；
- `development.enabled=true`、
  `development.gateway_allow_execution=false`、
  `development.require_human_final_approval=true`、
  `development.independent_review=true`；
- `development.repo_path` 指向中央受管 checkout；
- TeamControl、Workstation、Harness、Ouroboros、Development 和
  worktree root 均为中央本地绝对路径，且位于知识/Vault 之外；
- `development.verification_sandbox` 指向中央需要本地执行时使用的受审
  wrapper；与 `unsafe_host_verification` 不能同时启用；
- 飞书在没有真实 App 凭据前保持 `enabled=false`；
- 所有 `REPLACE_*`、示例域名、示例路径和示例秘密均已替换。

Governance 片段满足同时启用 Ouroboros、Development 和 Harness 时的角色
容量校验：

- Alice、Bob、Carol 都可参加 Seed/演化法定人数，任一人是候选作者时仍有
  两名其他 Reviewer；
- Alice 最多承担两个任务评审类型，Bob 最多承担两个，满足至少两名不同
  四审 Reviewer；
- Carol 只做 final/cancel，不参与四审；
- Alice 的 Harness approve 与 Bob 的 promote/rollback 分离。

## 4. 中央单实例与 TLS

先创建本地目录并安装中央服务，确保只有一个 unit 指向这些 root：

```bash
sudo install -d -o goclaw -g goclaw -m 0700 \
  /srv/goclaw-runtime \
  /srv/goclaw-workspace
sudo install -d -o goclaw -g goclaw -m 0750 \
  /srv/goclaw-repositories
```

调整 [`deploy/systemd/goclaw.service`](../deploy/systemd/goclaw.service) 的
用户、工作目录与 `ReadWritePaths`，先只让 systemd 重新加载 unit：

```bash
sudo systemctl daemon-reload
```

此时不要启动 Gateway。下一节的 `team bootstrap` 会直接写 TeamControl
store；若 Gateway 已启动，就会形成两个进程接触同一中央状态。完成 bootstrap
并启动 unit 后，如需前台诊断，也必须先停止 unit，再由同一个 `goclaw`
服务用户运行：

```bash
goclaw gateway run --verbose
```

生产访问使用 [`deploy/Caddyfile.example`](../deploy/Caddyfile.example)。
Caddy 必须把 `/dashboard`、`/dashboard/*`、`/assets/*`、
`/auth/session`、`/rpc`、`/ws` 和 `/health` 代理到同一个本地 Gateway
端口。不要把 `http://` 或 `ws://` 直接暴露给远程成员。

在每台控制端设置：

```bash
export GOCLAW_GATEWAY_HTTP_URL="https://goclaw.example.com"
export GOCLAW_GATEWAY_TOKEN="$(cat /secure/goclaw/gateway.token)"
export GOCLAW_USER_TOKEN="$(cat /secure/goclaw/me.token)"
```

非 loopback 的 `GOCLAW_GATEWAY_HTTP_URL` 必须使用 `https://`，且 URL
不得含用户名、密码、query 或 fragment。CLI 会在 base URL 后追加
`/rpc`。

## 5. 创建恰好三名用户、三个 Token、一个项目

Carol 作为首个管理员在中央主机、Gateway 尚未初始化时执行：

```bash
sudo -iu goclaw

goclaw team bootstrap \
  --root /srv/goclaw-runtime/team-control \
  --user carol \
  --name "Carol" \
  --email carol@example.com \
  --token-file /srv/goclaw-secrets/carol.team.token

exit
sudo systemctl enable --now goclaw
sudo systemctl status goclaw
```

启动 Gateway 后，Carol 使用该 Token 创建唯一团队与项目：

```bash
sudo -iu goclaw
export GOCLAW_GATEWAY_HTTP_URL="http://127.0.0.1:8080"
export GOCLAW_GATEWAY_TOKEN="$(cat /srv/goclaw-secrets/gateway.token)"
export GOCLAW_USER_TOKEN="$(cat /srv/goclaw-secrets/carol.team.token)"

goclaw team create \
  --id team-pilot \
  --name "Three-person Pilot"

goclaw team project-create \
  --team team-pilot \
  --id pilot-alpha \
  --key PILOT \
  --name "Pilot Alpha"

goclaw team repository-create \
  --project pilot-alpha \
  --id repo-main \
  --name "Pilot Repository" \
  --remote https://git.example.com/team/pilot.git \
  --local-path /srv/goclaw-repositories/pilot \
  --branch main
```

项目创建者 Carol 自动成为 project owner。再按
`user-create → member-add → token-issue → project-member-add` 的顺序
创建 Alice 和 Bob：

```bash
goclaw team user-create \
  --team team-pilot --id alice --name "Alice" --email alice@example.com
goclaw team member-add \
  --team team-pilot --user alice --role member
goclaw team token-issue \
  --user alice --label pilot \
  --token-file /srv/goclaw-secrets/alice.team.token
goclaw team project-member-add \
  --project pilot-alpha --user alice --role developer \
  --domain product --capacity 8

goclaw team user-create \
  --team team-pilot --id bob --name "Bob" --email bob@example.com
goclaw team member-add \
  --team team-pilot --user bob --role member
goclaw team token-issue \
  --user bob --label pilot \
  --token-file /srv/goclaw-secrets/bob.team.token
goclaw team project-member-add \
  --project pilot-alpha --user bob --role developer \
  --domain platform --capacity 8

exit
```

不要再签发第四个 active 用户或第二枚 active 个人 Token。明文 Team Token
通过安全渠道分别交给本人；Reviewer Token 与 Team Token 是两套不同秘密。

## 6. 三种 Runner 底座

### 6.1 原生 Linux

仓库、`CODEX_HOME`、device key 和 work root 都应由 Runner 用户拥有。
在该用户会话中执行自己的登录：

```bash
export CODEX_HOME=/home/alice/.codex
codex login
codex login status
```

### 6.2 Windows 成员：WSL2

使用专用 WSL2 发行版和
[`deploy/wsl2/wsl.conf.example`](../deploy/wsl2/wsl.conf.example)：

- 关闭 interop 与 Windows PATH 注入；
- 仓库、work root、device key、`CODEX_HOME` 全放在发行版虚拟磁盘；
- 禁止 `/mnt/c`、drvfs、9p 和 Windows 符号链接；
- 在 WSL2 内重新 `codex login`，不能复制 Windows 侧 OAuth；
- 使用 Linux 版 `goclaw`，不要用 `goclaw.exe runner work`。

完整模板见 [`deploy/wsl2/README_CN.md`](../deploy/wsl2/README_CN.md)。

### 6.3 macOS 成员：Lima

使用
[`deploy/lima/goclaw-runner.yaml.example`](../deploy/lima/goclaw-runner.yaml.example)
创建专用 Linux guest：

- 保持 host mount 关闭；
- 在 guest 磁盘重新 clone 仓库；
- 在 guest 内以本人账号重新 `codex login`；
- 不从 macOS Home 复制 OAuth，不启用 virtiofs/9p/sshfs；
- macOS 原生 `goclaw` 只运行控制命令。

完整模板见 [`deploy/lima/README_CN.md`](../deploy/lima/README_CN.md)。

三台机器的 OAuth 不同步到中央、Vault、Git、飞书或彼此。Codex 主进程
创建独立 HOME/XDG/TMP，并通过显式 `CODEX_HOME` 使用本人的订阅登录；
模型命令的 named permission profile 对真实目录 `deny`，每次模型调用前
必须通过 read-deny canary。GoClaw/Reviewer/Runner/Codex Token、SSH
agent、Docker/Kubernetes 与云凭据不会传入模型命令或 verifier。

## 7. 安装 wrapper、注册、Doctor、启动

三台 Linux substrate 必须安装发布包中同一份 wrapper。管理员执行：

```bash
sudo install -d -o root -g root -m 0755 /usr/local/libexec/goclaw
sudo install -o root -g root -m 0755 \
  scripts/verify-sandbox-bwrap.sh \
  /usr/local/libexec/goclaw/verify-sandbox-bwrap.sh
sha256sum /usr/local/libexec/goclaw/verify-sandbox-bwrap.sh
```

文件及所有可写父级必须由 root 控制，不能让 Runner 用户、group 或 other
改写。三台必须得到完全相同的 SHA-256。

`runner doctor` 要求已有 device key，因此真实顺序是：
`root-owned wrapper → register 生成 key → doctor → work`。注册时也传入
wrapper，以便控制面记录平台与 sandbox metadata：

```bash
goclaw runner register \
  --id runner-alice \
  --name "Alice Runner" \
  --key-file /home/alice/.config/goclaw/runner.key \
  --project pilot-alpha \
  --capability codex \
  --capability go \
  --verification-sandbox /usr/local/libexec/goclaw/verify-sandbox-bwrap.sh
```

Bob、Carol 分别使用自己的个人 Team Token、不同 Runner ID 与不同 key
文件重复注册。网络结果不明确时保留已生成 key，并以原参数加
`--reuse-key` 重试；不要为同一个 Runner ID 静默生成新 key。

随后用与服务完全相同的参数运行 Doctor：

Doctor 不是静态信息展示：它会实际执行 `codex login status` 和 wrapper
协议探针，检查 device key 属于当前 UID，并检查 repo、work root、
`CODEX_HOME` 及其父路径的 owner/写权限和本地文件系统边界；WSL2 还会拒绝
interop、Windows PATH 与共享挂载。任何一项失败都不能用登记 metadata
绕过。

```bash
goclaw runner doctor \
  --key-file /home/alice/.config/goclaw/runner.key \
  --work-root /home/alice/.local/share/goclaw/work \
  --repo repo-main=/home/alice/src/pilot \
  --codex-command /usr/local/bin/codex \
  --verification-sandbox /usr/local/libexec/goclaw/verify-sandbox-bwrap.sh \
  --json
```

只有 `ready=true` 才能启动：

```bash
goclaw runner work \
  --id runner-alice \
  --key-file /home/alice/.config/goclaw/runner.key \
  --work-root /home/alice/.local/share/goclaw/work \
  --repo repo-main=/home/alice/src/pilot \
  --project pilot-alpha \
  --codex-command /usr/local/bin/codex \
  --codex-model default \
  --verification-sandbox /usr/local/libexec/goclaw/verify-sandbox-bwrap.sh \
  --heartbeat 30s \
  --poll 5s
```

可采用
[`deploy/systemd/goclaw-runner.service.example`](../deploy/systemd/goclaw-runner.service.example)
和 [`deploy/runner.env.example`](../deploy/runner.env.example)。不要启动两个
使用同一 Runner ID/device key 的循环。

`goclaw pilot check` 对三台 Runner 的硬门是：

- 都是 online 或 busy，且 owner 分别为 Alice、Bob、Carol；
- 都含 capability `goclaw-runtime-linux-v1`；
- `runner_goos=linux`，`runner_goarch` 只能是 `amd64` 或 `arm64`；
- `host_profile` 分别可为 `native-linux`、`wsl2` 或 `lima`；
- `isolation_backend=bwrap`；
- `sandbox_sha256` 都是合法的小写 64 位十六进制摘要，且三台完全一致。

原生 Windows/macOS、不同 wrapper、缺失 metadata 或离线 Runner 都会使
试点检查失败。虽然通用 Runner CLI 支持在一次性隔离环境中显式使用
`--unsafe-host-verification`，但未提供 wrapper 的注册 metadata 是
`unconfigured`，unsafe Doctor/work 报告的是 `external-vm`；两者都不能
通过本试点要求 `isolation_backend=bwrap` 的 Gate。

## 8. 建立首个可追踪任务

Issue、WorkItem 和 Assignment 可在 Team Web Console 创建，或用通用 RPC。
准备 JSON 参数文件时至少保证：

- Issue 含 `id`、`project_id`、`type`、`title`、`severity`、`priority`；
- WorkItem 的 ID 与开发任务 Plan 中的 WorkItem ID 完全相同；
- WorkItem 恰好一个 active `owner`，且该 owner 与任务 `assignee` 相同；
- Issue、WorkItem、Task 都属于 `pilot-alpha`。

首个 smoke task 可用以下三个参数文件：

```json
{
  "id": "pilot-issue-smoke-001",
  "project_id": "pilot-alpha",
  "type": "improvement",
  "title": "三人试点端到端 smoke",
  "description": "验证 Wave、四审、Runner、Evidence 与 final 闭环",
  "severity": "medium",
  "priority": "p2",
  "module": "pilot",
  "expected": "任务只由已指派 Runner 执行并由独立 final 验收",
  "actual": "尚未执行"
}
```

```json
{
  "id": "implementation",
  "project_id": "pilot-alpha",
  "issue_id": "pilot-issue-smoke-001",
  "title": "完成试点 smoke 变更",
  "instructions": "只修改 Wave 允许的 docs 范围并执行冻结验证",
  "business_domain": "product",
  "priority": "p2",
  "estimate_points": 1,
  "verification_commands": [
    ["git", "diff", "--check"]
  ]
}
```

```json
{
  "id": "assign-pilot-smoke-alice",
  "project_id": "pilot-alpha",
  "target_type": "work_item",
  "target_id": "implementation",
  "user_id": "alice",
  "role": "owner"
}
```

Carol 以 project owner 身份依次调用：

```bash
export GOCLAW_USER_TOKEN="$(cat /secure/goclaw/carol.team.token)"

goclaw team rpc issue.create \
  --params /secure/goclaw/issue-smoke.json
goclaw team rpc work.create \
  --params /secure/goclaw/work-smoke.json
goclaw team rpc assignment.create \
  --params /secure/goclaw/assignment-smoke.json
```

先取得当前策略哈希：

```bash
goclaw team rpc policy.status --params /secure/goclaw/policy-status.json
```

其中 `policy-status.json` 为：

```json
{
  "project_id": "pilot-alpha",
  "repository_id": "repo-main"
}
```

把返回的 `effective_version` 作为 `POLICY_SHA`。精简 `dev create` 会生成
名为 `implementation` 的单个 WorkItem，因此首个 smoke task 要先创建并
指派同名 TeamControl WorkItem；后续正式任务建议通过 `--spec` 为每个
WorkItem 使用全局稳定且不复用的 ID。

Alice 创建任务时只声明 active Wave 的 Step；客户端不得填写 plan path、
revision 或哈希：

```bash
export GOCLAW_USER_TOKEN="$(cat /secure/goclaw/alice.team.token)"
TASK_ID=task-pilot-smoke-001

goclaw dev create \
  --id "$TASK_ID" \
  --project pilot-alpha \
  --repository-id repo-main \
  --assignee alice \
  --module pilot \
  --issue pilot-issue-smoke-001 \
  --title "三人试点 smoke change" \
  --request "在已批准范围内完成最小可回滚变更" \
  --base main \
  --wave-step PILOT-W00-S07 \
  --policy-hash "$POLICY_SHA" \
  --allow-path 'docs/**' \
  --max-files 4 \
  --max-lines 160 \
  --verify '["git","diff","--check"]' \
  --json
```

Gateway 根据注册仓库的 `local_path` 在精确 base commit 上读取
`docs/waves/wave-registry.json`，要求唯一 active、依赖完成、plan
approved/active、Step 已声明且允许产品变更，然后由服务器冻结：

- Wave ID、plan revision、Step；
- Registry/plan path 与 SHA-256；
- `base_ref` 解析出的精确 commit；
- `created_by=planner-service` 与真实 `requested_by`。

客户端提交的 repo path、Team ID、Wave revision/path/hash 或模糊 branch
都不能成为权威值。Registry、plan、Step、范围或 base 不一致时创建失败。

## 9. Alice、Bob、Carol 的审批闭环

Alice 用自己的 Team Token 与 Reviewer Token批准场景和容量：

```bash
export GOCLAW_USER_TOKEN="$(cat /secure/goclaw/alice.team.token)"
export GOCLAW_REVIEWER_TOKEN="$(cat /secure/goclaw/alice.reviewer.token)"

goclaw dev review "$TASK_ID" \
  --kind scenario --decision approved --reviewer alice \
  --comment "目标、边界和成功条件已与 Wave Step 对齐" \
  --counterargument "smoke 场景不能代表全部真实业务分支" \
  --evidence-ref "wave:PILOT-W00-S07"

goclaw dev review "$TASK_ID" \
  --kind capacity --decision approved --reviewer alice \
  --comment "文件、行数和验证预算适合当前试点" \
  --counterargument "首次运行可能暴露未计入的工具链成本" \
  --evidence-ref "task:$TASK_ID"
```

Bob 使用自己的两枚 Token批准风险和成本：

```bash
export GOCLAW_USER_TOKEN="$(cat /secure/goclaw/bob.team.token)"
export GOCLAW_REVIEWER_TOKEN="$(cat /secure/goclaw/bob.reviewer.token)"

goclaw dev review "$TASK_ID" \
  --kind risk --decision approved --reviewer bob \
  --comment "禁止动作、隔离边界和回滚路径均已明确" \
  --counterargument "真机隔离仍可能与测试环境存在差异" \
  --evidence-ref "doctor:runner-alice"

goclaw dev review "$TASK_ID" \
  --kind cost --decision approved --reviewer bob \
  --comment "执行时限和修复预算在试点上限内" \
  --counterargument "网络抖动可能增加重试和人工等待" \
  --evidence-ref "task:$TASK_ID"
```

Alice 作为 assignee 冻结并入队：

```bash
export GOCLAW_USER_TOKEN="$(cat /secure/goclaw/alice.team.token)"
unset GOCLAW_REVIEWER_TOKEN

goclaw dev freeze "$TASK_ID" --reviewer alice
goclaw dev enqueue "$TASK_ID" \
  --priority 10 \
  --capability codex \
  --capability go \
  --max-attempts 3
```

服务器会自动追加 `goclaw-runtime-linux-v1` capability，并从冻结任务构造
ExecutionPack；客户端不能自造执行包、队列 ID 或幂等键。只有
`runner-alice` 能领取 Alice 的任务。

Runner 完成、证据导入且 DoneGate 通过后，Carol 独立 final：

```bash
export GOCLAW_USER_TOKEN="$(cat /secure/goclaw/carol.team.token)"
export GOCLAW_REVIEWER_TOKEN="$(cat /secure/goclaw/carol.reviewer.token)"

goclaw dev show "$TASK_ID" --json
goclaw runner evidence "${TASK_ID}-r1"

goclaw dev accept "$TASK_ID" \
  --reviewer carol \
  --comment "签名 diff、冻结验证和 DoneGate 均与任务 revision 一致" \
  --counterargument "本地 smoke 仍不覆盖外部 CI 与生产依赖" \
  --evidence-ref "runner:${TASK_ID}-r1"
```

验收不会自动 commit/push。开发者应在正常 Git 流程中应用已验证 patch，
提交、创建 PR 并跑现有 CI；中央 checkout 可见该 commit 后，才可用
`goclaw dev link-pr` 做本地内容校验和关联登记。

## 10. 一致性 Gate

在首次入队、每次备份前和每次恢复后执行：

```bash
goclaw team rpc control.consistency.check \
  --params /secure/goclaw/consistency.json
```

`consistency.json`：

```json
{
  "project_id": "pilot-alpha"
}
```

报告有 critical finding 时，`dev enqueue` 和 `dev accept` 都会失败关闭。
不要手工修改 TeamControl、Workstation 或 Development JSON 来“消除”
finding；应从 Issue/WorkItem/Assignment/task revision 的权威操作修复，
再重新检查。

## 11. Credential attestation 与加密冷备

`pilot check` 要求一个 `0600` 的历史凭据闭环证明。示例结构：

```json
{
  "schema_version": "goclaw.credential-attestation/v1",
  "issue_id": "FE-ISSUE-007",
  "status": "rotated",
  "attested_by": "credential-owner",
  "attested_at": "2026-07-27T12:00:00Z",
  "evidence_ref": "secret-manager:audit/rotation-2026-07-27"
}
```

`status` 只能是 `revoked`、`rotated` 或 `never_valid`。该文件不能由
group/other 访问。示例时间、身份和 evidence 必须替换为真实凭据 owner 的
闭环记录，不能为了通过 Gate 虚构：

```bash
chmod 0600 /srv/goclaw-secrets/credential-attestation.json
```

在离线介质生成 age identity，并把 identity 与备份分开保管：

```bash
umask 077
age-keygen -o /offline/goclaw-pilot.agekey
age-keygen -y /offline/goclaw-pilot.agekey
```

冷备命令必须由中央服务账号 `goclaw` 执行。实现会按**执行命令的操作系统
用户**解析 `~/.goclaw/sessions`，并在该用户的
`~/.goclaw/pilot-maintenance.lock` 取锁；改用 root 会取错 session root，
也不能与服务账号形成同一维护锁边界。让 `/backup` 对该账号可写，并只在
操作期间挂载其可读的 `0600` age identity；不要通过放宽 group/other 权限
解决访问问题。

创建备份前还要确认以下源真实存在：

- 始终需要 config、credential attestation、TeamControl、Workstation、
  Development、服务账号的 sessions 和 workspace；
- 本文启用了 Harness、Ouroboros、Memory Catalog 与 knowledge root，因此
  `harness`、`ouroboros`、catalog 数据库和 knowledge tree 也必须存在；
- Workstation 备份内必须有 `tasks`、`runners`、`credentials`、`evidence`
  四个目录；
- 上述任一文件树都不能含 symlink 或 socket/device/FIFO 等特殊文件。

复制阶段会跳过不存在的源，但后续语义验证会拒绝缺少必需源的恢复点；所以
“backup 命令输出了文件”不等于可恢复。中央 Git 仓库还应通过 `--repo`
显式打包，且必须 clean。

创建冷备前先完成一致性检查，再停止 Gateway。`pilot backup` 会取得维护锁，
也会再次拒绝配置的本机 HTTP Gateway 端口仍在监听。它不单独探测
WebSocket 端口，因此必须通过同一 unit 停掉两个 listener：

```bash
sudo systemctl stop goclaw
sudo -iu goclaw

goclaw pilot backup \
  --config /home/goclaw/.goclaw/config.json \
  --output /backup/goclaw-pilot-20260727.tar.age \
  --age-recipient 'age1REPLACE_WITH_REAL_RECIPIENT' \
  --credential-attestation /srv/goclaw-secrets/credential-attestation.json \
  --repo pilot=/srv/goclaw-repositories/pilot

goclaw pilot verify-backup \
  --archive /backup/goclaw-pilot-20260727.tar.age \
  --age-identity /offline/goclaw-pilot.agekey

exit
```

备份输出必须是新建的 `.age` 文件，最终权限为 `0600`。Git 仓库必须 clean；
未提交工作应先保存在签名 Evidence 中。验证会解密并检查 manifest
hash、成员 digest/mode/type、TeamControl schema、Runner 凭据与 Evidence
签名、Development 事件链和 Git bundle 是否包含冻结 HEAD。归档内含配置、
凭据摘要和项目状态，即使已加密也应按最高敏感级别保存。

重启中央服务和三个 Runner，等待三台 online 后运行完整试点 Gate：

```bash
sudo systemctl start goclaw
sudo -iu goclaw

export GOCLAW_GATEWAY_HTTP_URL="http://127.0.0.1:8080"
export GOCLAW_GATEWAY_TOKEN="$(cat /srv/goclaw-secrets/gateway.token)"
export GOCLAW_USER_TOKEN="$(cat /srv/goclaw-secrets/carol.team.token)"

goclaw pilot check \
  --project pilot-alpha \
  --credential-attestation /srv/goclaw-secrets/credential-attestation.json \
  --wave-registry /srv/goclaw-repositories/pilot/docs/waves/wave-registry.json \
  --backup /backup/goclaw-pilot-20260727.tar.age \
  --backup-age-identity /offline/goclaw-pilot.agekey \
  --json

exit
```

这不是“备份文件存在”检查：它会实际解密并做语义验证。只有报告
`ready=true` 才能进入限时技术试点。`pilot check` 没有 `--config` 参数，
会从当前用户的标准路径加载配置；因此不要换成 root 或另一个运维账号执行。

### 恢复演练

恢复只允许写到不存在或为空的新目录，不能原地覆盖运行数据：

```bash
sudo systemctl stop goclaw
sudo -iu goclaw

goclaw pilot restore \
  --archive /backup/goclaw-pilot-20260727.tar.age \
  --age-identity /offline/goclaw-pilot.agekey \
  --target /srv/goclaw-runtime/restore-drill-20260727

exit
```

失败的部分恢复会被清理。成功后检查
`/srv/goclaw-runtime/restore-drill-20260727/data/`。为演练建立独立的恢复
服务账号与标准配置路径，复制并调整恢复出的 config，使每个 root 都指向
新目录，并使用隔离端口；不能覆盖 live 用户的 config。原实例保持停止，
临时把三台 Runner 指向恢复实例，依次执行
`control.consistency.check`、`pilot check` 和一个只读查询。演练结束后停止
恢复实例，把 Runner 指回原实例再恢复服务。任何时刻都不能让原实例与恢复
实例同时写同一 root。

## 12. Web Console、飞书与 Obsidian

### Web Console

默认入口：

```text
https://goclaw.example.com/dashboard/
```

登录时输入 Gateway Token 与本人的 Team Token。服务器换成
HttpOnly/SameSite 短期会话；Reviewer Token 只在当前页面内存中输入。
项目切换器只列出本人可访问项目，项目对话按
`project_id + topic_id` 使用无碰撞会话键。浏览器的本地存储、URL 和
Markdown 都不能保存任何 Token。

试点至少人工验证：

- 三人可登录且只看到 `pilot-alpha`；
- 对话 history、增量事件和重连不会串到其他项目/topic；
- Issue、WorkItem、Assignment、任务、Runner 和 Evidence 看板一致；
- Alice/Bob 的四审和 Carol final 能按角色成功，越权操作明确失败；
- 刷新后 Reviewer Token 已清空。

### 飞书

飞书只是渠道适配器，不是授权源。只有配置真实 App ID/secret、白名单/
pairing、长连接事件订阅和 `harness.routes` 后才能验收。当前适配器使用
飞书官方 WebSocket 长连接接收事件，不要求把公网 webhook 回调代理到
Gateway。路由把
`channel/account/chat` 映射到项目，但不能替代个人 Team Token、项目 RBAC
或 Reviewer Token；聊天工具不开放审批、编译或 Runner 执行。

未配置真实飞书凭据和长连接事件订阅时，只能说“代码支持飞书适配器”，不能说
“飞书机器人已打通”。

### Obsidian

Obsidian 是可选桌面适配器和 Markdown 知识编辑器，不是中央队列或状态库：

- Vault 可通过 Obsidian Sync 在多台电脑同步 Markdown；
- TeamControl、lease、SessionEvent、Harness/Ouroboros runtime、Token、
  device key 和 OAuth 都不能进入 Vault；
- Web Console 是试点默认聊天、审批和进度窗口；
- 如安装插件，Gateway/Team/Reviewer Token 必须放 Obsidian
  SecretStorage，不能写 `data.json`；
- 插件或 Sync 故障不能阻止 CLI/Web Console 完成核心闭环。

没有在真实 Obsidian Desktop 做兼容性与同步冲突测试时，不得把可选适配器
列为试点放行条件已经通过。

## 13. 回滚与停机

按故障层级处理：

1. 单任务异常：停止对应 Runner；queued/failed 用
   `goclaw runner cancel QUEUE_TASK_ID --reason TEXT`；leased 任务不能
   强制伪造完成，等待 lease 恢复或先停中央调查。
2. 需求/范围变化：创建新的 Wave plan revision，更新 Registry 后用
   `goclaw dev revise`；旧 revision 重新四审，不能在 frozen task 上就地
   改字段。
3. Harness 回归：由 Bob 以独立 `harness_rollback` 身份执行受治理回滚；
   Alice 不能同时 approve 和 promote。
4. 控制面不一致：停止入队和验收，运行 consistency report，保存证据，
   停 Gateway 后从最近一次已验证冷备恢复到新目录。
5. 凭据疑似泄露：停止服务，撤销/轮换对应 Team Token、Reviewer Token、
   Gateway Token 或 Runner key，更新 credential attestation 后重建冷备。

不得删除事件链、证据或旧 revision 来让看板“变绿”。

## 14. 放行清单与未验证项

技术试点开始前必须同时满足：

- [ ] 单实例 Gateway 仅监听 loopback，Caddy HTTPS/WSS 通过；
- [ ] 恰好 3 个 active 成员、3 枚 active 个人 Token、1 个项目；
- [ ] 三名 Reviewer 的真实摘要已替换，角色与职责分离验证通过；
- [ ] 三台不同 owner Runner online/busy；
- [ ] 三台均通过 Doctor，并满足 Linux runtime/capability/metadata 合同；
- [ ] 三台使用同一 root-owned wrapper SHA-256；
- [ ] 每人的 Codex OAuth 只存在自己的 Linux substrate；
- [ ] active Wave、Step、Registry/plan hash 和冻结 base 可重验证；
- [ ] 一次 Alice/Bob 四审、Runner evidence、Carol final 闭环通过；
- [ ] `control.consistency.check` 无 critical finding；
- [ ] credential attestation 合法；
- [ ] age 冷备可实际解密、语义验证并恢复到新目录；
- [ ] 三人 Web Console 登录、RBAC、项目会话和断线恢复人工通过；
- [ ] 凭据保留、轮换、事件响应与停机负责人已指定。

本仓库的确定性测试或交叉编译不能替代以下现场证据：

- 当前构建环境尚未证明目标真机的 bwrap 隔离；
- WSL2 和 Lima 必须分别在真实 guest 运行 Doctor 与任务；
- ChatGPT Workspace/Codex OAuth 必须由三位成员及中央可选模型账号真实授权；
- 飞书 App、长连接事件订阅、白名单和路由必须使用真实租户配置；
- Obsidian 必须在真实 Desktop/Vault Sync 环境验收；
- TLS、备份介质和恢复时限必须在部署网络与存储上演练。

任一项缺失时，正确状态是“`0.8.0-pilot.1` 技术试点候选，现场 Gate
blocked”，而不是“已上线”。
