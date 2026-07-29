# TC-W02 Contract Review Remediation Task Freeze r002

状态：`frozen`

| 字段 | 冻结值 |
|---|---|
| Task-ID | `TC-W02-REPLAN-001` |
| Task-Revision | `r002` |
| Work-Item | `team-control-authority-replan` |
| Project-ID | `goclaw-team-runtime` |
| Repository-ID | `goclaw-team-runtime-recovered` |
| Repository | `hvritual/goclaw-team-runtime` |
| Assignee | `Codex root agent` |
| Branch | `codex/tc-w02-replan-r001` |
| Base commit | `d2df5fd59fffb2d38838364614974ebce62bbda1` |
| Base tree | `af6d487039df7086daa4224d341bfdbb9422772d` |
| Issue | `TC-ISSUE-002` |
| Wave-ID | `TC-W02` |
| Wave revision | `r002` |
| Steps | `TC-W02-S06`–`TC-W02-S08` |
| Policy-Bundle | `TC-W02-R002-POLICY` |
| Policy manifest | `docs/waves/team-runtime/tc-w02/POLICY_BUNDLE_SHA256SUMS-r002.txt` |
| Policy manifest SHA-256 | `04b76061b2dd61fe995568e1c8646f5eb5d7684cd0f3fcb12c2f7eee4033536d` |
| Prior failed Evidence | `TC-EVID-W02-003` |
| Frozen at | `2026-07-29` |

## Base validity

该 base 已包含并可解析：

- active `TC-W02 r002` Plan；
- Registry 唯一 active `TC-W02`；
- `TC-W02-R002-POLICY` manifest；
- r001 exact review BLOCK Evidence 与 remediation requirements。

这修正了 r001 manual binding 的自授权偏差。r001 governance commit 和 failed
review 只读保留，不称为 frozen implementation/remediation Task。

## Exact scope

```text
docs/waves/**
```

允许：

- 目标合同、当前 API 映射、迁移路线和 proposed Wave drafts 的前向修正；
- base→candidate deterministic validator；
- r002 verification/review Evidence 与 append-only journal/index。

禁止：

- 任何产品代码、TypeScript、运行时配置或真实数据变更；
- 访问运行中的 Team Control、Runner 或真实知识库；
- 自动批准、MCP 实现或真实迁移。

## Acceptance

采用 [`plan-r002.md`](plan-r002.md) 的完整 Additional acceptance，特别是：

1. architecture/security/docs r001 全部 P1 前向关闭；
2. validator 对本 base 到 candidate commit 做非空确定性检查；
3. r002 final 三路 review P0=0/P1=0；
4. 产品代码 diff 为空。

## Deterministic verification

```bash
sha256sum -c docs/waves/team-runtime/tc-w02/POLICY_BUNDLE_SHA256SUMS-r002.txt
node docs/waves/validate-wave-docs.mjs \
  --base d2df5fd59fffb2d38838364614974ebce62bbda1 \
  --candidate <candidate-commit>
git diff --check d2df5fd59fffb2d38838364614974ebce62bbda1...<candidate-commit>
```

## Rollback

- remediation commit 可回到本 base；
- r001/r002 activation 和 failed review history 不改写；
- 任一 P1 未关闭，TC-W02 保持 active，TC-W03 不激活；
- 不触碰产品数据或 RN 实现。
