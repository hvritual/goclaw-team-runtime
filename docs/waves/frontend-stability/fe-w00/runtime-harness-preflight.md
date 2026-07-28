# FE-W00 运行夹具预检

- Wave：`FE-W00`
- Step：`FE-W00-S01` / `FE-W00-S03`
- Evidence：`FE-EVID-W00-006`
- 日期：2026-07-26
- 状态：`collecting`
- 产品代码变更：无

本文件确定“怎样测试才不会误判”，不证明前台异常已复现。

## 首选运行拓扑

浏览器测试应优先使用 Go Gateway WebSocket server 提供的嵌入式同源入口：

```text
http(s)://<test-gateway-host>:<websocket-port>/dashboard/
  ├─ /auth/session
  ├─ /rpc
  ├─ /ws
  ├─ /health
  └─ /assets
```

依据：

- `gateway/server.go:278-295` 在 WebSocket server 上同时注册 `/ws`、
  `/rpc`、`/auth/session`、`/health`、`/api/channels`、`/dashboard` 和
  `/assets`；
- `gateway/ui.go` 把 `gateway/ui_dist` 嵌入该入口；
- `gateway/web_sessions.go:261-274` 要求浏览器 `Origin` 的 host（包含
  port）与请求 `Host` 一致；
- 生产文档也要求上述路径进入同一个 Origin。

静态文件服务器只能检查 HTML/JS/CSS 是否能加载，不能验证 session、
Cookie、CSRF、RPC、项目 RBAC、WebSocket 或命令状态机。

## 开发模式的未验证风险

当前开发配置存在一个必须运行复现的静态契约差异：

1. `ui/vite.config.ts` 把 `/auth`、`/rpc`、`/health` 和 `/api/channels`
   代理到 HTTP gateway，把 `/ws` 代理到 WebSocket gateway；
2. `ui/src/team/client.ts:7-11` 在 `DEV` 模式不使用同源 `/ws`，而是直接
   连接页面 hostname 的 28789 端口；
3. `checkWebSocketOrigin` 比较完整 host:port，因此 5173 页面发往 28789
   的 WebSocket Origin 与 Host 不同；
4. Vite 的 `changeOrigin` 与浏览器保留的 Origin 如何共同影响
   `POST /auth/session` 也必须在真实 Vite/Gateway 环境验证。

可能结果是开发模式登录或 WebSocket 被同源策略拒绝，但在获得真实 HTTP
状态、Gateway 日志和浏览器 console 前，该项只能保持
`unverified contract observation`，不能登记为已确认 Bug。

## 一次性夹具要求

运行态复现不得使用生产 root 或真实项目。最小夹具需要：

| 资源 | 最低要求 |
|---|---|
| runtime root | 新建、独立、可整体销毁 |
| Gateway | 测试用连接 Token；同源 HTTP/WebSocket 入口 |
| TeamControl | 首个测试管理员；不复用真实用户 |
| 项目 | authorized project A 与 unauthorized project B |
| principal | owner、maintainer、developer、reviewer、viewer |
| 数据 | empty、normal、large/long text |
| 状态 | W00 plan 中列出的 Dev、Ouroboros、Harness、Runner 和会话状态 |
| Reviewer | 只用于测试的独立治理身份 |
| 清理 | 停止进程、撤销 Token、删除一次性 root，并保存脱敏证据引用 |

当前目录只有 example 配置，没有已冻结的一次性配置或可执行重置脚本。
Go 工具链缺失也使 Gateway 和 CLI 无法从当前源码启动。因此
`FE-W00-S03` 尚未开始。

## 现有自动化覆盖

当前 Go 源码包含以下底层测试：

- `gateway/web_sessions_test.go`
  - session lifecycle；
  - revocation；
  - WebSocket Origin policy；
  - CSRF header；
  - cross-site login rejection。
- `gateway/server_auth_test.go`
  - Gateway WebSocket/HTTP 认证；
  - HttpOnly cookie；
  - short-lived Web session perimeter；
  - public but hardened Team Console shell；
  - proxy loopback 信任边界。
- `gateway/team_guard_test.go`
  - legacy/unknown RPC deny-by-default；
  - Memory Catalog project scope。
- `gateway/team_runtime_test.go`
  - personal token layers；
  - project chat isolation；
  - development/runner/issue lifecycle 的后端状态机。

由于当前环境没有 Go 1.25.5，这些测试本轮未执行。即使以后全部通过，它们
也不能替代九个页面的浏览器 loader、命令和恢复测试。

Web Console `ui/package.json` 没有 UI test 或 E2E script；只有 TypeScript
和 Vite build。本轮 build 已通过，但这不是交互证据。

## 当前浏览器阻塞

按前端测试规范已优先选择 Cloud Browser。该通道明确拒绝访问本地预览
地址；没有改用普通 Playwright、原始 CDP 或其他浏览器通道规避。

继续运行复现需要以下二者之一：

1. 一个经过授权、Cloud Browser 可访问、只承载一次性测试数据的同源
   Gateway URL；或
2. 用户明确允许前端测试规范所述的普通 Playwright fallback，并且当前
   环境具备浏览器依赖。

在此之前，截图、DOM、console、网络请求和交互项均保持 `not-run`。

## 放行条件

- 冻结真实 Git commit；
- 提供 Go 1.25.5；
- 建立可重复创建/销毁的一次性运行夹具；
- 使用嵌入式同源入口完成 Browser 页面身份检查；
- 分别验证 Vite dev 与 embedded production 拓扑；
- 把开发模式 Origin/Host 观察复现为具体 Issue，或用证据标记
  `not-a-bug`；
- 证据脱敏后登记到 evidence index。
