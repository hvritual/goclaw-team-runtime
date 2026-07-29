# Team Control 与 Runner 双应用构建使用指南

适用路线：`TR-W00` 及后继 Team Runtime Waves。

## 1. 应用定位

| 应用 | 部署位置 | 主要职责 | 不负责 |
|---|---|---|---|
| `goclaw-team-control` | 中央 Linux 服务；macOS/Windows 可作管理客户端 | Gateway、成员/Token、项目/仓库、策略、任务、审批、知识、Harness、Runner scheduler、Web Console 和渠道 | 本地 Runner work loop |
| `goclaw-runner` | 开发成员电脑；`strict` 用 Linux/WSL2/Lima，`codex-delegated` 可用原生 Windows/macOS/Linux | 注册、doctor、领取任务、调用本机 Codex OAuth、目录限定执行、验证、签名 Evidence 与本地 release 切换 | 团队 bootstrap、成员/Token/策略管理、最终验收 |
| `goclaw` | 兼容期 | 保留原命令面，便于迁移 | 不作为新部署的首选入口 |

命令面隔离用于减少误操作；真正授权边界仍是 Gateway TLS/Token、个人 Team
Token、服务端项目 RBAC、Runner device key、ExecutionPack、目录检查和
sandbox。

## 2. 本机开发构建

要求 Go `1.25.5`。

```bash
make build-team-control
make build-runner

./goclaw-team-control version
./goclaw-runner version
```

如果没有 `make`：

```bash
go build -buildvcs=false -o goclaw-team-control ./cmd/team-control
go build -buildvcs=false -o goclaw-runner ./cmd/runner
```

## 3. 一次构建全部平台

本节是从 Linux/Bash 主机构建六个 `GOOS/GOARCH` 目标的交叉编译证明，
不是原生 Windows PowerShell installer、macOS `.pkg/.dmg` 或签名/公证
流程。原生安装、签名、升级与回滚归 `REL-W01`。输出目录必须不存在或
为空；脚本在同级 task-specific staging 中完成精确清单验证后才发布。

```bash
./scripts/build-apps.sh \
  --output /tmp/goclaw-apps \
  --version 0.9.0-wave.0

(cd /tmp/goclaw-apps && sha256sum -c SHA256SUMS)
```

输出包含：

- Linux amd64/arm64；
- macOS amd64/arm64；
- Windows amd64/arm64；
- 每个平台的 `goclaw-team-control`、`goclaw-runner` 和兼容 `goclaw`；
- 一个覆盖全部二进制的 `SHA256SUMS`。

Windows 文件带 `.exe`。构建脚本设置 `CGO_ENABLED=0`，不会把本地配置、
OAuth 或 Token 主动写入产物；若进程环境存在 Team/Gateway/Reviewer、
Runner device key、Codex access/refresh 或 GitHub Token，脚本会逐
candidate binary 拒绝包含其原值的产物。

## 4. Team Control 部署

中央配置继续使用 `~/.goclaw/config.json`。至少启用：

```json
{
  "team_control": {
    "enabled": true,
    "root": "/var/lib/goclaw/teamcontrol"
  },
  "workstation": {
    "enabled": true,
    "root": "/var/lib/goclaw/workstation",
    "lease_duration_seconds": 120,
    "runner_offline_seconds": 300,
    "default_max_attempts": 3,
    "max_idempotency_receipts": 128
  },
  "development": {
    "enabled": true,
    "root": "/var/lib/goclaw/development",
    "gateway_allow_execution": false
  }
}
```

启动：

```bash
goclaw-team-control gateway run
```

systemd 模板：

```text
deploy/systemd/goclaw-team-control.service.example
```

首次创建管理员、团队、成员和项目仍使用：

```bash
goclaw-team-control team bootstrap ...
goclaw-team-control team create ...
goclaw-team-control team user-create ...
goclaw-team-control team token-issue ...
goclaw-team-control team project-create ...
goclaw-team-control team repository-create ...
```

完整参数以 `goclaw-team-control team --help` 为准。

中央预算、Knowledge/Skill/Runner release Registry 与 Context Bundle
操作见 [`Team Control Registry、预算与 Context Bundle`](TEAM_CONTROL_REGISTRY_CN.md)。

## 5. Runner 部署

Runner 有两个不可静默互换的 execution profile：

- `strict` 是默认值，用于 Linux native、WSL2 或 Lima，要求受审 verifier
  wrapper；高保证任务和三人试点使用它；
- `codex-delegated` 可在原生 Windows、macOS、Linux 的 amd64/arm64 上运行，
  但必须由项目策略显式允许。它保留目录边界、独立 worktree、最小环境、
  credential canary 和 diff 后验检查，不提供 GoClaw OS 级进程/网络隔离。

注册与执行示例：

```bash
goclaw-runner runner register \
  --id alice-project-alpha \
  --name "Alice / Project Alpha" \
  --key-file ~/.config/goclaw/alice-project-alpha.key \
  --project project-alpha \
  --capability codex

goclaw-runner runner doctor \
  --key-file ~/.config/goclaw/alice-project-alpha.key \
  --work-root ~/.local/share/goclaw/worktrees-alpha \
  --repo repo-alpha=$HOME/src/project-alpha \
  --codex-command /usr/local/bin/codex \
  --verification-sandbox /usr/local/libexec/goclaw/verify-sandbox-bwrap.sh

goclaw-runner runner work \
  --id alice-project-alpha \
  --key-file ~/.config/goclaw/alice-project-alpha.key \
  --work-root ~/.local/share/goclaw/worktrees-alpha \
  --repo repo-alpha=$HOME/src/project-alpha \
  --project project-alpha \
  --codex-command /usr/local/bin/codex \
  --verification-sandbox /usr/local/libexec/goclaw/verify-sandbox-bwrap.sh
```

