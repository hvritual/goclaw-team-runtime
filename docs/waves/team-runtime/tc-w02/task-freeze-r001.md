# TC-W02 Manual Documentation Task Freeze r001

状态：`frozen (repository-manual, not Team runtime enqueued)`

| 字段 | 冻结值 |
|---|---|
| Task-ID | `TC-W02-REPLAN-001` |
| Task-Revision | `r001` |
| Work-Item | `team-control-authority-replan` |
| Project-ID | `goclaw-team-runtime` |
| Repository-ID | `goclaw-team-runtime-recovered` |
| Repository | `hvritual/goclaw-team-runtime` |
| Assignee | `Codex root agent` |
| Branch | `codex/tc-w02-replan-r001` |
| Base commit | `d6a166ceb1f445e7098855841d20bf3903f0d3d5` |
| Base tree | `02698c5cc262b247ac37c53e80fa807ab5cb2c15` |
| Issue | `TC-ISSUE-002` |
| Wave-ID | `TC-W02` |
| Wave revision | `r001` |
| Steps | `TC-W02-S01`–`TC-W02-S04` |
| Policy-Bundle | `TC-W02-R001-POLICY` |
| Policy manifest | `docs/waves/team-runtime/tc-w02/POLICY_BUNDLE_SHA256SUMS-r001.txt` |
| Policy manifest SHA-256 | `59b3782f2ea0d6109f2ecad329b8df5a8a25836edd21027005ab08ee0f3539b0` |
| Authorization source | `user-directive-2026-07-29-team-control-authority-replan` |
| Frozen at | `2026-07-29` |

## Manual binding limitation

本任务在 repository path 手工遵循 Wave 政策，没有调用运行中的 Team
Control freeze/enqueue，也不声称由 runtime 在 activation commit 自动解析。
它只授权用户明确列出的规划文档工作。产品实现必须等待后继 Wave 的
runtime/manual exact activation 与新 Task Freeze。

## Exact scope

```text
docs/waves/**
```

产品代码、运行时配置、真实知识数据和真实服务访问全部禁止。

## Acceptance

1. Registry 可解析且恰好一个 active TC-W02；
2. RN-W01/INT-W01 superseded 历史和原因可追溯；
3. 当前责任矩阵、目标合同、迁移与 Wave 路线完整；
4. Policy 冲突矩阵和 Knowledge/Context/MCP/Evidence 状态机完整；
5. 本次 diff 只在 `docs/waves/**`；
6. deterministic Gate 先通过；
7. architecture/security/docs 独立审查 P0=0/P1=0；
8. 不修改产品代码，不迁移或访问真实数据。

## Deterministic verification

```bash
sha256sum -c docs/waves/team-runtime/tc-w02/POLICY_BUNDLE_SHA256SUMS-r001.txt
node docs/waves/validate-wave-docs.mjs
git diff --check
git diff --name-only -- . ':!docs/waves/**'
```

## Stop and rollback

- 任一 secret、自动晋升、跨项目 scope 扩大、active 直接写或数据删除设计：
  立即停止；
- 任一 P0/P1 未关闭：TC-W02 保持 active，不激活实现；
- 恢复方式：撤销未合并的规划提交，恢复 RN-W01 先前 active Registry
  状态；不触碰产品数据或 RN 实现历史。
