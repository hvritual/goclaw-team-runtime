# Team Control Registry、预算与 Context Bundle

适用：`TC-W01 r004`。所有命令由 `goclaw-team-control` 执行，并要求
`GOCLAW_USER_TOKEN`；`goclaw-runner` 不暴露这些中央管理命令。

## 1. 数据权威

| 资源 | Team Control 保存 | 不保存 |
|---|---|---|
| Token Budget | 项目/成员、hard limit、累计 usage、幂等 event | ChatGPT/Codex OAuth、provider token |
| Knowledge Source | 安全 URI、revision、SHA-256、批准状态、typed metadata | Vault 正文、同步凭据 |
| Skill Release | URI、version、SHA-256、兼容版本、批准状态 | 执行时 secret |
| Runner Release | channel、平台、版本、URI、SHA-256、最小协议 | 下载后的 binary、签名私钥 |
| Context Bundle | policy、budget snapshot、批准资源引用、canonical hash | OAuth、device key、知识正文 |

Context Bundle 使用 `goclaw-context/v1` 编译器。相同输入得到相同 ID/hash；
预算使用量、Policy hash、Knowledge revision 或 Skill version 改变时产生新
bundle。Bundle 不会自动下载 URI。

## 2. 预算

创建或更新成员预算：

```bash
goclaw-team-control team budget-put \
  --project project-alpha \
  --id budget-alice-month \
  --user alice \
  --limit 2000000
```

查看中央摘要：

```bash
goclaw-team-control team control-summary --project project-alpha
```

usage event 由受信控制流程以稳定 `event_id` 写入；相同 payload 重试不重复
计费，也不会改变 state revision、更新时间或文件字节；不同 payload 复用 ID
会冲突，超限整笔失败。单项目所有预算之和不得超过
`9007199254740991`（JavaScript safe integer）：

```json
{
  "id": "usage-task-42-r3",
  "project_id": "project-alpha",
  "budget_id": "budget-alice-month",
  "tokens": 18420,
  "task_id": "task-42",
  "metadata": {
    "provider": "workspace",
    "model": "codex",
    "operation": "development"
  }
}
```

可用通用入口提交：

```bash
goclaw-team-control team rpc budget.usage.record --params usage.json
```

## 3. Knowledge、Skill 与 Runner release

三个 Registry 均使用 `(project_id, id)` 复合身份；两个项目可以使用相同
external ID，且不会通过 get/delete 的错误差异探测另一个项目。均使用通用
JSON RPC CLI。

Knowledge Source 完整输入：

```json
{
  "id": "knowledge-architecture-v3",
  "project_id": "project-alpha",
  "name": "Architecture",
  "uri": "file:///srv/knowledge/project-alpha/architecture.md",
  "revision": "git:0123456789abcdef",
  "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
  "status": "approved",
  "metadata": {
    "content_type": "text/markdown",
    "source_kind": "obsidian",
    "repository_id": "repo-alpha",
    "owner_id": "alice",
    "visibility": "project",
    "language": "zh-CN",
    "category": "architecture",
    "license": "internal"
  }
}
```

Skill Release 完整输入：

```json
{
  "id": "skill-go-style-v2",
  "project_id": "project-alpha",
  "name": "go-style",
  "version": "2.0.0",
  "uri": "git+https://github.com/example/team-skills",
  "sha256": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
  "min_runner_version": "0.8.0",
  "status": "approved",
  "metadata": {
    "source_kind": "git",
    "repository_id": "repo-alpha",
    "visibility": "project",
    "category": "coding"
  }
}
```

Runner Release 完整输入：

```json
{
  "id": "runner-0.8.1-darwin-arm64",
  "project_id": "project-alpha",
  "channel": "pilot",
  "version": "0.8.1",
  "os": "darwin",
  "arch": "arm64",
  "uri": "https://downloads.example.invalid/goclaw-runner-0.8.1-darwin-arm64.tar.gz",
  "sha256": "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
  "min_protocol": "1",
  "status": "draft"
}
```

