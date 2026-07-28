# FE-EVID-W01-019 — R6 traceability preflight

## 证据主体

| 字段 | 值 |
|---|---|
| 时间 | 2026-07-26 |
| Project-ID | `goclaw-team-runtime` |
| Wave / Plan | `FE-W01` / `plan-r008` |
| Task | `FE-W01-TRANSPORT-R1` revision `6` |
| Task Base | `047306b3f4113804c04a39725a0d5ee25bcb87b7` |
| Freeze commit | `90278f4a2b84d01566f3286bb097a2bd85945b05` |
| Evidence commit | `d7617219f0419f3889ad0ac725435ce4e10642df` |
| Branch | `repair/fe-w01-transport-r6` |
| Actor / role | `wave_docs_validate` 发现；`Codex root agent` 记录 |
| Review context | R6 independent governance review；不向 rev 6 追溯新增 Step |
| Environment | local Git worktree；Linux `6.12.13` x86_64 |
| 发现阶段 | R6 确定性 Gate 后、S11 下载工件预检和任何 R6 credential/runtime 前 |

## 动作、预期与实际

### 动作

只读检查 r008 Task freeze tuple、AGENTS Traceability 条款，以及 activation、
freeze、Evidence 三个 commit 的 message/trailers。检查只读取 commit metadata
和文档字段，不读取或输出 legacy channel raw diff/history。

### 预期

- 冻结 Task 同时包含 project、repository、assignee、base commit、
  policy bundle hash、acceptance criteria 和 deterministic verification；
- 每个 commit 至少包含 `Task-ID`、`Project-ID`、`Task-Revision`、
  `Work-Item`，并在本 Task 中包含 Wave、Issue、Repository 与
  `Policy-Bundle` trailers。

### 实际

- r008 Plan/Journal freeze 有 Project、Assignee、Base、policy bundle label、
  acceptance 与 verification，但没有 Repository-ID，也没有不可变 policy
  bundle hash；
- `047306b`、`90278f4`、`d761721` 只有 subject，没有任何 mandatory
  trailers；
- 11 个产品文件仍未 staged/commit；`FE-EVID-W01-017` 的确定性结果本身
  已由代码 Reviewer 复核通过，但不能替代缺失的 Task/commit traceability。

因此 R6 不可进入 S11，也不能作为后续产品提交或发布的合规执行链。

## 日志与安全停止

- 未创建 R6 Gateway/Team Token、synthetic config、TeamControl database、
  browser profile/context、Trace/HAR/video/screenshot；
- 未启动 Gateway、Vite 或 Chromium；
- 没有产品 commit；
- 本证据不保存 commit raw diff，只记录公开 commit SHA、缺失字段类别与
  文档引用。

## 恢复输入

候选 r009 应：

1. 保留 R6 branch/worktree 只读，不 amend/rebase/reset/delete；
2. 以最后一个带 Task trailers 的 R5 Evidence commit
   `5160273fb17502cf02cd10e1a17f5a47b7eb30be` 为新 revision 起点；
3. 冻结 Repository-ID `repo-goclaw-source-review`；
4. 冻结 policy bundle `wave-governance-v1` 为当前 `AGENTS.md` 内容
   SHA-256 `98bacd6013032cbaffd15095012ed6fc7cd274b62a78d3fdd738aeeadff94ebf`；
5. 从 activation 起所有 commit 使用完整 trailers；
6. 在新 R7 中重新执行 source-first 迁移和全部确定性 Gate，不复用 R6
   测试结果充当绿色验收。

Registry 在 r009 独立批准和原子激活前仍指向 r008。

`5160273...` 只作为可重复的历史内容/ancestry anchor；该选择不追认 R5
或更早提交已满足 r009 新冻结的 Repository 与 policy hash 合同。
