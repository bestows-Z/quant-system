# 真实持仓与市场智能分析实施计划

设计依据：`docs/superpowers/specs/2026-07-16-portfolio-market-intelligence-design.md`

## 执行原则

- 每个批次按数据库迁移、RPC、gateway、前端、容器、文档、测试的顺序完成。
- gateway-api 不访问数据库；用户身份只从 JWT 信任上下文传入 RPC metadata。
- 先写领域测试和接口契约，再实现业务逻辑；每个自然检查点运行相关测试。
- 外部数据不可用时返回明确状态，不构造虚假行情、龙虎榜、事件或候选结论。
- 旧功能全量回归必须保持通过。

## 第一批：真实持仓与个股均衡工作台

### 1. 数据迁移与领域模型

文件：

- `backend-go/migrations/002_portfolio_and_stock_analysis.sql`
- `deploy/init.sql`
- `docs/database.md`

任务：

1. 新增 `real_portfolios`、`real_positions`、`position_import_tasks`、`portfolio_snapshots`、`portfolio_analysis_records`、`support_resistance_levels`。
2. 添加 user_id、portfolio_id、symbol、quote_time、version 的索引和唯一约束。
3. 定义软删除、乐观版本、时间字段和金额精度。
4. 编写可重复执行的迁移验证脚本。

验收：空库初始化成功；重复执行迁移不破坏数据；跨用户唯一约束正确。

### 2. portfolio-rpc 契约

文件：

- `backend-go/apps/portfolio-rpc/portfolio.proto`
- 生成的 `portfolio/`、`portfolioclient/`
- `backend-go/apps/portfolio-rpc/etc/portfolio.yaml`

RPC：

- CreatePortfolio / ListPortfolios / GetPortfolio
- UpsertPosition / DeletePosition / ListPositions
- PreviewImport / ConfirmImport
- AnalyzePortfolio / GetPortfolioAnalysis

验收：proto 可生成；服务与 client 编译；user_id 不出现在公开 REST 请求体。

### 3. portfolio-rpc 领域实现

文件：

- `backend-go/apps/portfolio-rpc/internal/domain/`
- `backend-go/apps/portfolio-rpc/internal/repository/`
- `backend-go/apps/portfolio-rpc/internal/logic/`
- `backend-go/apps/portfolio-rpc/internal/server/`
- `backend-go/apps/portfolio-rpc/internal/svc/`

任务：

1. 实现用户隔离的组合与持仓 CRUD。
2. CSV 预览执行表头映射、证券校验、数值和日期校验，返回逐行错误。
3. 确认导入使用文件哈希和幂等键，并在事务中写入。
4. 通过 market-rpc 获取最新价，计算市值、浮盈亏和权重。
5. 计算集中度、行业暴露、组合收益波动、最大回撤和简化风险贡献。
6. 保存带版本和行情时间的分析快照。

测试：repository 事务、跨用户访问拒绝、CSV 重复导入、数值边界、空持仓和行情缺失降级。

### 4. 支撑压力分析

文件：

- `quant-python/app/indicators/support_resistance.py`
- `quant-python/app/services/market_service.py`
- `quant-python/app/api/routes.py`
- `quant-python/tests/test_support_resistance.py`
- market-rpc proto/client/logic 对应文件

任务：

1. 实现枢轴点、摆动高低点、ATR 带和成交量价格分布。
2. 合并相近价位并输出 support、pivot、resistance、strength、method、version、valid_until。
3. 新增 Python API，并由 market-rpc 集中 client 调用和缓存。
4. gateway 暴露 `GET /api/stocks/:symbol/support-resistance`。

测试：上涨、下跌、横盘、数据不足、零成交量和极端波动样本。

### 5. gateway-api 接入

文件：

- gateway config/servicecontext/routes/handlers/types
- `docs/api.md`

任务：

1. zRPC 接入 portfolio-rpc。
2. JWT metadata 注入和统一错误映射。
3. 暴露组合、持仓、CSV 预览/确认和分析 API。
4. 扩展个股分析与支撑压力接口。
5. 上传大小、CSV 类型、行数和频率限制。

测试：未登录、越权、错误 CSV、重复确认、组合不存在和正常流程。

### 6. 个股均衡工作台前端

文件：

