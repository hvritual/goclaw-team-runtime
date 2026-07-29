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

## 2026-07-29 — S02–S04 实现与 S05 确定性验证

- 新增 Token Budget/hard limit、幂等 usage ledger 和 overflow Gate；
- 新增 Knowledge Source、Skill Release、Runner Release Registry；
- 新增 `goclaw-context/v1` canonical Context Bundle，输入和 budget snapshot
  进入稳定 hash；
- 新 map 对旧 state 文件向前初始化，file store atomic write 合同不变；
- Gateway RPC、`team budget-put/control-summary/context-compile` CLI、
  Team Web Console central summary 和操作文档已接入；
- project RBAC、跨项目、checksum/status、并发、幂等、旧 state、hash
  determinism 正负例通过；
- 全仓 Go test/vet、关键包 race、UI 9/9 与 build 通过；
- `TC-EVID-W01-001` 进入 collecting；`S02`–`S04` complete，`S05`
  deterministic passed、independent final pending。

## 2026-07-29 — r002 三路独立验收 BLOCK

- reviewed exact commit：
  `6aab01f2508a4c162147da20b399172e01a83d7d`；
- code P0=0/P1=2/P2=3，security P0=0/P1=2/P2=0，
  docs/governance P0=0/P1=3/P2=1；
- 主要 P1 为跨项目裸 ID、Context target user 丢失、URI/metadata/policy
  secret channel、CRUD/UI Evidence 缺口与 Work-Item trailer 偏差；
- r002 implementation 保留为 `TC-EVID-W01-002` failed，不 amend/rebase；
- 激活 r003 做前向修复，RN-W01 继续 planned。

## 2026-07-29 — r003 Task Freeze

- activation commit：
  `6cc3334c68cb6b45b4f43c688e0ac0e674e02a7f`；
- activation tree：`b78c6374101d3f6fa41ff47e39673eb75a9378e2`；
- Task：`TC-W01-ACCEPTANCE-004` r003；
- canonical Work-Item：`tc-w01-acceptance-remediation`；
- Policy manifest SHA-256：
  `c9f6bb4e72166b9faf9511404e3ceca7f31b1aa2e8d73738581395756b7ec6c1`；
- freeze 后只做 r003 已列 P1/P2 修复，不进入 RN/INT/REL。

## 2026-07-29 — r003 S07 确定性 Gate 通过

- 复合 key、Context target identity、secret-safe schema、Registry CRUD、
  usage no-op、JS-safe budget 和 UI 五状态均已实现；
- r002 裸 key 与旧 Context budget user 可前向迁移；legacy unsafe
  Registry/Policy/Context 在读取或编译前失败关闭；
- 全仓 Go test/vet、TeamControl/Gateway race、UI `10/10` 与 production
  build 全部通过；
- `TC-EVID-W01-003` 进入 collecting；下一步推送 exact implementation SHA，
  由独立 code/security/docs reviewer 验收；
- 在三路均 P0=0/P1=0 前，`TC-W01` 保持 active，`RN-W01` 保持 planned。

## 2026-07-29 — r003 exact review 1 BLOCK

- reviewed exact remote commit `86f6823544c72c4c8f070fe851ac3d4cd4e5f1b3`，
  tree `b2d1675c1c98927e311d30e7255f90cc40ce3e0a`；
- code P0=0/P1=4/P2=2，security P0=0/P1=2/P2=3，
  docs P0=0/P1=2/P2=1；
- `TC-EVID-W01-004` 保存失败结论；不重写该 commit；
- P1 聚焦 Registry 状态机、legacy usage/预算、UNC URI、metadata collision、
  migration/CRUD Evidence 和完整 JSON 合同；
- findings 均在 r003 frozen scope 内，继续前向修复；新 exact SHA 必须重新
  运行 Gate 与三路独立 review。

## 2026-07-29 — r003 review remediation 2 Gate 通过

- Registry transition、legacy usage/预算、UNC URI、metadata collision、
  六类 migration、三类 CRUD/RBAC/persistence 和 JSON 合同 P1 已修复；
- 同时补齐 Context no-op、migration 持久化、Policy/Context hash 和 state
  文件权限防御；
- 全仓 Go test/vet、关键 race、UI `10/10` 与 build 再次通过；
- `TC-EVID-W01-005` collecting，等待新的远端 exact SHA 和三路重新验收。

## 2026-07-29 — r003 exact review 2 BLOCK

- reviewed exact `961a93291e6867adc977b930e6c483b49a1f7861`，
  tree `c560545338cdde96dd7939e15d858273ba93f4f3`；
- code P0=0/P1=0 PASS；security P0=0/P1=1 BLOCK；overall 已 BLOCK，
  因此未启动 docs final；
- parsed `file:` URI 的 UNC/device 编码变体仍可绕过 raw boundary；
- `TC-EVID-W01-006` 保留失败结果；继续最小前向修复并对新 SHA 重跑三路。

## 2026-07-29 — r003 review remediation 3 Gate 通过

- `file:` parse/unescape 后的 UNC/device 变体已拒绝并加入回归；
- 父目录 real-directory 与 Unix non-owner-write boundary 已加入；
- 全仓 Go test/vet、关键 race、UI `10/10` 与 build 再次通过；
- `TC-EVID-W01-007` collecting，等待第三个 exact SHA 完整三路验收。

## 2026-07-29 — r003 exact review 3 BLOCK

- reviewed exact `2bce7317f810fbe6ebf3a1874dccab02fb4f670a`，
  tree `4f9daf57198b5e4cd71386c7c25fce88fd515a1c`；
- code P0=0/P1=0/P2=3 PASS；security P0=0/P1=1/P2=2 BLOCK；
  overall 已 BLOCK，因此未启动 docs final；
- `C:\NUL`、`file:///C:/NUL`、`/dev/zero` 等设备或伪文件系统路径仍可
  进入本地 Registry；
- `TC-EVID-W01-008` 保留失败结果；继续最小前向修复并对新 SHA 重跑三路。

## 2026-07-29 — r003 review remediation 4 Gate 通过

- Windows DOS/NT device 与 Unix `/dev`、`/proc`、`/sys` 路径已拒绝并
  加入回归；
- URI/metadata/Policy 校验错误不再回显不受信任字段内容；
- Policy canonical marshal 使用 trim 后的字段值；
- 全仓 Go test/vet、关键 race、UI `10/10` 与 build 再次通过；
- `TC-EVID-W01-009` collecting，等待新的 exact SHA 完整三路验收。

## 2026-07-29 — r003 exact review 4 BLOCK，激活 r004

- reviewed exact `e879b0e2b7194d256d29955d554c464fd12d72bf`，
  tree `7caff593f58b0d6bfeb6358e88eecaac45ccc521`；
- code P0=0/P1=1/P2=2，security P0=0/P1=2/P2=2，
  docs P0=0/P1=1/P2=1，三路均 BLOCK；
- lexical `..`、percent-encoded traversal 和 Windows superscript DOS
  device 是代码/安全 P1；
- docs review 发现 r002 修改 `teamcontrol/service.go`，但 r002 Plan/Task
  scope 未授权，RBAC provenance 是治理 P1；
- `TC-EVID-W01-010` 保留失败结果；`TC-EVID-W01-005/007/009` 标记为
  superseded，不修改其历史正文；
- 激活 `TC-W01 r004`，先推送 activation，再以远端 exact SHA 冻结 Task；
  RN-W01 继续 planned。
