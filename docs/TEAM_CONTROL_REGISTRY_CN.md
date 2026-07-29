# Team Control Registry、预算与 Context Bundle

适用：`TC-W01 r002`。所有命令由 `goclaw-team-control` 执行，并要求
`GOCLAW_USER_TOKEN`；`goclaw-runner` 不暴露这些中央管理命令。

## 1. 数据权威

| 资源 | Team Control 保存 | 不保存 |
|---|---|---|
| Token Budget | 项目/成员、hard limit、累计 usage、幂等 event | ChatGPT/Codex OAuth、provider token |
| Knowledge Source | URI、revision、SHA-256、批准状态、metadata | Vault 正文、同步凭据 |
| Skill Release | URI、version、SHA-256、兼容版本、批准状态 | 执行时 secret |
| Runner Release | channel、平台、版本、URI、SHA-256、最小协议 | 下载后的 binary、签名私钥 |
| Context Bundle | policy、budget snapshot、批准资源引用、canonical hash | OAuth、device key、知识正文 |

Context Bundle 使用 `goclaw-context/v1` 编译器。相同输入得到相同 ID/hash；
预算使用量、Policy hash、Knowledge revision 或 Skill version 改变时产生新
bundle。Bundle 不会自动下载 URI。

## 2. 预算

创建或更新成员预算：

```bash
goclaw-team-control team budget-put \
  --project project-alpha \
  --id budget-alice-month \
  --user alice \
  --limit 2000000
```

查看中央摘要：

```bash
goclaw-team-control team control-summary --project project-alpha
```

usage event 由受信控制流程以稳定 `event_id` 写入；相同 payload 重试不重复
计费，不同 payload 复用 ID 会冲突，超限整笔失败：

```json
{
  "id": "usage-task-42-r3",
  "project_id": "project-alpha",
  "budget_id": "budget-alice-month",
  "tokens": 18420,
  "task_id": "task-42"
}
```

可用通用入口提交：

```bash
goclaw-team-control team rpc budget.usage.record --params usage.json
```

## 3. Knowledge、Skill 与 Runner release

三个 Registry 均使用通用 JSON RPC CLI。示例：

```json
{
  "id": "knowledge-architecture-v3",
  "project_id": "project-alpha",
  "name": "Architecture",
  "uri": "file:///srv/knowledge/project-alpha/architecture.md",
  "revision": "git:0123456789abcdef",
  "sha256": "64-lowercase-hex",
  "status": "approved"
}
```

```bash
goclaw-team-control team rpc knowledge.source.put --params knowledge.json
goclaw-team-control team rpc knowledge.source.list --params project.json
goclaw-team-control team rpc skill.release.put --params skill.json
goclaw-team-control team rpc runner.release.put --params runner-release.json
```

`draft`、`approved`、`disabled` 是合法状态。只有 `approved` 且 SHA-256
完整的 Knowledge/Skill 能进入 Context Bundle。批准是项目管理权限，不是
文件存在性或内容安全的自动证明。

## 4. 编译 Context Bundle

```bash
goclaw-team-control team context-compile \
  --project project-alpha \
  --repository repo-alpha \
  --user alice \
  --budget budget-alice-month \
  --knowledge knowledge-architecture-v3 \
  --skill skill-go-style-v2
```

跨项目 ID、未批准资源、缺失 checksum、用户与预算不匹配或无管理权限都会
失败关闭。Runner 在 `INT-W01` 才通过 project-scoped MCP 消费这些引用；
当前 Wave 只建立中央权威合同。

## 5. Team Web Console

“团队”页的“中央上下文治理”卡片投影：

- Token budget 已用/总量；
- Knowledge 与 Skill 总数/批准数；
- Runner release 数；
- Context Bundle 数。

浏览器只读取 Gateway 返回的项目投影，不持久化个人 Token、device key 或
Codex OAuth。空 Registry 显示明确 empty state；权限拒绝或网络错误使用
页面统一 error/retry 状态。

## 6. 备份与升级

新增状态仍在 TeamControl 单写 JSON snapshot 内，写入使用临时文件、
`fsync` 和 atomic rename。旧 schema v1 文件缺少新 map 时会初始化为空，
原用户、项目、任务和策略保持不变。升级前仍应按停机冷备流程保存并校验
完整 state；不要在同步盘上运行活动 control-plane state。
