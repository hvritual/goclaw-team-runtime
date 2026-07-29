# GoClaw 跨平台源码构建、成员 Token 与多项目并发手册

适用源码版本：`0.8.0-pilot.1`

本文以当前仓库源码和实际 CLI 为准，覆盖：

- Windows、macOS、Linux 的源码构建；
- 中央 Gateway/TeamControl 与成员工作站部署；
- Gateway Token、个人 Team Token、Runner device key、Codex OAuth 的边界；
- 成员开户、授权、Token 轮换、撤销和离职处理；
- Team、Project、Repository、项目角色与容量管理；
- Issue、WorkItem、Assignment、受治理开发任务的完整闭环；
- 多项目串行与并发 Runner 的正确部署方式。

## 1. 先确认平台边界

当前版本保留 `goclaw.runner/linux-v1` strict 合同，并新增显式降级的
`codex-delegated`。平台支持矩阵如下：

| 平台 | 构建应用 | `strict` | `codex-delegated` |
|---|---:|---:|---:|
| Linux amd64/arm64 | 是 | 原生 + bwrap | 原生；无 GoClaw OS sandbox |
| Windows amd64/arm64 | 是 | WSL2 guest | 原生；无 GoClaw OS sandbox |
| macOS Intel/Apple Silicon | 是 | Lima guest | 原生；无 GoClaw OS sandbox |

因此：

1. 完整 Team Runtime 发布包应在 Linux、WSL2 或 Linux CI 中构建。
2. Windows/macOS 原生 Runner 只有在 Team Control policy 显式允许
   `codex-delegated` 时才可执行；它不是 strict 或 sandbox 的替代证明。
3. Windows 仓库、work root、device key 与 Codex OAuth 必须位于 WSL2
   虚拟磁盘，不得放在 `/mnt/c`。
4. macOS 的执行资产必须位于 Lima guest 磁盘，不得启用 host mount。
5. 三人试点的 `pilot check` 固定要求一个项目、三名成员和三台 Runner；
   常规多项目模式不能把这个特定检查当作全局并发调度器。

## 2. 源码与工具链

在源码根目录执行后续命令：

```bash
cd /path/to/goclaw-r7-pilot
```

推荐使用与本版本验证记录一致的工具链：

| 工具 | 基线 | 用途 |
|---|---|---|
| Go | `1.25.5` | 后端、CLI、Gateway、Runner |
| Node.js | `24.x` | Web Console 与 Obsidian 插件 |
| npm | `11.x` | 前端依赖与构建 |
| Git | 当前受支持版本 | 源码、worktree、证据 |
| Bash + GNU tar/coreutils | Linux 环境 | 正式发布脚本 |
| bubblewrap | Linux 当前发行版 | 冻结验证隔离 |
| Codex CLI | 当前订阅可用版本 | 每位成员本地执行 |

基础检查：

```bash
go version
node --version
npm --version
git --version
```

源码声明的 Go 版本可直接检查：

```bash
grep '^go ' go.mod
```

## 3. 构建结果与目录

源码包含三类不同产物：

| 产物 | 用途 | 是否可执行 Runner |
|---|---|---:|
| `goclaw`/`goclaw.exe` | Gateway、控制 CLI 与 Runner | strict 仅 Linux substrate；delegated 可三平台 |
| `goclaw-team-runtime-linux-*.tar.gz` | Linux/WSL2/Lima Runner 包 | 是 |
| `obsidian-goclaw-plugin-*.tar.gz` | Obsidian 侧边栏、审批与进度界面 | 否 |

正式发布脚本还生成源码归档、校验文件和构建信息。它会拒绝把常见二进制、
本地状态、环境文件和疑似凭据装进源码包。

## 4. Linux 完整构建

### 4.1 安装依赖

Ubuntu 24.04 示例：

```bash
sudo apt-get update
sudo apt-get install -y \
  ca-certificates git curl build-essential bubblewrap tar
```

另外安装 Go `1.25.5`、Node.js `24.x`、npm `11.x` 和 Codex CLI。

### 4.2 开发构建

如果 Web Console 没有变化，可以直接使用源码中已有的
`gateway/ui_dist`：

```bash
go build -buildvcs=false \
  -ldflags="-X main.Version=0.8.0-pilot.1" \
  -o ./goclaw .

./goclaw version
```

如果修改了 Web Console，必须先重建并同步嵌入文件：

```bash
(
  cd ui
  npm ci
  npm test
  npm run build
)

find gateway/ui_dist -mindepth 1 -type f -delete
find gateway/ui_dist -mindepth 1 -type d -empty -delete
cp -R ui/dist/. gateway/ui_dist/

go build -buildvcs=false \
  -ldflags="-X main.Version=0.8.0-pilot.1" \
  -o ./goclaw .
```

### 4.3 正式发布构建

只构建核心 Runtime 和源码包：

```bash
./scripts/build-release.sh
```

同时构建 Obsidian 插件：

```bash
INCLUDE_OBSIDIAN_PLUGIN=1 ./scripts/build-release.sh
```

只生成经过内容检查的源码归档：

```bash
SOURCE_ONLY=1 ./scripts/build-release.sh
```

该脚本会：

1. 执行 `npm ci` 并构建 Web Console；
2. 同步 `ui/dist` 到 `gateway/ui_dist`；
3. 运行确定性的 Go 发布测试集；
4. 构建 Linux amd64/arm64 Runtime；
5. 交叉编译 Windows/macOS 控制 CLI，验证可移植性；
6. 打包 Linux Runner、WSL2/Lima 模板和 bwrap wrapper；
7. 生成经过恢复检查、成员路径检查和凭据扫描的源码包；
8. 生成 SHA-256 校验文件。

查看产物：

```bash
find dist -maxdepth 1 -type f -printf '%f\n' | sort
```

校验：

```bash
cd dist
sha256sum -c SHA256SUMS
```

实际校验文件名以 `dist/` 结果为准。

### 4.4 Linux 安装

控制面或 Runner 安装：

```bash
sudo install -o root -g root -m 0755 \
  dist/goclaw-linux-amd64 \
  /usr/local/bin/goclaw
```

ARM64 将文件名替换为 `goclaw-linux-arm64`。

安装冻结验证 wrapper：

