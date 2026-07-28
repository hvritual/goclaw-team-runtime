# FE-EVID-W01-018 — S11 syscall Gate environment-blocked

## 证据主体

| 字段 | 值 |
|---|---|
| 时间 | 2026-07-26 |
| Project-ID / Repository-ID | `goclaw-team-runtime` / `repo-goclaw-source-review` |
| Wave / Plan | `FE-W01` / `plan-r009` |
| Task / Revision | `FE-W01-TRANSPORT-R1` / `7` |
| Step / Issues | `FE-W01-S11` / `FE-ISSUE-003`、`004`、`005`、`010` |
| Task Base | `8006415eb59952823434740ba2d855b6c66990f9` |
| Preflight HEAD | `2b0ee1a76cc55c954dd4bd759b98990cc39e884d` |
| Actor | `Codex root agent` |
| Environment | Linux `6.12.13` x86_64；`strace 6.8`；ptrace denied |
| Manifest tooling | GNU find `4.9.0`；GNU sort `9.4`；GNU xargs `4.9.0`；GNU sha256sum `9.4`；npm `11.9.0`；Node `v24.14.0` |
| 状态 | `failed` / `environment-blocked` |

## 预期

在创建任何 synthetic token、config、database 或 runtime 前，确认下载工件
完整，并证明当前环境能够使用 `strace -f -e trace=connect` 对 Gateway 和
子进程执行方向感知的 syscall 零出站 Gate。Socket 轮询不能替代该 Gate。

## 下载工件预检

下载阶段 npm cache 与其中 debug log 已按批准合同从精确 root 删除；该缓存
只能通过重新下载恢复。清理后 root 顶层仅含 `tooling` 与 `browsers`。

| Gate | 结果 |
|---|---|
| Playwright version | `1.55.0`；matched |
| tooling package-lock SHA-256 | `53622035b305ccadd941f377f72f9231deb8394810387cea36196b2fb6a7e3fe`；matched |
| Playwright package SHA-256 | `bb26592b48d8a2157291e96a8a23ca39e3def369165283a5e7c883b24faa41b4`；matched |
| Playwright Core package SHA-256 | `36ca1b094edaa37835521c008b26cab5375cbab895ad4e7a9ab6577db23abec5`；matched |
| node_modules regular manifest | `4a87376e407b7093d8dfa42b3051ffbac1b4f5ec86ac05fddc7da08654968988`；456 files；matched |
| node_modules symlink manifest | `ee2013d54217dc845b497a758e9c010bed74001f1fc3cffd487e70a946587f2e`；2 links；matched |
| Chromium build/version | `1187` / `140.0.7339.16`；matched |
| Chromium executable SHA-256 | `2fa605e3639b8cfbe8037d0b8e0324dbf7f9e6ad7beb345374ecd26764e2d92b`；matched |
| Chromium regular manifest | `0f88026f00f407c0c858d3ed95da311baec3320450da01cf12fc97363c7b20e7`；468 files；matched |
| Chromium symlink manifest | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`；0 links；matched |
| quiet offline `npm ls` | passed |
| owner/mode/setid/hardlink/symlink containment | passed；symlink pseudo-mode 不计入 writable file Gate |
| runtime artifact scan | passed；0 token/config/database/profile/cookie/trace/HAR/video/screenshot/log |
| loopback TCP bindability | ports `9/5173/8080/28789` passed |

Canonical manifest 算法、root、版本和冻结 SHA 与 `plan-r008` 完全一致。未输出
文件名清单。由于 syscall Gate 随后失败，package/browser 目录未进入运行
使用阶段，也没有创建新的 browser user-data-dir。

## Syscall Gate 实际结果

先以 `strace -f -e trace=connect` 包裹无网络的 `/bin/true` 做能力测试：

1. sandbox 内调用在创建可用 trace 前被 `PTRACE_TRACEME` 权限拒绝；
2. 经用户授权的宿主级重试仍被同一 ptrace 权限拒绝；
3. 第二次进程被定向终止；两个能力检查日志和临时目录已删除；
4. 没有把 socket 快照、端口轮询或其他自动化框架冒充 syscall Evidence。

因此当前环境不能证明 Gateway/子进程没有短暂或失败的 `connect()`。S11
按计划标记 `environment-blocked`，不能产生绿色 inert-provider 结论。

## 用户触发的同合同重试

2026-07-26，用户再次明确要求启动真实浏览器测试。执行仍严格沿用
`plan-r009` 与 `FE-DEC-014`，没有更换或降低证明机制：

1. 在创建 credential/runtime 前，再次以授权宿主级
   `strace -f -e trace=connect` 包裹 `/bin/true`；
2. 仍在生成可用 trace 前被 `PTRACE_TRACEME`、`PTRACE_SETOPTIONS`
   权限拒绝；
3. 挂起的探测进程被定向终止，退出码 `130`；
4. 精确临时日志存在但为 `0` 行，随后已删除；
5. 未执行 `chmod` 运行阶段切换，未启动 Gateway、Vite、Chromium 或
   Playwright，未创建 credential、config、database、profile 或截图。

这次重试再次确认 `FE-ISSUE-010` 是当前执行环境能力阻断，不是浏览器用例
失败。`FE-EVID-W01-018` 继续保持 `failed/environment-blocked`；只有先在
ptrace-capable 的受控本地环境通过相同能力测试，才能继续 S11 并进入
S10/S05。

## 安全停止

- 未创建 Gateway Token、Team Token、sentinel、synthetic config、
  TeamControl database、browser profile/context 或运行日志；
- 未启动 Gateway、Vite、Chromium 或 provider/model 请求；
- 产品补丁保持 unstaged/uncommitted；
- 仓库内没有 Playwright config、report、test-results、HAR、Trace 或 video；
- `FE-W01-S10/S05` 未执行，不能声称页面身份、登录、WebSocket、刷新或退出
  已通过真实浏览器。

## 恢复条件

保持 r009 的验证机制不变时，只能在一个先通过相同 `/bin/true`
`strace -f -e trace=connect` 能力测试的受控本地环境重新执行 S11。若要改用
其他网络证明机制，必须先建立并批准新的 plan revision；不得在 r009 内
降级或绕过 syscall Gate。

`FE-W01-S08 / FE-EVID-W01-011` 仍独立阻断产品 commit、W01 complete 和发布。
