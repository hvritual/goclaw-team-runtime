# 认知治理与完整闭环

本文描述 v0.7.0 的认知偏差防护、审批身份、职责分离、结果反馈和停止条件。编目记忆治理见 [`LIBRARY_MEMORY_CN.md`](LIBRARY_MEMORY_CN.md)，部署步骤以 [`DEPLOYMENT_CN.md`](DEPLOYMENT_CN.md) 为准。

## 1. 不变量

Go Core 强制以下边界：

1. 人工批准是进入下一阶段的权限，不是正确性证明。
2. 聊天记忆和 Vault 笔记是上下文，不是验收证据。
3. 模型只能提出 Seed、演化和 Harness Candidate；不能直接激活、写知识、验收任务或提交代码。
4. 机械门失败不能被模型多数决或人工“证据裁决”覆盖。
5. 同一人的提案、审批、任务评审、最终验收和 Harness 提升按策略分离。
6. 失败、取消、回滚和无反馈都进入参考类；同一任务的新结果覆盖旧结果的统计口径，但保留完整历史。
7. 令牌、改动范围、修复次数、模型调用数和模型令牌按会话累计；达到停止条件时失败关闭。
8. Gateway 默认不能启动开发任务，也不能执行命令型 Harness Eval。

## 2. 闭环

```text
原始需求
  → 多视角需求评估
  → 分歧/灰区/同模型相关性升级给人
  → 冲突逐条解决
  → Seed：备选方案、未行动成本、证伪条件、预演失败、参考类预测、停止条件
  → 身份认证 + 审批法定人数
  → 四类任务评审 + 冻结 Revision/范围/命令/预算
  → 隔离 worktree 执行
  → Falsifier / Prediction / Kill Check + EvidencePackage
  → Go 机械门
  → 语义门 + 盲化评审 + 关键发现否决
  → 分歧时人工只裁决证据，不验收任务
  → 独立最终验收
  → 记录通过/失败/取消/回滚/无反馈
  → 参考类与最近评估历史进入下一代候选
  → 收敛、继续、停止或回滚
```

每个需要人工判断的动作都记录：

- 已认证的 `reviewer_id`、角色和来源；
- 决策、理由、最强反方论点；
- 证据引用和时间；
- 事件链或状态文件中的结构化决策记录。

## 3. 认知偏差对应控制

| 风险 | Go 侧控制 |
|---|---|
| 单一叙事、过早收敛 | 至少两个备选方案，且只允许一个被选择 |
| 自动化偏见 | 人工权限与正确性门分离；机械门不可覆盖 |
| 伪多样性 | 保存每个评估器的模型身份；多调用但同模型时升级人工 |
| 群体思维 | skeptical/operator/risk/status-quo 视角；保存分数分布 |
| 阈值附近的虚假确定性 | gray zone 与 score spread 升级 |
| 利益相关方被平均掉 | Claim 与 Conflict 独立保存，未解决冲突阻止结晶 |
| 确认偏差 | 每项验收标准必须有 falsifier 和检查结果 |
| 计划谬误 | Seed 包含参考类基准、分位数与失败率 |
| 沉没成本 | 预注册可机器计算的 kill condition |
| 幸存者偏差 | 失败、取消、回滚、无反馈均进入分母 |
| 双重计数 | 同一任务的后续结果以 `supersedes_id` 取代旧统计样本 |
| Goodhart 定律 | Golden/Holdout 与 Protected Paths；Candidate 不得改裁判 |
| 自我批准 | `forbid_self_approval` 和独立审批角色 |
| 权力集中 | 每位任务评审者可承担的评审类型上限、最终验收分离、Harness 批准/提升分离 |
| 无界循环 | 代数、调用数、令牌、修复次数和范围硬上限 |

## 4. Reviewer 身份

Gateway Token 只认证客户端连接；Reviewer Token 认证具体的人和角色，两者不能互相替代。配置只保存 Reviewer Token 的 SHA-256：

```bash
REVIEWER_TOKEN="$(openssl rand -hex 32)"
printf '%s' "$REVIEWER_TOKEN" | sha256sum
```

