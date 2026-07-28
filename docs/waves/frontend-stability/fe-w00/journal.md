# FE-W00 Journal

本文件只追加；当前计划见 [`plan-r003.md`](plan-r003.md)。r002 经独立
复核后被拒绝执行；r001、r002 均保留历史。

## 状态事件

| Seq | 时间 | Actor | From | To | 原因 | Evidence |
|---:|---|---|---|---|---|---|
| 1 | 2026-07-26 | user |  | `active` | 要求前台修复前建立 Wave 文档并记录全部逐步计划 | `FE-ISSUE-001` |
| 2 | 2026-07-26 | Wave governance | `active` | `complete` | r003 门禁与独立复核通过；首批问题迁移到 W01 | `FE-EVID-W00-007`、`FE-DEC-007` |

## Change log

| Change ID | 时间 | 提出人 | 内容 | Material | 影响 | 新 Plan revision | 决策 |
|---|---|---|---|---:|---|---|---|
| `FE-CHG-W00-001` | 2026-07-26 | user | 建立 Wave-first 文档治理 | yes | W00 禁止产品代码；W01–W05 先保持 planned | `plan-r001` | `FE-DEC-001` |
| `FE-CHG-W00-002` | 2026-07-26 | Codex | 新增环境基线与静态契约预备证据 | no | 不改变目标、范围或 Gate；暴露 Git、Go、Gateway 与 Browser 环境阻塞 |  |  |
| `FE-CHG-W00-003` | 2026-07-26 | Codex | 新增一次性运行夹具预检 | no | 不改变计划；明确 embedded same-origin 首选路径和 dev Origin 待验证项 |  |  |
| `FE-CHG-W00-004` | 2026-07-26 | Codex | 建立独立 0.7.0 Git 基线、官方 Go 工具链并完成真实 Vite 协议复现 | yes | W00 可在首批基础 Issue 拆分后迁移；完整页面矩阵由聚合 Issue 跨 Wave 继续跟踪 | `plan-r002` | `FE-DEC-005`、`FE-DEC-006` |
| `FE-CHG-W00-005` | 2026-07-26 | independent reviewers | 拒绝 r002 的 Wave 切换，要求保持原 Step 含义并补安全/范围 Gate | yes | 产品代码继续禁止；新建 W00/W01 r003 | `plan-r003` | `FE-DEC-007` |

## Decision log

| Decision ID | 时间 | 问题 | 备选方案 | 选择 | 最强反方论点 | Evidence | 决策人 |
|---|---|---|---|---|---|---|---|
| `FE-DEC-001` | 2026-07-26 | 是否先直接修前台 | 立即修、先诊断无文档、Wave-first | Wave-first | 文档会延迟首个修复 | 用户要求且当前报告缺少逐项证据 | user |
| `FE-DEC-002` | 2026-07-26 | W00 是否允许顺手修复 | 允许小修、完全禁止 | 完全禁止 | 明显小问题可能被延后 | 避免未复现与范围漂移 | user |
| `FE-DEC-005` | 2026-07-26 | 以哪套源码开始修复 | 覆盖旧脏树、继续无 Git、独立冻结 0.7.0 | 独立冻结 0.7.0 | 新仓库没有上游历史 | 产品源码与发布归档一致，旧树缺少 Team Web Console | Codex |
| `FE-DEC-006` | 2026-07-26 | Browser 被拒时是否等待全部页面再修基础层 | 无限等待、绕过 Browser、协议证据后渐进修复 | 协议证据后渐进修复 | 首批修复不能代表全部页面可用 | `FE-EVID-W00-007`，且 W01 Browser gate 保留 | user directive + Codex |
| `FE-DEC-007` | 2026-07-26 | r002 是否足以激活 W01 | 直接执行、原地改 r002、新建 r003 | 新建 r003 | 继续修订增加文档开销 | 两路独立复核均确认负例与 Gate 不足 | independent reviewers |

## Evidence ledger

