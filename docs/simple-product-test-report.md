# 简单产品多绑定与并发心跳测试报告

生成时间：2026-06-26 10:48:47 +08:00

## 结论

本轮基于简单产品链路的 API 端到端测试通过。测试通过真实 Gin 路由创建产品、发布版本、创建 License、注册多个节点、发送心跳，并验证绑定数量限制、在线并发限制、重复并发心跳和解绑后的重新绑定能力。

## 新增测试用例

| 测试用例 | 覆盖目标 | 结果 |
| --- | --- | --- |
| `TestSimpleProductMultiBindingHeartbeatAPI` | 3 个节点绑定同一个 License，3 个节点进行 4 轮并发重复心跳，校验在线统计、心跳列表和节点列表 | 通过 |
| `TestSimpleProductBindingAndConcurrentLimitsAPI` | License 最大绑定数为 3、最大在线并发为 2，校验第 4 个节点绑定被拒绝、第 3 个在线心跳被拒绝、解绑后新节点可重新绑定 | 通过 |

## 关键场景

### 多绑定成功

- 创建简单产品 `simple-product-api`。
- 创建并立即发布版本 `1.0.0`。
- 创建 License，设置 `max_nodes = 3`、`max_concurrent = 3`。
- 注册 `simple-node-a`、`simple-node-b`、`simple-node-c`。
- 每次注册均返回 `binding_established = true`。
- 注册后 `current_node_count` 依次为 `1`、`2`、`3`。

### 多并发心跳成功

- 3 个已绑定节点执行 4 轮并发心跳，共 12 次心跳请求。
- 所有心跳均返回 HTTP `200`、业务码 `200`、`online = true`。
- `/monitor/online` 返回 `total_online = 3`。
- `/monitor/nodes/heartbeats?page=1&page_size=10` 返回至少 3 条节点心跳记录。
- `/nodes?page=1&page_size=10` 返回 3 个节点。

### 绑定数量限制

- 创建 License，设置 `max_nodes = 3`、`max_concurrent = 2`。
- 注册 `limit-node-a`、`limit-node-b`、`limit-node-c` 成功。
- 注册第 4 个节点 `limit-node-d` 被拒绝。
- 预期返回：HTTP `409`、业务码 `409`。

### 在线并发限制

- `limit-node-a` 心跳成功。
- `limit-node-b` 心跳成功。
- `limit-node-c` 作为第 3 个在线节点心跳被拒绝。
- 预期返回：HTTP `409`、业务码 `409`。

### 解绑后恢复绑定能力

- 解绑 `limit-node-c` 与 License 的绑定关系。
- 再次注册 `limit-node-d` 成功。
- `/licenses?product_id=<product_id>&page=1&page_size=10` 可查询到该 License。

## 执行命令

| 检查项 | 命令 | 结果 |
| --- | --- | --- |
| 简单产品专项测试 | `go test -count=1 ./api -run "TestSimpleProduct" -v` | 通过 |
| 全量 Go 测试 | `go test -count=1 ./...` | 通过 |
| 前端构建验证 | `npm run build`（目录：`web`） | 通过 |

## 执行摘要

```text
=== RUN   TestSimpleProductMultiBindingHeartbeatAPI
--- PASS: TestSimpleProductMultiBindingHeartbeatAPI
=== RUN   TestSimpleProductBindingAndConcurrentLimitsAPI
--- PASS: TestSimpleProductBindingAndConcurrentLimitsAPI
PASS
ok   nexus-core/api
```

全量测试结果：

```text
ok   nexus-core/api
ok   nexus-core/domain/service
ok   nexus-core/persistence/repository
```

前端构建结果：

```text
✓ built
```

## 说明

- 测试中的 `record not found` 日志来自首次注册节点时查询旧节点和旧绑定的探测流程，是预期路径，不影响结果。
- 部分 `SLOW SQL` 日志出现在并发心跳测试中，测试仍通过；当前测试 SQLite 使用临时库，且连接串行化会放大并发写入耗时。
- 本轮测试未接入真实外部 MQTT broker；节点控制协议链路仍由既有控制链路测试覆盖。
