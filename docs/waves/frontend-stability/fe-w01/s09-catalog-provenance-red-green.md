# FE-EVID-W01-013 — Catalog Provenance Red/Green and R4 Gates

- 日期：2026-07-26
- Wave：`FE-W01`
- Step：`FE-W01-S09`、`FE-W01-S04`
- Issue：`FE-ISSUE-008`
- Project：`goclaw-team-runtime`
- Fixture project：synthetic `alpha`
- Actor：synthetic `importer`
- Runtime principal/role：不适用；本次为本地 package test，不经过鉴权路径
- Task：`FE-W01-TRANSPORT-R1` revision 4
- Task Base：`dec9b07bece5e76e20130c9262a273edc41e851f`
- Worktree freeze commit：`72110e0`
- 日志脱敏：只保留命令、断言与摘要；不含凭据、生产数据或 raw deletion diff
- 结论：Revision 4 确定性 Gate 通过；独立代码与安全复核待完成

## 红相

只修改 `memory/catalog/service_test.go`，新增五类推断 case：

1. 空 scheme/kind 的默认 Markdown；
2. 显式 Markdown scheme、空 kind；
3. 显式 kind 保留；
4. `git+markdown` 加 synthetic revision；
5. synthetic `obsidian` 自定义 scheme。

每个 case 同时断言 ProjectID、Source URI 的项目 namespace 和默认 collection。
在未修改 `ingest.go` 时，目标测试精确失败：

- 既有稳定根目录测试：实际 kind 为 `markdown-markdown`；
- 新矩阵的 default Markdown：实际 `markdown-markdown`，期望 `markdown`；
- 新矩阵的 explicit Markdown：实际 `markdown-markdown`，期望 `markdown`；
- 没有出现范围外失败。

## 最小实现

`memory/catalog/ingest.go` 只在 `SourceKind` 为空时按 scheme 分支：

- `markdown` → `markdown`；
- `git+markdown` → `git-markdown`；
- 其他合法 scheme → `<scheme>-markdown`。

显式 kind、Source URI、revision、collection、稳定根目录、schema、既有 records、
审批和项目隔离均未修改。

## 绿相

| Gate | 结果 |
|---|---|
| Catalog 目标矩阵 `count=2` | passed；0.309s |
| 全 `memory/catalog` | passed；0.304s |
| `memory/catalog` race | passed；1.917s |
| channels 普通 | passed；0.010s |
| channels race | passed；1.036s |
| TeamConsole targeted | passed |
| 全 Gateway | passed |
| Gateway race | passed |
| 全仓 `go test ./... -count=1 -timeout=5m` | passed |
| `go vet ./...` | passed |
| R4 独立 `npm ci` | passed；未复用其他 worktree `node_modules` |
| UI transport tests | passed；2/2 |
| TypeScript | passed |
| Vite build | passed；48 modules；输出到一次性 `/tmp` 目录 |
| gofmt/source gates | passed |
| Task Base + cumulative base + untracked scope | passed；精确 11 个产品路径及 `docs/waves/**` |
| package lock SHA | `46fd937f66b1b7a16950df8347619831948e9dded477b7d4ba8139018974bdbb` |

## 仍未关闭的门禁

- 独立代码与安全 Reviewer 尚未签核当前 Revision 4 patch。
- `FE-W01-S08` / `FE-EVID-W01-011` 仍等待 credential owner。
- Cloud Browser 仍拒绝 localhost，未执行真实 Browser/视觉回归。
- 因此外部阻断，W01 不能 complete，产品代码不能提交或发布，也不能宣称全部
  页面交互已经恢复。
