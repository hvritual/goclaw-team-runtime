# FE-W00 环境与构建基线

- Wave：`FE-W00`
- Step：`FE-W00-S01`
- Evidence：`FE-EVID-W00-005`
- 采集时间：2026-07-26 03:56 CEST
- 状态：`blocked`
- 产品代码变更：无

本文件记录本次可重复核验的环境事实。`blocked` 表示当前证据不足以冻结
后续代码 Wave 的执行基线，不表示已经证明任何前台缺陷。

## Git 与交付物身份

| 检查 | 结果 | 影响 |
|---|---|---|
| 当前目录 `.git` | 不存在 | 不能独立验证或冻结 `base_commit` |
| `git rev-parse --verify HEAD` | 非 Git repository | 文档中的上游 commit 说明不能替代当前工作树证明 |
| `dist/goclaw-team-runtime-source-0.7.0.tar.gz` | SHA-256 `874c82a9d5991a7d404def8a79f5a917c017ecf101216699ccc88f01ac50f757` | 只能标识旧归档内容 |
| 旧归档包含 `docs/waves/` | 否 | 旧归档早于本轮 Wave 治理 |
| 当前 `AGENTS.md` | SHA-256 `98bacd6013032cbaffd15095012ed6fc7cd274b62a78d3fdd738aeeadff94ebf` | 当前仓库政策快照 |
| 旧归档内 `AGENTS.md` | SHA-256 `b4f49eb45c6f790b368da4413d994a7994eab997cb2ef4cf1ff2c11f45c75f26` | 与当前政策不同 |

因此，现有 `0.7.0` 源码归档不能作为本轮 Wave 文档落盘后的可执行 Git
基线。进入 `FE-W01` 前必须在真实 Git 仓库冻结与当前内容对应的 commit。

## 工具链

| 工具 | 实际状态 |
|---|---|
| OS | Linux 6.12.13 x86_64 |
| Git | 2.51.1；源码目录无 metadata |
| Go / gofmt / golangci-lint | 缺失；`go.mod` 要求 Go 1.25.5 |
| Node | v24.14.0 |
| npm / npx | 11.9.0 |
| UI TypeScript | 5.9.3 |
| UI Vite | 5.4.21 |
| Chromium / Chrome /本地 Playwright | 未发现 |
| Docker | 未发现 |
| bubblewrap | 可用 |
| jq / sha256sum / tar / rg / make / gcc | 可用 |

`npm --prefix ui ls --depth=0 --omit=optional` 和
`npm --prefix plugins/obsidian-goclaw ls --depth=0 --omit=optional` 均为
exit 0。npm 报告环境中存在将弃用的 `http-proxy` 配置名；未读取或记录
该配置值。

Web Console 的 `package.json` 只有 `dev`、`build`、`preview`，没有
`test` 或 E2E 脚本。现有 CI 对 Web Console 执行依赖安装和 build，但没有
浏览器交互测试。

## 内容快照

| 对象 | SHA-256 |
|---|---|
| `go.mod` | `75d678dafdeed2fa5409947216037c9c29e5a42c5e156986894b5662a1b7882c` |
| `go.sum` | `e932340bdc08fef8c13a52fea85045ad552cab5a477f2d48e93127f0c7c579b8` |
| `ui/package.json` | `f33858e1e918cb750364f43571d53b56976518f5d4aae581fe18d8c9af7fbb3f` |
| `ui/package-lock.json` | `46fd937f66b1b7a16950df8347619831948e9dded477b7d4ba8139018974bdbb` |
| `ui/vite.config.ts` | `6ce093c3f08247846ec1eb56acb97e8ad661bdcc6158520e533b66e5992c2ec2` |
| `ui/src` 文件哈希列表的有序聚合 | `3b116918ffe13e577ffb3f183fb0bd86fcb12d2cc307989a815e024a57dc9074` |

这些内容哈希只能标识当前展开目录，不能替代 Git commit 或签名来源。

## 确定性前端构建

为避免覆盖仓库产物，构建输出写入临时目录：

```text
npm run build -- --outDir <temporary-directory>
```

结果：

- TypeScript 与 Vite build exit 0；
- Vite 处理 48 个模块；
- 临时构建与 `ui/dist` 经 `diff -qr` 无差异；
- 临时构建与 `gateway/ui_dist` 经 `diff -qr` 无差异。

| 构建文件 | SHA-256 |
|---|---|
| `index.html` | `02fa94e01374fe6303723755a948bd9fe2ef73db5a1bc4a42966cb9cf18963f7` |
| `assets/index-CyBpFmaN.css` | `c09da52aa0b04578b79efb34221859e423aece31fb5db75351ddf82c86f1e64e` |
| `assets/index-BCJyISDE.js` | `1ff3956ddf6382e52891ad62325dee0dd2b0ee9dedefa3fdb45a8f66ddc99215` |

这证明当前 Web Console 源码能生成现有三份静态资源；它不证明 Gateway、
登录、RPC、WebSocket 或页面命令可用。

## 运行与浏览器检查

基线采集开始时，5173、8080 和 28789 均未监听，没有可复用 Gateway 测试
实例。

仓库 Vite dev 脚本在当前沙箱启动失败：

```text
SystemError [ERR_SYSTEM_ERROR]:
uv_interface_addresses returned Unknown system error 1
```

静态构建可由只读 HTTP server 启动，但云浏览器安全策略明确拒绝访问本地
预览地址。按前端测试规范，没有切换到本地 Playwright、原始 CDP 或其他
浏览器通道。因此本轮没有 DOM snapshot、console、截图或交互通过证据。

以上两项均登记为测试环境阻塞，不能归类为 GoClaw 前台 Bug。

## 脱敏拓扑

```text
Browser
  ├─ session HTTP
  ├─ JSON-RPC
  ├─ WebSocket
  └─ channel HTTP
        ↓
Go Gateway
        ↓
Production: embedded gateway/ui_dist
Development: Vite proxy to local Gateway endpoints
```

本轮只检查示例和构建配置文件名，没有读取或保存 Token、Cookie、实际部署
地址、真实 `.env` 或凭据值。

## S01 阻塞与解除条件

| 阻塞 | 解除证据 |
|---|---|
| 当前源码没有 Git metadata | 在真实仓库记录并独立验证 commit SHA |
| Go 1.25.5 与格式/静态检查工具缺失 | 工具版本记录及 Go build/test 基线 |
| 没有可运行的隔离 Gateway 测试实例 | 一次性 root、项目 A/B、角色和重置命令 |
| Browser 无法访问本地测试地址 | 获准且可访问的测试 URL，或用户明确批准规范允许的替代验证通道 |
| 没有 Web Console 测试脚本 | 后续 Wave 批准的测试基础设施计划；W00 不补产品代码 |

`FE-W00-S01` 在上述解除证据齐备前保持 `blocked`。静态契约盘点可以作为
预备输入继续收集，但 `FE-W01` 不得激活。