```bash
sudo install -d -o root -g root -m 0755 /usr/local/libexec/goclaw
sudo install -o root -g root -m 0755 \
  scripts/verify-sandbox-bwrap.sh \
  /usr/local/libexec/goclaw/verify-sandbox-bwrap.sh

sudo test "$(stat -c '%U:%G:%a' \
  /usr/local/libexec/goclaw/verify-sandbox-bwrap.sh)" = "root:root:755"
```

## 5. Windows 源码构建与 WSL2 Runner

### 5.1 Windows 原生控制 CLI

安装：

- Go `1.25.5`；
- Node.js `24.x` 与 npm `11.x`；
- Git for Windows；
- PowerShell 7，推荐但非强制。

PowerShell 中执行：

```powershell
Set-Location C:\src\goclaw-r7-pilot

go version
node --version
npm --version
git --version
```

如未修改 UI，构建原生控制 CLI：

```powershell
$Version = "0.8.0-pilot.1"
$env:CGO_ENABLED = "0"
$env:GOOS = "windows"
$env:GOARCH = "amd64"

New-Item -ItemType Directory -Force .\dist | Out-Null
go build -buildvcs=false -trimpath `
  -ldflags "-s -w -X main.Version=$Version" `
  -o .\dist\goclaw-windows-amd64.exe .

.\dist\goclaw-windows-amd64.exe version
Get-FileHash .\dist\goclaw-windows-amd64.exe -Algorithm SHA256
```

Windows on ARM 把 `GOARCH` 和文件名改为 `arm64`。

如修改了 UI：

```powershell
Push-Location .\ui
npm ci
npm test
npm run build
Pop-Location

Get-ChildItem .\gateway\ui_dist -Force | Remove-Item -Recurse -Force
Copy-Item .\ui\dist\* .\gateway\ui_dist\ -Recurse -Force

go build -buildvcs=false -trimpath `
  -ldflags "-s -w -X main.Version=$Version" `
  -o .\dist\goclaw-windows-amd64.exe .
```

可执行与平台无关的关键测试：

```powershell
go test -count=1 `
  ./memory ./memory/catalog ./governance ./ouroboros `
  ./orchestratorlite ./harness ./teamcontrol ./workstation `
  ./providers ./gateway ./agent ./agent/tools ./config `
  ./cli ./cli/commands ./internal/start
```

Windows 原生 strict 会按设计失败。只有项目已批准 delegated、Runner 以同一
profile 注册且 Doctor 通过 Windows ACL/reparse/Codex canary 检查时，才可
显式运行：

```powershell
.\dist\goclaw-windows-amd64.exe runner doctor `
  --execution-profile codex-delegated `
  --key-file C:\secure\goclaw\runner.key `
  --work-root C:\goclaw\work `
  --repo repo-alpha=C:\src\alpha

.\dist\goclaw-windows-amd64.exe runner work `
  --execution-profile codex-delegated `
  --id alice-alpha-windows `
  --key-file C:\secure\goclaw\runner.key `
  --work-root C:\goclaw\work `
  --repo repo-alpha=C:\src\alpha `
  --project project-alpha
```

不要传 `--verification-sandbox` 或 `--unsafe-host-verification`。高风险任务
继续使用下一节的 WSL2 strict。

### 5.2 Windows 完整 Runtime 在 WSL2 构建

以管理员 PowerShell 创建专用发行版：

```powershell
wsl --install -d Ubuntu-24.04
```

在 WSL2 内把 `deploy/wsl2/wsl.conf.example` 安装为 `/etc/wsl.conf`：

```bash
sudo install -o root -g root -m 0644 \
  deploy/wsl2/wsl.conf.example \
  /etc/wsl.conf
```

然后回到 Windows：

```powershell
wsl.exe --shutdown
```

重启发行版后确认：

```bash
cat /etc/wsl.conf
```

其中必须是：

- `systemd=true`；
- `interop.enabled=false`；
- `appendWindowsPath=false`；
- `automount.enabled=false`。

源码、仓库、Runner work root、device key 和 `CODEX_HOME` 都要放在 WSL2
虚拟磁盘，例如 `/home/alice/...`。不要从 `/mnt/c` 构建或执行任务。

随后在 WSL2 内按“Linux 完整构建”执行：

```bash
INCLUDE_OBSIDIAN_PLUGIN=1 ./scripts/build-release.sh
```

### 5.3 WSL2 Runner 安装

在专用发行版内：

```bash
sudo apt-get update
sudo apt-get install -y bubblewrap ca-certificates git

sudo install -o root -g root -m 0755 \
  ./goclaw \
  /usr/local/bin/goclaw

sudo install -d -o root -g root -m 0755 /usr/local/libexec/goclaw
sudo install -o root -g root -m 0755 \
  ./scripts/verify-sandbox-bwrap.sh \
  /usr/local/libexec/goclaw/verify-sandbox-bwrap.sh
```

以成员本人的 ChatGPT 订阅登录：

```bash
codex login
codex login status
```

不要从 Windows Home 或其他成员复制 `.codex`。

## 6. macOS 源码构建与 Lima Runner

### 6.1 macOS 原生控制 CLI

安装 Xcode Command Line Tools、Go `1.25.5`、Node.js `24.x`、npm `11.x`
和 Git。

Intel Mac：

```bash
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 \
  go build -buildvcs=false -trimpath \
  -ldflags="-s -w -X main.Version=0.8.0-pilot.1" \
  -o dist/goclaw-darwin-amd64 .
```

Apple Silicon：

```bash
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 \
  go build -buildvcs=false -trimpath \
  -ldflags="-s -w -X main.Version=0.8.0-pilot.1" \
  -o dist/goclaw-darwin-arm64 .
```

检查：

```bash
./dist/goclaw-darwin-arm64 version
shasum -a 256 ./dist/goclaw-darwin-arm64
```

如修改了 UI，先执行 Linux 章节中的 `npm ci`、`npm test`、`npm run build`
和 `gateway/ui_dist` 同步，再编译 Go。

macOS 原生 strict 不能执行 `runner work`。项目明确接受 delegated 降级时，
可在 register/doctor/work 三处传
`--execution-profile codex-delegated`；高风险任务继续使用 Lima strict。

### 6.2 可选 Tauri 桌面壳

仓库还保留 Tauri `1.5` 桌面壳。它是可选的桌面封装，不是 Team Runtime
Runner，也不替代浏览器 Web Console。

