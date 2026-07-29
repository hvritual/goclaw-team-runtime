# TR-EVID-W00-001 — Application Boundary Verification

状态：`deterministic-passed / independent-review-pending`

## Authority

| 字段 | 值 |
|---|---|
| Task | `TR-W00-APP-SPLIT-001` r001 |
| Wave / Step | `TR-W00` r001 / `S02`–`S05` |
| Activation base | `e87aa3ac3330f3259a12d2a7f3198ee2726d6814` |
| Freeze commit | `fce193e` |
| Main implementation | `76df7a0` |
| Release cleanup fixes | `cafc72c`、`1de4a4b` |
| Release source commit | `1de4a4b70efe351bf2b39d5bb3ebb1494ce909af` |
| Release source tree | `b94f0c9343af2f76ad9f4af6b6ddb5c7aa78dfd6` |
| Draft PR | `https://github.com/hvritual/goclaw-team-runtime/pull/1` |

工具链：Go `1.25.5`、Node `24.14.0`、npm `11.9.0`。

## Application contract

真实 host binary 的 `--help` 和 `version` 通过：

- `goclaw-team-control` 显示中央命令且不含顶层 `runner`；
- `goclaw-runner` 显示 `runner/config/health/status/version`，不含
  `team/gateway/dev/harness/ouroboros`；
- Cobra 内建 `help/completion` 两端均保留；
- 兼容 `goclaw` 未过滤命令；
- 版本首行分别使用三个 binary 名称。

`cli/application_test.go` 对 unified、Team Control 和 Runner allowlist 做
确定性表驱动测试，未知 mode 失败关闭。

## Deterministic results

| Gate | 结果 |
|---|---|
| r001 Policy manifest | 3/3 SHA-256 OK |
| `go test ./cli ./cmd/team-control ./cmd/runner` | passed |
| `go vet ./cli ./cmd/team-control ./cmd/runner` | passed |
| `go test -race ./cli ./teamcontrol ./workstation ./gateway` | passed |
| host binary help/version probes | passed |
| `scripts/build-apps.sh` | 18 binaries + `SHA256SUMS`; 19 files；全部 OK |
| Web UI | 8/8 tests；production build passed |
| Obsidian adapter | 6/6 tests；production build passed |
| `bash -n` build scripts | passed |
| `git diff --check` | passed |

跨平台矩阵为 Linux、macOS、Windows × amd64、arm64；每个目标都构建
Team Control、Runner 和兼容入口。

## Release Gate

最终成功运行：

```text
RELEASE_VERSION=0.9.0-wave.2 INCLUDE_OBSIDIAN_PLUGIN=0 \
  ./scripts/build-release.sh
```

退出码为 `0`。脚本重新执行 Web 8/8、build、目标 Go package suite、双架构
Linux 三 binary、Windows/macOS 交叉构建、归档合同、源码凭据扫描、tracked
bundle 一致性和发布 SHA 复算。

`0.9.0-wave.2`：

| Artifact | SHA-256 |
|---|---|
| Linux amd64 | `9859d01e01e4c1103bea5389e7a79f2d8efe6458e151e7e34ee16f1fe64ca13e` |
| Linux arm64 | `9553effec151e96d02a0d1c965faac690da967ec1f14cd96794248133fea6031` |
| Source | `e55e075a872330d8cf33e389a1daeae66a787d5dad011510511f8b58d4ae02a2` |
| Release manifest | `8c00e64abd8ee494f64f6a96cc2dcd0a38c723c1eaf95451f49f66953df0e92d` |

两个 Linux 包均包含 executable mode 的：

```text
goclaw
goclaw-team-control
goclaw-runner
```

并包含两个 systemd 模板。归档没有包含 Obsidian binary；插件在本次已单独
完成 test/build。

## Security observations

- 新入口没有读取或持久化 GitHub、Team、Gateway、Reviewer、Runner 或
  Codex OAuth；
- release source scanner passed；
- `build-apps.sh` 会在对应环境变量存在时逐 binary 查找其原值，只报告变量
  名和 artifact，不回显 secret；
- 命令过滤不被描述为授权边界；服务端 RBAC/device key/ExecutionPack 保持；
- Runner 仍只在 Linux/WSL2/Lima substrate 执行，原生 Windows/macOS 只作
  管理/注册客户端。

## Environment cleanup observation

共享工作区文件系统在 release process 结束后曾重新物化已经隔离删除的
`.release-*` stage。发布目录已先原子完成且四项 SHA 重验全部通过；精确
临时 stage 随后被人工删除并验证。该行为不改变 Git tree 或归档内容，但在
普通本地文件系统/CI 再验证前，不把“宿主自动清理”列为跨环境 passed。

## Remaining gate

作者已完成确定性验证，不能替代独立 final。TR-W00 保持 `active`，
`TR-EVID-W00-001` 保持 `collecting`，直到独立 code/security/docs review
在 exact review commit 上给出 P0=0/P1=0。