把 64 位摘要填入 `governance.reviewers.<id>.token_sha256`。团队模式还可为描述性的 Reviewer key 绑定唯一 TeamControl 用户：

```json
{
  "governance": {
    "reviewers": {
      "erin-final": {
        "team_user_id": "erin",
        "token_sha256": "<64-hex-sha256>",
        "roles": ["task_accept", "task_cancel"]
      }
    }
  }
}
```

`team_user_id` 是可选字段，大小写不敏感且不能被两个 Reviewer key 重复绑定。启用 TeamControl 后，Gateway 从个人 `GOCLAW_USER_TOKEN` 得到 principal `erin` 并忽略客户端自报的 Reviewer ID。治理策略先按 principal 精确匹配 Reviewer map key；没有精确 key 时，再按大小写不敏感的 `team_user_id` 找到 `erin-final` 的 Token 摘要和角色。决策记录仍保留真实 TeamControl principal，描述性 key 只负责策略映射。没有绑定时，Reviewer key 必须与 principal ID 一致。

原始 Reviewer Token：

- Obsidian：写入插件设置的 Reviewer Token，保存在 Obsidian SecretStorage；
- 本机 CLI：仅通过当前进程环境变量传入：

```bash
export GOCLAW_REVIEWER_TOKEN='<original-token>'
```

示例配置里的全零摘要是故意设置的失败关闭占位符，必须逐一替换；配置校验会拒绝全零摘要、多个身份共用同一摘要，以及多个 Reviewer key 绑定同一 `team_user_id`。

支持角色：

| 范围 | 角色 |
|---|---|
| 规格 | `seed_approve`, `evolution_approve`, `readiness_override`, `conflict_resolve`, `evaluation_resolve`, `outcome_record`, `kill_switch`, `session_cancel` |
| 开发 | `scenario_review`, `capacity_review`, `risk_review`, `cost_review`, `task_accept`, `task_cancel` |
| Harness/知识 | `harness_approve`, `harness_promote`, `harness_rollback`, `knowledge_approve` |
| 紧急管理 | `*`，仅建议离线 break-glass 身份使用 |

启动时会验证未知角色、法定人数和明显的职责分离死锁。启用认证后，旧的“仅传 actor/reviewer 字符串”API 会失败关闭。

## 5. 决策输入

所有批准命令都应附上理由、最强反方论点和证据引用：

```bash
export GOCLAW_REVIEWER_TOKEN='<token>'

goclaw ouroboros approve-seed <session-id> \
  --reviewer alice-spec \
  --comment "目标、边界、证伪条件与停止条件可执行" \
  --counterargument "参考类样本仍少，成本分位数可能偏乐观" \
  --evidence-ref "adr:ADR-0042" \
  --evidence-ref "seed:<sha256>"
```

拒绝不要求“反对拒绝的反方论点”；如果全局启用了 `require_counterargument`，批准、接受、提升和回滚仍必须提供。

## 6. 需求评估分歧

默认两个需求评估器并行给出清晰度、问题框架、利益相关方主张与冲突。以下任一条件阻止自动 readiness：

- 最大分差超过 `assessment_max_spread`；
- 总歧义度位于阈值的 `assessment_gray_zone`；
- 多个评估器实际只使用一个模型身份；
- 存在未解决的利益相关方冲突；
- 仍有阻塞问题。

先解决冲突，再裁决 readiness：

```bash
goclaw ouroboros resolve-conflict <session-id> \
  --conflict <conflict-id> \
  --resolution "优先保证数据完整性，延迟目标改为 p95 < 500ms" \
  --reviewer alice-spec \
  --comment "产品与运维已共同确认" \
  --counterargument "该选择会牺牲高峰吞吐"

goclaw ouroboros resolve-readiness <session-id> \
  --ready \
  --reviewer alice-spec \
  --comment "阻塞问题和利益冲突均已关闭" \
  --counterargument "同模型评估仍可能遗漏未知约束"
```

人工 readiness 只能处理系统明确升级的灰区或相关性问题，不能跳过阻塞问题或开放冲突。