```bash
goclaw-team-control team rpc knowledge.source.put --params knowledge.json
goclaw-team-control team rpc knowledge.source.list --params project.json
goclaw-team-control team rpc knowledge.source.get --params knowledge-get.json
goclaw-team-control team rpc knowledge.source.delete --params knowledge-get.json
goclaw-team-control team rpc skill.release.put --params skill.json
goclaw-team-control team rpc skill.release.get --params skill-get.json
goclaw-team-control team rpc skill.release.list --params project.json
goclaw-team-control team rpc skill.release.delete --params skill-get.json
goclaw-team-control team rpc runner.release.put --params runner-release.json
goclaw-team-control team rpc runner.release.get --params runner-get.json
goclaw-team-control team rpc runner.release.list --params project.json
goclaw-team-control team rpc runner.release.delete --params runner-get.json
```

引用文件的完整内容：

```json
{"project_id":"project-alpha"}
```

`project.json` 使用上式。三个 get/delete 文件分别是：

```json
{"project_id":"project-alpha","knowledge_id":"knowledge-architecture-v3"}
```

```json
{"project_id":"project-alpha","skill_id":"skill-go-style-v2"}
```

```json
{"project_id":"project-alpha","runner_release_id":"runner-0.8.1-darwin-arm64"}
```

Knowledge、Skill、Runner 的 get/put 响应分别使用以下完整 presenter-safe
对象；list 响应是同类对象数组。注意输入 metadata 被中央存储验证，但
Gateway 有意不返回 `metadata`：

```json
{
  "id": "knowledge-architecture-v3",
  "project_id": "project-alpha",
  "name": "Architecture",
  "uri": "file:///srv/knowledge/project-alpha/architecture.md",
  "revision": "git:0123456789abcdef",
  "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
  "status": "approved",
  "created_by": "alice",
  "updated_by": "alice",
  "created_at": "2026-07-29T08:00:00Z",
  "updated_at": "2026-07-29T08:00:00Z"
}
```

```json
{
  "id": "skill-go-style-v2",
  "project_id": "project-alpha",
  "name": "go-style",
  "version": "2.0.0",
  "uri": "git+https://github.com/example/team-skills",
  "sha256": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
  "min_runner_version": "0.8.0",
  "status": "approved",
  "created_by": "alice",
  "updated_by": "alice",
  "created_at": "2026-07-29T08:00:00Z",
  "updated_at": "2026-07-29T08:00:00Z"
}
```

```json
{
  "id": "runner-0.8.1-darwin-arm64",
  "project_id": "project-alpha",
  "channel": "pilot",
  "version": "0.8.1",
  "os": "darwin",
  "arch": "arm64",
  "uri": "https://downloads.example.invalid/goclaw-runner-0.8.1-darwin-arm64.tar.gz",
  "sha256": "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
  "min_protocol": "1",
  "status": "approved",
  "created_by": "alice",
  "updated_by": "alice",
  "created_at": "2026-07-29T08:00:00Z",
  "updated_at": "2026-07-29T08:00:00Z"
}
```

成功删除响应统一为
`{"id":"对应资源 ID","deleted":true}`。删除不存在或其他项目的 ID 返回
not found；没有项目权限先返回 forbidden，不能据此探测资源是否存在。

`draft`、`approved`、`disabled` 是合法状态。只有 `approved` 且 SHA-256
完整的 Knowledge/Skill 能进入 Context Bundle。`approved` 资源不能直接
删除，必须先用对应 `put` 把相同 immutable identity 改为 `disabled`，再
调用 `delete`。

Registry URI 只允许绝对本地路径、`file`、`https`、`git+https`，并拒绝
userinfo、query、fragment、opaque URI、明文 HTTP、未知 scheme、Windows
设备路径和 Unix `/dev`、`/proc`、`/sys` 伪文件系统路径。raw path 与
`file:` 解码路径均在词法折叠 `.`/`..` 后复用同一边界；URI 最长 4096
bytes 且不能包含控制字符。所有 rooted path 都逐段拒绝 DOS device name，
不依赖 Team Control 宿主操作系统，并拒绝旧式 `C|` drive、NTFS ADS、
Win32 尾空格/尾点别名和非法 UTF-8；远端 URI 解码路径也复查字符边界。
raw URI 的首尾空白直接拒绝，不做静默 trim。校验错误不会回显不受信任的
scheme、URI、metadata key/value 或 policy key。Metadata
只允许上例字段以及非秘密的 `secret_ref` 标识；Gateway 响应不回显 metadata。
实际凭据必须由 Runner 本机安全存储解析，不能写进 JSON、Vault、URI 或
Context Bundle。

