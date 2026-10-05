# API 摘要

所有网关接口统一返回：

```json
{"code":0,"message":"success","data":{},"trace_id":"..."}
```

行情接口保留底层数据源字段：

- `data_mode`: `real` 或 `mock`
- `source`: `tencent`、`sina`、`akshare`、`efinance`、`database`、`cache` 或 `mock`

## 行情

- `GET /api/stocks?page=1&page_size=50&keyword=平安`
- `GET /api/stocks/search?keyword=000001`
- `GET /api/market/realtime?symbols=000001,600519`
- `GET /api/market/realtime/{symbol}`
- `GET /api/market/timeshare/{symbol}`：当日交易分钟价格、均价、分钟成交量、累计量额
- `GET /api/market/orderbook/{symbol}`：最新价及买卖五档
- `GET /api/kline/{symbol}?period=daily&start_date=2026-01-01&end_date=2026-07-04&adjust=qfq`
- `GET /api/indicator/{symbol}?type=macd`
- `GET /api/indicator/full/{symbol}`
- `GET /api/stocks/{symbol}/support-resistance`：枢轴点、摆动高低点、ATR 和成交量价格分布综合压力支撑
- `GET /api/market/overview`：市场涨跌家数、涨跌停、成交额、温度和涨跌分布
- `GET /api/market/sectors?type=industry&sort_by=change_percent&order=desc`：全部可识别行业板块、行情覆盖率、计算口径、涨跌家数、成交额和领涨股票；`type=concept` 在真实 Provider 未配置时返回 `SOURCE_PENDING`，不生成模拟成分
- `GET /api/market/sectors/{code}`：板块详情、广度、强度、覆盖率和行情时间
- `GET /api/market/sectors/{code}/constituents?page=1&page_size=100`：真实成分关系及成分股最新行情
- `GET /api/market/sectors/{code}/timeshare`：板块分钟涨跌幅、覆盖率和计算方法；覆盖不足时明确返回数据不足
- `GET /api/market/screener?sort_by=change_percent&order=desc&limit=50`：按涨跌幅、成交额、换手率或价格排序选股
- `GET /api/market/dragon-tiger?trade_date=2026-07-17`：龙虎榜；真实源未接通时明确返回 `SOURCE_PENDING`

## 智能候选池

以下接口要求 JWT：

- `GET /api/candidates?sort_by=total&industry=&min_confidence=0.5&limit=30`
- `GET /api/candidates/{symbol}/evidence`

候选记录包含技术、资金、情绪、板块、风险五项分数，以及 `reasons`、`negative_factors`、`evidence`、`factor_version`、`scoring_version`、`quote_time` 和 `data_source`。分数用于研究排序，不构成投资建议；数据覆盖不足会降低 `confidence`。

真实源失败时返回 `code != 0`，message 包含数据源不可用原因。前端应显示错误状态，不应白屏。

## 技术指标

`/api/indicator/full/{symbol}` 返回：

- `indicators.ma`: MA5 / MA10 / MA20 / MA60
- `indicators.macd`: DIF / DEA / MACD histogram
- `indicators.rsi`: RSI6 / RSI12 / RSI24
- `indicators.boll`: upper / middle / lower
- `indicators.kdj`: K / D / J

所有指标基于同一段真实日 K 计算。仅显式 mock 模式下会基于 mock K 线计算。

## 真实持仓

真实持仓与模拟账户严格分离，不连接券商：

- `POST /api/portfolios`
- `GET /api/portfolios`
- `GET /api/portfolios/{id}`
- `DELETE /api/portfolios/{id}`
- `GET /api/portfolios/{id}/positions`
- `POST /api/portfolios/{id}/positions`
- `PUT /api/portfolios/{id}/positions/{position_id}`
- `DELETE /api/portfolios/{id}/positions/{position_id}`
- `POST /api/portfolios/{id}/import/preview`：multipart CSV，需 `Idempotency-Key`
- `POST /api/portfolios/{id}/import/confirm`
- `GET /api/portfolios/{id}/analysis`
- `POST /api/portfolios/{id}/analyze`

CSV 必填列为 `symbol/股票代码`、`quantity/持仓数量`、`avg_cost/成本价`。导入先预览校验，确认后在事务中写入；重复确认返回幂等结果。

组合分析返回实时行情覆盖率、数据源、盈亏、集中度、行业暴露和逐票警告。行情缺失的股票不使用虚假价格补齐，并降低分析覆盖率。

`advanced_json` 是版本化的高级分析结果，当前 `portfolio-v2` 包含：

