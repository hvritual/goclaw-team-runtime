# FE-EVID-W01-020 — R7 traceable deterministic revalidation

## 证据主体

| 字段 | 值 |
|---|---|
| 时间 | 2026-07-26 |
| Project-ID | `goclaw-team-runtime` |
| Repository-ID | `repo-goclaw-source-review` |
| Wave / Plan | `FE-W01` / `plan-r009` |
| Task | `FE-W01-TRANSPORT-R1` revision `7` |
| Step | `FE-W01-S12` |
| Issues | `FE-ISSUE-002`–`FE-ISSUE-009` |
| Reconstruction base | `5160273fb17502cf02cd10e1a17f5a47b7eb30be` |
| Task Base / activation | `8006415eb59952823434740ba2d855b6c66990f9` |
| Freeze commit | `ad4882080fe2e35b5f61574a645e6d0f1b811bfc` |
| Cumulative W01 diff base | `697f50e5f428769b75061dfd859d2549dd1c330d` |
| Branch | `repair/fe-w01-transport-r7` |
| Assignee | `Codex root agent` |
| Policy | `wave-governance-v1` / `98bacd6013032cbaffd15095012ed6fc7cd274b62a78d3fdd738aeeadff94ebf` |
| Environment | Linux `6.12.13` x86_64；Go `1.25.5`；Node `24.14.0`；npm `11.9.0` |
| 状态 | `collecting`；等待独立代码、安全与文档复核 |

## Traceability 与重建证明

- activation 的 parent 精确为 reconstruction base `5160273...`；
- freeze 的 parent 精确为 Task Base `8006415...`；
- activation 与 freeze commit 均含 Task、Project、revision、Work-Item、
  Wave、全部八个 Issue、Repository 和 policy hash trailers；
- repository-root `AGENTS.md` SHA-256 精确为冻结值；
- `plan-r008.md` SHA-256 精确为
  `dd25cb6397aeef4db1442ef79fea4e0a36fd3dcc2a11f5db87919f4904993392`；
- `plan-r009.md` SHA-256 精确为
  `ef7b3f829fbfa915e8a312459e083a0e78c9414ff948b8ac478a9c714338caa8`；
- Journal 前 `26641` bytes SHA-256 精确为
  `33a50e1bbd028ca06adcee3e18df0ea62f405ff72a6e982b318720c11bccf997`，
  新权威记录只追加在 EOF；
- R6 仍保留为技术通过、治理失败的只读证据；本次没有 amend、rebase、
  reset、删除或从 R6 commit 派生。

## Source-first 安全迁移

1. 在任何 Go 测试前，只以 SHA-256 核对 reconstruction base 的 legacy
   channel test，结果与冻结值一致；未输出文件内容、raw deletion diff 或
   历史版本；
2. 使用 Delete/Add 写入已审查的 synthetic constructor test；
3. 在运行任何 Go 程序前，credential source gate 与
   no-network/no-wait source gate 均无输出通过；
4. 随后只迁移其余 10 个批准产品文件；
5. 未运行 legacy channel test，未访问 WeWork 或其他外部 channel。

## 11 文件 manifest

| 路径 | SHA-256 |
|---|---|
| `ui/src/team/client.ts` | `ce72f1b15721a5be6ed9fe5625182d94bd3274c6181ce82d615c7c62d81813f6` |
| `ui/vite.config.ts` | `f335cbbc7c10201930c4e360ec95df99b7817891907fe98f21a7ad5bcf3868f3` |
| `ui/package.json` | `bc254d466186012ca175eab62e42a5be6ec81e3977a9997464e31e0fd93c2ee0` |
| `ui/tests/team-transport.test.mjs` | `0724cba8b1474263e9698f842d9529d15b9f6a9abb6e3d63d8772e65d89530ae` |
| `gateway/team_runtime_test.go` | `548ca84c27b667d100dffa0f54b5e895847a2ca747af4e39707922fe3dd40b09` |
| `gateway/web_sessions_test.go` | `920bbb7d55d6930f9de068a03939f17f961127b578c485d29fb8b3c5e7219405` |
| `gateway/ui.go` | `42287ecacc994c3d592ac870d9ebab9a4526c66caa23d36caa4383361eb4bbfa` |
| `gateway/server_auth_test.go` | `78eefc861967ad21603ecc122df1eb92ec53adaa2810a20c567ace3fe7d07215` |
| `channels/weworkwsbot_test.go` | `69b89a9fe880a68ae1a4ae50db0b0b3266a40cc405b38ceb57934c8901f6a92c` |
| `memory/catalog/ingest.go` | `391acede0cde724274281beaa854e8c07d170111503077a9fb2197ae30663032` |
| `memory/catalog/service_test.go` | `9a5a068fe7a29afebf0f9937c95b8194b5768fd89dfb42191e9bd0c43b482e64` |

结果为 `11/11`。产品 patch 保持 unstaged/uncommitted；本 Evidence 不授权
产品 commit。

## 确定性 Gate

| Gate | 结果 |
|---|---|
| `gofmt` changed Go files | passed；无输出 |
| channel synthetic credential/network/wait source Gate | passed；无输出 |
| `go test ./channels -count=1 -v -timeout=30s` | passed；package `0.030s` |
| `go test -race ./channels -count=1 -timeout=30s` | passed；package `1.036s` |
| Catalog inference target，`count=2` | passed |
| `go test ./memory/catalog -count=1 -timeout=30s` | passed |
| `go test -race ./memory/catalog -count=1 -timeout=30s` | passed |
| Gateway Team Console target | passed |
| `go test ./gateway -count=1` | passed |
| `go test -race ./gateway -count=1` | passed |
| `go test ./... -count=1 -timeout=5m` | passed；20 个有测试的包 |
| `go vet ./...` | passed；无输出 |
| `npm --prefix ui run test:transport` | passed；2/2 |
| `npm --prefix ui run build` | passed；48 modules |
| Task Base 与累计 base `git diff --check` | passed |
| tracked/untracked allowlist | passed；11 个产品路径 + `docs/waves/**` |
| `ui/package-lock.json` | SHA-256 `46fd937f66b1b7a16950df8347619831948e9dded477b7d4ba8139018974bdbb`；未修改 |
| repo Playwright artifacts | absent；无 config/report/test-results/HAR/video |

UI 使用 R7 独立 `npm ci` 和任务专用 cache
`/workspace/scratch/afe5d81cd055/.cache/npm-fe-w01-r7`。Go 使用冻结的
Go 1.25.5 工具链与隔离 `GOMODCACHE/GOCACHE`。依赖安装未修改 lockfile。

## 边界与下一 Gate

- 尚未清理或复用下载阶段 Playwright 工件；
- 尚未创建 R7 token、sentinel、synthetic config、database 或 browser
  profile/context；
- 尚未启动 Gateway、Vite 或 Chromium；
- `FE-EVID-W01-020` 只有在独立代码、安全和文档复核全部通过后，才允许
  进入 S11 的下载工件与 inert-provider 预检；
- `FE-W01-S08 / FE-EVID-W01-011` 仍未解决，因此 W01、产品 commit 与发布
  继续阻断。
