# Go Backend

企业级量化智能体平台 Go 后端，采用 go-zero API Gateway + zrpc 微服务结构。

服务拆分：
- `apps/gateway-api`：REST、JWT、统一响应、限流、中间件和跨 RPC 聚合
- `apps/user-rpc`：用户注册登录、角色权限、JWT 所需用户信息
- `apps/market-rpc`：股票、行情、K 线、指标、自选股，代理 quant-python
- `apps/strategy-rpc`：策略配置和策略信号
- `apps/backtest-rpc`：回测任务和结果
- `apps/paper-rpc`：模拟账户、订单、成交、持仓
- `apps/risk-rpc`：个股、账户、持仓、回测风险评分和预警
- `apps/rag-rpc`：知识库、文档、检索、mock embedding、Qdrant 预留
- `apps/agent-rpc`：量化分析智能体编排、mock LLM、真实工具调用
- `apps/task-rpc`：采集任务、异步任务、任务日志
- `apps/notify-rpc`：站内通知和事件提醒
- `apps/ws-api`：WebSocket 连接、订阅和推送

当前阶段已实现 user、market、strategy、backtest、risk、paper、rag、agent 和 gateway 的第一版可运行闭环。

## Verification

```bash
cd quant-system/backend-go
GOCACHE=$PWD/.gocache go mod tidy
GOCACHE=$PWD/.gocache go test ./...
GOCACHE=$PWD/.gocache go build ./...
```
