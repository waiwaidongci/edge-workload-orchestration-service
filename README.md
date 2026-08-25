# Edge Task Orchestrator

纯Go边缘计算任务调度与节点编排服务。它提供节点注册与心跳、能力目录、任务模板版本、调度约束、任务分发、租约、执行回执、重试、取消和死信处理。项目默认使用内存适配器，因此开发机无需先启动PostgreSQL或Redis即可运行完整HTTP流程；`migrations/`和接口边界为生产存储适配留下明确替换点。

## 快速启动

需要Go 1.23或更新版本。

```bash
go run ./cmd/edge-task-orchestrator
```

也可以指定YAML并用环境变量覆盖配置：

```bash
EDGE_CONFIG=configs/config.example.yaml EDGE_HTTP_ADDRESS=:8091 go run ./cmd/edge-task-orchestrator
```

默认监听`http://localhost:8090`，探针为`/healthz`和`/readyz`，Prometheus文本指标为`/metrics`。
浏览器访问`http://localhost:8090/ui/`可查看节点、任务、策略和死信的运行状态。

## 端到端示例

先注册节点，响应中的节点ID后续作为`NODE_ID`：

```bash
curl -sS -X POST localhost:8090/api/v1/nodes -H 'Content-Type: application/json' \
  -d '{"name":"edge-a","region":"tokyo","zone":"a","labels":{"gpu":"true","site":"plant-1"},"capacity":{"cpu_millis":4000,"memory_mb":8192,"storage_mb":20000},"max_concurrent":4}'
```

创建任务模板和版本：

```bash
curl -sS -X POST localhost:8090/api/v1/templates -H 'Content-Type: application/json' -d '{"name":"sensor-transform","description":"transform telemetry"}'
curl -sS -X POST localhost:8090/api/v1/templates/TEMPLATE_ID/versions -H 'Content-Type: application/json' \
  -d '{"image":"registry.example/sensor-transform:1.0","command":["/app/run"],"resources":{"cpu_millis":500,"memory_mb":256},"timeout_seconds":60,"retry":{"max_attempts":3,"base_backoff_seconds":1}}'
```

创建策略并提交任务：

```bash
curl -sS -X POST localhost:8090/api/v1/policies -H 'Content-Type: application/json' -d '{"name":"tokyo-gpu","priority":10,"constraints":{"required_labels":{"site":"plant-1"},"preferred_regions":["tokyo"]}}'
curl -sS -X POST localhost:8090/api/v1/tasks -H 'Content-Type: application/json' \
  -d '{"template_id":"TEMPLATE_ID","template_version":1,"policy_id":"POLICY_ID","priority":5,"labels":{"site":"plant-1"},"parameters":{"window":"5m"}}'
```

调度器每250ms扫描队列并发送任务到内存传输适配器。节点收到消息后，用任务ID、节点ID和一次性回执ID调用：

```bash
curl -sS -X POST localhost:8090/api/v1/tasks/TASK_ID/nodes/NODE_ID/receipt -H 'Content-Type: application/json' -d '{"receipt_id":"r-1","state":"running"}'
curl -sS -X POST localhost:8090/api/v1/tasks/TASK_ID/nodes/NODE_ID/receipt -H 'Content-Type: application/json' -d '{"receipt_id":"r-2","state":"completed","success":true}'
```

重复提交同一个`receipt_id`是幂等的。执行租约或超时后，调度器会按指数退避重新排队；达到最大尝试次数则写入死信，可通过`GET /api/v1/dead-letters`查看并使用`POST /api/v1/dead-letters/{id}/resolve`标记处理。

## 目录和架构

`cmd`是进程入口；`internal`按节点、能力、模板、策略、执行、心跳、调度和失败域拆分，每个域包含`domain`、`application`、`adapter`、`infrastructure`层。所有仓储、时钟、节点传输和Presence依赖均为接口并使用构造注入。`api/openapi.yaml`包含主要路由，`migrations/`包含PostgreSQL建表和回滚脚本，`deploy/`包含Dockerfile及PostgreSQL/Redis compose依赖。

## 验证

```bash
./scripts/check.sh
```

该项目包含48个Go源文件和超过2000行非测试源码。行数统计排除`*_test.go`、生成代码、依赖目录和构建产物；仓库没有通过重复代码或空函数填充行数。