| Evidence ID | 时间 | Step/Issue/Task | Artifact/Trace | SHA-256 | 声明 | 结果 | 生成者 | 复核者 |
|---|---|---|---|---|---|---|---|---|
| `FE-EVID-W00-003` | 2026-07-26 | `FE-W00-S02` / `FE-ISSUE-001` | [`contract-inventory-preliminary.md`](contract-inventory-preliminary.md) | `73bd0a0b3576fe872d6f9570742c85f6dbe8a48bc3933d6a252ef22aefc3cee8` | 3 HTTP、16 query、23 command 和 `chat.event` 均找到静态 Gateway 路径 | `collecting`；尚无运行契约证据 | Codex + frontend audit | unassigned |
| `FE-EVID-W00-005` | 2026-07-26 | `FE-W00-S01` / `FE-ISSUE-001` | [`baseline-manifest.md`](baseline-manifest.md) | `3ce02d8bd1c7d8c1dbb3a296964af6952998c484a8d06a69cccf5a52d217dac1` | 冻结 Git、工具链、构建和浏览器环境事实 | `collecting`；S01 blocked | Codex + environment audit | unassigned |
| `FE-EVID-W00-006` | 2026-07-26 | `FE-W00-S01/S03` / `FE-ISSUE-001` | [`runtime-harness-preflight.md`](runtime-harness-preflight.md) | `2530c2a9cfc37986908a169adfbea2218f524422767bb1b297f6894a8dc8eb31` | 确定安全同源入口、夹具要求、现有测试与浏览器边界 | `collecting`；运行态 not-run | Codex + runtime audit | unassigned |
| `FE-EVID-W00-007` | 2026-07-26 | `FE-W00-S01/S03/S06` / `FE-ISSUE-002`–`004` | [`authority-runtime-reproduction.md`](authority-runtime-reproduction.md) | `43b52a4c2855058e25c9b14867444839e779e0de16017d49f5e84c3e05447acd` | 独立 Git、官方 Go、可重建 build、测试编译失败与 Vite HTTP/WS 复现 | `collecting`；等待独立复核 | Codex + parallel runtime audits | unassigned |
| `FE-EVID-W00-007` | 2026-07-26 | correction: `FE-W00-S08/S09/S10` / `FE-ISSUE-002`–`004` | [`authority-runtime-reproduction.md`](authority-runtime-reproduction.md) | `43b52a4c2855058e25c9b14867444839e779e0de16017d49f5e84c3e05447acd` | 更正旧 ledger 的 Step 绑定；根因、构建与协议证据获独立复核 | `passed` | Codex + parallel runtime audits | `wave_transition_review` |
| `FE-PLAN-W00-R003` | 2026-07-26 | `FE-W00-S11` | [`plan-r003.md`](plan-r003.md) | `b765790c2c961a925ab07a8416141fa3910171af943e8aaf27f4d3456c02eac6` | 保留稳定 Step 并冻结渐进迁移门禁 | `passed` | Codex | `wave_transition_review` |

## 进度事件

| 时间 | Step ID | 状态变化 | 实际结果/阻塞 | 下一动作 |
|---|---|---|---|---|
| 2026-07-26 | documentation gate | `planned → active` | Wave registry、模板、Issue/Decision/Evidence 索引已建立；尚未开始运行复现 | 执行 `FE-W00-S01` |
| 2026-07-26 | `FE-W00-S01` | `planned → blocked` | UI 可确定性重建并与两个 dist 副本一致；但源码无 Git metadata、Go 1.25.5 缺失、无隔离 Gateway，Browser 安全策略拒绝本地预览 | 在真实 Git 仓库冻结 commit，准备 Go/Gateway/获准浏览器环境 |
| 2026-07-26 | `FE-W00-S02` | `planned（预备证据）` | 43 个前端操作均找到 Gateway 路径；权限、项目、返回结构和运行行为仍未验证 | S01 解除后采集真实脱敏请求/响应并完成契约矩阵 |
| 2026-07-26 | `FE-W00-S03` | `planned（预检完成）` | 已确定 embedded same-origin 首选入口与夹具维度；一次性配置、Go CLI 和重置命令尚不可运行 | 解除 S01 工具链与测试 URL 阻塞后建立夹具 |
| 2026-07-26 | `FE-W00-S01` | `blocked → complete` | 0.7.0 独立 Git commit、官方 Go 1.25.5、UI 可重建性和产品 Go build 已冻结 | 独立复核 `FE-EVID-W00-007` |
| 2026-07-26 | `FE-W00-S08` | `planned → complete` | 建立独立 Git、官方 Go 与可重建 build；保留原 S03 含义 | 复核 W01 r003 |
| 2026-07-26 | `FE-W00-S09` | `planned → complete` | 真实 Vite HTTP/WS 探针复现 `/auth` 403、直连 WS 403，并证明同源 WS proxy 101 | 将 `FE-ISSUE-003/004` 绑定 W01 r003 |
| 2026-07-26 | `FE-W00-S10` | `planned → complete` | 拆分 `FE-ISSUE-002`–`004`；聚合 `FE-ISSUE-001` 继续覆盖未验证表面 | 独立复核后切换唯一 active Wave |
| 2026-07-26 | `FE-W00-S11` | `review → changes-requested` | Evidence 根因获批，但 r002 Step 身份、回滚、负例与范围 Gate 不足 | 新建并复核 r003；产品代码保持不变 |
| 2026-07-26 | `FE-W00-S11` | `changes-requested → complete` | W00/W01 r003 通过 Wave 与安全独立复核 | 执行 S12 原子状态切换 |
| 2026-07-26 | `FE-W00-S12` | `planned → complete` | Registry、Issue、Evidence、README、Track 与两个 Journal 同步；W01 成为唯一 active Wave | 在专用 worktree 执行 W01-S01 |
## 2026-07-28 — r004 完成状态投影

- 独立文档复核发现 Registry 的 `complete` 与当前 r003 frontmatter 的
  `active` 不一致；
- 新建 r004，只同步已确认的 complete 终态，不改变历史证据和范围；
- 产品代码继续禁止。

## 2026-07-28 — r005 批准归因更正

- r004 把第一轮 BLOCK reviewer 误写入 `approved_by`；
- 新建 r005，只保留用户授权，reviewer 身份继续用于 Evidence；
- complete 状态、范围、历史证据和产品代码禁令均不改变。

## 2026-07-28 — Recovery append-only 完整性恢复

- import tag 中本 Journal 的前 7574 bytes 已恢复为原 SHA-256
  `bc0d4ef39e3f76ec932451b695b7a2bd1a80f7d3cf00987ee7c3bb8c903f34aa`；
- r004/r005 事件保留在冻结前缀之后；当前权威状态见 Registry 和
  `plan-r005.md`，不再改写本文件顶部历史文字。
