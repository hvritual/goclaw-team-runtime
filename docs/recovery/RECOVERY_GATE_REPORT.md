# GoClaw 恢复 Gate 报告

Wave：`MVP-W00 r001`  
状态：执行中  
日期：2026-07-28

## Gate 矩阵

| Gate | 命令/对象 | 状态 | 证据摘要 |
|---|---|---|---|
| Archive SHA | `sha256sum -c SHA256SUMS-0.8.0-pilot.1.txt` | `pass` | 四个归档全部 OK |
| Archive compare | `tar --compare` | `pass` | 611 个成员与解压树一致 |
| Git import | import commit/tree | `pass` | 611 个文件已追踪 |
| Toolchain | Go/Node/npm/Git | `pass` | 版本已冻结 |
| Go all packages | `go test ./...` | `pending` | 待执行 |
| Go race | 关键包 `go test -race` | `pending` | 待执行 |
| Go vet | 关键包 `go vet` | `pending` | 待执行 |
| Web tests | `npm test` | `pending` | 待执行 |
| Web build | `npm run build` | `pending` | 待执行 |
| Obsidian tests | `npm test` | `pending` | 待执行 |
| Obsidian build | `npm run build` | `pending` | 待执行 |
| Release build | `scripts/build-release.sh` | `pending` | 待执行 |
| Source scan | release script recoverable scan | `pending` | 待执行 |
| Independent review | code/security/docs | `pending` | 待执行 |

## 通过规则

- `pass` 只能由本次实际命令的零退出码产生；
- 历史发布报告只能用作对比，不能替代本次验证；
- 外部凭据、浏览器、bwrap、WSL2、Lima 和三台物理电脑不属于 Recovery
  的伪造 Gate，它们继续留在 MVP/Pilot 实机阶段；
- 任一失败保持原始输出并把 Wave 置为 `blocked`，不得直接修改产品代码。
