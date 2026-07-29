# RN-W01 Journal

## 2026-07-29 — r002 activation

- TC-W01 exact `a25628ae...` / tree `7bd25828...` 的 code/security/docs
  均 P0=0/P1=0；
- 用户选择保留 `strict` 并扩展 `codex-delegated`，接口缺省仍为 strict；
- Team Control 负责 profile policy、release pin、版本/capability/posture
  投影与 kill switch，Runner 负责本机 doctor、目录边界、执行、更新回滚和
  Evidence；
- RN-W01 激活只建立授权，不提前修改产品代码；下一步先冻结 exact Task。

## 2026-07-29 — r002 Task Freeze

- Task `RN-W01-LIFECYCLE-001` r002 从远端 activation exact
  `f7b30062468919db5ca8c4fcb5148f4493188832` /
  tree `4eeb0a92368f29e9bfa5aa0a7cd9fb2ff62cd8b2` 冻结；
- Policy manifest `3/3`，SHA-256 `37c0f491...`；
- 冻结后才允许实现双 profile、跨平台 doctor、版本更新/回滚和并发 Evidence。

## 2026-07-29 — r003 scope correction

- 实现前确认 Team mode 禁止 raw `runner.enqueue`，唯一受信入口是
  `dev.task.enqueue`；
- r002 exact scope 遗漏 `gateway/development.go` 与 `cli/dev.go`，若不
  修订，Team Control 无法选择或约束 delegated profile；
- 保留 r002 activation/freeze 和已授权 workstation/runner 工作，不改写
  历史；r003 前向补充最小 Gateway/CLI scope，重新 Freeze 后继续。

## 2026-07-29 — r003 Task Freeze

- Task r003 从远端 activation exact `ebced0ac55176e67a9ae28351c43255d170bab86` /
  tree `bfa32ace1e6ec1af3c8f7aa006b21524842c73b6` 冻结；
- Policy manifest `3/3`，SHA-256 `5d06456c...`；
- 后继实现必须使用 r003 tuple；r002 产品改动尚未提交，可在本授权下前向
  完成。

## 2026-07-29 — r004 release identity scope correction

- RunnerRelease 缺 `size_bytes`，本地 stage 无法同时绑定中央大小与 SHA；
- r003 scope 未授权 `gateway/team_control.go` 投影新增非秘密字段；
- r004 前向增加完整 artifact identity 与最小 Gateway scope；此前产品
  变更仍未提交，不改写 r002/r003 历史。

## 2026-07-29 — r004 Task Freeze and stop boundary

- Task r004 从远端 activation exact `400c8005a9d3152f083fbd4b51ca2b6253590679` /
  tree `78da2dfc56586057ac1c583331f77fd0f15659fc` 冻结；
- Policy manifest `3/3`，SHA-256 `9f01bb43...`；
- 用户要求完成当前 RN-W01 后暂停；不自动激活 INT/REL，不自动 merge。

## 2026-07-29 — first independent acceptance reproduction

- 审查对象：remote exact `d6a166ceb1f445e7098855841d20bf3903f0d3d5` /
  tree `02698c5cc262b247ac37c53e80fa807ab5cb2c15`；角色为独立
  code/security/docs reviewers，项目 `goclaw-team-runtime`，动作是
  RN-W01 r004 最终验收；
- 期望：r004 冻结门禁全部通过，真实 Team Control 可写入 Runner lifecycle
  policy，claim 对最新 policy 失败关闭，运行进程绑定已激活 release，Runner
  不持有成员 Token，Windows ownership 合同成立，legacy zero-size release
  只读；
- 实际：code `P0=0/P1=4`、security `P0=0/P1=5/P2=1`、docs/ops
  `P0=0/P1=4/P2=1`。复现包括
  `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go test -c ./workstation`
  因 `doctor_test.go` 使用 `syscall.Stat_t` 编译失败；policy 写入拒绝
  `runner.*`；claim 未重校验 repository policy；release projection 未校验
  当前 executable；Windows 未读取 owner SID；Runner work 要求成员
  `GOCLAW_USER_TOKEN`；legacy zero-size record 仍可更新；
- 处理：以上均为 r004 已冻结合同的实现偏差，保持 Task/Wave/scope 不变，
  修复后生成新的 exact candidate，并重新执行全部确定性门禁及三类独立
  验收。Windows Job Object 为 P2，保留为未关闭限制，不把 best-effort
  process-tree 终止宣称为强隔离。

## 2026-07-29 — r005 authentication scope correction

- r004 scope 足以修复 policy、claim、release、Windows 和文档 P1，但未授权
  HTTP Runner device authentication 所需的 `gateway/server.go`、
  `gateway/principal.go`、`gateway/team_guard.go`、`cli/system.go`；
- r005 完整继承 r002–r004，只增加上述认证边界、`README.md` 和
  `cli/runner_security_test.go`；不扩展到 Job Object、远程下载、installer
  或后续 Wave；
- r005 activation 与 Task Freeze 分离提交。冻结 base 必须包含本 plan、
  Registry 和 policy manifest，之后才提交验收修复。
