# GoClaw Project Console

Obsidian Desktop 插件，提供项目级 GoClaw 聊天、Ouroboros 规格访谈、认知分歧裁决、Seed/演化候选与知识/Harness/开发任务审批、开发 DoneGate 看板、团队只读控制台、Trace 进度和 Harness 状态。

## 构建

```bash
npm ci
npm test
npm run build
```

将 `manifest.json`、`main.js`、`styles.css`、`versions.json` 复制到：

```text
<Vault>/.obsidian/plugins/goclaw-project-console/
```

完整部署见 [../../docs/DEPLOYMENT_CN.md](../../docs/DEPLOYMENT_CN.md)，认知治理见 [../../docs/GOVERNANCE_CLOSED_LOOP_CN.md](../../docs/GOVERNANCE_CLOSED_LOOP_CN.md)，规格闭环见 [../../docs/OUROBOROS_GO_CN.md](../../docs/OUROBOROS_GO_CN.md)。

批准的 Ouroboros Seed 可从“规格”页单向编译为开发任务；插件可完成四类评审、冻结和最终验收。长任务执行默认仅允许主机 CLI，只有显式启用 `development.gateway_allow_execution` 后才开放侧边栏运行按钮。

“团队”页按当前 `project_id` 并行读取以下服务端授权数据。任一模块不可用时只显示该模块错误，不阻断其他模块：

| 模块 | 只读 RPC |
|---|---|
| 成员与负载 | `team.members` |
| 项目任务 | `work.items` |
| Bug 状态 | `issue.list` |
| Runner 与租约 | `runner.list` |
| 生效策略 | `policy.status` |
| 文档概览 | `docs.summary` |
| 共享组件 | `components.summary` |

团队页不提供状态变更按钮。项目身份必须由 Gateway 认证和服务端授权，不能仅信任插件设置中的自由文本。

## 安全

- Gateway Token 使用 Obsidian SecretStorage。
- Reviewer Token 使用独立 SecretStorage 项；Reviewer ID 与服务端角色表匹配。
- 插件不会将这两个 Token 写入 `data.json`。
- 审批需要理由、最强反方论点和证据引用；证据争议裁决不等于任务验收。
- 非本机连接必须使用 `wss://`。
- “Vault 就绪”不等价于 Obsidian Sync 远端无冲突；请以 Sync 面板为准。
