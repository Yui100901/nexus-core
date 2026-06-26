# 只读 Shell 命令测试产品报告

生成时间：2026-06-26 11:01:57 +08:00

## 结论

`cmd/shell-demo-product` 已完成端到端验证。测试产品可以注册为 Nexus 节点，上报 HTTP 控制能力，接收服务端下发的控制指令，执行白名单内的只读 Shell 命令，并将输出作为控制指令结果返回服务端。

## 覆盖命令

| 命令 | 类型 | 结果 |
| --- | --- | --- |
| `echo` | 只读输出 | 通过 |
| `dir` | 目录读取 | 通过 |
| `del` | 非白名单命令 | 拒绝，通过 |

当前白名单：

- `echo`
- `dir`
- `pwd`
- `whoami`

## 测试用例

| 测试用例 | 覆盖目标 | 结果 |
| --- | --- | --- |
| `TestShellDemoProductEndToEnd` | 产品创建、版本发布、License 创建、节点注册、心跳、能力上报、服务端下发 `echo` 命令、节点返回输出 | 通过 |
| `TestReadOnlyShellCommandGuard` | 白名单保护，验证 `del` 被拒绝，`echo` 可执行 | 通过 |
| `TestShellDemoDirCommandThroughServer` | 服务端下发 `dir` 命令，节点执行目录读取并返回结果 | 通过 |

## 执行命令

```powershell
go test -count=1 ./cmd/shell-demo-product -v
```

执行结果摘要：

```text
=== RUN   TestShellDemoProductEndToEnd
command: id=1 status=3
result: {"ok":true,"command":"echo","output":"nexus shell demo\r\n","exit_code":0}
--- PASS: TestShellDemoProductEndToEnd
=== RUN   TestReadOnlyShellCommandGuard
--- PASS: TestReadOnlyShellCommandGuard
=== RUN   TestShellDemoDirCommandThroughServer
--- PASS: TestShellDemoDirCommandThroughServer
PASS
ok   nexus-core/cmd/shell-demo-product
```

## 安全约束

- 只允许白名单命令。
- `dir`、`pwd`、`whoami` 不接受外部参数。
- `echo` 参数限制数量和长度。
- 参数拒绝 `&`、`|`、`<`、`>`、换行等 Shell 特殊字符。
- 每个命令设置 3 秒执行超时。
- 测试仅使用临时 SQLite 和 `httptest` 服务，不污染 `data/test.db`。
