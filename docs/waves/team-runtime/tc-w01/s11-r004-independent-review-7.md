# TC-W01 S11 r004 Independent Review 7

Evidence ID：`TC-EVID-W01-016`

状态：`failed`

Reviewed exact remote commit：
`a7dd4b27be22948053cdc84888f80ef41256f677`

Reviewed tree：
`ebba9a4b8cce6546877638c0c7f6a7e4b6b8d466`

| Reviewer | P0 | P1 | P2 | 结论 |
|---|---:|---:|---:|---|
| code | 0 | 2 | 1 | BLOCK |
| security | 0 | 2 | 3 | BLOCK |
| docs/governance | 0 | 0 | 1 | PASS |

## P1

- Registry raw/file absolute 判断依赖宿主 `filepath.IsAbs`，Linux/Windows
  结果不一致；
- local Registry 接受 NTFS ADS；
- repository relative path 接受 Win32 会剥离为 `.`/`..` 或别名的尾空格/
  尾点 segment；
- relative path 接受非法 UTF-8，JSON roundtrip 可能改变字节。

Remote HTTPS decoded control、state path 与 Registry canonical record hash 为
P2。

## 处理决定

- 本 exact review 保留为失败 Evidence；
- 使用平台中立 portable-absolute helper，不依赖 build host；
- local path 拒绝 ADS、单反斜杠 rooted、尾空格/尾点别名；
- raw/decoded path 拒绝非法 UTF-8，HTTPS/git+HTTPS 解码后复查；
- relative path 逐段执行 Windows portable-name boundary；
- 新 exact SHA 重新执行三路验收。
