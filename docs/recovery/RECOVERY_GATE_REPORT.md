# GoClaw 恢复 Gate 报告

Wave：`MVP-W00 r002`

状态：确定性 Gate 已通过；独立复核 `BLOCK`，进入 `plan-r002`
日期：2026-07-28

## Gate 矩阵

| Gate | 命令/对象 | 状态 | 证据摘要 |
|---|---|---|---|
| Archive SHA | `sha256sum -c SHA256SUMS-0.8.0-pilot.1.txt` | `pass` | 四个归档全部 OK |
| Import tree compare | `verify-source-import.sh` | `pass` | 固定 import tree；611/611、内容/执行位/extra=0 |
| Git import | import commit/tree | `pass` | 611 个文件已追踪 |
| Toolchain | Go/Node/npm/Git | `pass` | 版本已冻结 |
| Go all packages | `go test -count=1 ./...` | `pass` | 所有包通过 |
| Go race | 关键包 `go test -race` | `pass` | 6 个关键包通过 |
| Go vet | 关键包 `go vet` | `pass` | 零输出、零退出码 |
| Web tests | `npm test` | `pass` | 8/8 |
| Web build | `npm run build` | `pass` | TypeScript + Vite 通过 |
| Obsidian tests | `npm test` | `pass` | 6/6 |
| Obsidian build | `npm run build` | `pass` | TypeScript + esbuild 通过 |
| Release build | `scripts/build-release.sh` | `pass` | Linux 双架构和控制端交叉编译通过 |
| Source scan | release script recoverable scan | `pass` | 路径、类型、二进制和凭据特征扫描通过 |
| Independent review | code/security/docs | `blocked` | P0=0；去重后 8 个 P1 待关闭 |

## 通过规则

- `pass` 只能由本次实际命令的零退出码产生；
- 历史发布报告只能用作对比，不能替代本次验证；
- 外部凭据、浏览器、bwrap、WSL2、Lima 和三台物理电脑不属于 Recovery
  的伪造 Gate，它们继续留在 MVP/Pilot 实机阶段；
- 任一失败保持原始输出并把 Wave 置为 `blocked`，不得直接修改产品代码。

## 本次执行命令

```bash
scripts/recovery/verify-source-import.sh \
  /immutable/input/goclaw-team-runtime-source-0.8.0-pilot.1.tar.gz

go test -count=1 ./...
go test -race -count=1 \
  ./session ./orchestratorlite ./teamcontrol ./workstation ./gateway ./cli
go vet \
  ./session ./orchestratorlite ./teamcontrol ./workstation ./gateway ./cli

(cd ui && npm ci && npm test && npm run build)
(cd plugins/obsidian-goclaw && npm ci && npm test && npm run build)

INCLUDE_OBSIDIAN_PLUGIN=1 ./scripts/build-release.sh
(cd dist && sha256sum -c SHA256SUMS-0.8.0-pilot.1.txt)
git diff --exit-code -- gateway/ui_dist plugins/obsidian-goclaw/main.js
```

Go 命令使用冻结的 `go1.25.5` 和 `/tmp` 下隔离的 GOPATH、module/cache；
npm 使用 UI、插件和 release 三个独立的 `/tmp` cache。首轮 npm 因默认
`/root/.npm` 不可写而失败，修正缓存位置后完整重放通过；没有把首轮环境
失败隐去或描述成代码缺陷。

## 发布重建结果

| 制品 | SHA-256 |
|---|---|
| `SHA256SUMS-0.8.0-pilot.1.txt` | `cb02be08f065274855ee6a0b9935a567b3a70b6aca23ea58be7f304047b26e7e` |
| Linux amd64 runtime | `c6bc5d551e4953288c350cca543c9391928fb39671e3d6c476b02847cca499bd` |
| Linux arm64 runtime | `be1297eddc219e2294ae918fa14a96d556f098b9bab96fa77b9891dbc2d2aff4` |
| 恢复分支 source archive | `619889bc7eaf14b1ab3094648e0c4068d8969321d67c4d25ae0ce5bdc6b30da3` |
| Obsidian adapter | `02eb2f4ac000fefb8c013b67020a26721a4f777c40fb1fabd4e568e48158aae8` |

重建制品内部校验全部通过，但这些 SHA 只是 `MVP-W00-S04` 的非最终构建
快照，禁止发布或覆盖 7 月 27 日同名归档。恢复分支新增了来源和 Wave 文档，
且 r001 构建未固定 gzip/归档时间；`plan-r002` 将以新版本名完成两次位级
一致的原子构建。构建完成后，受追踪的 UI bundle 和 Obsidian `main.js`
无差异。
