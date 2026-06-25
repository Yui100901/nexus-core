# API 示例

本文档补充 License、节点、访问接入、监控和审计接口的常用调用示例。示例假设服务端地址为 `http://localhost:8080`。

## 通用查询参数

列表接口统一支持：

- `page`：页码，默认 `1`。
- `page_size`：每页数量，默认 `50`，最大 `200`。
- `limit`：兼容旧调用，传入后等同于 `page_size`。

例如：

```bash
curl "http://localhost:8080/monitor/nodes/heartbeats?page=1&page_size=20"
```

## 产品与版本

创建产品：

```bash
curl -X POST http://localhost:8080/products \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Demo Product",
    "description": "用于接口联调的测试产品"
  }'
```

创建并立即发布版本：

```bash
curl -X POST http://localhost:8080/products/versions \
  -H "Content-Type: application/json" \
  -d '{
    "product_id": 1,
    "version_code": "1.0.0",
    "method": 0
  }'
```

## License 管理

创建 License：

```bash
curl -X POST http://localhost:8080/licenses \
  -H "Content-Type: application/json" \
  -d '{
    "product_id": 1,
    "validity_hours": 720,
    "max_nodes": 2,
    "max_concurrent": 1,
    "remark": "demo license"
  }'
```

批量创建 License：

```bash
curl -X POST http://localhost:8080/licenses/batch \
  -H "Content-Type: application/json" \
  -d '{
    "product_id": 1,
    "validity_hours": 720,
    "max_nodes": 2,
    "max_concurrent": 1,
    "count": 10
  }'
```

查询、更新、续期、吊销和恢复：

```bash
curl http://localhost:8080/licenses/1
curl http://localhost:8080/license-keys/YOUR_LICENSE_KEY

curl -X PATCH http://localhost:8080/licenses/1 \
  -H "Content-Type: application/json" \
  -d '{"max_nodes": 3, "max_concurrent": 2, "feature_mask": "control"}'

curl -X POST http://localhost:8080/licenses/1/renew \
  -H "Content-Type: application/json" \
  -d '{"extra_hours": 168}'

curl -X POST http://localhost:8080/licenses/1/revoke
curl -X POST http://localhost:8080/licenses/1/restore
```

## 注册与心跳

节点首次接入：

```bash
curl -X POST http://localhost:8080/access/register \
  -H "Content-Type: application/json" \
  -d '{
    "device_code": "demo-node-001",
    "license_key": "YOUR_LICENSE_KEY",
    "product_id": 1,
    "version_code": "1.0.0"
  }'
```

节点心跳：

```bash
curl -X POST http://localhost:8080/access/heartbeat \
  -H "Content-Type: application/json" \
  -d '{
    "device_code": "demo-node-001",
    "license_key": "YOUR_LICENSE_KEY",
    "product_id": 1,
    "version_code": "1.0.0"
  }'
```

心跳响应会包含 `pending_control`，用于提示节点是否存在待处理控制任务摘要。

## 节点管理

查询和更新节点：

```bash
curl http://localhost:8080/nodes/1
curl http://localhost:8080/node-devices/demo-node-001

curl -X PATCH http://localhost:8080/nodes/1 \
  -H "Content-Type: application/json" \
  -d '{"metadata": "{\"os\":\"windows\",\"version\":\"1.0.0\"}"}'
```

封禁、解封和解绑：

```bash
curl -X POST http://localhost:8080/nodes/1/ban
curl -X POST http://localhost:8080/nodes/1/unban

curl -X DELETE http://localhost:8080/node-bindings \
  -H "Content-Type: application/json" \
  -d '{"node_id": 1, "license_id": 1}'
```

## 监控与审计

在线摘要：

```bash
curl http://localhost:8080/monitor/online
```

节点最近心跳：

```bash
curl "http://localhost:8080/monitor/nodes/heartbeats?page=1&page_size=20"
```

审计日志：

```bash
curl "http://localhost:8080/audit-logs?resource_type=license&resource_id=1&page=1&page_size=20"
```
