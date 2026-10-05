# Agent 量化研究、候选池与分析档案设计

## 1. 目标

将现有单次个股 Agent 调用升级为可复用的企业级研究流水线。系统在未配置大模型时生成完整量化报告；配置模型后，在量化证据、公告资讯、知识库检索和风险结果之上生成可追溯的模型报告。所有输出仅供研究，不构成投资建议。

首批覆盖个股、候选池、真实持仓、模拟账户、策略信号、回测和风险分析。候选池每个交易日收盘后更新，全市场先运行确定性量化评分，只对前 30 名自动运行模型深度分析，其余候选保留量化报告并支持按需补充模型分析。

## 2. 方案选择

采用“量化全覆盖 + Agent 分层增强”的混合方案：

- 不采用所有页面同步调用模型。该方案延迟高、成本不可控，且页面刷新容易重复调用。
- 不采用全市场每日模型分析。该方案会产生大量无价值 Token 消耗，免费数据源也无法保证相同时间点的全量证据完整性。
- 采用收盘后异步批处理与用户按需分析结合。确定性计算负责全市场，模型负责前 30 名和用户明确请求。

## 3. 服务边界

### task-rpc

- 注册交易日 `15:30 Asia/Shanghai` 的候选池刷新任务。
- 生成任务幂等键 `candidate-close:{trade_date}`。
- 发布候选刷新事件，并跟踪量化、Agent、归档三个阶段状态。
- 非交易日不执行；失败允许人工重试，但同一交易日只能存在一个成功快照。

### feature-rpc

- 从统一收盘行情快照生成全市场量化评分。
- 保存候选快照、因子版本、行情来源、证据摘要和数据覆盖率。
- 返回前 30 名给 Agent 流水线。
- 盘中页面只读取最近一个成功收盘快照，不因页面刷新重新计算全市场。

### agent-rpc

- 支持 `STOCK`、`CANDIDATE`、`PORTFOLIO`、`PAPER_ACCOUNT`、`SIGNAL`、`BACKTEST`、`RISK` 七类报告。
- 无模型配置时由本地量化报告构建器生成详细报告，不返回寥寥数语。
- 有模型配置时调用 model-gateway-rpc，并将量化结果、RAG 片段、资讯事件和数据源清单作为证据。
- 自动选择系统研究知识库，并在用户指定时叠加用户知识库。
- 每份报告保存原始上下文摘要、模型原始输出、引用、数据源、模型、Token、延迟、量化版本和证据完整度。

### rag-rpc

- 增加系统级 `quant-system-research` 知识库，普通用户只读。
- 收录指标公式、策略说明、风险规则、A 股交易规则、公告和资讯事件摘要。
- 公告/资讯按股票代码、行业、发布日期和事件类型写入 metadata。
- 检索使用 Qdrant 向量召回与 MySQL 关键词召回合并，去重后返回前 5 至 8 条证据。
- Agent 未显式传知识库时自动检索系统库，避免 RAG 永久处于跳过状态。

### model-gateway-rpc

- 持久化每次连接测试与生成调用。
- 记录输入 Token、输出 Token、总 Token、首字延迟或总延迟、状态、请求类型、模型配置和 trace_id。
- Provider 返回 usage 时使用真实值；未返回时使用本地估算并标记 `token_source=ESTIMATED`。
- 汇总接口返回调用次数、成功率、失败数、输入/输出/总 Token、平均延迟和 P95 延迟。
- 增加按日期、请求类型、模型分页查询调用明细。
- `max_tokens` 仅表示最大输出长度；模型上下文容量单独维护。DeepSeek v4 pro 可配置 1M 上下文，但系统仍会将原始时间序列转换为统计摘要、关键拐点和最新窗口，避免噪声淹没有效证据。

### market-rpc / quant-python

- 资金流使用真实多源适配器：东方财富主源、新浪备用源、配置 Token 后使用 Tushare 增强源。
- Redis 保存最近一次成功结果、来源和采集时间。
- 新鲜数据失败时允许返回最近成功的真实缓存并明确标记 `stale-cache`。
- 所有源均失败且没有缓存时返回 `SOURCE_UNAVAILABLE`，不根据盘口或成交量推算并冒充真实资金流。

