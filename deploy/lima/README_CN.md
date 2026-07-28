# Lima Runner 试点模板

Lima 是 macOS 成员的 Linux 执行底座，不是原生 macOS Runner。模板关闭
host mount；仓库、work root、device key 和 Codex OAuth 都必须在 guest
磁盘中单独创建。

1. 用 `goclaw-runner.yaml.example` 创建专用 guest。
2. 进入 guest，安装 Linux `goclaw`，再以该成员自己的 ChatGPT 订阅执行
   `codex login`；不得从 macOS Home 复制 OAuth。
3. 在 `/var/lib/goclaw-runner/src` 内重新 clone 授权仓库。不要启用
   virtiofs、9p、sshfs 或其他 host share。
4. 安装受审 bwrap wrapper 和通用 systemd unit，填写
   `runner.env.example`。
5. `goclaw runner doctor --json` 必须报告 `host_profile=lima`、
   `ready=true`，然后才注册并启动服务。

macOS 上的控制 CLI 可以管理项目，但 `runner work` 会失败关闭。试点
任务只会分配给声明 `goclaw-runtime-linux-v1` 的 guest Runner。
