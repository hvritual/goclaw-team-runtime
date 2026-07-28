# FE-W01 S06 — Dashboard Shell Red/Green Evidence

日期：2026-07-26
Task：`FE-W01-TRANSPORT-R1` revision 2
Task Base：`1cc3c1188271f084e6412d62ef18d4edaf775193`
Plan：`FE-W01 plan-r004`
Issue：`FE-ISSUE-005`
Evidence：`FE-EVID-W01-008`

本 Evidence 只使用 `httptest`、Go embedded FS 和脱敏 loopback URL。没有读取
或记录配置、Token、Cookie、项目数据和用户数据。

## Red phase

在不修改 `gateway/ui.go` 的前提下，扩展
`gateway/server_auth_test.go`，冻结以下合同：

- `/dashboard` 单次 301 到 `/dashboard/`；
- GET shell 200、无 Location、HTML MIME、完整 CSP 与安全头、no-store；
- HEAD shell 200、无 body、与 GET 一致的 headers 和 Content-Length；
- embedded hashed JS/CSS 的正确 MIME、nosniff 与 immutable；
- unknown route 404、无 Location、无 shell marker。

命令：

```text
go test ./gateway \
  -run '^TestTeamConsole(ShellIsPublicButHardened|CanonicalRouteAndAssetCaching)$' \
  -count=1 -v
```

结果：

```text
TestTeamConsoleShellIsPublicButHardened
  GET team console status = 301, location = "./"     FAIL

TestTeamConsoleCanonicalRouteAndAssetCaching
  canonical_dashboard_redirect                       PASS
  hashed_assets_are_immutable_and_correctly_typed    PASS
  unknown_route_does_not_return_the_shell            PASS
```

红测只在已复现的 slash shell 合同失败；canonical redirect、真实 asset 与
unknown-route 负例均通过。该结果精确授权 r004 S06 对 `gateway/ui.go` 的
最小实现，不授权修改 `DashboardHandler`、Origin、CSRF、session、身份或
RBAC。

## Green phase

最小实现只修改 `gateway/ui.go`：

- `/dashboard/` 从 embedded FS 读取 `ui_dist/index.html`；
- 保留公开 URL path，不再把内部 `/index.html` path 交给 FileServer；
- 使用 `http.ServeContent` 保留 HEAD 与 Content-Length 语义；
- assets、unknown paths、旧 `DashboardHandler` 和 auth/RBAC 均未修改。

同一目标命令转绿：

```text
TestTeamConsoleShellIsPublicButHardened             PASS
TestTeamConsoleCanonicalRouteAndAssetCaching        PASS
  canonical_dashboard_redirect                      PASS
  hashed_assets_are_immutable_and_correctly_typed   PASS
  unknown_route_does_not_return_the_shell           PASS
```

补充回归：

| Gate | 结果 |
|---|---|
| `go test ./gateway -count=1` | PASS，`0.869s` |
| Node transport | PASS，2/2 |
| UI `tsc && vite build` | PASS，48 modules |
| 临时 production bundle | JS/CSS 文件名与现有 `ui/dist`、`gateway/ui_dist` 一致 |

Gateway race 和全仓 Go 属于 S04，不由本 Evidence 代替。S06 确定性合同已
red→green，当前状态：`passed`，等待独立复核。
