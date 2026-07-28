# GoClaw 0.8.0-pilot.1 恢复来源证明

状态：`MVP-W00` 收集中  
日期：2026-07-28

## 权威来源

| 项 | 值 |
|---|---|
| 源码归档 | `goclaw-team-runtime-source-0.8.0-pilot.1.tar.gz` |
| 归档 SHA-256 | `cf327169e7654d2284c98482e4d885085ed6068152f5ae9cbd103ea5ffd78c8f` |
| 归档成员数 | `611` |
| 原始 Git 历史 | 不存在于归档，未声称恢复 |
| 新 Git 根提交 | `e4783a4f2bc7a6ce8df1405787c44ed636b195d3` |
| import tag | `v0.8.0-pilot.1-import` |
| import Git tree | `38f798c2a652eaf99d5ad1ca145e50c176ee4c58` |
| Wave 激活提交 | `fd1d67ef32f5b3e6131e454e569077723096dd15` |

根提交完整导入归档中的 611 个文件。提交信息明确说明这是新的归档导入
历史，不能替代、推断或伪造 2026-07-27 丢失的原始 commit。

## 已执行验证

```bash
sha256sum -c SHA256SUMS-0.8.0-pilot.1.txt
tar -tzf goclaw-team-runtime-source-0.8.0-pilot.1.tar.gz
tar --compare --gzip \
  --file goclaw-team-runtime-source-0.8.0-pilot.1.tar.gz \
  --directory goclaw-0.8.0-pilot.1-source
git ls-files
git rev-parse 'v0.8.0-pilot.1-import^{}^{tree}'
```

结果：

- 四个发布归档的 SHA-256 全部通过；
- 源码归档无绝对路径、`..` 穿越、`.git` 或链接成员；
- 解压目录与归档逐文件比较通过；
- Git 根提交追踪 611 个文件；
- 原有 `goclaw/` 脏工作树未被修改。

## 冻结工具链

| 工具 | 版本 | 当前恢复环境 |
|---|---|---|
| Go | `1.25.5` | 官方 Linux amd64 归档，本地临时工具链 |
| Go 归档 SHA-256 | `9e9b755d63b36acf30c12a9a3fc379243714c1c6d3dd72861da637f336ebb35b` | 已验证 |
| Node.js | `24.14.0` | 已安装 |
| npm | `11.9.0` | 已安装 |
| Git | `2.51.1` | 已安装 |

Go 工具链只用于恢复验证，没有进入源码包。正式 CI/开发机应根据
`.tool-versions` 安装相同版本。

## 后续绑定规则

1. 后续 Task、Wave、Evidence、PR 和 Release 只能绑定恢复完成后的唯一
   base commit。
2. 历史文档中的旧 commit SHA 仅作为不可解析的历史引用保留。
3. 恢复标签只能在全部确定性 Gate 和独立复核通过后创建。
4. 任一产品目录 diff、凭据扫描命中或 Gate 失败都必须停止恢复完成声明。