原生 Windows/macOS 降级运行时，注册、doctor 和 work 三个命令都必须显式
使用同一参数：

```text
--execution-profile codex-delegated
```

该 profile 不得同时使用 `--verification-sandbox` 或
`--unsafe-host-verification`。Team Control 的 resolved policy 必须包含：

```json
{
  "runner.execution_profiles": ["strict", "codex-delegated"],
  "runner.target_version": "0.9.0",
  "runner.target_release_id": "runner-090-darwin-arm64",
  "runner.release_channel": "pilot",
  "runner.rollout_paused": false
}
```

未配置该规则时只允许 `strict`。客户端传入的 ExecutionPack 不是权威，
Gateway 会重建 profile 并追加匹配 capability；Runner claim 与本地执行还会
再次核对，无法从 strict 静默降级。配置 target 时，Gateway 把目标
version/release 转成 claim capability；Runner 每次工作循环心跳上报当前
binary/release/profile，项目视图显示 `unmanaged/update_required/compliant`
或 `paused`。pause 同时阻止新 enqueue 和已有队列的新 claim。

Runner 只让 Codex 主进程使用本机 `CODEX_HOME` 的 ChatGPT/Codex OAuth；
控制面不接收 OAuth 文件。模型生成的命令使用 named permission profile：
worktree 可写、命令网络关闭、真实 `CODEX_HOME` 为 OS sandbox `deny`。
每次调用模型前，Runner 先执行不访问模型的 read-deny canary；Codex CLI
不支持 permission profile、sandbox 启动失败或 canary 能读取该目录时，
任务失败关闭。个人 Team Token、Gateway Token 和 device key 应通过
Secret Store、systemd credential 或权限为 `0600` 的环境文件注入。

## 6. Runner 本地 release 生命周期

Team Control 的 Runner release Registry 保存 immutable
`id/version/os/arch/min_protocol/size_bytes/sha256`。Runner 不自动访问 Registry
URI；operator 必须先用可信渠道取得本地 artifact，再对中央 `approved`
记录做完整身份匹配：

```bash
goclaw-runner runner release stage-from-control RELEASE_ID \
  --project project-alpha \
  --work-root /absolute/runner-root \
  --artifact /absolute/goclaw-runner

goclaw-runner runner release activate RELEASE_ID \
  --work-root /absolute/runner-root
goclaw-runner runner release path --work-root /absolute/runner-root
goclaw-runner runner release confirm RELEASE_ID \
  --work-root /absolute/runner-root
```

`activate/confirm/rollback` 要求工作循环已停止；运行进程锁或并发 release
mutation 锁存在时失败关闭。activate 后必须由外部 service manager 用
`runner release path` 返回且已重新校验的路径做 smoke，再 confirm。失败时：

```bash
goclaw-runner runner release rollback --work-root /absolute/runner-root
```

本 Wave 只提供可恢复的本地选择状态，不会下载 URI、覆盖运行中 binary、
重启 service 或绕过 macOS/Windows 代码签名策略。

## 7. 命令面验证

```bash
goclaw-team-control --help
goclaw-runner --help
```

预期：

- Team Control 没有顶层 `runner`；
- Runner 只有 `runner`、`config`、`health`、`status`、`version`；
- Runner 不能发现 `team`、`gateway`、`dev`、`harness`、`ouroboros`。

## 8. 从兼容入口迁移

| 旧命令 | 新命令 |
|---|---|
| `goclaw gateway run` | `goclaw-team-control gateway run` |
| `goclaw team ...` | `goclaw-team-control team ...` |
| `goclaw dev ...` | `goclaw-team-control dev ...` |
| `goclaw runner ...` | `goclaw-runner runner ...` |

`goclaw` 在当前兼容周期仍可构建。先替换 systemd/launch 脚本，再移除旧
binary；不要让两个中央进程同时写同一个 TeamControl/Workstation root。

## 9. GitHub 作为恢复来源

权威仓库：

```text
https://github.com/hvritual/goclaw-team-runtime
```

每个 Wave 的 Plan、Task Freeze、commit trailers、Evidence 和使用文档都应
推送到该私有仓库。检查授权：

```bash
gh auth status
git remote -v
git ls-remote origin HEAD
```

若授权过期或被撤销：

```bash
gh auth login --hostname github.com --git-protocol https
gh auth setup-git
git push
```

不得把 `hosts.yml`、GitHub Token、浏览器 OAuth、Team Token 或 Runner key
提交到仓库。授权中断时保留本地 commit，恢复授权后从同一分支继续推送。

## 10. 当前 Wave 边界

TR-W00 只完成双应用边界和构建。以下能力由后继 Wave 依次交付：

1. `TC-W01`：预算、知识源、Skill、Runner release 与 Context Compiler，
   已通过三路 exact review；
2. `RN-W01`：双 execution profile、release stage/activate/confirm/rollback
   和多项目生命周期，当前为候选验收状态；
3. `INT-W01`：项目作用域 MCP、知识/Skill 与 Context Bundle；
4. `REL-W01`：正式跨平台归档、安装升级、回滚和三人试点 Evidence。