- 历史净值、年化收益、年化波动率、最大回撤、Sharpe、Sortino。
- 历史模拟 VaR 95% 与 CVaR 95%。
- HHI、有效持仓数、逐票风险贡献、逐票收益贡献。
- 相关系数矩阵与历史行情来源、模式、覆盖率。
- 市场普跌、高波动、最大行业下跌、最大持仓下跌四类压力情景。

至少 80% 当前权重具备当日收益时，该交易日才进入组合收益序列；有效观察少于 30 日时会返回置信度警告。

## AI 模型网关

以下接口均要求 JWT，API Key 只在创建或更新时提交，响应永远只返回 `********`：

- `GET /api/models/providers`
- `GET /api/models/configs`
- `POST /api/models/configs`
- `PUT /api/models/configs/{id}`
- `DELETE /api/models/configs/{id}`
- `POST /api/models/configs/{id}/test`
- `POST /api/models/configs/{id}/activate`
- `GET /api/models/usage?days=30`
- `GET /api/models/usage/records?days=30&request_type=&limit=50&offset=0`

支持 OpenAI、DeepSeek、Gemini、豆包、通义千问和 Kimi 的 OpenAI-compatible 接口。配置必须通过连接测试后才能设为当前模型；没有可用模型或模型调用失败时，Agent 降级到本地规则报告引擎，不阻塞行情、策略或风控链路。

调用汇总包含输入 Token、输出 Token、总 Token、成功率、平均延迟和 P95 延迟。调用明细标记 Token 来自模型供应商还是本地估算；Agent 报告生成成功后，调用记录通过 `report_id` 精确关联报告档案。

## Agent 分析档案

以下接口要求 JWT：

- `POST /api/agent/analyze-stock`
- `GET /api/agent/reports?symbol=&report_type=&start_date=&end_date=&limit=50&offset=0`
- `GET /api/agent/reports/{id}`
- `DELETE /api/agent/reports/{id}`：删除当前用户单份报告，模型用量审计继续保留
- `DELETE /api/agent/reports`：请求体 `{"ids":[1,2]}`，批量删除当前用户报告

报告保存模型供应商、模型、输入/输出 Token、延迟、证据来源和证据完整度。未选择个人知识库时自动使用系统量化研究库；没有配置大模型时生成同章节的本地证据报告，不会伪造缺失数据。

候选池默认读取最近一次不可变收盘快照，只有显式刷新才重新计算。交易日 15:30 后生成当日快照，并为当前用户默认模型异步分析前 30 名候选；任务按用户、快照、股票和报告类型幂等。

## AI 研究聊天

以下 REST 接口要求 JWT：

- `POST /api/agent/chat/sessions`：创建持久化研究会话
- `GET /api/agent/chat/sessions`：列出当前用户会话
- `PUT /api/agent/chat/sessions/{id}`：重命名会话
- `DELETE /api/agent/chat/sessions/{id}`：删除会话
- `GET /api/agent/chat/sessions/{id}/messages`：读取消息与证据历史
- `DELETE /api/agent/chat/sessions/{id}/messages`：清空当前会话消息

连接 `GET /ws/agent-chat?token={jwt}` 后发送：

```json
{"type":"chat.ask","session_id":1,"request_id":"客户端唯一编号","content":"结合行情和公告分析主要风险","symbol":"600176"}
```

服务端依次推送 `chat.accepted`、`chat.tool.started`、`chat.tool.completed`、`chat.delta` 和 `chat.completed`。消息、模型用量、延迟、引用及工具调用均会持久化；工具链只读，不提供下单、修改账户或启停自动交易能力。

## 实时推送

连接 `GET /ws/market?token={jwt}` 后订阅单只股票：

```json
{"type":"subscribe","channel":"market","symbol":"600176"}
```

行情更新消息：

```json
{
  "type":"market.quote.updated",
  "channel":"market",
  "data":{"symbol":"600176","price":47.71,"change_pct":-0.0998,"source":"tencent"}
}
```

前端先用 HTTP 获取完整分时和盘口快照，WebSocket 更新最新行情并触发增量刷新；断线时使用 15 秒 HTTP 兜底。

连接建立后也可发送批量订阅：

```json
{"type":"subscribe","channel":"market","symbols":["600176","000001"],"sector_codes":["SEC-..."]}
```

单连接最多订阅 30 只股票和 10 个板块。服务端生产者负责拉取并缓存真实报价、盘口与分时，页面、模拟撮合和自动策略读取同一份带时间戳快照。免费公网 Provider 的目标是秒级研究看盘，不承诺交易所授权 Level-1/Level-2 的毫秒时延。

