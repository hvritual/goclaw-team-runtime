# TC-W02 S04 Deterministic Verification

状态：`superseded`

验证基线：

- base commit:
  `d6a166ceb1f445e7098855841d20bf3903f0d3d5`
- base tree:
  `02698c5cc262b247ac37c53e80fa807ab5cb2c15`
- branch: `codex/tc-w02-replan-r001`
- scope: `docs/waves/**` only

## Gate

| 检查 | 通过条件 | 当前结果 |
|---|---|---|
| Registry JSON | 可解析 | pending final run |
| active Wave | 恰好一个，且是 TC-W02 | pending final run |
| route/status | RN/INT superseded；TC-W03–W06 proposed；REL depends TC-W06 | pending final run |
| plan/document | document 存在，active/proposed/planned frontmatter 与 Registry 相容 | pending final run |
| contract completeness | Policy/Knowledge/Context/MCP/Evidence、冲突矩阵、迁移路线齐全 | pending final run |
| links | changed Markdown 本地链接存在 | pending final run |
| append-only | journal 无删除行 | pending final run |
| scope | diff 全部在 `docs/waves/**` | pending final run |
| formatting | `git diff --check` 通过 | pending final run |
| product code | 本次产品代码 diff 为空 | pending final run |

r001 validator 在 commit 后可能得到 0-file vacuous PASS，未满足 frozen
base→candidate 证明要求，因此本记录 superseded，不能作为 passed
Evidence。修正后的 r002 结果见
[`s07-r002-deterministic-verification.md`](s07-r002-deterministic-verification.md)。
