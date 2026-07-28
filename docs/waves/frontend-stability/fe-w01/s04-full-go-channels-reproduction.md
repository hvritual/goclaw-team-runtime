# FE-W01 S04 — Full-Go Channels Gate Reproduction

日期：2026-07-26
Task：`FE-W01-TRANSPORT-R1` revision 2
Task Base：`1cc3c1188271f084e6412d62ef18d4edaf775193`
发现步骤：`FE-W01-S04`
Issues：`FE-ISSUE-006`、`FE-ISSUE-007`
Evidence：`FE-EVID-W01-009`

本 Evidence 不复制 test source 中的 Bot ID 或 Secret ID literal，不判断它们
是否仍有效，也没有对外部服务执行登录、轮换或撤销。

## 1. 全仓 Gate 阻断

在 shell、完整 Gateway、transport 和 UI build 已通过后执行：

```text
go test ./... -count=1
```

前几个 package 正常完成，但 `channels` 超过三分钟没有结束。停止该次运行后，
以 30 秒失败关闭超时隔离：

```text
go test ./channels -count=1 -v -timeout=30s
```

实际结果：

```text
TestJSONFileStorage                               PASS
TestJSONFileStorageCleanupExpired                 PASS
TestJSONFileStorageFilePath                       PASS
TestJSONFileStorageEmptyDir                       PASS
TestNewWeWorkWsBotChannel                         TIMEOUT after 30s

goroutine:
  channels.TestNewWeWorkWsBotChannel
  channels/weworkwsbot_test.go:64
  time.Sleep(10 * time.Minute)
```

该失败不是 race、shell 或 transport assertion；它由测试主动等待十分钟造成。

## 2. 根因与外部副作用

`channels/weworkwsbot_test.go` 的测试：

1. 在源码中直接构造 credential-shaped Bot ID 与 Secret ID literal；
2. 启动 MessageBus goroutine；
3. 调用真实 WeWork WebSocket channel 的 `Start(ctx)`，但忽略返回 error；
4. 无条件执行 `time.Sleep(10 * time.Minute)`。

即使网络连接立即失败，被忽略的 error 也不会结束测试，因而全仓 Gate 至少阻塞
十分钟。受控环境未证明外部连接成功。

源码只记录以下脱敏结构：

| 字段 | 事实 |
|---|---|
| Bot ID literal | 版本化源码中的 35 字符 credential-shaped value；内容不复制 |
| Secret ID literal | 版本化源码中的 43 字符 credential-shaped value；内容不复制 |
| `channel.Start(ctx)` | line 63；返回值未检查 |
| ten-minute sleep | line 64 |
| 当前/base SHA-256 | `2514948eb0a9fdee39c084ec0cde09eab2b144e2cf9a95511b562c8e4c01f01b` |

该文件与累计 base `697f50e`、r004 activation base `1cc3c118` 完全相同，
且当前没有未提交 diff。问题不由 FE-W01 transport 或 shell patch 引入。

## 3. Issue 拆分

- `FE-ISSUE-006`：真实网络型测试和固定十分钟 sleep 阻断
  `go test ./...`，状态 `root-caused`；
- `FE-ISSUE-007`：版本化测试中存在 credential-shaped Bot/Secret literal，
  有效性未知，按潜在 S0 安全材料处理，状态 `root-caused`。

修复方向必须只改测试：

- 使用明确的 synthetic placeholder；
- 只验证 constructor、默认值、必填字段和初始化状态；
- 不调用 `Start`、不建立网络连接、不启动 goroutine、不 sleep；
- 不在新文档、日志、测试输出或 commit message 中复制旧 literal。

删除当前源码 literal 不能清除 Git 历史。如果这些值曾经有效，负责人必须在
外部 WeWork 管理面撤销/轮换；本任务没有相应权限，也不能以代码 patch 代替
轮换。

## 4. 计划影响

`channels/weworkwsbot_test.go` 不在 r004 allowlist。S04 已再次停止，不能通过
跳过 `channels`、提高 timeout 或删除全仓 Gate 放行。修复前必须建立新的
Plan revision、Task revision 和独立 worktree。
