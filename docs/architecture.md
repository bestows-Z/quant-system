# Architecture

Quant System 是一个前后端分离的量化分析与模拟交易平台。

第一版架构：

```text
Vue 3 Frontend
  | REST / WebSocket
  v
go-zero API Gateway
  | gRPC / go-zero zrpc
  +--> auth-rpc
  +--> stock-rpc
  +--> market-rpc ---- HTTP/gRPC ----> Python Quant Service
  +--> strategy-rpc
  +--> backtest-rpc ---- HTTP/gRPC ----> Python Quant Service
  +--> paper-rpc
  +--> rag-rpc
  +--> agent-rpc ---- zrpc ----> market-rpc / strategy-rpc / backtest-rpc / risk-rpc / rag-rpc / paper-rpc
  |       +---- zrpc ----> model-gateway-rpc ---- HTTPS ----> configured LLM provider
  +--> portfolio-rpc ---- zrpc ----> market-rpc
  +--> feature-rpc ---- MySQL ----> realtime quotes / daily bars / factor snapshots
  +--> model-gateway-rpc
  +--> risk-rpc
  +--> task-rpc
  +--> notify-rpc
  +--> ws-api ---- RabbitMQ ----> quant.events

MySQL stores business data, historical bars, strategies, backtests, paper accounts, orders and risk records.
Redis stores realtime quotes, task states and WebSocket push cache.
```

边界：
- API Gateway 负责 REST、WebSocket、JWT 鉴权、跨 RPC 聚合和统一错误响应。
- Go RPC 微服务负责业务能力拆分：认证、股票、行情、策略、回测、模拟交易、风控、任务。
- Python Quant Service 作为独立行情量化微服务，负责行情源适配、指标计算、策略辅助和简单回测。
- Go 与 Go 之间采用 go-zero zrpc / gRPC 通讯。
- Go 与 Python 第一版保留 HTTP 调试接口，后续通过 gRPC/proto 形成稳定内部 RPC 边界。
- Agent 服务负责量化分析编排，真实调用 market/strategy/backtest/risk/rag/paper。
- 系统只做数据分析和模拟交易，不接入真实交易，不构成投资建议。
- RabbitMQ 将行情、策略、回测、模拟交易、风控、Agent 等事件分发给 task-rpc、notify-rpc 和 ws-api。

## 微服务职责

| 服务 | 职责 |
| --- | --- |
| `api/gateway` | 对外 REST、WebSocket、JWT 中间件、统一响应、请求聚合 |
| `rpc/auth` | 注册、登录、用户信息、角色预留、JWT payload 数据 |
| `rpc/stock` | 股票列表、搜索、详情、自选股 |
| `rpc/market` | 实时行情、K 线、指标接口代理、Redis 行情缓存 |
| `rpc/strategy` | 策略定义、运行策略、策略信号保存与查询 |
| `rpc/backtest` | 回测任务创建、调用 Python 回测、结果/交易/权益曲线落库 |
| `rpc/paper` | 模拟账户、下单、撤单、虚拟撮合、持仓和成交 |
| `portfolio-rpc` | 用户真实持仓、CSV 幂等导入、组合收益与高级风险分析，不连接券商 |
| `feature-rpc` | 大盘广度、板块强度、排序选股、多维候选评分、证据快照与龙虎榜数据契约 |
| `model-gateway-rpc` | 多模型配置、AES-GCM 密钥加密、连通性测试、调用审计与模型选择 |
| `rpc/risk` | 个股风险、账户风险、仓位集中度、预警记录 |
| `rpc/task` | 采集任务、定时任务状态、数据源状态 |
| `rpc/notify` | 通知中心、未读计数、事件转通知 |
| `ws-api` | WebSocket 连接、订阅和实时推送 |
| `quant-python` | 数据源适配、指标计算、简单回测、策略辅助 |

`feature-rpc` 的候选评分由技术、资金、情绪、板块和风险五个维度组成，结果记录因子版本、评分版本、行情时间、数据源和正负证据。候选池只用于研究筛选，不输出确定性买卖建议。龙虎榜在没有可验证数据时返回 `SOURCE_PENDING`，不会使用模拟数据填充。

## 模型调用边界

- 模型配置按用户隔离，API Key 使用 `MODEL_ENCRYPTION_KEY` 派生的 AES-GCM 密钥加密落库。
- 模型 RPC 禁用请求体统计日志，Gateway 到模型 RPC 的请求时长中间件也禁用，防止密钥进入日志。
- 模型用量单独记录 Provider、模型、Token、延迟、状态和 TraceID，不记录明文密钥。
- LLM 只负责研究解释与报告合成；实时行情计算、交易约束和风控判断不依赖 LLM。
- 所有业务 RPC 只加入 Compose 内部网络，不映射宿主机端口；外部请求必须经过带 JWT 的 Gateway。
- Gateway 使用不记录请求体和 Authorization 的自定义访问日志；内置完整请求日志、认证 RPC 客户端耗时日志和认证服务请求体统计日志关闭，避免 JWT 和密码进入日志。
- 前端 Nginx 与 `ws-api` 均关闭包含查询串的默认访问日志，改为只记录 `URL.Path`；WebSocket 仍可用查询参数完成浏览器握手，但 JWT 不会写入新日志。
- 单机 Docker 开发环境关闭基于宿主 CPU 的自适应丢弃，避免镜像构建造成健康检查和行情请求被误杀；连接上限、请求超时和业务限流继续生效。生产集群应由容器 CPU 限额、网关限流和编排层扩缩容共同保护。
- 自定义 Base URL 拒绝环回、私网、链路本地、`.local/.internal` 和云元数据目标，并在 DNS 解析后再次校验 IP；模型 HTTP 客户端不跟随重定向。
- 模型输入会压缩长时间序列、保留最近观测并截断超长文本，避免完整 K 线和原始 JSON 消耗上下文；模型网关返回调用记录主键，Agent 保存报告后按主键建立可审计关联。

## Agent 与系统知识库

- `rag-rpc` 启动时幂等初始化公共系统量化研究库，包含技术指标、因子与回测验证、A 股 T+1 执行和证据报告规范。
- 文档切块生成本地 384 维向量并写入 Qdrant；检索结果同时返回文档、切块、分数和来源，Qdrant 不可用时保留明确降级状态。
- Agent 并行调用行情、指标、策略、回测、风控、RAG 和可选模拟账户工具，再合成固定章节报告。LLM 只做解释与报告合成，不进入实时交易和风控决策路径。
- 每个交易日收盘后，`feature-rpc` 固化候选快照；`agent-rpc` 对前 30 名执行限并发、可重试、幂等的异步研究任务。
- 资金流只采用东方财富或配置 Token 后的 Tushare Pro 真实数据，全部不可用时返回 `SOURCE_UNAVAILABLE`，不使用盘口或价格推算冒充真实资金流。

## 阶段验收要求

每个实现阶段必须运行对应测试：
- Python 阶段：`python -m compileall app`、FastAPI 健康检查、核心接口调用。
- Go 阶段：`go test ./...`、服务启动检查、健康检查。
- 前端阶段：`npm run build`，必要时运行页面访问检查。
- Docker 阶段：`docker compose config`、`docker compose up`，并检查 MySQL、Redis、Go、Python、前端健康状态。

发现错误必须继续修复，不能把已知未修复问题留给使用者。
