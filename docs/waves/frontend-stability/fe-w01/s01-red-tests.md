# FE-W01 S01 Red-test Evidence

日期：2026-07-26
Task：`FE-W01-TRANSPORT-R1` revision 1
Plan：`FE-W01 plan-r003`
Base：`b288564361fac4f09d65e2a6a7ff80362a5cc12e`
Branch：`repair/fe-w01-transport-r1`

本 Evidence 记录修复实现前的确定性失败与安全正/负控制。测试只使用
loopback、临时端口、mock transport 和一次性 TeamControl 数据；未读取或记录
真实配置、Token、Cookie、Authorization、请求正文或用户数据。

## 测试变更

S01 只做下列测试基础工作：

- 新增 `ui/tests/team-transport.test.mjs`；
- 新增 `npm run test:transport` 脚本，不增加依赖；
- 修正 `team_runtime_test.go` 两个旧返回值接收方式，并断言 personal-token
  路径的 `browserSession == nil`；
- 增加跨端口 Origin、Sec-Fetch-Site、CSRF 和 browser-session RPC 测试。

此时 `ui/src/team/client.ts` 与 `ui/vite.config.ts` 相对 base 均无变化。

## Node/Vite red tests

命令：

```text
npm --prefix ui run test:transport
```

结果：2 tests，0 pass，2 fail。失败与 W00 根因一一对应：

```text
TeamClient:
actual   ws://127.0.0.1:28789/ws
expected ws://127.0.0.1:5173/ws

Vite /auth:
observed Host   127.0.0.1:<temporary-target-port>
observed Origin http://127.0.0.1:5173
actual status   403
expected status 204
```

测试中的真实 `/ws` proxy 对照在断言 auth status 前已完成；最终 green run
仍必须单独证明 101 与 Host/Origin 一致。

## Gateway security controls

目标 Gateway 测试恢复编译并通过：

```text
TestTeamPersonalTokenAuthenticationLayers                 PASS
TestWebSocketOriginPolicy/vite_direct_cross-port         PASS
TestCSRFHeader                                            PASS
TestBrowserSessionRPCRequiresCSRF/missing                 PASS
TestBrowserSessionRPCRequiresCSRF/wrong                   PASS
TestBrowserSessionRPCRequiresCSRF/correct                 PASS
TestWebSessionCreateRejectsUnsafeBrowserContext/
  same_hostname_different_port                            PASS
TestWebSessionCreateRejectsUnsafeBrowserContext/
  browser_reports_cross-site                              PASS
```

该结果证明测试编译阻断已修正，同时现有严格 Origin/CSRF/身份行为没有为了让
前端 red tests 通过而被放宽。

## 环境偏差

第一次 `npm ci` 因环境默认 cache 指向不可写位置而失败，并留下不完整、
Git 忽略的 `node_modules`。该目录被移动到 `/tmp` 保留诊断痕迹；第二次使用
任务专用 npm cache 与锁定 lockfile 成功安装 250 packages。没有修改
`package-lock.json`，没有把环境失败解释为产品失败。

## S01 结论

- `FE-ISSUE-002` 的测试编译阻断已由测试代码修正；
- `FE-ISSUE-003`、`FE-ISSUE-004` 已获得可执行 red tests；
- 允许进入 S02/S03 的最小实现；
- 尚不能标记任何 Issue fixed，Browser gate 仍未运行。