额外安装 Rust/Cargo 和 Tauri CLI 后：

```bash
make setup-tauri
make build-tauri-current
```

签名、公证和 DMG 流程见 `docs/release-macos.md`：

```bash
make release-tauri-macos
make verify-tauri-macos
make notarize-tauri-macos
make staple-tauri-macos
```

注意 Tauri 配置仍有独立的旧版本号；试点正式交付以 Go CLI、Web Console
和 Linux Runner 包为准。

### 6.3 Lima Linux Runner

安装 Lima 后，从源码根目录创建无 host mount 的专用 guest：

```bash
limactl start \
  --name=goclaw-pilot \
  deploy/lima/goclaw-runner.yaml.example

limactl shell goclaw-pilot
```

模板使用 Ubuntu 24.04、关闭所有 host mount，并安装 Git、CA 证书和
bubblewrap。进入 guest 后：

1. 把 Linux `goclaw` 包复制或下载到 guest；
2. 在 guest 磁盘的 `/var/lib/goclaw-runner/src` 重新 clone 授权仓库；
3. 安装 root-owned bwrap wrapper；
4. 以该成员自己的订阅执行 `codex login`；
5. 不要添加 virtiofs、9p、sshfs 或其他 macOS 目录共享。

## 7. 中央服务最小配置

中央服务推荐部署在专用 Linux 主机，以一个 GoClaw 进程独占写入本地
TeamControl 和 Workstation root。

`config.json` 最小关键段：

```json
{
  "gateway": {
    "websocket": {
      "enabled": true,
      "auth_token": "<gateway-boundary-token>"
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

- 非 loopback 访问必须使用 HTTPS/WSS；
- `team_control.root` 与 `workstation.root` 必须位于非同步本地磁盘；
- 不能由两个 GoClaw 进程或共享盘同时写入同一 root；
- 备份、维护和恢复期间应先进入维护锁；
- Gateway Token 至少 24 个字符，不能提交进 Git、Vault 或群聊。

启动：

```bash
goclaw gateway run
```

成员 CLI 使用：

```bash
export GOCLAW_GATEWAY_HTTP_URL="https://goclaw.example.com"
export GOCLAW_GATEWAY_TOKEN="$(cat /secure/goclaw/gateway.token)"
export GOCLAW_USER_TOKEN="$(cat /secure/goclaw/me.team.token)"
```

Windows PowerShell：

```powershell
$env:GOCLAW_GATEWAY_HTTP_URL = "https://goclaw.example.com"
$env:GOCLAW_GATEWAY_TOKEN = Get-Content `
  "$env:USERPROFILE\.goclaw\secrets\gateway.token" -Raw
$env:GOCLAW_USER_TOKEN = Get-Content `
  "$env:USERPROFILE\.goclaw\secrets\me.team.token" -Raw
```

## 8. 四类凭据必须分开

| 凭据 | 作用 | 保存位置 | 是否中央保存明文 | 轮换方式 |
|---|---|---|---:|---|
| Gateway Token | 连接 Gateway 的外层边界 | 受控客户端/服务 Secret Store | 配置中需要 | 更换服务配置并滚动客户端 |
| 个人 Team Token | 用户身份与项目 RBAC | 每位成员自己的 Secret Store | 否，只存 SHA-256 摘要 | 签发新 Token，切换后撤销旧 Token |
| Runner device key | Runner 与证据 HMAC | 该 Runner Linux substrate | 注册时共享，之后不回显 | Runner 空闲时 `rotate-key` |
| Codex OAuth | ChatGPT 订阅模型登录 | 每位成员本地 `CODEX_HOME` | 否 | 成员自行 `codex login/logout` |

还可能配置独立 Reviewer Token。它属于 GoClaw 治理层的审批认证，不应复用
Gateway Token、个人 Team Token 或 Runner device key。

禁止：

- 把任何 Token 或 OAuth 目录放进 Obsidian Vault；
- 多人共用一个个人 Team Token；
- 多人共用一个 Codex OAuth；
- 把 device key 当个人登录 Token；
- 把 Token 明文写进 task spec、ExecutionPack、EvidenceBundle、Git 提交或日志；
- 通过 `--allow-env` 向 Codex 透传 GoClaw Token、SSH agent、Docker/
  Kubernetes socket 或云凭据。

## 9. 首个管理员、团队与成员开户

### 9.1 首次 bootstrap

只在 TeamControl 为空时执行一次：

```bash
install -d -m 0700 /secure/goclaw

goclaw team bootstrap \
  --root /srv/goclaw-runtime/team-control \
  --user admin \
  --name "Team Admin" \
  --email admin@example.com \
  --label bootstrap \
  --token-file /secure/goclaw/admin.team.token
```

`--root` 必须与 Gateway 配置的 `team_control.root` 完全一致。明文只写入新建
的 `0600` 文件；服务端只保存摘要。初始化后不能再次 bootstrap。

```bash
export GOCLAW_USER_TOKEN="$(cat /secure/goclaw/admin.team.token)"
```

### 9.2 创建 Team

```bash
goclaw team create \
  --id team-product \
  --name "Product Engineering" \
  --description "产品研发团队"
```

### 9.3 创建成员并签发 Token

开户顺序不能颠倒：

```text
user-create → member-add → token-issue → project-member-add
```

示例：

```bash
goclaw team user-create \
  --team team-product \
  --id alice \
  --name "Alice" \
  --email alice@example.com

goclaw team member-add \
  --team team-product \
  --user alice \
  --role member

goclaw team token-issue \
  --user alice \
  --label laptop-2026 \
  --expires 2027-07-27T00:00:00Z \
  --token-file /secure/goclaw/alice.team.token
```

注意：

- 目标用户至少要有一个 active team membership 才能签发 Token；
- `planner-service` 不允许登录，也不能获取 Token；
- Token 到期时间必须是未来的 RFC3339 时间；
- 签发者必须是该用户所属**每一个活动团队**的 owner/admin；
- 普通成员当前不能自助签发、列出或撤销自己的 Token；
- Token 文件只出现一次，不可从服务端恢复明文。

Windows 建议把秘密放到受 ACL 保护的目录：

```powershell
$SecretDir = "$env:USERPROFILE\.goclaw\secrets"
New-Item -ItemType Directory -Force $SecretDir | Out-Null
icacls $SecretDir /inheritance:r `
  /grant:r "${env:USERNAME}:(OI)(CI)F"
```

