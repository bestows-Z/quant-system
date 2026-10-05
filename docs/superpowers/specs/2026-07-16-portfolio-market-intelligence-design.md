# 真实持仓与市场智能分析平台设计规格

日期：2026-07-16

## 1. 目标与边界

本阶段将现有量化分析与模拟交易系统扩展为覆盖市场发现、个股研判、真实持仓分析、事件知识和多模型解释的企业级量化智能平台。

系统继续坚持以下边界：

- 不连接券商账户，不提交真实委托。
- 真实持仓只支持手动维护和 CSV 导入。
- 所有买卖入口只连接 paper trading。
- “智能推荐”统一命名为“智能候选池”，只表示研究候选，不构成投资建议。
- 免费公共行情以秒级刷新和明确延迟为目标，不承诺交易所级毫秒行情。
- LLM 不进入实时评分和撮合快路径，只负责异步证据解释和报告生成。

## 2. 已确认的产品结构

### 2.1 导航分类

市场洞察：

- 市场总览
- 板块中心
- 龙虎榜
- 智能候选池
- 行情中心
- 热点事件

组合与交易：

- 真实持仓
- 持仓分析
- 模拟账户
- 模拟下单
- 委托记录
- 成交记录

量化研究：

- 因子实验室
- 策略中心
- 回测中心
- 风险中心

智能与知识：

- 智能体分析
- 事件知识库
- 检索问答
- 模型管理

### 2.2 个股综合研判工作台

采用用户确认的 C 方案，即均衡工作台。

首屏同时保留：

- 股票名称、价格、涨跌幅、成交量、成交额和行情时间。
- 分时、K 线、成交量和主要指标切换。
- 五档盘口和实时成交信息。
- 技术面、资金面、情绪面、消息面、板块强度和风险分项评分。
- 压力位、枢轴位、支撑位及计算时间。
- 加入自选、关联真实持仓、模拟买入和模拟卖出入口。

深度内容通过“行情、研判、资讯、持仓影响”标签页分开呈现。移动端按行情、研判、盘口、交易的优先级排列为单列，次要区域折叠或进入标签页。

### 2.3 真实持仓

真实持仓和模拟账户是两个独立领域，不共享订单、成交或资金余额。

用户可以：

- 创建一个或多个真实持仓组合。
- 手动录入股票、持仓数量、可用数量、成本价、建仓日期和备注。
- 通过 CSV 执行“上传、预览、校验、确认”导入流程。
- 查看当前市值、浮动盈亏、盈亏比例、单票集中度和行业暴露。
- 查看组合波动率、最大回撤、相关性、风险贡献、事件影响和压力测试。
- 请求 Agent 对脱敏后的持仓和结构化证据进行组合解释。

原始持仓必须按 JWT 用户隔离。RPC 从可信鉴权上下文获取 user_id，不接受前端传入任意用户编号。

### 2.4 智能候选池

候选池按综合评分或指定分项排序，至少展示：

- 技术面、资金面、消息面、情绪面、板块强度和风险评分。
- 入选原因、负面因素、支撑位、压力位、置信度和数据时间。
- 因子版本、评分版本、模型版本和结构化证据引用。
- 加入自选、打开个股研判和进入模拟下单的操作。

候选池不能使用“必涨、稳赚、推荐买入”等确定性文案。数据覆盖不足或关键数据过期时，系统降低置信度或返回“暂不可分析”。

## 3. 服务架构

### 3.1 增强现有服务

- `gateway-api`：REST 聚合、JWT、参数校验、统一响应和限流；禁止访问数据库。
- `ws-api`：推送行情、指数、板块、候选评分和持仓风险事件。
- `market-rpc`：扩展指数、板块、涨跌排行、龙虎榜、实时成交和筛选排序。
- `agent-rpc`：编排结构化工具证据，生成解释、报告和审计记录。
- `paper-rpc`：承接个股页面的模拟买卖，继续执行资金、持仓、手续费、滑点和 T+1 检查。
- `risk-rpc`：增加真实组合风险评估。
- `rag-rpc`：增加公告、新闻和研究文档的来源、版本和引用能力。
- `task-rpc` / `notify-rpc`：承接导入、批量分析、事件解析和因子评估任务。

### 3.2 新增服务