## 模拟交易 T+1

- A 股买入数量必须为 100 股整数倍。
- 买入成交后增加 `volume` 和 `today_buy_volume`，不增加 `available_volume`。
- 当日卖出超过 `available_volume` 时返回“当日买入不可卖出”。
- 下一个工作日访问持仓或下单时，系统自动将今买数量结转为可用数量。
- 当前交易日历按周末和 2026 年上交所公布的休市安排处理；跨年度生产运行前需要同步对应年度官方日历，不能只按工作日推断。

## 自动策略模拟交易

以下接口要求 JWT，且只操作绑定的模拟账户：

- `GET /api/auto-trader/accounts`
- `POST /api/auto-trader/accounts`
- `PUT /api/auto-trader/accounts/{id}`：仅暂停状态可编辑风控参数
- `DELETE /api/auto-trader/accounts/{id}`：审计保留式归档，不删除历史决策和成交
- `GET /api/auto-trader/accounts/archived`
- `GET /api/auto-trader/accounts/{id}/strategies`
- `POST /api/auto-trader/accounts/{id}/strategies/generate`
- `GET /api/auto-trader/strategies/{id}`
- `POST /api/auto-trader/strategies/{id}/evaluate`
- `POST /api/auto-trader/strategies/{id}/promote`
- `POST /api/auto-trader/accounts/{id}/start`
- `POST /api/auto-trader/accounts/{id}/pause`
- `GET /api/auto-trader/accounts/{id}/runs`
- `GET /api/auto-trader/accounts/{id}/decisions`
- `GET /api/auto-trader/accounts/{id}/deliveries`：逐笔模拟成交交割单
- `GET /api/auto-trader/accounts/{id}/deliveries/export`：UTF-8 CSV 导出

AI 只允许选择白名单策略及参数，生成版本先进入 `SHADOW`。评估接口使用真实历史 K 线调用 backtest-rpc，保存样本股票、交易次数、年化收益、最大回撤、胜率和证据；未通过门槛的版本不能晋级。运行阶段不调用 LLM，而是按分钟读取真实候选、规则信号、风控和不超过 3 秒的五档盘口。当前每个自动账户只有一个执行中的晋级版本；多策略权重组合仍属于后续阶段。

## 回测与研究记录治理

- `DELETE /api/backtest/{id}`：事务删除当前用户回测任务、结果、委托、交易和权益曲线（硬删除，不支持页面恢复）
- `DELETE /api/backtest`：请求体 `{"task_ids":[1,2]}` 批量删除
- `DELETE /api/backtest?scope=failed`：清理当前用户失败记录
- `POST /api/backtest/run`：支持 `adjust=qfq`、佣金率/最低佣金、印花税、过户费、滑点、成交量参与率、目标仓位和买入单位；返回版本、质量报告、成交和委托证据。
- `params_json` 必须是对应策略的白名单数值对象；未知键、越界值、非整数周期或快慢周期颠倒会在创建任务前被拒绝，不会悄悄退回默认值。
- `GET /api/backtest/{id}/orders`：查询当前用户回测委托，包括未成交与取消原因。
- `GET /api/backtest/{id}/result`：包含 `engine_version`、`execution_model`、`quality_status`、`quality_json`、费用拆分和盈亏指标。旧记录缺少这些证据时，页面明确标记为旧版结果。
- `DELETE /api/risk/records/{id}`：删除当前用户单条风控记录
- `DELETE /api/risk/records`：请求体 `{"ids":[1,2]}` 批量删除
- `DELETE /api/strategies/{id}`：软删除当前用户创建的策略；系统策略不可删除

成功回测必须保存可审计的真实 `data_source`、`data_mode=real` 和实际参数。系统不会把 mock、bootstrap、fixture 或 synthetic 记录展示为真实回测。历史涨跌停和证券状态缺乏逐日可信证据时，结果标为 `PARTIAL`，不可用于自动策略晋级。

## RAG 内容生命周期

- `DELETE /api/rag/documents/{id}`：删除当前用户文档、切块、向量映射；Qdrant 暂时不可达时写入清理任务重试
- `DELETE /api/rag/knowledge-bases/{id}`：级联删除当前用户知识库内容

系统公共知识库不可由普通用户删除。删除后检索会立即排除软删除内容，不等待异步向量清理完成。

通知中心支持 `DELETE /api/notifications/{id}` 删除单条，以及 `DELETE /api/notifications?scope=read|all` 清理已读或全部通知；重复删除按幂等成功处理。
