# FE-EVID-W01-010 — Deterministic WeWork Constructor Test

- 日期：2026-07-26
- Wave：`FE-W01`
- Step：`FE-W01-S07`
- Issue：`FE-ISSUE-006`、`FE-ISSUE-007`
- Task：`FE-W01-TRANSPORT-R1` revision 3
- Task Base：`2b0ead819b0a2b276b8c9de6779beb03d84767b5`
- Worktree freeze commit：`91d0ac5`
- 结论：S07 确定性代码 Gate 通过；独立代码与安全复核待完成

## 安全执行顺序

1. 在 Revision 3 未运行任何 Go test 前，只以 SHA-256 确认旧 test blob
   与冻结 Evidence 一致；未再次运行旧 test。
2. 将 `channels/weworkwsbot_test.go` 替换为 constructor-only 测试。
3. 先执行无输出 source gates，确认：
   - `BotID`、`SecretID` 只使用 `test-` 前缀的 synthetic values；
   - 不调用 Start、连接、发送或流式发送路径；
   - 不含 sleep、ticker、after、test-owned goroutine、HTTP/WebSocket import
     或 endpoint literal。
4. source gates 通过后才执行普通与 race 测试。
5. 未输出或归档 raw deletion diff；本文件不复制旧 material。

## 通过结果

| Gate | 结果 |
|---|---|
| synthetic credential source gate | passed |
| no-network/no-wait source gate | passed |
| `go test ./channels -count=1 -v -timeout=30s` | passed；package 0.010s |
| `go test -race ./channels -count=1 -timeout=30s` | passed；package 1.031s |
| Gateway TeamConsole targeted | passed |
| 全 Gateway | passed |
| Gateway race | passed |
| R3 独立 `npm ci` | passed；未复用 R1/R2 `node_modules` |
| UI transport tests | passed；2/2 |
| TypeScript + Vite build | passed；48 modules；输出到一次性 `/tmp` 目录 |
| package lock SHA | `46fd937f66b1b7a16950df8347619831948e9dded477b7d4ba8139018974bdbb` |

## 未声明通过的边界

- `go test ./...` 在范围外的 `memory/catalog` 基线失败，见
  [`FE-EVID-W01-012`](s04-memory-catalog-reproduction.md)；因此 S04 和 W01
  尚未通过。
- Cloud Browser 仍拒绝 localhost；没有执行 Browser/视觉 Gate。
- credential owner 尚未提供 `FE-EVID-W01-011`，不能证明历史 material
  已撤销、轮换或从未有效。
- 本证据不宣称页面功能已经全部恢复，也不授权产品提交或发布。
