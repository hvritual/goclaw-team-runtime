# MVP-W00 Recovery Task Freeze r003

状态：`frozen`

| 字段 | 冻结值 |
|---|---|
| Task-ID | `MVP-W00-RECOVERY-003` |
| Task-Revision | `r003` |
| Work-Item | `recovery-governance-finalization` |
| Project-ID | `goclaw-team-runtime` |
| Repository-ID | `goclaw-team-runtime-recovered` |
| Repository | local Git repository on branch `recovery/0.8.0-pilot.1` |
| Assignee | `Codex root agent` |
| Base commit | `6fa9607f97715660271ea1356797d4dffaf78f62` |
| Base tree | `f3d30badaeb83c91622b4585d73923f6e787f0fb` |
| Issue | `MVP-ISSUE-001` |
| Wave-ID | `MVP-W00` |
| Wave revision | `r003` |
| Steps | `MVP-W00-S06A`–`MVP-W00-S06D` |
| Policy-Bundle | `MVP-W00-R003-POLICY` |
| Policy manifest | `docs/recovery/POLICY_BUNDLE_SHA256SUMS-r003.txt` |
| Policy manifest SHA-256 | `1d725fc514890338a8d6e7ad338287474674a709b650bacf329ec0ff2f07e0d2` |
| Frozen at | `2026-07-28` |

## 授权范围

- `docs/waves/**`
- `docs/recovery/**`

`product_code_changes_allowed=false`。本 Task 不授权修改已经通过安全/代码复核
的发布实现，也不授权任何 Go、UI、Obsidian 运行时行为。

## Acceptance criteria

1. Registry、current Plan、Track 和总 README 的 revision/status/dependency/
   scope/product flag 唯一一致。
2. 第一轮 BLOCK reviewer 只记作 finding trigger，不记作 revision approver。
3. `MVP-ISSUE-001`、Task、Evidence、commit 和 Wave Step 可追溯。
4. Review 对象使用 exact commit/tree，不使用 `working-tree`。
5. 来源、Go、Web、Obsidian、archive negative、release 双构建全部重验通过。
6. code/security/docs 三路最终复核 P0=0、P1=0。
7. 最终 release manifest commit/tree 等于 tag target，checksum 通过且工作树
   clean。

## Deterministic verification

```bash
sha256sum -c docs/recovery/POLICY_BUNDLE_SHA256SUMS-r003.txt
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
- 回滚只撤回 r003 投影，不重写 import/r001/r002 历史，不覆盖原始归档。