长期运行应使用 Windows Credential Manager、macOS Keychain、Linux Secret
Service/systemd credential 或同等级秘密管理器，而不是普通文本文件。

## 10. 个人 Token 查询、轮换、撤销与离职

`team rpc` 的参数必须来自 JSON 文件；不要把秘密写入该文件。

### 10.1 查询 Token 元数据

`/secure/goclaw/token-list-alice.json`：

```json
{
  "user_id": "alice"
}
```

```bash
goclaw team rpc team.token.list \
  --params /secure/goclaw/token-list-alice.json
```

返回的是 credential ID、label、状态和到期信息，不返回明文或摘要。

### 10.2 零停机轮换

1. 管理员为同一用户签发新 Token，使用新 label 和未来到期时间。
2. 通过安全渠道把新 Token 交给本人。
3. 成员替换本地注入值并重启自己的控制 CLI/Runner 服务。
4. 用新 Token 调用 `project.list` 验证身份和项目权限。
5. 管理员撤销旧 credential ID。
6. 验证旧 Token 已失败，新 Token 正常。
7. 把轮换日期、credential ID、操作者和验证结果记入审计记录，不记录明文。

签发新 Token：

```bash
goclaw team token-issue \
  --user alice \
  --label laptop-2027 \
  --expires 2028-07-27T00:00:00Z \
  --token-file /secure/goclaw/alice.team.token.next
```

验证新 Token：

```bash
GOCLAW_USER_TOKEN="$(cat /secure/goclaw/alice.team.token.next)" \
  goclaw team rpc project.list
```

撤销参数 `/secure/goclaw/token-revoke.json`：

```json
{
  "credential_id": "<credential-id-from-token-list>"
}
```

```bash
goclaw team rpc team.token.revoke \
  --params /secure/goclaw/token-revoke.json
```

### 10.3 禁用成员

`/secure/goclaw/user-disable.json`：

```json
{
  "team_id": "team-product",
  "user_id": "alice",
  "status": "disabled"
}
```

```bash
goclaw team rpc team.user.status \
  --params /secure/goclaw/user-disable.json
```

禁用用户后其认证失败。系统拒绝禁用最后一个活动 team owner。

### 10.4 离职/设备丢失顺序

1. 停止并禁用该成员的 Runner；
2. 等待或处理活动 lease，不要在 leased 状态强行换 key；
3. 撤销该用户的全部个人 Token；
4. 禁用用户；
5. 在工作站执行 `codex logout`，销毁本地 OAuth、device key 和 work root；
6. 移除仓库访问、VPN、SSH 和其他外部平台权限；
7. 保留中央事件、EvidencePackage 和审计记录。

## 11. 项目、仓库和角色管理

### 11.1 创建多个项目

```bash
goclaw team project-create \
  --team team-product \
  --id project-alpha \
  --key ALPHA \
  --name "Alpha Platform" \
  --description "核心交易平台"

goclaw team project-create \
  --team team-product \
  --id project-beta \
  --key BETA \
  --name "Beta Operations" \
  --description "运营自动化平台"
```

建议所有 ID 使用稳定、全局不复用的格式：

```text
[A-Za-z0-9][A-Za-z0-9._-]{0,63}
```

Project 创建者自动成为 project owner。

### 11.2 登记仓库

一个 Project 可以登记多个 Repository：

```bash
goclaw team repository-create \
  --project project-alpha \
  --id repo-alpha-api \
  --name "Alpha API" \
  --remote https://git.example.com/product/alpha-api.git \
  --local-path /srv/goclaw/checkouts/project-alpha/alpha-api \
  --branch main

goclaw team repository-create \
  --project project-beta \
  --id repo-beta-ops \
  --name "Beta Ops" \
  --remote https://git.example.com/product/beta-ops.git \
  --local-path /srv/goclaw/checkouts/project-beta/beta-ops \
  --branch main
```

中央 `local_path` 是中央策略/Wave/commit 校验所见的 checkout。每台成员
工作站仍要通过 `--repo REPOSITORY_ID=/absolute/local/path` 建立自己的映射。

不要让两个项目登记并写入同一个工作副本。即使 remote 相同，也应使用独立的
managed checkout，避免策略、base、worktree 和证据串项目。

### 11.3 项目角色

| 角色 | 适合对象 | 主要权限 |
|---|---|---|
| `owner` | 项目负责人 | 全部项目操作，可授予 owner |
| `maintainer` | 技术负责人/交付负责人 | 几乎全部项目操作，但不能授予 owner |
| `developer` | 开发成员 | 读、Issue/WorkItem、Artifact/Document/Component 写入 |
| `reviewer` | 独立评审 | 读、Issue 流转、Artifact/Document 写入 |
| `viewer` | 观察者/业务方 | 只读 |

最终 `dev accept` 需要 `project.manage + task_accept`，通常由 owner 或
maintainer 执行。`reviewer` 角色名称不等于最终任务验收权限。

项目成员必须先是活动 Team 成员：

```bash
goclaw team project-member-add \
  --project project-alpha \
  --user alice \
  --role developer \
  --domain billing \
  --domain api \
  --capacity 80

goclaw team project-member-add \
  --project project-beta \
  --user bob \
  --role developer \
  --domain operations \
  --capacity 60
```

`--domain` 可重复；`capacity` 范围是 `0..10000`。当前 capacity 和
business domain 用于校验、计划和看板，不是自动优化排程器。

### 11.4 项目查询

当前用户可访问的项目：

```bash
goclaw team rpc project.list
```

`project-get.json`：

```json
{
  "project_id": "project-alpha"
}
```

```bash
goclaw team rpc project.get --params project-get.json
goclaw team rpc project.members --params project-get.json
goclaw team rpc repository.list --params project-get.json
```

`team.members` 实际是项目级团队看板投影，也使用 `project_id`：

```bash
goclaw team rpc team.members --params project-get.json
```

## 12. 多项目并发模型

### 12.1 三个硬约束

1. 一个 `runner work` 进程一次只消费一个 `--project` 队列。
2. 一个 Runner ID 同时最多持有一个活动 lease。
3. 任务 `assignee_id` 必须等于 Runner owner；一个成员的 Runner 不能领取
   指派给另一成员的任务。