- `portfolio-rpc`：真实持仓、CSV 导入、组合快照、风险暴露和分析记录。
- `feature-rpc`：增量因子、支撑压力、综合评分和候选池。
- `news-event-rpc`：公告与热点采集、实体映射、事件分类、方向、置信度和衰减。
- `factor-lab-rpc`：因子注册、版本、IC、Rank IC、ICIR、相关性、样本外评估和生命周期。
- `model-gateway-rpc`：GPT、DeepSeek、Gemini、豆包、通义、Kimi 适配、密钥加密、重试、熔断、成本统计和降级。

所有 Go 服务采用 go-zero zRPC，gateway 只通过 zRPC 调用领域服务。Python 量化计算能力通过集中 client 或 RPC 适配访问，不允许业务逻辑散落 `http.Get`。

## 4. 实时与异步数据流

### 4.1 实时快路径

行情事件进入 Redis Streams，`feature-rpc` 使用 Redis 中的滚动窗口状态执行增量计算，产生分项评分、支撑压力和结构化理由码，再由 WebSocket 增量推送。

该路径不调用 LLM。目标响应时间仅计算本地行情事件进入系统后的规则与因子处理，不包含公网行情源延迟。

### 4.2 异步解释路径

Agent 从行情、因子、风险、持仓、新闻事件和 RAG 获取结构化证据，通过 `model-gateway-rpc` 调用用户启用的模型，生成可追溯解释。模型超时、不可用或证据覆盖不足时降级为规则解释。

### 4.3 消息分工

- Redis Streams 承载高频行情和因子事件。
- RabbitMQ 承载任务、通知、文档解析和分析完成等业务事件。
- WebSocket 只推送客户端订阅的增量结果。

## 5. 数据存储

### 5.1 MySQL

新增主要表：

- `real_portfolios`
- `real_positions`
- `position_import_tasks`
- `portfolio_snapshots`
- `portfolio_analysis_records`
- `market_indices`
- `index_quotes`
- `market_sectors`
- `sector_quotes`
- `dragon_tiger_records`
- `dragon_tiger_seats`
- `market_breadth_snapshots`
- `factor_definitions`
- `factor_versions`
- `factor_snapshots`
- `support_resistance_levels`
- `candidate_pool_snapshots`
- `candidate_evidence`
- `news_sources`
- `raw_documents`
- `market_events`
- `event_security_relations`
- `event_factor_snapshots`
- `document_ingestion_tasks`
- `model_providers`
- `model_configs`
- `model_usage_records`
- `analysis_runs`
- `analysis_evidence`
- `recommendation_audits`

所有业务表包括创建和更新时间。用户数据包含 user_id 索引；历史快照包含 symbol、trade_date 或 quote_time 组合索引；来源数据包含 source、source_key 和采集时间唯一约束或幂等约束。

### 5.2 Redis

保存最新行情、盘口、指数、板块、滚动因子状态、WebSocket 订阅和短期候选池缓存。所有缓存键必须包含版本并设置合理 TTL。

### 5.3 ClickHouse

保存分钟线、Tick、盘口快照、盘中因子快照和板块强度快照。第一批不依赖 ClickHouse 才能启动；在第六批启用后切换高频历史数据写入。

### 5.4 Qdrant 与 MinIO

Qdrant 保存公告、新闻和研究文档的向量及来源元数据。MinIO 保存公告 PDF、CSV 原始文件和解析产物。第一批 CSV 可先使用受控本地临时文件完成预览，正式确认后只保存导入摘要；第六批接入 MinIO。

## 6. 分析方法

### 6.1 支撑与压力

首版综合使用：

- 经典枢轴点及 S1/S2、R1/R2。
- 指定窗口内摆动高低点。
- ATR 波动带。
- 成交量价格分布的高成交密集区。

结果保存方法、输入周期、数据时间、版本和有效期。后续由因子实验室比较不同方法在样本外的触达率和突破有效性。

### 6.2 候选评分

首版使用可解释加权评分，不让 LLM直接产生分数：

- 技术面：趋势、动量、波动和量价结构。
- 资金面：成交额、换手率、量比、资金流和盘口不平衡。
- 情绪面：涨跌分布、板块强度、龙虎榜和市场宽度。
- 消息面：公告与事件方向、置信度、影响范围和时间衰减。
- 风险：波动、回撤、流动性、涨跌停、ST/退市等字段。

