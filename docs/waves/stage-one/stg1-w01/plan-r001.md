---
schema: goclaw.wave/v1
wave_id: STG1-W01
track_id: TEAM-DELIVERY-STAGE1-2026-08
title: MVP Daily Development Kernel
revision: 1
plan_status: approved
wave_state: active
approved_by:
  - user-directive-2026-08-09-complete-stage-one
owner: Codex root agent
issue_id: STG1-ISSUE-001
task_id: STG1-DELIVERY-001
depends_on:
  - TC-W01
created_at: 2026-08-09
updated_at: 2026-08-09
product_code_changes_allowed: true
---

# STG1-W01 r001 — 阶段一：团队日常研发内核

本修订响应用户明确指令“完成阶段一的事项”。为遵守单一 active Wave 和
独立验收门禁，原路线图中的 `KRN-W01 → TC-W01 r002 → REQ-W01 → QLT-W01`
在本 Wave 中冻结为四个顺序 Step；已独立验收的 `TC-W01 r004` 作为管理内核
基线复用，不重复实现。PR #6 的 `TCF-W01` 前端候选作为 UI 输入合入本 Wave，
其桌面/移动渲染和独立 Review 门禁被继承，不将其伪装为已完成。

## 目标

1. 以 append-only Delivery Event Journal 和确定性 Reducer 建立可重放的研发事实流；
2. 保留 TC-W01 已验收的成员、RBAC、预算、Registry、Policy 与 Context Compiler；
3. 打通 Request → IntentContract → SolutionSpec/ADR → 四审 → Freeze → WorkItem；
4. 为 Defect 与 Risk 提供不同状态机、处置决策、证据要求和 WorkItem 关联；
5. 让 Team Control 只通过真实项目级 RPC 读投影、发命令，不建立浏览器事实库；
6. 形成需求和 Bug 各一条可重放、可双向追溯的确定性验收用例。

## 不做

- 不实现 GitHub Webhook、自动 Merge、自动 Release 或 Runner 生产化；
- 不允许 Agent 自行通过四审、Freeze、独立 Review、风险接受或 DoneGate；
- 不将 Event Journal、任务状态或 Token 写入同步 Vault/Web Storage；
- 不改变 GitHub 作为 Commit、PR、CI 原始对象权威的边界；
- 不使用 mock/fallback 伪造成功链路；
- 不关闭独立代码、安全、文档 Review 或真实浏览器回归门禁。

## 冻结 Step

| Step | 路线图映射 | 交付 |
|---|---|---|
| `STG1-W01-S01` | Governance | 复用 TC-W01 accepted base，吸收 TCF-W01，冻结 Schema/命令/状态迁移 |
| `STG1-W01-S02` | KRN-W01 | DeliveryEvent、hash chain、CommandEnvelope、Reducer、Replay/Integrity Check |
| `STG1-W01-S03` | TC-W01 r002 | 保留已验收 Registry/Budget/Context 能力，补充驾驶舱投影和 revision |
| `STG1-W01-S04` | REQ-W01 | Request、IntentContract、SolutionSpec、Review、Freeze、ChangeIntent、WorkItem |
| `STG1-W01-S05` | QLT-W01 | Defect/Risk 独立状态机、RCA/止损/风险处置、复审、任务关联 |
| `STG1-W01-S06` | Gateway/UI | 项目级真实 RPC、明确 denied/conflict/stale/error、Team Control 页面 |
| `STG1-W01-S07` | Verification | Go 单测/race/vet、重放一致性、前端契约/build、浏览器 QA、Evidence |
| `STG1-W01-S08` | Acceptance | 独立 code/security/docs Review；只有 P0=0/P1=0 才可完成 Wave |

## 允许修改

- `teamcontrol/**`
- `gateway/team_control.go`
- `gateway/team_runtime_test.go`
- `ui/src/**`
- `ui/tests/**`
- `docs/waves/**`
- `docs/IMPLEMENTATION_STATUS_CN.md`
- `docs/TEAM_DEVELOPMENT_CN.md`
- `docs/TEAM_WEB_CONSOLE_CN.md`

其他运行时目录、Runner、发布脚本、凭据与部署配置禁止修改。

## 核心合同

### Delivery Event

每个事件必须含 `id/project_id/stream_id/stream_version/sequence/schema_version/
event_type/actor_id/command_id/payload/occurred_at/previous_hash/hash`。事件按项目
形成全局 sequence 和 hash chain；未知 schema/event、sequence 缺口、stream
version 冲突、hash 不一致必须失败关闭。

### Command

所有阶段一领域 mutation 只能进入 `ExecuteDeliveryCommand`。Command 必须带
`id/project_id/type/actor_id/expected_revision/payload`。Reducer 是投影唯一更新
路径；相同 command id 幂等返回原结果，revision/CAS 不匹配返回 conflict。

### Human gates

- IntentContract 未批准不得提交 SolutionSpec；
- Scenario/Capacity/Risk/Cost 四审未全通过不得 Freeze；
- Freeze 后不得原位修改目标和验收标准，只能创建 ChangeIntent 新 revision；
- Risk 接受必须有责任人、理由和未来复审时间；
- Defect 未有复现证据不得进入修复中，未有验证证据不得 resolved；
- WorkItem done 不自动替代 DoneGate 或独立最终验收。

## 验收标准

- [ ] 事件追加、hash chain 校验、从空状态 Replay 结果确定性一致；
- [ ] 未知 schema/event、损坏事件、sequence/revision 冲突全部失败关闭；
- [ ] Request 到 FrozenPlan/WorkItem 的完整链路通过；
- [ ] Defect 和 Risk 使用不同状态机与必填证据；
- [ ] 任意 WorkItem 可反查来源合同、方案/处置、负责人、证据和状态；
- [ ] TC-W01 的 Budget/Registry/Policy/Context 测试无回归；
- [ ] 前端只使用真实 RPC，无 Web Storage Token 或本地任务事实库；
- [ ] Go test/race/vet、前端契约/build、桌面与移动 QA 有索引证据；
- [ ] 独立 code/security/docs Review 为 P0=0/P1=0。

## 风险与回滚

- 事件与现有 snapshot 不一致：新领域先双写同一原子 state 文件并以事件重放校验，
  不迁移旧对象；失败时拒绝启动/命令，不静默修复。
- 跨聚合事务过大：单 Command 只产生同一项目内的一组原子事件。
- UI 超前后端：RPC 未注册或服务端拒绝时显示错误，不提供 fallback。
- 回滚单位：STG1-W01 分支 commit；不改写 TC-W01 accepted candidate 历史。