因此有两种正确部署：

| 模式 | Runner 注册 | 进程 | 效果 |
|---|---|---|---|
| 多项目串行 | 一个 Runner ID 授权多个项目 | 每次只启动一个项目队列 | 省资源，但不能并行 |
| 多项目并发 | 每个活动项目独立 Runner ID/key/work root | 每个项目一个进程 | 同一 Linux substrate 可并发 |

绝对不要用同一个 Runner ID 启动两个 `runner work` 进程。

### 12.2 推荐命名

```text
Runner ID:     <member>-<project>-<substrate>
Key file:      ~/.config/goclaw/<runner-id>.key
Work root:     ~/.local/share/goclaw/<runner-id>/worktrees
Repository:    ~/src/<project>/<repository>
Service:       goclaw-runner-<project>.service
```

示例：

```text
alice-project-alpha-wsl2
alice-project-beta-wsl2
```

### 12.3 串行多项目 Runner

注册时可重复 `--project`：

```bash
goclaw runner register \
  --id alice-shared-linux \
  --name "Alice Shared Linux" \
  --project project-alpha \
  --project project-beta \
  --capability codex \
  --verification-sandbox \
    /usr/local/libexec/goclaw/verify-sandbox-bwrap.sh \
  --key-file ~/.config/goclaw/alice-shared-linux.key
```

但同一时刻只运行一个队列：

```bash
goclaw runner work \
  --id alice-shared-linux \
  --key-file ~/.config/goclaw/alice-shared-linux.key \
  --work-root ~/.local/share/goclaw/alice-shared-linux/worktrees \
  --repo repo-alpha-api=/home/alice/src/project-alpha/alpha-api \
  --project project-alpha \
  --codex-command "$(command -v codex)" \
  --verification-sandbox \
    /usr/local/libexec/goclaw/verify-sandbox-bwrap.sh
```

切换到 Beta 前先停止上述进程，再用同一 ID 启动 `--project project-beta`。

`runner update --project ...` 会**替换**授权项目集合，不是追加：

```bash
goclaw runner update \
  --id alice-shared-linux \
  --project project-alpha \
  --project project-beta
```

### 12.4 同一成员双项目并发

为两个项目分别注册：

```bash
goclaw runner register \
  --id alice-project-alpha-linux \
  --name "Alice Alpha Runner" \
  --project project-alpha \
  --capability codex \
  --verification-sandbox \
    /usr/local/libexec/goclaw/verify-sandbox-bwrap.sh \
  --key-file ~/.config/goclaw/alice-project-alpha-linux.key

goclaw runner register \
  --id alice-project-beta-linux \
  --name "Alice Beta Runner" \
  --project project-beta \
  --capability codex \
  --verification-sandbox \
    /usr/local/libexec/goclaw/verify-sandbox-bwrap.sh \
  --key-file ~/.config/goclaw/alice-project-beta-linux.key
```

分别预检：

```bash
goclaw runner doctor \
  --key-file ~/.config/goclaw/alice-project-alpha-linux.key \
  --work-root ~/.local/share/goclaw/alice-project-alpha-linux/worktrees \
  --repo repo-alpha-api=/home/alice/src/project-alpha/alpha-api \
  --codex-command "$(command -v codex)" \
  --verification-sandbox \
    /usr/local/libexec/goclaw/verify-sandbox-bwrap.sh \
  --json

goclaw runner doctor \
  --key-file ~/.config/goclaw/alice-project-beta-linux.key \
  --work-root ~/.local/share/goclaw/alice-project-beta-linux/worktrees \
  --repo repo-beta-ops=/home/alice/src/project-beta/beta-ops \
  --codex-command "$(command -v codex)" \
  --verification-sandbox \
    /usr/local/libexec/goclaw/verify-sandbox-bwrap.sh \
  --json
```

只有两个报告都显示 `ready=true` 才启动。用两个独立终端或两个 systemd
服务运行：

```bash
goclaw runner work \
  --id alice-project-alpha-linux \
  --key-file ~/.config/goclaw/alice-project-alpha-linux.key \
  --work-root ~/.local/share/goclaw/alice-project-alpha-linux/worktrees \
  --repo repo-alpha-api=/home/alice/src/project-alpha/alpha-api \
  --project project-alpha \
  --codex-command "$(command -v codex)" \
  --verification-sandbox \
    /usr/local/libexec/goclaw/verify-sandbox-bwrap.sh \
  --heartbeat 30s \
  --poll 5s
```

```bash
goclaw runner work \
  --id alice-project-beta-linux \
  --key-file ~/.config/goclaw/alice-project-beta-linux.key \
  --work-root ~/.local/share/goclaw/alice-project-beta-linux/worktrees \
  --repo repo-beta-ops=/home/alice/src/project-beta/beta-ops \
  --project project-beta \
  --codex-command "$(command -v codex)" \
  --verification-sandbox \
    /usr/local/libexec/goclaw/verify-sandbox-bwrap.sh \
  --heartbeat 30s \
  --poll 5s
```

同一机器并发时还应限制 CPU、内存和磁盘，避免两个 Codex/测试进程相互挤压。

### 12.5 多成员、多项目容量建议

10 人团队可采用：

- 每人一个主责项目 Runner；
- 只有确有跨项目职责的人再创建第二个 project-scoped Runner；
- 每个活跃 Project 至少有一名 owner/maintainer、一名 developer 和一名
  独立最终验收人；
- 高风险项目使用独立 VM，而不是在同一 substrate 上叠加 Runner；
- 中央 Gateway 保持单写者，Runner 才是横向并发执行面。

当前系统不是多 Leader 集群，也没有数据库共识或自动故障转移。多个 Runner
可以并发领取不同任务，但不能让多个 Gateway 实例写同一 TeamControl root。

## 13. 项目任务完整闭环

Team 模式不能直接使用 `dev run/resume/repair` 在 Gateway 宿主执行。正确路径：

```text
Issue
  → WorkItem
  → Assignment
  → dev create
  → scenario/capacity/risk/cost 四审
  → freeze
  → enqueue
  → project-scoped runner work
  → Evidence/DoneGate
  → human accept
  → link-pr/CI/release evidence
```

### 13.1 ID 规则

Issue、WorkItem、Assignment、Task、Repository 和 Runner ID 都使用全局稳定且
不复用的名称，建议加项目短码：

