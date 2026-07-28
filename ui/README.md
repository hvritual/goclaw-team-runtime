# GoClaw Team Console

`ui/` 是 GoClaw Team Runtime `0.7.0` 的默认人工控制面，使用 React、TypeScript 与 Vite 构建，并在发布时嵌入 Go 二进制。

## 工作区

- 总览：WorkItem、Issue、Runner、Policy 和 Evidence。
- 对话：按 `project_id + topic_id` 共享会话。
- 规格：Ouroboros 访谈、Seed、编译、评估与演化。
- 记忆：Catalog 状态、approved 检索、来源和候选。
- 审批：知识、记忆、Seed、开发任务和 Harness 治理。
- 开发：冻结任务、工作站队列、DoneGate 和修复 revision。
- 团队：成员负载、任务、Bug、Runner、文档与组件。
- 进度：任务状态、证据覆盖与 Trace 健康度。
- Harness：版本、实验、提升与回滚。

## 安全模型

- 登录 Token 仅用于 `POST /auth/session`，不会进入浏览器持久存储。
- Gateway 返回 HttpOnly、SameSite=Strict 的短期会话 Cookie。
- 浏览器 `/rpc` 请求必须携带会话 CSRF Token。
- WebSocket 复用同源 Cookie并拒绝跨站 Origin。
- Reviewer Token 仅保存在当前 React 页面内存，刷新即清空。
- 项目切换只改变请求上下文；授权始终由 Gateway 服务端执行。

## 本地构建

```bash
npm ci
npm run build
```

生产构建由 `scripts/build-release.sh` 自动复制到 `gateway/ui_dist`，随后由 `go:embed` 打入 `goclaw`。不要手工编辑 `gateway/ui_dist` 中的哈希文件。

完整使用与验收见 [`../docs/TEAM_WEB_CONSOLE_CN.md`](../docs/TEAM_WEB_CONSOLE_CN.md)。
