# 实现状态矩阵

## 1. 产品功能矩阵

| 功能域 | Web | Desktop | `backend/**` | 遗留 `server/**` | 当前判断 |
| --- | --- | --- | --- | --- | --- |
| 登录/账号/PAT | 有入口 | 有登录与深链接 | Auth 登录未迁移 | 有兼容 API | 部分实现：产品可见，canonical 未接管 |
| 工作区 | 创建、切换、设置 | 同步共享，并有标签组 | identity 及部分服务 opt-in | 完整兼容链 | 部分实现 |
| 成员/邀请/角色 | 有完整设置与邀请页 | 共享 | Member opt-in；邀请未迁移 | 完整兼容链 | 部分实现 |
| 项目 | 列表、详情、资源、需求基线 | 同步共享 | Project/Relationship opt-in | 完整兼容链 | 部分实现 |
| Issue | 多视图、评论、附件、属性等 | 同步共享 | 主线 Create/Get/List/Update/Status opt-in | 完整兼容链 | 部分实现，前端成熟度高于 canonical backend |
| 普通任务 | CRUD + 四状态 | 同步共享 | Todo CRUD opt-in | 有兼容链 | 部分实现 |
| Skill 资料 | 列表、详情、文件管理 | 同步共享 | Workspace binding opt-in；System Skill 占位 | 有兼容与执行注入 | 部分实现 |
| 知识治理 | 发布区、提议、审核 | 同步共享 | Knowledge/Requirement 部分 opt-in；控制面有知识流 | 有兼容链 | 部分实现 |
| Team Control | 项目控制页 | 同步共享 | SQLite + HTTP/SSE + 事件内核 | 非主要依赖 | 已实现的端到端切片 |
| 智能体 | 无当前路由 | 无当前路由 | Auth Agent 占位 | 有 | 仅遗留/文档漂移 |
| Runtime/Daemon | 无当前路由 | 无已确认管理页 | Execution 模块未落地 | 有 | 仅遗留 |
| 自动化 | 无当前路由 | 无当前路由 | 无 | 有 | 仅遗留 |
| Chat | 无当前路由 | 无当前路由 | 无 | 有 | 仅遗留 |
| Inbox/通知 | 仅邀请入口，不是通知中心 | 同左 | 无 | 有 | 仅遗留 |
| Asset 服务 | 前端有附件兼容链 | 同步共享 | Space Asset 占位 | 有 | 仅遗留/占位 |
| Agent Release | 无 | 无 | System Agent Release 占位 | 未纳入本次盘点 | 占位 |
| Mobile | 无目录 | 不适用 | 不适用 | 不适用 | 文档/脚本残留，当前无实现 |

## 2. Canonical backend 模块矩阵

| 模块/入口 | 已实现内容 | 未完成内容 | 状态 |
| --- | --- | --- | --- |
| `cmd/server` | HTTP/gRPC、健康检查、四模块 Ping、优雅停机 | 默认业务持久化与真实业务 service selection | 部分实现 |
| Auth | SQLite owner 初始化、成员查询、角色修改 | 登录、会话、个人资料、邀请、成员删除；Agent 服务 | 部分实现/Agent 占位 |
| Workspace identity | SQLite identity reader | 默认运行时选择、完整租户生命周期 | 部分实现 |
| Project/Relationship | CRUD、搜索、关系管理和删除清理 | 默认组合、兼容 HTTP、PostgreSQL、实时 | 部分实现 |
| Todo | 完整 CRUD、状态、过滤排序 | 默认组合、兼容 HTTP、生产 cutover | 部分实现 |
| Issue | Create/Get/List/Update/Status | Delete、Search、评论/附件/标签/属性等完整兼容域 | 部分实现 |
| Knowledge | Create/Get | 完整审核/发布兼容链 | 部分实现 |
| Requirement | Save version/Get | 完整状态机兼容边界 | 部分实现 |
| Setting | 工作区设置、skill binding | 默认组合与兼容 API | 部分实现 |
| Space Asset | 只有协议与生成适配器 | 上传、版本、读取实现 | 占位 |
| System Skill | 只有协议与生成适配器 | 发布与读取实现 | 占位 |
| System Agent Release | 只有协议与生成适配器 | 发布与升级解析 | 占位 |
| `cmd/controlplane` | SQLite、身份边界、工作区/成员读取、命令、投影、SSE | 与完整产品 API 合并、默认 Postgres 运行选择 | 已实现独立切片 |
| Delivery Kernel | 命令幂等、CAS、事件链、重放、图约束、证据、检查、DoneGate | 更广产品域与发布验收 | 已实现 |
| P2 flows | Requirement、Quality、Review、Knowledge、Run | 完整外部执行/部署集成 | 已实现核心状态流 |

## 3. 验证等级

| 范围 | 已有证据 | 本次新增验证 | 仍缺少 |
| --- | --- | --- | --- |
| Canonical backend | 单元、SQLite、bufconn、HTTP/SSE、内核与身份测试 | 只读调查代理在本次运行了 `go test ./...` 并报告通过 | 默认业务服务运行验收、PostgreSQL 集成、真实身份上游、容器运行 |
| Team Control 前端 | Core/Views 单测、Playwright 和历史生产构建记录 | 本次静态检查路由、客户端与页面 | 本次未重新启动浏览器或服务 |
| 共享产品前端 | 大量同目录 Vitest 与 E2E 文件 | 本次静态检查路由、调用链、页面与设置 | 全量 `pnpm test`、Playwright 和真实 API 联调 |
| 遗留六域 | 旧基线记录窄测和 typecheck 通过 | 本次仅核查其证据路径指向 `server/**` | 不能转化成 canonical backend 验收 |
| 部署与集成 | Compose、Helm、CI、安装脚本和配置文档 | 本次静态检查 | 真实 Docker/Kubernetes、OAuth、邮件、GitHub/VCS、对象存储联通 |

## 4. 就绪度结论

- **Baseline Ready（功能概览层）**：达到。主要入口、模块和版本断层已有证据索引。
- **Runtime Verified（整个产品）**：未达到。本次没有启动 Web、桌面、遗留服务或外部集成。
- **Canonical Cutover Ready**：未达到。默认 `cmd/server` 仍使用占位业务服务，根产品启动仍依赖 `server/**`。
- **Release Ready**：无法从本次静态盘点得出。
