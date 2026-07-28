# FE-EVID-W01-014 — R5 确定性重验

## 证据主体

| 字段 | 值 |
|---|---|
| 时间 | 2026-07-26 |
| Project | `goclaw-team-runtime` |
| Wave / Plan | `FE-W01` / `plan-r007` |
| Task | `FE-W01-TRANSPORT-R1` revision `5` |
| Task Base | `2f9cd8289d7c05e44d30b70a07e7991036229bbf` |
| Freeze commit | `6bf8018` |
| Branch | `repair/fe-w01-transport-r5` |
| Worktree | `/workspace/scratch/afe5d81cd055/worktrees/fe-w01-transport-r5` |
| Plan SHA | `183cfd53e79841384af7323cc37cf2da924f5164bdece2b97684009e424770ec` |
| Cumulative W01 base | `697f50e5f428769b75061dfd859d2549dd1c330d` |
| Policy | `wave-governance-v1` |

## 安全迁移

- Task Base 的 legacy channel test 只执行无输出 SHA-256 比对，匹配计划冻结值；
- 以 Delete/Add 换成 R4 已验收 synthetic constructor test；
- source-first Gate 在任何 Go 程序或测试前通过；
- 未运行旧 channel test，未输出或归档 raw deletion diff/history；
- 其余 10 个产品文件随后迁移；
- 全部 11 个产品文件与 r007 冻结的 R4 SHA-256 manifest 精确匹配。

## 确定性 Gate

| Gate | 结果 |
|---|---|
| `gofmt` changed Go files | passed；无输出 |
| channel source credential/network/wait Gate | passed；无输出 |
| `go test ./channels -count=1 -v -timeout=30s` | passed |
| `go test -race ./channels -count=1 -timeout=30s` | passed |
| Catalog inference target，`count=2` | passed |
| `go test ./memory/catalog` | passed |
| `go test -race ./memory/catalog` | passed |
| Gateway Team Console target | passed |
| `go test ./gateway` | passed |
| `go test -race ./gateway` | passed |
| `go test ./... -count=1 -timeout=5m` | passed |
| `go vet ./...` | passed；无输出 |
| `npm --prefix ui run test:transport` | passed；2/2 |
| `npm --prefix ui run build` | passed；48 modules |
| Task Base 与累计 base `git diff --check` | passed |
| tracked/untracked allowlist | passed；11 个产品路径 + `docs/waves/**` |
| `ui/package-lock.json` | SHA-256 `46fd937f66b1b7a16950df8347619831948e9dded477b7d4ba8139018974bdbb`；未修改 |
| Playwright repo artifact absence | passed；无 config、report 或 test-results |

Go 使用任务工具链 `1.25.5`、隔离 `GOMODCACHE/GOCACHE`。UI 在 R5 使用独立
`npm ci` 和任务专用 npm cache；未复用 R4 `node_modules`。

## 结论与边界

R5 的 source-first、普通/race、全仓、vet、UI、scope、lockfile 和 manifest
门禁全部通过，允许进入 r007/S10 的仓库外 Playwright 准备阶段。

本证据不证明真实页面通过，不解除 `FE-W01-S08 / FE-EVID-W01-011`，不授权
产品提交或发布。Browser 结果必须进入 `FE-EVID-W01-015` 并独立复核。
