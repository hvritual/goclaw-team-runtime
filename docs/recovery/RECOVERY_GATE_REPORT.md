# GoClaw 恢复 Gate 报告

Wave：`MVP-W00 r002`

状态：r002 确定性 Gate 已通过；第二轮独立复核待执行
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
| Archive negative tests | `test-release-archive-lib.sh` | `pass` | duplicate/extra/link/traversal 全部 fail closed |
| Release build | `scripts/build-release.sh` × 2 | `pass` | 首次原子发布；第二次整目录 identical |
| Source scan | release script recoverable scan | `pass` | 路径、类型、二进制、凭据和精确成员合同通过 |
| Independent review | code/security/docs | `collecting` | 第一轮 P0=0/P1=8；r002 第二轮待执行 |

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

scripts/recovery/test-release-archive-lib.sh

RELEASE_VERSION=0.8.0-pilot.1-recovered.1 \
  INCLUDE_OBSIDIAN_PLUGIN=1 ./scripts/build-release.sh
RELEASE_VERSION=0.8.0-pilot.1-recovered.1 \
  INCLUDE_OBSIDIAN_PLUGIN=1 ./scripts/build-release.sh

(cd dist/releases/0.8.0-pilot.1-recovered.1 &&
  sha256sum -c SHA256SUMS-0.8.0-pilot.1-recovered.1.txt)
git diff --exit-code -- gateway/ui_dist plugins/obsidian-goclaw/main.js
```

Go 命令使用冻结的 `go1.25.5` 和 `/tmp` 下隔离的 GOPATH、module/cache；
npm 使用 UI、插件和 release 三个独立的 `/tmp` cache。首轮 npm 因默认
`/root/.npm` 不可写而失败，修正缓存位置后完整重放通过；没有把首轮环境
失败隐去或描述成代码缺陷。

## S04 历史构建快照

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

## S05D 第二轮复核候选

候选 commit：`792b599c56852e26623bca83313f56b3a0693f2b`。以下值由同一 commit
连续两次洁净执行产生；第二次输出明确为 `Verified identical existing
release`。复核结论写回后会再从最终文档 commit 构建，因此最终发布必须以
版本目录内的 manifest/checksum 为准，不能把本表当成最终 locator。

| 制品 | SHA-256 |
|---|---|
| Linux amd64 runtime | `be234e3f424d1c548851dcfc98a97ca4aab4ade26a4c0a054aad82906e8f5132` |
| Linux arm64 runtime | `2bd8094d78f783b3ae6b91e20a3ae88aeebe6d45be82d863705904b7381c5e97` |
| recovered source | `3157e998e8fc7b153c458dc28917d01e637831e808c77d079b5e8a77f2858e55` |
| Obsidian `0.6.0` | `b4eb4051b161eba11ff95146a346e8b7440a8e71c3c9ecd53e7d86e6784568b6` |
| release manifest | `f2c4cfe786a0c562ed4d0e983002e10e653bcb1f77ecf3a4c2fbbf0e34253305` |

版本目录为 `dist/releases/0.8.0-pilot.1-recovered.1/`；原始输入仍位于独立
只读目录，未被覆盖。recovered source 已包含 `.tool-versions`，release
manifest 将 runtime `0.8.0-pilot.1-recovered.1` 映射到 Obsidian `0.6.0`。

## S06C r003 可追溯重验

r003 frozen Task：`MVP-W00-RECOVERY-003 r003`；base
`6fa9607f97715660271ea1356797d4dffaf78f62`；Policy-Bundle
`1d725fc514890338a8d6e7ad338287474674a709b650bacf329ec0ff2f07e0d2`。

在 projection 修复 commit `3c209a411333d31bdac44896a40d256bde33e3b0`
上重新执行来源、Policy、Go 全包/race/vet、Web、Obsidian、archive negative
和 release 双构建，全部通过。第二次 release 输出为 `Verified identical
existing release`；manifest 绑定：

- commit：`3c209a411333d31bdac44896a40d256bde33e3b0`
- tree：`a8ff1a18f430a5c7e15709e6f8a3066a25890950`
- manifest SHA-256：
  `16ea5c12c1bdc9334f3eef8ee444148e50cd4aabade11a1deb60c8adcfe81965`

这是 final review 前的 r003 候选，不是最终 tag 制品。独立复核写回会产生
新的 docs commit；S06D 必须从该最终 commit 再构建两次并核对 manifest，
然后才能创建 tag。