## 7. 评估争议

评估顺序为机械门、语义门、盲化角色评审、Go 多数决。关键发现、分差过大或评审模型相关时设置 `human_decision_required`，在裁决前：

- 不记录通过或失败结果；
- 不允许生成演化候选；
- 不把争议样本加入参考类分母。

裁决：

```bash
goclaw ouroboros resolve-evaluation <session-id> \
  --evaluation <evaluation-id> \
  --accept \
  --reviewer alice-spec \
  --comment "diff、测试日志和证伪检查支持验收标准" \
  --counterargument "评审仍来自同一模型供应链" \
  --evidence-ref "artifact:diff" \
  --evidence-ref "artifact:test-log"
```

`--accept` 只接受这次证据判读。它不能：

- 覆盖机械门或语义门失败；
- 代替 Orchestrator Lite 的 `task_accept`；
- 触发提交、部署、知识写入或 Harness 提升。

不带 `--accept` 表示驳回争议评估，并记录失败结果。

## 8. 结果、预测与停止

手工记录部署后结果时必须关联本会话的 evaluation、task 或 Seed：

```bash
goclaw ouroboros record-outcome <session-id> \
  --kind rolled_back \
  --task <task-id> \
  --reason "发布后错误率超过预注册阈值" \
  --reviewer alice-spec \
  --comment "监控和回滚记录已核对" \
  --counterargument "异常也可能由无关的上游服务引起" \
  --evidence-ref "monitor:incident-2026-07-24"
```

`kind` 支持 `passed`、`failed`、`cancelled`、`rolled_back`、`no_feedback`。同一任务的新记录通过 `supersedes_id` 替代旧统计结果，但旧记录仍保留在审计历史中。

预注册停止条件可由 Go DoneGate 自动检查，也可由有权身份手工触发：

```bash
goclaw ouroboros trigger-kill <session-id> \
  --condition <condition-id> \
  --reason "changed_files 已超过阈值" \
  --reviewer alice-spec \
  --comment "范围扩张已达到停止条件" \
  --counterargument "继续一次修复可能仍能缩小范围" \
  --evidence-ref "evidence:<run-id>"
```

参考类：

```bash
goclaw ouroboros reference-class --project project-alpha
```

## 9. Harness 与取消操作

Harness 批准者不能同时提升同一个 Candidate。回滚也是认证决策：

```bash
goclaw harness rollback \
  --reviewer grace-harness-operator \
  --comment "Golden 线上代理指标出现回归" \
  --counterargument "回滚会暂时恢复旧的已知缺陷" \
  --evidence-ref "trace:regression-42"
```

Ouroboros 会话和开发任务取消同样需要专门角色、理由和审计记录。命令型 Harness Eval 只能从本机 CLI 使用 `--execute` 启动，Gateway 会拒绝远程执行。

## 10. 单模型订阅的诚实边界

ChatGPT Workspace + Codex app-server 可以提供多个独立上下文，但如果返回的模型身份相同，系统不会把它们伪装成真正独立的评审。需求评估和实现评估会升级给人工。

如果配置多个 `assessment_models` 或 `evaluation_models`，Go Core 只验证返回的模型身份是否不同；这仍不是供应商、训练数据或组织利益上的独立性证明。高风险环境应接入真正独立的评审源或外部审计。

## 11. 仍然存在的边界

- Reviewer Token 是共享秘密 + SHA-256 校验，不是硬件密钥、个人证书或不可抵赖签名。
- 本地事件链可发现篡改，但不是外部 WORM、透明日志或远程时间戳。
- 单 Leader 只有进程内互斥，没有分布式队列、租约和 HA。
- Reference Class 是按项目的结构化频率统计，不自动解决样本选择、环境漂移或因果归因。
- 人工理由和反方论点能提高可审计性，不能消除组织压力、共识偏差或恶意串通。
- Obsidian Sync 的远端冲突状态仍以 Obsidian 自身面板为准。

这些限制应进入部署风险登记，不应通过增加模型调用次数来掩盖。
