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
