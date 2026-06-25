# 节点控制接口示例

本文档记录节点控制链路的最小调用顺序。示例假设服务端地址为 `http://localhost:8080`，并且已经完成产品、版本、License 创建，以及节点注册，已拿到 `node_id`。

## 1. 创建控制服务

控制服务是服务端定义的能力，例如 `restart_process`。

```bash
curl -X POST http://localhost:8080/control-services \
  -H "Content-Type: application/json" \
  -d '{
    "product_id": 1,
    "identifier": "restart_process",
    "name": "Restart Process",
    "service_type": "command",
    "input_schema": {
      "type": "object",
      "properties": {
        "process_name": {"type": "string"},
        "delay_seconds": {"type": "integer"}
      },
      "required": ["process_name"]
    },
    "output_schema": {
      "type": "object"
    }
  }'
```

## 2. 上报节点能力

节点能力声明某个节点支持的服务、节点侧字段 Schema 和通信协议。

### HTTP 节点

HTTP 协议下，`endpoint` 表示节点接收命令的 URL。

```bash
curl -X POST http://localhost:8080/node-capabilities \
  -H "Content-Type: application/json" \
  -d '{
    "node_id": 1,
    "service_identifier": "restart_process",
    "protocol": "http",
    "endpoint": "http://127.0.0.1:19090/control/restart",
    "schema": {
      "fields": {
        "proc": {"source": "process_name", "type": "string", "required": true},
        "delay": {"source": "delay_seconds", "type": "integer", "default": 0}
      }
    }
  }'
```

### MQTT 节点

MQTT 协议下，`endpoint` 表示服务端发布命令的 topic。服务端需要在 `config-dev.yml` 中配置 `mqtt.broker_url`。

```bash
curl -X POST http://localhost:8080/node-capabilities \
  -H "Content-Type: application/json" \
  -d '{
    "node_id": 1,
    "service_identifier": "restart_process",
    "protocol": "mqtt",
    "endpoint": "nodes/1/control/restart_process",
    "schema": {
      "fields": {
        "proc": {"source": "process_name", "type": "string", "required": true}
      }
    }
  }'
```

### WebSocket 节点

WebSocket 协议下，节点不需要提供 `endpoint`，而是主动连接服务端：

```text
ws://localhost:8080/node-control/ws?node_id=1
```

服务端会在创建控制指令时按 `node_id` 找到该连接并发送命令。

## 3. 创建并下发控制指令

```bash
curl -X POST http://localhost:8080/control-commands \
  -H "Content-Type: application/json" \
  -d '{
    "node_id": 1,
    "service_identifier": "restart_process",
    "payload": {
      "process_name": "worker",
      "delay_seconds": "3"
    }
  }'
```

服务端会根据节点能力中的 `schema.fields` 把标准 payload 转换为节点 payload。例如上面的请求会被转换为：

```json
{
  "proc": "worker",
  "delay": 3
}
```

## 4. 协议消息格式

MQTT 和 WebSocket 下发给节点的消息格式一致：

```json
{
  "command_id": 10,
  "node_id": 1,
  "service_identifier": "restart_process",
  "payload": {
    "proc": "worker",
    "delay": 3
  }
}
```

WebSocket 节点执行后返回：

```json
{
  "command_id": 10,
  "status": "success",
  "result": {
    "ok": true
  }
}
```

`status` 可取 `success`、`running`、`failed`、`timeout`。

## 5. 查询指令结果

```bash
curl http://localhost:8080/control-commands/10
```

状态值说明：

- `0`：pending，待发送。
- `1`：sent，已发送。
- `2`：running，执行中。
- `3`：success，执行成功。
- `4`：failed，执行失败。
- `5`：timeout，执行超时。