```text
alpha-issue-2026-001
alpha-work-2026-001
alpha-assign-2026-001-alice
alpha-task-2026-001
```

精简 `dev create` 会生成名为 `implementation` 的单 WorkItem。它只适合
一次性 smoke。正式多项目任务必须使用 `--spec`，为 WorkItem 指定全局唯一 ID。

### 13.2 创建 Issue、WorkItem 与 Assignment

`issue-alpha-001.json`：

```json
{
  "id": "alpha-issue-2026-001",
  "project_id": "project-alpha",
  "type": "bug",
  "title": "订单重复创建",
  "description": "相同 Idempotency-Key 可能创建两笔订单",
  "severity": "high",
  "priority": "p1",
  "module": "orders",
  "expected": "相同幂等键只创建一笔订单",
  "actual": "并发请求创建两笔订单"
}
```

`work-alpha-001.json`：

```json
{
  "id": "alpha-work-2026-001",
  "project_id": "project-alpha",
  "issue_id": "alpha-issue-2026-001",
  "title": "实现订单幂等校验",
  "instructions": "只修改订单模块与对应测试",
  "business_domain": "billing",
  "priority": "p1",
  "estimate_points": 3,
  "verification_commands": [
    ["go", "test", "./internal/orders/..."],
    ["git", "diff", "--check"]
  ]
}
```

`assignment-alpha-001.json`：

```json
{
  "id": "alpha-assign-2026-001-alice",
  "project_id": "project-alpha",
  "target_type": "work_item",
  "target_id": "alpha-work-2026-001",
  "user_id": "alice",
  "role": "owner"
}
```

由项目管理者执行：

```bash
goclaw team rpc issue.create --params issue-alpha-001.json
goclaw team rpc work.create --params work-alpha-001.json
goclaw team rpc assignment.create --params assignment-alpha-001.json
```

每个准备入队的 WorkItem 应且仅应有一个 active owner。被冻结任务的
`assignee_id` 要与这个 owner 以及最终领取任务的 Runner owner 一致。

### 13.3 获取策略哈希

`policy-alpha.json`：

```json
{
  "project_id": "project-alpha",
  "repository_id": "repo-alpha-api"
}
```

```bash
goclaw team rpc policy.status --params policy-alpha.json
```

把返回的 `effective_version` 写入任务 spec 的 `policy_bundle_hash`。

### 13.4 正式任务 spec

以 `deploy/dev-task.example.json` 为模板，至少补齐下列项目级字段：

```json
{
  "id": "alpha-task-2026-001",
  "team_id": "team-product",
  "project_id": "project-alpha",
  "repository_id": "repo-alpha-api",
  "module": "orders",
  "assignee_id": "alice",
  "issue_ids": ["alpha-issue-2026-001"],
  "document_refs": ["vault://project-alpha/adr/order-idempotency"],
  "policy_bundle_hash": "<effective-version-from-policy-status>",
  "title": "为订单接口增加幂等校验",
  "repo_path": "/srv/goclaw/checkouts/project-alpha/alpha-api",
  "base_ref": "main",
  "request": {
    "raw_request": "增加 Idempotency-Key 幂等校验，保持成功响应兼容。",
    "source": "obsidian"
  },
  "goal": {
    "objective": "重复请求只创建一个订单",
    "non_goals": ["不重写订单存储层"],
    "success_tests": [
      "订单测试通过",
      "git diff --check 通过"
    ]
  },
  "plan": {
    "summary": "实现、测试并验证幂等边界",
    "milestones": [
      {
        "id": "alpha-m1",
        "title": "实现与验证",
        "work_items": [
          {
            "id": "alpha-work-2026-001",
            "title": "实现订单幂等校验",
            "instructions": "只修改 internal/orders 与对应测试",
            "assignee_id": "alice",
            "issue_ids": ["alpha-issue-2026-001"],
            "scope_paths": ["internal/orders/**"],
            "acceptance_criteria": [
              "重复和并发请求测试通过"
            ],
            "capability_manifest": {
              "executor": "codex-exec",
              "tools": ["filesystem", "shell"],
              "sandbox": "workspace-write"
            },
            "verification_commands": [
              {
                "name": "订单测试",
                "argv": ["go", "test", "./internal/orders/..."]
              },
              {
                "name": "补丁格式",
                "argv": ["git", "diff", "--check"]
              }
            ]
          }
        ]
      }
    ]
  },
  "evidence_plan": {
    "required": ["diff", "policy", "verification", "independent_review"],
    "commands": [
      {
        "name": "订单测试",
        "argv": ["go", "test", "./internal/orders/..."]
      },
      {
        "name": "补丁格式",
        "argv": ["git", "diff", "--check"]
      }
    ]
  },
  "scope": {
    "allowed_paths": ["internal/orders/**"],
    "denied_paths": [".env", ".env.*", "**/.env*"],
    "max_changed_files": 12,
    "max_changed_lines": 500,
    "allow_new_dependency": false
  },
  "risk": {
    "level": "medium",
    "forbidden": [
      "修改生产配置",
      "推送远端分支",
      "创建或删除数据库"
    ],
    "rollback": "丢弃隔离 worktree 或回退验收后的本地提交。",
    "human_escalates": [
      "需要修改公开 API",
      "需要增加依赖",
      "冻结验证不可用"
    ]
  },
  "cost": {
    "max_repair_attempts": 2,
    "max_input_tokens": 200000,
    "max_output_tokens": 30000
  },
  "done_gate": {
    "require_changed_files": true,
    "require_all_verifications": true,
    "require_policy_pass": true,
    "require_independent_review": true,
    "require_human_acceptance": true,
    "require_work_item_traceability": true
  },
  "created_by": "alice",
  "requested_by": "alice"
}
```

Team Gateway 还要求声明 active Wave Step。实际创建时：

```bash
goclaw dev create \
  --project project-alpha \
  --spec /secure/goclaw/alpha-task-2026-001.json \
  --wave-step <active-wave-step>
```

Gateway 会解析并冻结权威 Registry、plan revision/hash 与精确 Git base；
不要在客户端伪造这些权威字段。Team Gateway 还会使用登记的 Repository
`local_path` 覆盖客户端 `repo_path`，因此这里应写中央 checkout，不能把
成员工作站路径当作控制面权威路径。

### 13.5 四审、冻结和入队