- `frontend-vue/src/views/market/StockDetail.vue`
- `frontend-vue/src/components/market/StockWorkbench*.vue`
- `frontend-vue/src/components/paper/QuickPaperOrder.vue`
- `frontend-vue/src/api/market.ts`
- `frontend-vue/src/api/paper.ts`

任务：

1. 按方案 C 重组为行情、研判、资讯、持仓影响标签页。
2. 首屏组合分时/K线、成交量、盘口、实时成交、压力支撑和分项评分。
3. 内嵌模拟买卖抽屉，复用 paper API 和 T+1 错误展示。
4. WebSocket 增量更新价格、盘口和时间，不整页刷新。
5. 桌面双列、平板自适应、手机单列和底部交易抽屉。

验收：360、768、1024、1440、1920 五个宽度无覆盖和整页横向滚动。

### 7. 真实持仓前端

文件：

- `frontend-vue/src/views/portfolio/PortfolioList.vue`
- `frontend-vue/src/views/portfolio/PortfolioDetail.vue`
- `frontend-vue/src/views/portfolio/PortfolioAnalysis.vue`
- `frontend-vue/src/components/portfolio/PositionEditor.vue`
- `frontend-vue/src/components/portfolio/ImportPositionsDialog.vue`
- `frontend-vue/src/api/portfolio.ts`
- router/sidebar

任务：组合管理、持仓编辑、CSV 预览确认、盈亏与暴露图表、逐票风险与证据入口。

### 8. 容器、文档与第一批验收

文件：Docker Compose、环境变量、README、API/数据库/架构文档、验收脚本。

测试：

- `python -m pytest -q`
- `go test ./...`
- `go build ./...`
- `npm run build`
- Docker 全量启动和健康检查
- 浏览器桌面/手机交互
- 扩展端到端验收：持仓 CRUD、CSV 幂等、分析、支撑压力、页面内模拟买卖、WebSocket。

## 第二批：市场洞察与智能候选池

1. 扩展 market-rpc 指数、市场宽度、行业/概念板块、涨跌排序、龙虎榜契约和缓存。
2. 新建 feature-rpc，计算技术、资金、情绪、板块和风险分项评分。
3. 实现覆盖率、流动性和风险过滤，保存候选及证据快照。
4. 建设市场总览、板块中心、龙虎榜、智能候选池和高级选股器页面。
5. 通过 Redis Streams 与 ws-api 推送市场和评分增量。
6. 增加数据过期、来源不可用和排序稳定性测试。

## 第三批：新闻事件与 Qdrant 知识构建

1. 新建 news-event-rpc 和事件数据迁移。
2. 对接合法公告源，保存原文来源、哈希和采集时间。
3. 实现事件分类、证券实体映射、方向、置信度和时间衰减。
4. 扩展 rag-rpc：真实 embedding provider 可配置，mock 仅限显式开发模式。
5. 将公告/新闻切块写入 Qdrant，检索结果必须返回来源与时间。
6. 建设热点事件和个股资讯证据页。

## 第四批：增量因子与因子实验室

1. 新建 feature-rpc 增量窗口状态和 factor-lab-rpc。
2. 实现因子定义、版本、快照、实验和生命周期表。
3. 计算 IC、Rank IC、ICIR、衰减、换手率、覆盖率和相关性。
4. 实现中性化、Walk-forward、Purged CV、PBO/Deflated Sharpe。
5. 建设因子实验室页面，线上晋级必须人工确认。

## 第五批：多模型网关与 Agent

1. 新建 model-gateway-rpc 和模型配置迁移。
2. 实现 OpenAI-compatible 基础适配器与 OpenAI/Gemini 原生扩展点。
3. 加密 API Key，提供连接测试、超时、熔断、重试、限额、成本和降级。
4. Agent 使用真实行情、因子、事件、持仓、风险和 RAG 证据。
5. 保存提示版本、工具调用、证据引用、模型响应和使用成本。
6. 建设模型管理与分析审计页面。

## 第六批：高频存储与最终加固

1. Docker Compose 引入 ClickHouse 与 MinIO，提供可选 profile 和健康检查。
2. 高频数据双写、校验、回补和切换读取。
3. 优化前端所有页面的 360/768/1024/1440/1920 响应式布局。
4. 执行负载、故障注入、外部源断连、Redis/RabbitMQ/Qdrant 降级演练。
5. 完善部署、备份、恢复、容量规划、监控和安全文档。
6. 完整端到端验收和已知问题清零。