分项权重版本化。候选产生后必须经过覆盖率阈值、流动性过滤和风险上限过滤。

### 6.3 因子生命周期

因子状态为 `CANDIDATE -> SHADOW -> PROMOTED -> RETIRED`。自动流程只生成候选和影子结果，不能未经样本外验证自动替换线上评分。

评估包括 IC、Rank IC、ICIR、衰减、换手率、覆盖率、相关性、行业与市值中性化、Walk-forward、Purged CV、PBO 或 Deflated Sharpe。

## 7. API 概要

市场：

- `GET /api/market/overview`
- `GET /api/market/sectors`
- `GET /api/market/sectors/:code`
- `GET /api/market/dragon-tiger`
- `GET /api/market/screener`
- `GET /api/stocks/:symbol/analysis`
- `GET /api/stocks/:symbol/support-resistance`
- `GET /api/stocks/:symbol/trades`

真实持仓：

- `POST /api/portfolios`
- `GET /api/portfolios`
- `GET /api/portfolios/:id`
- `POST /api/portfolios/:id/positions`
- `PUT /api/portfolios/:id/positions/:position_id`
- `DELETE /api/portfolios/:id/positions/:position_id`
- `POST /api/portfolios/:id/import/preview`
- `POST /api/portfolios/:id/import/confirm`
- `GET /api/portfolios/:id/analysis`
- `POST /api/portfolios/:id/analyze`

候选与模型：

- `GET /api/candidates`
- `GET /api/candidates/:symbol/evidence`
- `GET /api/models/providers`
- `POST /api/models/configs`
- `POST /api/models/configs/:id/test`
- `GET /api/models/usage`

所有 API 使用既有 `{code,message,data,trace_id}` 响应格式。

## 8. 错误处理、安全与审计

- 外部源不可用时返回真实错误或明确标记的缓存数据，不伪造行情。
- 分析数据过期或覆盖不足时返回可识别的分析状态并降低置信度。
- CSV 以文件哈希和幂等键防止重复导入，失败行不写入正式持仓。
- 模型 API Key 仅后端加密保存，前端只显示掩码；日志不得输出密钥或完整持仓原始文件。
- 候选与 Agent 报告保存输入快照、证据引用、模型版本、提示版本和耗时。
- 每个用户只能访问自己的真实组合、模型配置和分析记录。
- 所有交易按钮继续明确显示“模拟交易”。

## 9. 交付批次与验收

### 第一批：真实持仓与个股工作台

- `portfolio-rpc`、数据库迁移和 REST API。
- 手动持仓、CSV 预览/确认和持仓分析。
- 个股均衡工作台、压力支撑、实时成交和页面内模拟买卖。
- 桌面与手机响应式验收。

### 第二批：市场洞察与候选池

- 指数、大盘宽度、板块、排序选股和龙虎榜。
- `feature-rpc` 首版评分和智能候选池。
- WebSocket 增量评分推送。

### 第三批：新闻事件与知识库

- `news-event-rpc`、公告和合法热点源采集。
- Qdrant 文档构建、实体映射、事件因子和证据检索。

### 第四批：增量因子与实验室

- Redis 滚动状态、因子注册和版本。
- IC、Rank IC、相关性、样本外评估和生命周期界面。

### 第五批：多模型与 Agent

- `model-gateway-rpc`、模型配置、密钥加密、测试、降级和成本审计。
- Agent 使用真实因子、事件、持仓和 RAG 证据生成解释。

### 第六批：高频存储与最终加固

- ClickHouse、MinIO 和高频历史数据迁移。
- 全端响应式完善、容量测试、故障演练和最终文档。

每批必须完成数据库迁移、RPC、gateway、前端、容器配置、接口文档和自动化测试。验收至少包括单元测试、RPC 集成测试、前端构建、桌面与手机浏览器检查、Docker 健康检查和端到端回归。

## 10. 完成定义

功能只有在以下条件同时满足时才算完成：

- 使用真实代码和可执行迁移，不是 README 或伪代码。
- 容器可以从干净数据卷按文档启动。
- 前后端契约一致，错误和空状态可用。
- 数据来源、时间、模式和置信度可见。
- 关键业务有自动化测试和浏览器交互验证。
- 不存在未修复的构建错误、启动错误或已知阻断缺陷。
