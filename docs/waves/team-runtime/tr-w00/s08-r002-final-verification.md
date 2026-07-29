# TR-W00 r002 Final Verification

Evidence ID：`TR-EVID-W00-003`

## Authority

| 字段 | 值 |
|---|---|
| Repository | `hvritual/goclaw-team-runtime` |
| Branch | `agent/tr-w00-acceptance-fixes-002` |
| Draft PR | `https://github.com/hvritual/goclaw-team-runtime/pull/2` |
| Reviewed exact commit | `60465b59e559311b406586cf2a50bd6a285e450d` |
| Reviewed exact tree | `5dc081a514a2e6bc18b4b6dce0961cb415182b45` |
| Task | `TR-W00-ACCEPTANCE-002` r002 |
| Issue | `TR-ISSUE-002` |
| Policy manifest SHA-256 | `c8a370406594da4b7f5f6bfa1dbc1bcfb9a775bf989521291fdfde143f5e4815` |

PR #1 的提前合并只作为历史偏差保留，不构成本 Evidence 的 acceptance。
本 Evidence 绑定 PR #2 的上述 exact commit；三名 reviewer 均只读，没有
修改文件、分支或 PR。

## r001 findings closure

| Finding | 收敛结果 |
|---|---|
| source archive 缺双入口 | source allowlist 包含 `cmd`；解包后断言并编译两个入口 |
| cross-build 可混入陈旧文件 | 非空目标失败关闭；task-specific stage；精确 18 项 manifest |
| Codex 可读真实 OAuth home | named permission profile deny `CODEX_HOME`；模型前 read-deny canary |
| binary secret Gate 不完整 | 覆盖 Team/Gateway/Reviewer/Runner/Codex/GitHub secret；安全 `grep --` |
| current projection/state machine/README 错位 | Registry、Wave README、Decision、Issue 和根 README 前向修复 |
| Runner systemd strict 阻断写入 | 改为 `ProtectSystem=full`；其余 capability/kernel/device 加固保留 |
| canary 负向测试 marker 被净化 | fixture 显式允许 synthetic marker，错误进入模型分支可被观察 |

## Deterministic verification

在与 reviewed tree 内容相同的本地 tree 上执行：

- policy manifest、`bash -n`、`git diff --check`：通过；
- focused Go test、vet、关键包 race：通过；
- exact 18 binary Linux/macOS/Windows cross-build、manifest 与 checksum：
  通过；
- 非空输出目录和 synthetic leading-dash secret 负例：均失败关闭；
- source-only release 解包后的双入口构建：通过；
- UI：8/8 tests 与 production build 通过；
- Obsidian adapter：6/6 tests 与 production build 通过；
- 完整 `0.9.0-wave.3` release Gate：通过；
- final systemd/canary 修复后 `go test -count=1 ./...` 与 `go vet ./...`：
  通过。

完整 release 候选的主要 SHA-256：

| Artifact | SHA-256 |
|---|---|
| `SHA256SUMS-0.9.0-wave.3.txt` | `93b80b7d...` |
| Linux amd64 archive | `ec6f1445dfc2d95e673e13a0053744ad53e1ca0fc0c055a63a9961be226233d6` |
| Linux arm64 archive | `1bd862b42ee03b025a9be8775b22230c38784e23ec076443a2e181a9f7ca4a07` |
| source archive | `93f511a5a4e72bb7f49a71293a33319fd6ec4d24c1894a37ebfff48607c83be2` |
| release manifest | `355656b2acf2bc3c02ff7457c7e8942034353ba05b03505264b1bde873e1e884` |

这些是验证候选，不是 GitHub Release 或稳定发布声明。最终 systemd/canary
修复只改变 unit 模板、测试和 Journal；随后全仓 Go test/vet 已重跑。

## Independent final review

| Reviewer | 结论 | P0 | P1 | P2 |
|---|---|---:|---:|---:|
| `tr_w00_code_review` | PASS | 0 | 0 | 2 |
| `tr_w00_security_review` | PASS | 0 | 0 | 4 |
| `tr_w00_docs_review` | PASS | 0 | 0 | 0 new |

三个 reviewer 均绑定 exact commit
`60465b59e559311b406586cf2a50bd6a285e450d`。Code 与 Security 的 P2
进入后继 Wave 风险表，不阻断 TR-W00。

## External target-host boundary

当前容器没有可执行的 Codex CLI，也没有安装到
`/usr/local/bin/goclaw-runner` 的目标 unit。仓库测试证明参数合同和失败
关闭控制流，但不冒充真实 OS enforcement。Linux native、WSL2、Lima 上的
Codex read-deny canary、systemd user-unit smoke、签名/公证和原生安装器
仍由 `RN-W01`/`REL-W01` 与三人试点 Gate 收集。

## Result

`TR-W00` 的代码、安全、文档 final gate 为 P0=0/P1=0，双应用边界可以
作为后继 Team Runtime Wave 的冻结基线。该结论不等于生产 release 放行。
