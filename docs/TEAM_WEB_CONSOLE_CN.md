# Team Web Console

版本：GoClaw Team Runtime `0.7.0`。

状态：前台代码面已经存在，但由于收到“很多功能使用异常”的汇总报告，
运行可用性当前标记为 `implemented_unverified`。下表描述目标职责和已有
入口，不代表每项操作已经通过本轮运行验收。唯一当前激活的稳定性 Wave 是
[`FE-W00`](waves/frontend-stability/fe-w00/plan-r001.md)，该 Wave 只允许
复现、契约盘点和证据固化，尚未授权产品代码修复。

## 定位

Team Web Console 是十人团队的默认控制面，替代 Obsidian 插件承担以下强制职责：

| 工作区 | 权威数据 | 可执行操作 |
|---|---|---|
| 总览 | WorkItem、Issue、Runner、Policy、Trace | 查看项目健康度、阻塞与证据覆盖 |
| 对话 | 项目 SessionEvent | 按 `project_id + topic_id` 发送项目消息 |
| 规格 | Ouroboros Event/Seed | 访谈、回答、重评、结晶、编译、评估与演化 |
| 记忆 | Memory Catalog | 检索 approved 记忆、查看来源、待复核和候选 |
| 审批 | Governance Decision | Seed、知识、记忆、开发任务与 Harness 人工决策 |
| 开发 | Orchestrator Lite | 冻结任务入队、查看 DoneGate、创建修复 revision |
| 团队 | TeamControl/Workstation | 成员负载、任务、Bug、Runner、Policy、Docs、Components |
| 进度 | Task + Trace | 分别展示任务状态、证据覆盖和运行健康度 |
| Harness | Harness Registry | 查看版本与实验，批准、提升或回滚 |

页面不会把看板状态写回 Markdown，也不会另建一套任务真源。

## 异常修复 Wave 门禁

所有前台异常处理遵循 [Wave 更新管理](waves/README.md) 和
[前台稳定性轨道](waves/frontend-stability/index.md)：

1. 先在 `FE-W00` 固化运行环境、角色、项目、操作、预期、实际和脱敏日志；
2. 只有状态达到 `reproduced` 或 `root-caused` 的 Issue 才能进入修复 Wave；
3. 修复任务必须引用 `Wave-ID`、`Issue-ID`、计划修订和具体 Step；
4. 只有注册表中的 active Wave 且计划明确允许时，才能修改范围内产品代码；
5. 范围、契约、门禁、风险或回滚发生实质变化时，先创建新的计划修订；
6. 没有写入证据索引的验证结果，不能推动 Issue 或 Wave 状态；
7. 后续 Wave 的依赖未完成时不得提前激活。

当前门禁由仓库规则、执行者和 Reviewer 共同遵守，尚未由 Gateway 在任务
冻结时自动校验 Wave 修订或计划哈希。该运行时强制能力必须另立治理 Wave，
不能作为本轮已实现功能描述。

## 登录与凭据边界

浏览器登录提交 Gateway Token 和个人 Team Token，Gateway 验证后发放：

- `goclaw_team_session`：随机 256-bit、HttpOnly、SameSite=Strict 的短期会话；
- CSRF Token：只保存在当前 React 内存，用于 `/rpc` 变更请求；
- Reviewer Token：由用户在身份菜单临时输入，仅存在页面内存，刷新即清空。

长期 Gateway Token 只参与登录交换，不复制到 Cookie。交换成功后，
`goclaw_team_session` 同时满足外层 Gateway 认证与 Team principal 认证；
退出、过期或服务端撤销后立即失效。旧原生客户端仍使用 Gateway
Bearer/子协议 Token 加个人 Team Token 的双层认证。

Team Token、Gateway Token 和 Reviewer Token都不写入：

- LocalStorage；
- SessionStorage；
- IndexedDB；
- URL；
- Markdown；
- Catalog；
- Trace。

生产环境必须使用 HTTPS、VPN 或 SSH 隧道。静态 Console Shell 可以匿名读取，但所有项目数据均需浏览器会话、项目 RBAC 和 CSRF；跨站 WebSocket Origin 会被拒绝。

## 访问

默认 WebSocket 端口同时托管 Team Console：

```text
https://goclaw.example/dashboard/
```

反向代理必须把以下路径路由到同一个 GoClaw WebSocket 服务端口：

```text
/dashboard/
/assets/
/auth/session
/rpc
/ws
```

不要只代理静态页面，否则登录 Cookie、RPC 和 WebSocket 会跨 Origin。

## 项目隔离

左侧项目切换器只改变请求上下文，不授予权限。Gateway 根据个人 Team Token 解析 principal，并在服务端验证：

- 项目成员关系；
- 当前 RPC 所需权限；
- 资源实际所属项目；
- 高风险 Governance 角色；
- 创建者、审批者、执行者和验收者职责分离。

未授权项目必须返回拒绝态，不能展示空数据来伪装成功。

## 状态呈现约束

所有工作区都必须显式呈现：

- loading；
- empty；
- denied/error；
- disconnected；
- pending approval；
- blocked；
- stale/conflict。

“总进度”不通过平均多个不相干指标伪造。任务状态、DoneGate 覆盖、Runner 在线和 Trace 健康度分别显示。

## Obsidian 可选适配器

Obsidian 插件继续兼容 `0.6.0` RPC，但从 `0.7.0` 起：

- 不是生产部署前置条件；
- 不再是唯一审批入口；
- 不承担身份、任务、Bug、队列或运行状态存储；
- 默认发布构建不生成插件包；
- 插件移除不会删除 Markdown、Catalog、SessionEvent 或 Governance 历史。

确需使用时：

```bash
INCLUDE_OBSIDIAN_PLUGIN=1 ./scripts/build-release.sh
```

## 待执行验收清单

以下项目是 `FE-W05` 的发布放行条件，不是当前已经通过的测试结果：

1. 未登录访问 `/dashboard/` 只能看到登录页，不能读取任何项目 RPC。
2. 错误 Gateway Token 或个人 Team Token 返回 401。
3. 登录响应 Cookie 为 HttpOnly、SameSite=Strict；HTTPS 下带 Secure。
4. 没有 CSRF Header 的浏览器 `/rpc` 返回 403。
5. 跨站 Origin 的 WebSocket 和登录请求被拒绝。
6. 项目 A 成员不能读取项目 B 的成员、任务、Issue、记忆或 Trace。
7. Reviewer Token 刷新后清空。
8. 所有九个工作区完成 loading、empty、denied 和断线恢复检查。
9. 任务没有 EvidencePackage 或 DoneGate 未通过时不能验收。
10. 移除 Obsidian 插件后，上述操作仍可在 Web Console 完成。
