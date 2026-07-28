# WSL2 Runner 试点模板

WSL2 是 Windows 成员的 Linux 执行底座，不是原生 Windows Runner。

1. 新建一套专用于 GoClaw 的 WSL2 发行版，不复用日常开发发行版。
2. 把 `wsl.conf.example` 安装为 `/etc/wsl.conf`，从 Windows 执行
   `wsl.exe --shutdown` 后重启；`runner doctor` 必须确认 interop 和
   Windows PATH 已关闭。
3. 在发行版内安装 Git、Codex CLI、`bubblewrap` 和 Linux 版 `goclaw`。
4. 仓库、work root、device key 和 `CODEX_HOME` 全部放在发行版虚拟磁盘；
   不允许 `/mnt/c`、drvfs、9p 或 Windows 符号链接。
5. 复制 `runner.env.example` 和通用 systemd unit，权限分别设为 `0600`
   和 `0644`。wrapper 由 root 安装为 `root:root 0755`。
6. 先执行与 unit 完全相同参数的 `goclaw runner doctor --json`，只有
   `ready=true` 才注册并启动服务。

Windows 上的 `goclaw.exe` 只用于控制面命令和诊断展示，不能执行
`runner work`；试点任务要求 capability `goclaw-runtime-linux-v1`。
