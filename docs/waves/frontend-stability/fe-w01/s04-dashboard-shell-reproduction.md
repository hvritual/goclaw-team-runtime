# FE-W01 S04 — Dashboard Shell 301 自循环复现

日期：2026-07-26
适用产品基线：`b288564361fac4f09d65e2a6a7ff80362a5cc12e`
发现步骤：`FE-W01-S04` 全量 Gateway Gate
Issue：`FE-ISSUE-005`

本证据只使用内存 HTTP recorder、嵌入式静态文件和脱敏的 loopback URL。
没有启动外部服务，没有读取配置、Token、Cookie、项目数据或用户数据。

## 1. 环境与身份

| 字段 | 冻结值 |
|---|---|
| Runtime | Go `1.25.5 linux/amd64` |
| 源码 | 0.7.0；Task worktree HEAD `697f50e5f428769b75061dfd859d2549dd1c330d` |
| 产品基线 | `b288564361fac4f09d65e2a6a7ff80362a5cc12e` |
| Handler | `gateway.TeamConsoleHandler()` |
| 请求 | `GET http://localhost/dashboard/` |
| Principal / project / role | 不适用；该 route 只返回公开的 immutable shell，数据仍受 session、CSRF 与 RBAC 保护 |
| 浏览器 | 未使用；Cloud Browser 对 localhost 的策略阻塞仍然有效 |

## 2. 稳定复现

已有测试把 `/dashboard/` 的权威合同冻结为：

- 返回 `200 OK`；
- `Content-Security-Policy` 存在；
- `X-Frame-Options: DENY`；
- shell 使用 `Cache-Control: no-store`。

重复两次执行：

```text
go test ./gateway \
  -run '^TestTeamConsoleShellIsPublicButHardened$' \
  -count=2 -v
```

实际结果：

```text
=== RUN   TestTeamConsoleShellIsPublicButHardened
    server_auth_test.go:101: team console status = 301
--- FAIL: TestTeamConsoleShellIsPublicButHardened (0.00s)
=== RUN   TestTeamConsoleShellIsPublicButHardened
    server_auth_test.go:101: team console status = 301
--- FAIL: TestTeamConsoleShellIsPublicButHardened (0.00s)
FAIL
```

独立的内存 response probe 返回：

```json
{
  "cache_control": "no-store",
  "frame_options": "DENY",
  "location": "./",
  "status": 301
}
```

浏览器对相对位置的标准解析结果为：

```text
new URL("./", "http://localhost/dashboard/").href
=> http://localhost/dashboard/
```

因此客户端会再次请求同一个 URL，形成稳定的重定向自循环，Team Web Console
shell 无法加载。这不是视觉推断。

## 3. 基线归属

当前 worktree 中两个相关文件没有未提交 diff，且与冻结产品基线逐文件一致：

| 文件 | 当前 SHA-256 | 基线 SHA-256 |
|---|---|---|
| `gateway/ui.go` | `c201ef99b84acc0158554da31b7fe781a7f316b621b900e10981e72b32fe94b3` | `c201ef99b84acc0158554da31b7fe781a7f316b621b900e10981e72b32fe94b3` |
| `gateway/server_auth_test.go` | `3cbfe720b02da0d636e4cfe5173cb82a7290d534546b3525bb16ebfa5ff58967` | `3cbfe720b02da0d636e4cfe5173cb82a7290d534546b3525bb16ebfa5ff58967` |

该失败在 FE-W01 transport patch 之前已经存在，不由
`ui/src/team/client.ts`、`ui/vite.config.ts` 或新增 transport tests 引入。

## 4. 根因

`TeamConsoleHandler` 对 `/dashboard/` 执行以下映射：

```text
/dashboard/
→ /
→ /index.html
→ /ui_dist/index.html
→ http.FileServer
```

Go `http.FileServer` 对 URL path 以 `/index.html` 结尾的请求执行 canonical
redirect，将其重定向到 `./`。Handler 在交给 FileServer 前把公开 URL path
改成了内部 embed path，因而触发这个特殊重定向；返回给浏览器的 `./` 又解析
回 `/dashboard/`。

根因由测试、HTTP response、URL 解析和标准库行为共同支持，状态可进入
`root-caused`。

## 5. 安全边界与计划影响

最小修复方向是只对 `/dashboard/` 直接读取并返回嵌入的
`ui_dist/index.html`，不把内部 `/index.html` path 交给 FileServer；assets
继续由现有 FileServer 提供。

修复不得：

- 放宽 Gateway Origin、CSRF、browser session、Gateway/Team 双层身份或 RBAC；
- 把 Token 放入 URL、HTML、日志或 WebSocket subprotocol；
- 改变 `/dashboard` 到 `/dashboard/` 的单次 canonical redirect；
- 把 shell 或 `index.html` 改为长期缓存；
- 把 hashed assets 改为 `no-store`；
- 将未知路径静默回退为 shell。

`gateway/ui.go` 与 `gateway/server_auth_test.go` 不在 FE-W01 r003 allowlist。
因此 S04 已停止，产品修复必须先由新的 Plan revision 授权并通过独立评审。
