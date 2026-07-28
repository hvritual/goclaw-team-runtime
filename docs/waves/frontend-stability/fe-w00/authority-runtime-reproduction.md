# FE-W00 权威基线与首批运行复现

日期：2026-07-26  
适用基线：`b288564361fac4f09d65e2a6a7ff80362a5cc12e`

本证据只记录脱敏环境、确定性构建与一次性协议探针。没有读取真实配置，
没有使用生产 Token、Cookie、项目或用户数据。

## 1. 权威源码选择

检查到两套候选源码：

- 原开发仓库仍停留在上游 `e05c79d`，工作树包含大量既有未提交成果，
  Team Web Console 仍是 `0.6.0` 形态；
- 当前审阅副本的产品源码与
  `goclaw-team-runtime-source-0.7.0.tar.gz` 逐文件一致，并包含新的
  `ui/src/team/**`、Web Session 和 Team Console 路由。

为避免覆盖旧工作树或把 0.6.0 与 0.7.0 混合，当前审阅副本被初始化为
独立本地 Git 仓库。首个 commit 同时冻结：

- 未修改的 0.7.0 产品源码；
- 已批准的 FE-W00 文档治理；
- 既有可选截图工件。

该仓库没有配置 remote，不会自动推送到外部。旧工作树未被修改或重置。

## 2. 工具链与可重建性

| 项目 | 结果 |
|---|---|
| Go | 官方 `go1.25.5 linux/amd64` |
| Go 下载 SHA-256 | `9e9b755d63b36acf30c12a9a3fc379243714c1c6d3dd72861da637f336ebb35b`，与 Go 官方 JSON 清单一致 |
| Node | `v24.14.0` |
| npm | `11.9.0` |
| Vite | `5.4.21` |
| UI build | 通过，48 modules transformed |
| UI 内容聚合 SHA-256 | `08171d6fa7c60f0627b922172de86814cf7bacd819ec70f8adfaaf71a475d3e8` |
| UI 可重建性 | 临时 build、`ui/dist`、`gateway/ui_dist` 三者逐文件一致 |
| Go product build | 通过；临时二进制 SHA-256 `701d62f13665cc9c78d2d224806b20eb5199edbbb2b6889c1ec4110e325d0609` |

Go module 与 build cache 位于本任务专用 workspace cache；没有复用个人
`HOME`、Codex OAuth、SSH agent、云凭据或 Docker/Kubernetes socket。

## 3. Gateway 测试基线失败

命令：

```text
go test ./gateway -run 'Test(WebSession|WebSocketOrigin)' -count=1 -v
```

实际结果：

```text
gateway/team_runtime_test.go:166:17:
assignment mismatch: 2 variables but server.authenticateTeamHTTP returns 3 values

gateway/team_runtime_test.go:171:15:
assignment mismatch: 2 variables but server.authenticateTeamHTTP returns 3 values
```

产品包可以构建；失败只发生在 Gateway 测试编译。两个调用点仍使用旧的双返回
值契约，而实现已经返回 principal、browser session 与 error。该问题登记为
`FE-ISSUE-002`，不能通过跳过整个 Gateway 测试包来放行。

随后执行全仓只编译测试：

```text
go test ./... -run '^$' -count=1
```

除 `gateway` 的上述两个错误外，其余含测试的 Go package 均完成编译。这把
当前确定性编译阻断收敛到同一函数的两个旧调用点，不代表尚未实际执行的
全量测试已经通过。

## 4. `/auth` 开发代理复现

一次性探针使用真实 Vite proxy 和临时 HTTP target。target 按当前 Gateway
规则检查 `Origin.host == Host`，不读取请求正文或凭据。

实际观测：

```json
{
  "vite_port": 5173,
  "target_port": 41407,
  "status": 403,
  "observed": {
    "host": "127.0.0.1:41407",
    "origin": "http://127.0.0.1:5173",
    "path": "/auth/session"
  }
}
```

`ui/vite.config.ts` 的 `/auth.changeOrigin = true` 改写 Host，但浏览器 Origin
保持页面来源，确定性触发 Gateway 的严格同源拒绝。该问题登记为
`FE-ISSUE-003`。

## 5. WebSocket 直连与同源代理对照

一次性探针使用真实 HTTP Upgrade 和 Vite WebSocket proxy。临时 target
仍只检查当前 Gateway 的同源规则。

实际观测：

```json
{
  "vite_port": 5173,
  "target_port": 45027,
  "direct_status": 403,
  "proxied_status": 101,
  "observed": [
    {
      "host": "127.0.0.1:45027",
      "origin": "http://127.0.0.1:5173",
      "path": "/ws"
    },
    {
      "host": "127.0.0.1:5173",
      "origin": "http://127.0.0.1:5173",
      "path": "/ws"
    }
  ]
}
```

另以 Vite SSR 加载真实 `TeamClient`，mock 浏览器 transport 后捕获：

```json
{
  "captured": "ws://127.0.0.1:28789/ws",
  "expected": "ws://127.0.0.1:5173/ws",
  "matches": false
}
```

当前 DEV client 绕过已经配置的 `/ws` proxy；直连跨端口被严格 Origin
策略拒绝，同源代理则保留 Host/Origin 并完成 101。该问题登记为
`FE-ISSUE-004`。

## 6. 安全结论与最小修复方向

- 不允许放宽 `checkWebSocketOrigin`。
- `/auth` 开发代理必须保留页面 Host。
- DEV WebSocket 必须使用当前页面 host 的 `/ws`，由 Vite 代理到 Gateway。
- Gateway 回归测试必须明确断言跨端口直连仍被拒绝。
- 生产同源 URL、Cookie、CSRF、双层身份和项目 RBAC 均不在本修复的改变范围。

## 7. 浏览器边界

Cloud Browser 已拒绝访问本地预览 URL。依照当前前端测试规范，没有改用其他
浏览器通道规避该限制。因此：

- 本证据足以支持协议级失败测试和最小 transport 修复；
- 不足以声明真实页面交互或视觉回归已经通过；
- W01 可以实施确定性修复，但在获得可达浏览器前不得完成 browser exit gate。
