---
schema: goclaw.wave/v1
wave_id: TC-W01
track_id: TEAM-RUNTIME-2026-07
title: Team Control acceptance and multi-project isolation remediation
revision: 3
plan_status: approved
wave_state: active
approved_by:
  - user-directive-2026-07-29-independent-agents-and-auto-continue
owner: Codex root agent
reviewers:
  - independent_code_reviewer
  - independent_security_reviewer
  - independent_documentation_reviewer
depends_on:
  - TR-W00
supersedes:
  - docs/waves/team-runtime/tc-w01/plan-r002.md
created_at: 2026-07-29
updated_at: 2026-07-29
steps:
  - TC-W01-S06
  - TC-W01-S07
  - TC-W01-S08
allowed_change_scope:
  - docs/waves/**
  - docs/**
  - teamcontrol/types.go
  - teamcontrol/inputs.go
  - teamcontrol/store.go
  - teamcontrol/validation.go
  - teamcontrol/policy.go
  - teamcontrol/policy_test.go
  - teamcontrol/queries.go
  - teamcontrol/controlplane.go
  - teamcontrol/controlplane_test.go
  - gateway/team_control.go
  - gateway/team_runtime_test.go
  - cli/team.go
  - ui/src/team/TeamPage.tsx
  - ui/src/team/types.ts
  - ui/src/team/control-summary-state.ts
  - ui/tests/control-summary.test.mjs
product_code_changes_allowed: true
---

# TC-W01 r003 — 多项目隔离与验收前向修复

r002 implementation exact `6aab01f...` 的三路 review 均 BLOCK。r003 不改写
该 commit，而是修复已复现的隔离、secret、CRUD、UI Evidence 和 traceability
问题。

## 修复合同

### 项目身份

- Budget、Usage Event、Knowledge、Skill、Runner Release、Context Bundle
  的存储 key 使用 `(project_id, resource_id)` 复合域；
- 相同 external ID 可在两个项目独立创建、读取和删除；
- update/get/delete 只查调用者已授权项目的复合 key，不通过错误差异暴露
  其他项目存在性；
- normalize 将 r002 candidate 的裸 key 前向重建为复合 key；对象 ID 不变。

### Context identity

- Context canonical material 显式保存 `target_user_id`；
- project budget 的 owner 为空不覆盖 target user；
- 不同 target user 得到不同 hash/ID，相同完整输入保持 byte/hash 稳定。

### Secret-safe schema

- Registry URI 只允许批准的 local/HTTPS scheme，拒绝 userinfo、query、
  fragment、opaque URI、明文 HTTP 和未知 scheme；
- Registry/usage metadata 使用明确 key/type schema，不保存自由字段；
- Policy Rules 使用明确 key/type schema；legacy unknown/unsafe rule 在
  list/resolve/context compile 前失败关闭；
- Gateway presenter 不返回 Registry metadata；
- Context 只纳入通过上述 schema 的 URI、Policy 和 hash，不支持 inline
  secret；需要凭据的外部资源只保存非秘密 `secret_ref` 标识，实际解析留在
  Runner 本地安全存储且不进入 Bundle。

### CRUD 与 UI Evidence

- Knowledge/Skill/Runner release 实现 get/delete；approved 资源必须先
  disabled 才能删除；
- Gateway 注册 get/delete 并补 RBAC、跨项目、同 ID、持久化测试；
- Team 页对 control summary 单独显示 loading、empty、denied、error、
  ready；状态判定提取为可执行纯函数并由 Vite SSR test 运行；
- 项目预算总量限制在 JavaScript safe integer；usage replay 对 state
  revision/file bytes 严格 no-op；
- 文档给出 Knowledge/Skill/Runner release/Context JSON 完整样例。

## Steps

| Step | 内容 | 状态 |
|---|---|---|
| `TC-W01-S06` | 登记三路 findings、激活 r003、冻结 exact remediation Task | `active` |
| `TC-W01-S07` | 修复复合 key、Context identity、secret schema、CRUD/UI | `planned` |
| `TC-W01-S08` | 全量 Gate 与三路 exact-commit final review | `planned` |

## Acceptance

- [ ] r003 Task base 已包含本 Plan、Registry 和 Policy manifest；
- [ ] 两项目相同 ID 的 Budget/Usage/三个 Registry 均独立，无 existence oracle；
- [ ] r002 candidate 裸 key state 可前向加载为复合 key；
- [ ] project budget + 两 target users 产生不同 Context hash/ID；
- [ ] URI userinfo/query/fragment/http/unknown scheme、未批准 metadata/policy
      key/type 全部失败关闭且不进入 state/RPC/Context；
- [ ] Knowledge/Skill/Runner get/delete 与 approved-delete guard 通过；
- [ ] identical usage replay 的 state revision、UpdatedAt 和 file bytes 不变；
- [ ] Gateway/CLI presenter 不返回 metadata/secret；
- [ ] UI 五状态由可执行测试覆盖，total 不超过 JS safe integer；
- [ ] 全仓 Go test/vet、关键 race、UI test/build 通过；
- [ ] 后续所有 commit 使用 frozen canonical Work-Item，历史偏差只读保留；
- [ ] final exact commit 的 code/security/docs 均 P0=0/P1=0。

## 回滚

- 复合 key migration 冲突：加载失败，不覆盖 state；
- legacy Policy/URI 不符合安全 schema：读取/compile 失败关闭，由管理员先
  创建安全 revision，不静默脱敏后继续；
- approved Registry 删除：拒绝，先显式 disabled；
- 任一 P1 未关闭：TC-W01 保持 active，RN-W01 不激活。
