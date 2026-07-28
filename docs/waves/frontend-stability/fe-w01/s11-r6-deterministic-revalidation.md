# FE-EVID-W01-017 — R6 确定性重验

## 证据主体

| 字段 | 值 |
|---|---|
| 时间 | 2026-07-26 |
| Project | `goclaw-team-runtime` |
| Wave / Plan | `FE-W01` / `plan-r008` |
| Task | `FE-W01-TRANSPORT-R1` revision `6` |
| Task Base | `047306b3f4113804c04a39725a0d5ee25bcb87b7` |
| Freeze commit | `90278f4a2b84d01566f3286bb097a2bd85945b05` |
| Branch | `repair/fe-w01-transport-r6` |
| Worktree | `/workspace/scratch/afe5d81cd055/worktrees/fe-w01-transport-r6` |
| Plan SHA | `dd25cb6397aeef4db1442ef79fea4e0a36fd3dcc2a11f5db87919f4904993392` |
| Cumulative W01 base | `697f50e5f428769b75061dfd859d2549dd1c330d` |
| Actor / role | `Codex root agent` / assignee |
| Environment | Linux `6.12.13` x86_64；Go `1.25.5`；Node `24.14.0`；npm `11.9.0` |
| Policy | `wave-governance-v1` |

## 安全迁移

- Task Base 的 legacy channel test 只执行 SHA-256 比对，结果精确等于计划冻结
  值；未输出文件内容、raw deletion diff 或历史版本；
- 以 Delete/Add 写入已审查的 synthetic constructor test，并在任何 Go
  程序或测试前通过 credential/network/wait 两个无输出 source Gate；
- 随后只迁移其余 10 个批准产品文件；
- 11 个产品文件逐一与 R5/R4 已批准 manifest 比较，结果为 `11/11`；
- 未执行旧 channel test，未访问 WeWork 或其他外部 channel。

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

## 动作、预期与实际

预期：R6 的独立迁移在不改变 r008 产品合同、lockfile 或 allowlist 的情况下，
复现 R5 全部确定性绿色结果。

实际：

| Gate | 结果 |
|---|---|
| `gofmt` changed Go files | passed；无输出 |
| channel synthetic credential/network/wait source Gate | passed；无输出 |
| `go test ./channels -count=1 -v -timeout=30s` | passed |
| `go test -race ./channels -count=1 -timeout=30s` | passed |
| Catalog inference target，`count=2` | passed |
| `go test ./memory/catalog` | passed |
| `go test -race ./memory/catalog` | passed |
| Gateway Team Console/session/security target | passed |
| `go test ./gateway` | passed |
| `go test -race ./gateway` | passed |
| `go test ./... -count=1 -timeout=5m` | passed；16 个有测试的包 |
| `go vet ./...` | passed；无输出 |
| `npm run test:transport` | passed；2/2 |
| `npm run build` | passed；48 modules |
| Task Base 与累计 base `git diff --check` | passed |
| tracked/untracked allowlist | passed；11 个产品路径 + `docs/waves/**` |
| `ui/package-lock.json` | SHA-256 `46fd937f66b1b7a16950df8347619831948e9dded477b7d4ba8139018974bdbb`；未修改 |
| repo Playwright artifact absence | passed；无 config/report/test-results/HAR/video |

Go 使用任务工具链和隔离 `GOMODCACHE/GOCACHE`。UI 使用 R6 独立
`node_modules` 与 npm cache。两次以完全清空网络环境执行的 `npm ci` 因无法
解析锁文件下载 host 失败；没有 credential/runtime，lockfile 未改。随后仍
使用同一 R6 cache，继承受控网络代理并强制将 registry host 重写为允许的
`registry.npmjs.org`，`npm ci` 成功；该例外只适用于依赖下载，不放宽
S11/S10 的 `env -i` 和零出站合同。

## 日志与边界

测试输出仅包含测试名、状态、包名、耗时和构建资产公开摘要；没有 Token、
Cookie、synthetic config、Trace、HAR 或 raw channel deletion。npm 的失败
日志位于任务专用 cache，不作为 Evidence 索引，且产生于任何 R6 credential
之前。

本 Evidence 只证明 R6 source-first、manifest、确定性、scope 与 lockfile
Gate。它不证明 inert provider 或真实页面通过，不解除
`FE-W01-S08 / FE-EVID-W01-011`，不授权产品提交或发布。S11 绿色结果必须
进入 `FE-EVID-W01-018`，Browser 结果必须进入 `FE-EVID-W01-015`。
