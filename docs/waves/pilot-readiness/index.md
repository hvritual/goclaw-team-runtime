# 三人试点就绪 Track

- Track ID：`PILOT-READINESS-2026-07`
- 候选版本：`0.8.0-pilot.1`
- 当前 Wave：`PILOT-W00`
- 当前计划：[`PILOT-W00 plan-r005`](pilot-w00/plan-r005.md)
- 当前状态：`blocked`；等待 `MVP-W00` 和 `FE-W01` 完成，当前 revision
  仅允许文档变更。
- 目标：一个中央控制面、一个项目、三名真人、三台独立 Runner 的可控开发试点

本 Track 吸收尚未完成的前台读取/命令/韧性工作，并增加 Runner、Wave
运行时门禁、冷备与三人并发验收。它不宣称生产就绪，也不改变
`FE-W01` 中两个外部阻断事实：

- 历史 credential-shaped material 仍需责任人提供撤销、轮换或从未有效证明；
- 当前执行环境不允许 ptrace，因此旧的 syscall 零出站 Gate 仍无法在这里通过。

试点运行采用全新 Token、全新 Runner device key 和独立数据根。正式部署
前必须由 `goclaw pilot check` 对上述条件失败关闭。

当前实现和确定性发布 Gate 已收敛为 `0.8.0-pilot.1` 技术候选，但 Track
仍未完成：真实三台电脑、Codex OAuth、飞书、浏览器/Obsidian Desktop、
bwrap/WSL2/Lima 和 credential owner 证明尚未全部完成。代码完成不自动
等于试点已上线。

## 平台边界

| 成员电脑 | 试点执行环境 | 状态 |
|---|---|---|
| Linux x86_64 / arm64 | 原生 Linux + bwrap | 支持 |
| Windows 11 | 专用 WSL2 Ubuntu，全部数据位于 guest ext4 | 支持 |
| macOS Intel / Apple Silicon | 专用 Lima Ubuntu VM，禁止 host 目录共享 | 支持 |
| 原生 Windows | 只允许控制面命令；Runner 执行失败关闭 | 不支持 |
| 原生 macOS | 只允许控制面命令；Runner 执行失败关闭 | 不支持 |

## 试点成功定义

1. 三个独立用户只能看见授权项目，并各自使用本机 Codex OAuth；
2. 三个 Runner 同时领取绑定到各自主人的冻结任务，无串领和重复完成；
3. Task 在 freeze/enqueue 时绑定获批 active Wave、plan SHA、Step 和 Issue；
4. Web Console 切换项目不会显示旧项目数据，登录过期会回到登录页；
5. 项目聊天刷新后恢复共享历史，重复/乱序事件不重复拼接；
6. Issue、WorkItem、审批、Runner 和进度能由三人共同观察并按权限流转；
7. 任务、补丁、测试、Evidence、接受结果与 Issue/WorkItem 保持可追溯；
8. 冷备、校验和恢复演练可执行；任何一致性歧义阻止新 enqueue/accept；
9. Runner 取消会终止完整进程组，验证环境不继承宿主临时目录或危险 Git 配置；
10. 不自动 commit、push、开 PR 或 merge；这些仍由人工执行。
