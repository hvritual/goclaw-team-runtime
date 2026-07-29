# MC-W01 Multica Baseline Adoption Task Freeze r001

状态：`frozen`

| 字段 | 冻结值 |
|---|---|
| Task-ID | `MC-W01-BASELINE-001` |
| Task-Revision | `r001` |
| Work-Item | `multica-six-domain-baseline` |
| Project-ID | `multica-baseline` |
| Repository-ID | `goclaw-to-multica-transition` |
| Repository | `hvritual/goclaw-team-runtime` |
| Assignee | `Codex root agent` |
| Target branch | `codex/multica-six-domain-baseline` |
| Backup branch | `codex/backup-goclaw-pre-multica-20260729` |
| Base commit | `dfe8aa4a5d100d5225582c902dfe0f5537790e11` |
| Base tree | `697cf292bdf0a205a272f6e5dd0817389b6c808c` |
| Issue | `MC-ISSUE-001` |
| Wave-ID | `MC-W01` |
| Wave revision | `r001` |
| Steps | `MC-W01-S02`–`MC-W01-S05` |
| Policy-Bundle | `MC-W01-R001-POLICY` |
| Policy manifest | `docs/waves/multica-transition/mc-w01/POLICY_BUNDLE_SHA256SUMS-r001.txt` |
| Policy manifest SHA-256 | `430ae2d96db863bc261a4544f7bfc30fada092cd7677442af1ee4f3bca8b1324` |
| Multica upstream candidate | `beb3e9be65023f63bd5dfdbb0231ed41aa9f1cb8` |
| Frozen at | `2026-07-29` |

## Base validity

该 exact base 已包含并可解析：

- 唯一 active `MC-W01 r001`；
- approved Plan、Registry 和 Policy manifest；
- 用户 G3 确认、`MC-DEC-001`、`MC-ISSUE-001`；
- `TC-W02` superseded 且历史未改写。

## Exact scope

允许：

- 创建并验证本地 backup ref；
- 获取并冻结 Multica upstream exact commit/tree；
- 在目标分支做一次完整 tracked-tree replacement；
- 在 Multica 原生树中添加六域 baseline 文档、测试和已确认缺口修复；
- 记录 deterministic 与 independent Evidence。

禁止：

- 改写/推送 main、origin 或任何远端；
- 删除 backup；
- 访问真实数据、凭据或执行真实 Agent smoke；
- 引入 GoClaw TeamControl、Gateway、Runner、Obsidian 或双写适配层；
- 删除 Multica 其他原生域。

## Acceptance

采用 `plan-r001.md` 的完整退出门禁，特别是：

1. `BACKUP-VERIFIED` 通过后才允许 tree replacement；
2. 初始采用提交 tree 与 frozen upstream tree 完全一致；
3. 六域源码/合同/测试映射和确定性验证完整；
4. 最终 architecture/security/docs P0=0/P1=0；
5. main、origin、远端和仓库外文件保持不变。

## Deterministic verification

```bash
sha256sum -c docs/waves/multica-transition/mc-w01/POLICY_BUNDLE_SHA256SUMS-r001.txt
git rev-parse codex/backup-goclaw-pre-multica-20260729^{tree}
git diff --quiet <frozen-multica-commit>^{tree} <initial-adoption-commit>^{tree}
git diff --check dfe8aa4a5d100d5225582c902dfe0f5537790e11...<candidate-commit>
```

## Rollback

- tree replacement 前：停止并保留当前分支；
- tree replacement 后：切回 backup 分支；
- 不使用 destructive reset 清理用户数据；
- backup 在六域独立验收完成前不得删除。
