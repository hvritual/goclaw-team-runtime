# FE-EVID-W01-021 — r012 MVP 环境预检与失败关闭

日期：2026-07-28
Wave/Step：`FE-W01 r012` / `FE-W01-S13B`、`S13C`
Task：`FE-W01-MVP-BROWSER-012 r012`
Issues：`FE-ISSUE-007`、`FE-ISSUE-010`
证据状态：`failed/environment-blocked`

## 精确对象

- preflight HEAD：
  `5eb7437a26149325c81e5763537e612a5495186d`；
- tree：`3767025fd708f93b71d7fd8c0c97ed58ea919f85`；
- recovered tag：`v0.8.0-pilot.1-recovered.1`；
- Task base：`683a008caf3c99642f5ec32a71443c63092f7a4c`；
- r012 Policy manifest SHA-256：
  `0d1645ea6c8c6b347ce1514ad5962ee9cd289840f92e4f008d4d0cfa0b4c0595`。

运行前后 `git status --short` 均为空；没有产品代码、配置、credential、
profile、截图、trace 或临时 Playwright 脚本进入仓库。

## 工具链与确定性 Gate

| Gate | 结果 |
|---|---|
| Go | `go1.25.5` |
| Node | `v24.14.0` |
| npm | `11.9.0` |
| UI install/test/build | `npm ci`；8/8；production build PASS |
| Gateway/session/agent race | 三包 PASS |
| tracked UI bundle | 无 diff |

确定性 Gate 通过只证明现有 recovered source 可构建和回归，不替代真实
Browser、syscall 或 credential owner Gate。

## ptrace capability

先以 `strace 6.8` 执行：

```text
strace -f -e trace=connect -o <temporary-file> /bin/true
```

结果：

- `PTRACE_TRACEME: Operation not permitted`；
- `PTRACE_SETOPTIONS: Operation not permitted`；
- trace 文件为 0 bytes；
- stderr/trace 临时文件已删除。

能力测试在任何 Gateway/provider/runtime 前失败，因此未启动 Gateway，
未创建 inert config、public marker、session database 或 provider 进程，
也未用 socket polling 替代 syscall 证明。

## Browser availability 与失败

Browser plugin 与 `control-browser` skill 均可用；Browser runtime 成功绑定
Chrome/CDP，并命名 session 为 `GoClaw MVP-W01 browser gate`。

Vite 仅在 `127.0.0.1:5173` 启动，显示 ready 后，Browser plugin 尝试导航
到 `http://127.0.0.1:5173/`。Browser 安全策略拒绝该动作，返回的权威原因
是本地 URL 权限被拒，并明确要求不得通过 workaround、raw CDP、其他
browser surface 或 policy circumvention 实现同一结果。

因此：

- 没有获得页面 URL/title/DOM/Console/screenshot；
- 没有执行登录、WebSocket、刷新、401、退出或项目切换；
- 没有执行 Playwright fallback；
- Vite 已立即终止；
- 没有 Gateway/provider/credential/runtime。

r012 的一般 fallback 授权不能覆盖本次 action-specific Browser 安全拒绝。
该结果是安全失败关闭，不是 Browser 回归通过。

## credential owner

本次未获得历史 credential-shaped material 的责任人证明。Codex 未生成、
推断或代签撤销/轮换/从未有效结论，`FE-ISSUE-007` 保持外部阻断。

## 结论

| Gate | 状态 |
|---|---|
| deterministic UI/Go | `passed` |
| Browser rendered flow | `failed/environment-blocked` |
| ptrace/syscall zero-outbound | `failed/environment-blocked` |
| credential owner closure | `blocked/external-owner` |

`FE-W01` 不得 complete，`MVP-W02`/`PILOT-W00` 不得启动。后继执行需要：

1. 能允许本地 URL 的受控 Browser/Playwright 环境；
2. 能让相同 `/bin/true` strace capability test 通过的 ptrace-capable host；
3. credential owner 的撤销、轮换或从未有效证明。

不得通过降低浏览器、syscall、credential 或跨项目隔离门禁来关闭本证据。