四审命令示例：

```bash
goclaw dev review alpha-task-2026-001 \
  --project project-alpha \
  --kind scenario \
  --decision approved \
  --reviewer alice \
  --comment "场景覆盖正常、重复和并发请求" \
  --counterargument "唯一索引失败时仍需验证回滚" \
  --evidence-ref "vault://project-alpha/reviews/scenario-001"

goclaw dev review alpha-task-2026-001 \
  --project project-alpha \
  --kind capacity \
  --decision approved \
  --reviewer alice \
  --comment "容量与窗口允许执行" \
  --counterargument "并发测试可能延长执行时间" \
  --evidence-ref "vault://project-alpha/reviews/capacity-001"

goclaw dev review alpha-task-2026-001 \
  --project project-alpha \
  --kind risk \
  --decision approved \
  --reviewer bob \
  --comment "变更范围受控且可回滚" \
  --counterargument "数据库唯一约束可能影响旧数据" \
  --evidence-ref "vault://project-alpha/reviews/risk-001"

goclaw dev review alpha-task-2026-001 \
  --project project-alpha \
  --kind cost \
  --decision approved \
  --reviewer bob \
  --comment "成本上限可接受" \
  --counterargument "repair 超限应停止并修订任务" \
  --evidence-ref "vault://project-alpha/reviews/cost-001"
```

冻结：

```bash
goclaw dev freeze alpha-task-2026-001 \
  --project project-alpha \
  --reviewer bob
```

入队：

```bash
goclaw dev enqueue alpha-task-2026-001 \
  --project project-alpha \
  --priority 80 \
  --capability codex \
  --max-attempts 3
```

服务器从 frozen task 构造 secret-free ExecutionPack。只有：

- Runner owner 等于 `assignee_id`；
- Runner 被授权访问该 Project；
- capability 匹配；
- Runner 在线且没有活动 lease；

才可领取。

### 13.6 监控多项目

开发任务：

```bash
goclaw dev list --project project-alpha
goclaw dev list --project project-beta

goclaw dev show alpha-task-2026-001 --project project-alpha
```

Runner：

```bash
goclaw runner list --project project-alpha
goclaw runner list --project project-beta
```

项目级工作、Issue 和 Assignment：

```bash
goclaw team rpc work.items --params project-get.json
goclaw team rpc issue.list --params project-get.json
goclaw team rpc assignment.list --params project-get.json
```

Web Console 与 Obsidian 团队页也必须先选择 Project；不存在“无项目的全局
任务队列”。项目 A 的对话、任务、记忆、审批和 Runner 不应投影到项目 B。

### 13.7 验收

Runner 完成后，任务进入 `awaiting_acceptance`。具备最终权限且与执行职责
独立的人验收：

```bash
goclaw dev accept alpha-task-2026-001 \
  --project project-alpha \
  --reviewer carol \
  --comment "冻结测试、策略、diff 与独立评审均通过" \
  --counterargument "尚需外部 CI 验证目标环境" \
  --evidence-ref "vault://project-alpha/acceptance/alpha-task-2026-001"
```

验收完成后再由开发者把 accepted patch 应用到受控分支、提交并创建 PR。
中央 checkout 可见该 commit 后登记：

```bash
goclaw dev link-pr alpha-task-2026-001 \
  --project project-alpha \
  --commit <accepted-commit-sha> \
  --url <pull-request-url>
```

`link-pr` 不会自动 fetch、push、创建、批准或 merge PR，也不会等待 CI。

### 13.8 失败、取消与修订

- `queued`/`failed` 队列任务可以取消；
- 活动 lease 不能直接取消，应先安全停止/等待；
- 实质改变范围、策略、验证或目标时必须 `dev revise`，重新四审、冻结和入队；
- repair 次数、文件数、变更行数或 Token 成本超限时失败关闭；
- WorkItem 未完成或存在 cancelled 分支时，不应自动关闭共享 Issue。

示例：

```bash
goclaw dev revise alpha-task-2026-001 \
  --project project-alpha \
  --reason "需要扩大测试范围" \
  --reviewer bob \
  --spec /secure/goclaw/alpha-task-2026-001-r2.json
```

`dev revise --spec` 读取的是当前 `Task` 的 replacement JSON，而不是首次
创建使用的 `CreateRequest`。应先用 `dev show` 导出当前 Task，在不改变
Team、Project、Repository、assignee、Issue ID 和 WorkItem ID 的前提下修订
目标、范围或验证；Team Gateway 会重置并重新派生服务端权威字段。

## 14. systemd 持久运行与并发服务

源码提供：

- `deploy/runner.env.example`
- `deploy/systemd/goclaw-runner.service.example`

单项目：

```bash
install -d -m 0700 ~/.config/goclaw
install -m 0600 deploy/runner.env.example ~/.config/goclaw/runner.env
install -d -m 0755 ~/.config/systemd/user
install -m 0644 \
  deploy/systemd/goclaw-runner.service.example \
  ~/.config/systemd/user/goclaw-runner.service
```

编辑 `runner.env` 后：

```bash
systemctl --user daemon-reload
systemctl --user enable --now goclaw-runner.service
systemctl --user status goclaw-runner.service
journalctl --user -u goclaw-runner.service -f
```

多项目并发时，为每个项目建立独立 env 和 service，并修改：

- `GOCLAW_RUNNER_ID`；
- `GOCLAW_RUNNER_KEY_FILE`；
- `GOCLAW_RUNNER_WORK_ROOT`；
- `GOCLAW_PROJECT_ID`；
- `GOCLAW_REPOSITORY_ID/PATH`；
- unit 的 `EnvironmentFile`；
- unit 文件名。

不要让两个 unit 复用 Runner ID、key file 或 work root。

## 15. Runner key 轮换与维护

先停止服务并确认 Runner 空闲：

```bash
systemctl --user stop goclaw-runner.service
goclaw runner list --project project-alpha
```

然后按 CLI 帮助使用 `runner rotate-key` 创建新的排他 key 文件。活动 lease
期间轮换会被拒绝。完成后更新服务的 key file 路径、执行 `runner doctor`
并重启服务。

禁用/启用空闲 Runner：

```bash
goclaw runner update --id alice-project-alpha-linux --disable
goclaw runner update --id alice-project-alpha-linux --enable
```