## 4. 数据模型

### agent_analysis_reports 扩展

- `analysis_scope`：报告类型。
- `snapshot_no`：关联候选或市场快照。
- `model_config_id`、`provider_code`、`model`。
- `prompt_tokens`、`completion_tokens`、`total_tokens`、`latency_ms`。
- `quant_version`、`factor_version`、`knowledge_base_ids_json`。
- `data_sources_json`、`evidence_completeness`、`token_source`。
- 索引：`(user_id, created_at DESC)`、`(user_id, symbol, created_at DESC)`、`(snapshot_no, symbol)`。

### candidate_agent_reports

- 唯一约束 `(snapshot_no, symbol, report_type)`。
- 保存量化报告 ID、Agent 报告 ID、状态、失败原因和重试次数。

### model_usage_records 扩展

- `prompt_tokens`、`completion_tokens` 已存在并继续使用。
- 新增 `token_source`、`first_token_latency_ms`、`report_id`。
- 索引：`(user_id, created_at DESC)`、`(user_id, request_type, created_at DESC)`。

## 5. 报告结构

每份报告至少包含：

1. 数据覆盖与来源：行情时间、行情源、历史区间、指标版本、资讯与知识库引用数量。
2. 行情与量价：趋势、收益、波动、成交量、换手率、量比、支撑压力。
3. 技术指标：MA、MACD、RSI、KDJ、BOLL、ATR 等可用指标，不可用项明确说明。
4. 策略与回测：策略信号、形成原因、样本区间、收益、回撤和胜率。
5. 消息面与知识库：公告和资讯事件、检索片段、发布时间与引用链接。
6. 风险：波动、趋势、集中度、流动性、回撤和数据缺失风险。
7. 综合研判：证据一致性、冲突点和需要继续观察的数据，不输出确定性建议。

本地量化报告与模型报告使用同一结构。模型不可用时只缺少自然语言增强，不缺少量化章节。

## 6. API 与前端

- `GET /api/agent/reports`：按股票、报告类型、日期和状态分页查询档案。
- `GET /api/agent/reports/:id`：完整报告与证据元数据。
- `POST /api/candidates/refresh`：管理员手工触发收盘候选任务。
- `GET /api/candidates/snapshots/latest`：最近成功快照及更新时间。
- `POST /api/candidates/:symbol/analyze`：按需运行候选 Agent。
- `GET /api/models/usage`：增强汇总。
- `GET /api/models/usage/records`：调用明细。

前端新增“分析档案”和“模型调用记录”。候选池显示收盘快照时间、量化评分、Agent 状态、摘要、证据完整度和报告入口。主要分析页面统一提供“量化报告”和“Agent 增强”状态，不配置模型时明确显示“使用本地量化引擎”。

## 7. 错误处理与安全

- 模型失败不丢弃量化报告，Agent 状态标记为降级并保存失败原因。
- RAG 或资金流失败只降低证据完整度，不伪造数据。
- API Key 继续加密保存，日志禁止记录请求密钥、完整 Prompt 和模型原始响应。
- 模型输出经过 JSON 结构校验、非有限数值清洗和确定性建议用语过滤。
- 批处理限制并发、按用户和模型配置设置日 Token 预算。

## 8. 验收

- 收盘任务可幂等生成候选快照，并只对前 30 名创建 Agent 子任务。
- 配置模型账户能看到调用明细和非零 Token/延迟；未配置账户能看到完整本地量化报告。
- 个股与候选报告可检索、可打开、可追溯数据源和知识库引用。
- 系统知识库存在文档、切块、向量和检索日志，Agent 报告至少能引用可用的系统研究资料。
- 资金流至少一个真实源或真实缓存可用；不可用时状态和采集时间清晰。
- Go、Python、Vue 构建与测试通过，Docker 全量启动，桌面与 390px 页面无布局溢出。
