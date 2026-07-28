# MVP-W00 Recovery Task Freeze r004

状态：`frozen`

| 字段 | 冻结值 |
|---|---|
| Task-ID | `MVP-W00-RECOVERY-004` |
| Task-Revision | `r004` |
| Work-Item | `base-resolvable-append-only-recovery` |
| Project-ID | `goclaw-team-runtime` |
| Repository-ID | `goclaw-team-runtime-recovered` |
| Repository | local Git repository on branch `recovery/0.8.0-pilot.1` |
| Assignee | `Codex root agent` |
| Base commit | `91d47a874d3eb5f7823ea8b0e9df1c24656cccec` |
| Base tree | `8b6880ac3e3bf194786ab595c91f6a6cdc890eb9` |
| Issues | `MVP-ISSUE-001`, `MVP-ISSUE-002` |
| Wave-ID | `MVP-W00` |
| Wave revision | `r004` |
| Steps | `MVP-W00-S07B`–`MVP-W00-S07E` |
| Policy-Bundle | `MVP-W00-R004-POLICY` |
| Policy manifest | `docs/recovery/POLICY_BUNDLE_SHA256SUMS-r004.txt` |
| Policy manifest SHA-256 | `4479549a338b7c6859defb738f93ce37e518c6bddb9ee3d9a58598f1b3f7a1f4` |
| Frozen at | `2026-07-28` |

## Base 可解析性

本 Task 的 base commit 必须同时满足：

1. Registry 的 active Wave 为 `MVP-W00`，document 为
   `recovery/mvp-w00/plan-r004.md`；
2. base tree 中存在该 Plan 和 r004 Policy manifest；
3. 从 base tree 读取 manifest 后，三项 SHA-256 全部通过；
4. base commit/tree 与本表完全一致。

冻结前已对 `91d47a8` 执行上述检查。实施提交不得同时承担 activation 或
freeze，不得反向修改本冻结文件。

## 授权范围

- `docs/waves/**`
- `docs/recovery/**`

`product_code_changes_allowed=false`。本 Task 只授权治理投影、历史 Journal
恢复、追加式证据、确定性重验和最终恢复发布记录；不授权修改 Go、Web、
Obsidian 或发布脚本。

## Acceptance criteria

1. activation、freeze、implementation 分属三个顺序提交，Task 可从 exact
   base 解析 active Plan、Registry 和 Policy。
2. FE-W01 Journal 前 26641 bytes SHA-256 精确为
   `33a50e8f3a613b1de95cb76a63ae40e9514a16e9ee9bfa4fbae8c17ffbf44555`。
3. FE-W00、FE-W01、PILOT-W00、MVP-W00 Journal 恢复各自已提交历史前缀，
   Recovery 新事件只追加到 EOF。
4. Registry、Plan、Track、总 README 和 Gate report 的 current revision、
   状态、依赖、scope 与 product flag 唯一一致。
5. `MVP-ISSUE-001`、`MVP-ISSUE-002`、Task、Evidence、commit 和 Wave Step
   可追溯。
6. 来源、Policy、Go、Web、Obsidian、archive negative、release 双构建全部
   重验通过。
7. code/security/docs 三路最终复核 P0=0、P1=0。
8. 最终 release manifest commit/tree 等于 tag target，checksum 通过且工作
   树 clean。

## Deterministic verification

```bash
git show 91d47a874d3eb5f7823ea8b0e9df1c24656cccec:docs/waves/wave-registry.json
git show 91d47a874d3eb5f7823ea8b0e9df1c24656cccec:docs/recovery/POLICY_BUNDLE_SHA256SUMS-r004.txt
sha256sum -c docs/recovery/POLICY_BUNDLE_SHA256SUMS-r004.txt
head -c 26641 docs/waves/frontend-stability/fe-w01/journal.md | sha256sum
scripts/recovery/verify-source-import.sh /immutable/input/source.tar.gz
scripts/recovery/test-release-archive-lib.sh
go test -count=1 ./...
go test -race -count=1 \
  ./session ./orchestratorlite ./teamcontrol ./workstation ./gateway ./cli
go vet \
  ./session ./orchestratorlite ./teamcontrol ./workstation ./gateway ./cli
(cd ui && npm ci && npm test && npm run build)
(cd plugins/obsidian-goclaw && npm ci && npm test && npm run build)
INCLUDE_OBSIDIAN_PLUGIN=1 ./scripts/build-release.sh
INCLUDE_OBSIDIAN_PLUGIN=1 ./scripts/build-release.sh
git diff --check
git status --short
```

## 独立验收与回滚

- final reviewers：`recovery_code_review`、`recovery_security_review`、
  `recovery_docs_review`；
- assignee 不得替代三路独立复核；
- 任一 P0/P1 保持 BLOCK，不创建 recovered tag；
- 回滚只新增 superseding revision，不 amend/rebase r004 或更早历史；
- 不覆盖 2026-07-27 原始归档，不修改 import tag。