修改授权项目同样要求 Runner 空闲，并且 `--project` 列表是全量替换。

## 16. 构建与部署验收清单

### 16.1 构建

- [ ] Go 版本为 `1.25.5`；
- [ ] Web Console `npm test` 与 `npm run build` 通过；
- [ ] 关键 Go 发布测试集通过；
- [ ] Linux amd64/arm64 构建信息中的 GOOS/GOARCH 正确；
- [ ] Windows/macOS 应用可以执行 `version` 和控制命令；
- [ ] Windows/macOS 原生 strict 失败关闭；delegated Doctor 明确 warning，
      且只在项目 policy/capability 匹配时领取；
- [ ] 发布包 SHA-256 校验通过；
- [ ] 源码包不含 `.env`、Token、OAuth、本地 runtime 或二进制。

### 16.2 身份

- [ ] 一人一 Team Token，一设备/Runner 一 device key；
- [ ] 一人一本地 Codex OAuth；
- [ ] Team Token 明文只交付一次，中央只存摘要；
- [ ] Token 设置到期时间和轮换负责人；
- [ ] 旧 Token 在新 Token 验证后撤销；
- [ ] `planner-service` 没有 Token；
- [ ] secrets 不进入 Git、Vault、日志、任务 spec 或 ExecutionPack。

### 16.3 项目

- [ ] 每个项目使用稳定 Project ID 和 Key；
- [ ] 每个仓库有独立中央 checkout；
- [ ] 每位成员先有 Team membership，再有 Project membership；
- [ ] owner/maintainer、developer、独立验收人职责明确；
- [ ] business domain 与 capacity 已登记；
- [ ] `project.list` 只返回当前成员可访问的项目。

### 16.4 Runner 与并发

- [ ] WSL2 关闭 interop、Windows PATH 和 automount；
- [ ] Lima 关闭 host mounts；
- [ ] strict 执行路径在 Linux substrate 本地磁盘，bwrap wrapper 为
      `root:root 0755`；
- [ ] delegated 使用本机私有目录，Windows ACL/reparse 或 Unix owner/mode
      检查通过，且风险接受已记录；
- [ ] `runner doctor --json` 返回 `ready=true`；
- [ ] 每个并发项目使用独立 Runner ID/key/work root/process；
- [ ] 没有两个进程复用同一 Runner ID；
- [ ] Runner owner 与任务 assignee 一致；
- [ ] 中央 root 只有一个 GoClaw 写进程。

### 16.5 任务闭环

- [ ] Issue、WorkItem、Assignment、Task ID 全局稳定且不复用；
- [ ] 正式多项目任务使用 `--spec`，不复用 `implementation`；
- [ ] WorkItem 恰有一个 active owner；
- [ ] policy hash 与 active Wave Step 已冻结；
- [ ] 四审、freeze、enqueue 全部按 Project 执行；
- [ ] Runner 只消费自己的 Project 队列；
- [ ] Evidence/DoneGate 通过后由独立人员 accept；
- [ ] PR、CI、发布和回滚证据继续登记。

## 17. 常见错误

### Windows/macOS strict 提示不能执行 Runner

这是预期行为。高保证任务进入 WSL2/Lima strict；只有项目明确配置
`runner.execution_profiles` 且接受无 GoClaw OS sandbox 的边界时，才在
register/doctor/work 三处显式选择 `codex-delegated`。

### Runner 一直领不到任务

依次检查：

1. `runner list --project ...` 是否 online；
2. Runner owner 是否等于 Task `assignee_id`；
3. Runner 是否授权该 Project；
4. enqueue capability 是否与 Runner 匹配；
5. Runner 是否已有活动 lease；
6. 是否使用正确的 `--project` 队列；
7. task 是否已完成四审和 freeze。

### 同一机器第二个项目没有并发

若复用了同一 Runner ID，系统只允许一个活动 lease。为第二项目创建新的
Runner ID、device key、work root 和进程。

### 更新 Runner 后丢失原项目权限

`runner update --project` 是替换，不是追加。再次提交完整授权项目列表。

### Token 无法签发/列出/撤销

检查操作者是否是目标用户所属每一个活动团队的 owner/admin，而不只是当前
Team 管理员。

### Token 已丢失

服务端不能恢复明文。签发新 Token，验证后撤销旧 credential。

### `dev create` 出现 WorkItem 冲突

不要在正式多项目任务使用精简模式默认的 `implementation`。改用 `--spec`
并为 WorkItem 使用带项目短码的全局唯一 ID。

### 中央数据偶发冲突

确认没有第二个 Gateway/GoClaw 进程，也没有把 TeamControl/Workstation root
放在 NFS、SMB、Obsidian 同步盘或其他共享同步目录。

## 18. 当前能力边界

- 支持一个中央控制面协调多个项目、多个成员和多个并发 Runner；
- 支持项目级 RBAC、队列、lease、heartbeat、签名证据和人工最终验收；
- 不支持多个中央 Leader、数据库级分布式事务或自动故障转移；
- 不自动做容量最优排程，business domain/capacity 主要用于治理和看板；
- 不自动 commit、push、建 PR、批准、merge 或发布；
- 原生 Windows/macOS delegated 可执行，但不提供 GoClaw OS 级进程/网络
  隔离；strict 仍依赖 WSL2/Lima Linux substrate；
- device key 是共享 HMAC 秘密，不是 TPM 证明或不可抵赖设备证书；
- Obsidian 是项目交互和知识投影，不是秘密库、中央任务队列或 Runtime root。

## 19. 源码依据

主要实现和示例位于：

- `go.mod`
- `scripts/build-release.sh`
- `.goreleaser.yaml`
- `Makefile`
- `cli/team.go`
- `cli/runner.go`
- `cli/dev.go`
- `gateway/team_control.go`
- `gateway/workstation.go`
- `teamcontrol/`
- `workstation/`
- `orchestratorlite/types.go`
- `deploy/dev-task.example.json`
- `deploy/runner.env.example`
- `deploy/systemd/goclaw-runner.service.example`
- `deploy/wsl2/`
- `deploy/lima/`
- `scripts/verify-sandbox-bwrap.sh`
- `docs/TEAM_DEVELOPMENT_CN.md`
- `docs/WORKSTATION_RUNNER_CN.md`
- `docs/PILOT_3_PERSON_DEPLOYMENT_CN.md`
