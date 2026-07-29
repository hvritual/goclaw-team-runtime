# TC-W01 Journal

本文件只在末尾追加事件。当前状态以 Registry 和 current Plan 为准。

## 2026-07-29 — r002 路线激活

- 依赖 `TR-W00 r002` 已由 exact-commit 三路 P0=0/P1=0 review 关闭；
- 用户授权自动继续 `TC-W01 → RN-W01 → INT-W01 → REL-W01`；
- 当前只激活 `TC-W01 r002`，后继三个 Wave 保持 planned；
- 本 Wave 只负责中央 Registry、预算账本、Context Bundle、RPC/CLI 与最小
  Team Control projection；
- Runner 下载/更新、MCP 注入与原生发行分别保留给后继 Wave；
- 产品实现必须等待 activation commit 后的独立 Task Freeze。

| Step | 状态 | 结论 |
|---|---|---|
| `TC-W01-S01` | active | 等待 activation commit 后冻结 exact Task |
| `TC-W01-S02` | planned | 等待 S01 |
| `TC-W01-S03` | planned | 等待 S02 |
| `TC-W01-S04` | planned | 等待 S02–S03 |
| `TC-W01-S05` | planned | 等待 S02–S04 |

## 2026-07-29 — r002 Task Freeze

- activation base：
  `c29dabdee2f0551ad57996611e61e40521e3f7ee`；
- activation tree：`9f08542676dec3f4680107613027392974dd3d50`；
- Task：`TC-W01-CONTROL-003` r002；
- Policy manifest SHA-256：
  `0950b7ee6a2a15a0d8b7093cb4f7dbb513a1691c7c4f38755710a91a25d52b2b`；
- branch：`agent/tc-w01-control-003`；
- freeze 后只允许实现中央 Registry、预算、Context Bundle 和冻结的
  Gateway/CLI/UI projection，不进入 RN/INT/REL 范围。
