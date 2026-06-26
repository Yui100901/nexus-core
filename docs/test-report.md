# nexus-core 全量测试报告

生成时间：2026-06-25 23:02:48 +08:00

## 结论

本轮全量测试与 P1 发布验证通过。主服务、Demo 产品、协议转换测试产品、API 层、Service 层、Repository 层均完成编译或测试验证，未发现阻塞性问题。

## 执行结果

| 检查项 | 命令 | 结果 |
| --- | --- | --- |
| 全量测试（禁用缓存） | `go test -count=1 ./...` | 通过 |
| 全量编译 | `go build ./...` | 通过 |
| 测试用例枚举 | `go test -list . ./...` | 通过 |
| Race 检测 | `CGO_ENABLED=1 go test -race -count=1 ./...` | 通过 |
| 发布验证脚本 | `powershell -ExecutionPolicy Bypass -File .\scripts\release-validate.ps1 -Port 18081` | 通过 |
| Swagger UI 冒烟 | `/swagger/index.html`、`/swagger/doc.json` | 通过 |
| Demo 产品端到端验证 | `go run ./cmd/demo-product -server http://127.0.0.1:18081` | 通过 |
| 协议转换 Demo 端到端验证 | `go run ./cmd/protocol-demo-product -server http://127.0.0.1:18081` | 通过 |
| 定时发布重启恢复 | 创建未来定时版本后停服，跨过发布时间再重启并注册 | 通过 |
| SQLite 迁移重启演练 | 临时 SQLite 自动迁移、写入、同库重启健康检查 | 通过 |

## 覆盖范围

### API 层

已覆盖 4 个 API 集成测试：

- `TestControlAPIHTTPFlow`
- `TestControlAPIManageAndCompleteCommand`
- `TestP1AccessLicenseNodeAndMonitorAPI`
- `TestRegisterDefaultRoutes`

覆盖内容包括：

- 控制服务、节点能力、控制指令创建和查询。
- MQTT 异步回执接口。
- 注册、心跳、License 管理、节点管理。
- 监控和审计分页查询。
- 默认路由注册。

### Service 层

已覆盖 25 个 Service 测试：

- 控制链路：HTTP、MQTT、WebSocket 下发，节点在线校验，License scope 校验，异步回执，Schema 转换。
- 控制服务管理：创建、更新、启停、删除阻断。
- License 主链路：注册、激活、绑定、心跳、并发限制、过期和吊销。
- P2 核心管理：产品删除保护、版本定时发布恢复、最低版本限制、License 续期/清理、节点封禁原因和审计。
- P3 监控审计：在线摘要、节点最近心跳、审计日志查询。

### Repository 层

已覆盖 2 个 Repository 集成测试：

- `TestRepositoryGenericHelpers`
- `TestNodeAndLicenseRepositories`

覆盖内容包括：

- 通用查询、计数、更新、删除 helper。
- Node Repository 创建、按 ID 查询、按设备码查询。
- License Repository 创建、按 Key 查询、状态更新、按状态查询、批量删除。

## 当前验证到的关键能力

- 项目可完整编译：主服务、`cmd/demo-product` 和 `cmd/protocol-demo-product` 均通过构建验证。
- P0 控制链路闭环可用：HTTP/MQTT/WebSocket、异步回执、在线校验、License scope、结果转换均有测试覆盖。
- P1 工程交付能力可用：统一分页参数、API 示例、Repository 测试、Swagger UI 冒烟、Race 检测、发布验证脚本均已通过。
- P2 核心管理策略已固定：产品删除保护、定时发布持久化、License 续期边界、节点封禁原因和审计均有测试覆盖。
- 协议转换测试产品已完成端到端验证：HTTP 和 WebSocket 两条链路均验证了字段映射、类型转换、约束校验和输出转换。
- 定时发布跨进程恢复已完成真实演练：服务停止跨过发布时间后，重启时启动扫描释放到期版本，节点注册成功。
- SQLite 自动迁移已完成临时库演练：同一数据库跨启动复用，健康检查正常。

## 未覆盖或环境受限项

- 未做真实外部 MQTT broker 联调：当前 MQTT 测试使用 fake publisher 验证发布消息结构和状态流转。
- 当前环境未安装 MQTT broker、Docker 或 Podman；发布验证脚本已提供 `-MqttBrokerUrl` 参数，具备 broker 后仍需补充真实发布/订阅验收。
- Swagger UI 本轮覆盖页面和文档内容冒烟，未做逐接口交互点击。

## 建议

- 后续进入 P3 时，补充离线事件持久化、超并发事件、无效访问事件的集成测试。
- 发布前建议固定执行 `powershell -ExecutionPolicy Bypass -File .\scripts\release-validate.ps1 -Port 18081`。
- 如接入真实 MQTT broker，执行脚本时传入 `-MqttBrokerUrl tcp://host:1883`，并补充外部订阅端验收发布内容。