Policy Rules 只允许：

| key | 类型 |
|---|---|
| `style`、`code_style` | 非空字符串 |
| `max_files`、`max_changed_lines` | 1–1000000 整数 |
| `require_race_test`、`require_all_verifications`、`require_independent_review` | boolean |

未知 key 或类型错误会在写入时拒绝；旧 state 中的未知规则会在
list/resolve/context compile 时失败关闭。

## 4. 编译 Context Bundle

```bash
goclaw-team-control team context-compile \
  --project project-alpha \
  --repository repo-alpha \
  --user alice \
  --budget budget-alice-month \
  --knowledge knowledge-architecture-v3 \
  --skill skill-go-style-v2
```

跨项目 ID、未批准资源、缺失 checksum、用户与预算不匹配或无管理权限都会
失败关闭。Runner 在 `INT-W01` 才通过 project-scoped MCP 消费这些引用；
当前 Wave 只建立中央权威合同。

上述命令等价的完整 `context.compile` 参数：

```json
{
  "project_id": "project-alpha",
  "repository_id": "repo-alpha",
  "user_id": "alice",
  "budget_id": "budget-alice-month",
  "knowledge_ids": ["knowledge-architecture-v3"],
  "skill_ids": ["skill-go-style-v2"]
}
```

响应是 immutable `ContextBundle`，包含 `target_user_id`、`policy`、
`budget.budget_user_id`、批准的 Knowledge/Skill 引用、compiler version
和 canonical SHA-256。项目级预算的 `budget_user_id` 为空，但
`target_user_id` 始终保留，因此同一项目预算用于不同成员时会生成不同
Bundle ID/hash。

完整响应示例（时间与 hash 为格式示例，不是可复用校验值）：

```json
{
  "id": "ctx-dddddddddddddddddddddddddddddddd",
  "project_id": "project-alpha",
  "repository_id": "repo-alpha",
  "target_user_id": "alice",
  "compiler_version": "goclaw-context/v1",
  "policy": {
    "project_id": "project-alpha",
    "repository_id": "repo-alpha",
    "bundle_ids": ["policy-project-alpha-v1"],
    "bundle_hashes": [
      "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
    ],
    "rules": {
      "code_style": "gofmt",
      "max_files": 40,
      "require_race_test": true
    },
    "hash": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
  },
  "budget": {
    "budget_id": "budget-alice-month",
    "budget_user_id": "alice",
    "limit_tokens": 2000000,
    "used_tokens": 18420
  },
  "knowledge": [
    {
      "id": "knowledge-architecture-v3",
      "name": "Architecture",
      "version": "git:0123456789abcdef",
      "uri": "file:///srv/knowledge/project-alpha/architecture.md",
      "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
    }
  ],
  "skills": [
    {
      "id": "skill-go-style-v2",
      "name": "go-style",
      "version": "2.0.0",
      "uri": "git+https://github.com/example/team-skills",
      "sha256": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
    }
  ],
  "hash": "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
  "created_by": "alice",
  "created_at": "2026-07-29T08:00:00Z"
}
```

## 5. Team Web Console

“团队”页的“中央上下文治理”卡片投影：

- Token budget 已用/总量；
- Knowledge 与 Skill 总数/批准数；
- Runner release 数；
- Context Bundle 数。

浏览器只读取 Gateway 返回的项目投影，不持久化个人 Token、device key 或
Codex OAuth。control summary 独立于页面其他数据加载，明确区分 loading、
empty、denied、error/retry 和 ready，不会把权限拒绝误显示为空数据。

## 6. 备份与升级

新增状态仍在 TeamControl 单写 JSON snapshot 内，写入使用临时文件、
`fsync` 和 atomic rename。旧 schema v1 文件缺少新 map 时会初始化为空，
原用户、项目、任务和策略保持不变。r002 使用裸资源 ID 的 candidate state
会在内存中前向重建为复合 key；发现迁移冲突或不安全 legacy URI/Policy
时加载/读取失败关闭，不覆盖原文件。升级前仍应按停机冷备流程保存并校验
完整 state；不要在同步盘上运行活动 control-plane state。
