# Database Design

数据库使用 MySQL 8。基础初始化位于 `deploy/init.sql` 和 `backend-go/migrations/001_init.sql`；后续增量迁移按编号保存在 `backend-go/migrations/`，Docker 干净数据卷会依次执行挂载的迁移。

核心表：
- 用户权限：`users`、`roles`、`user_roles`
- 股票与行情：`stocks`、`stock_daily_bars`、`stock_minute_bars`、`realtime_quotes`
- 指标与自选：`indicators`、`watchlists`
- 策略：`strategies`、`strategy_signals`
- 回测：`backtest_tasks`、`backtest_results`、`backtest_trades`、`backtest_equity_curve`
- 模拟交易：`paper_accounts`、`paper_positions`、`paper_orders`、`paper_trades`、`paper_equity_curve`
- 风控与系统：`risk_records`、`collect_tasks`、`system_configs`
- 真实持仓：`real_portfolios`、`real_positions`、`position_import_tasks`、`portfolio_snapshots`、`portfolio_analysis_records`
- 个股分析：`support_resistance_levels`
- 模型网关：`model_providers`、`model_configs`、`model_usage_records`
- Agent 研究：`agent_analysis_reports`、`candidate_agent_reports`、`knowledge_bases`、`rag_documents`、`rag_chunks`、`rag_embeddings`
- 板块研究：`sector_minute_snapshots`（板块主数据与成分关系沿用 `market_sectors`、`sector_constituents`）
- 内容清理：`rag_vector_cleanup_tasks`
- 自动模拟交易：`auto_trade_accounts`、`auto_strategy_versions`、`auto_strategy_evaluations`、`auto_trade_runs`、`auto_trade_decisions`、`auto_trade_scheduler_state`

约束原则：
- 股票基础信息使用 `symbol + market` 唯一约束。
- 日 K 使用 `symbol + trade_date` 唯一约束。
- 分钟 K 使用 `symbol + period + trade_time` 唯一约束。
- 自选股使用 `user_id + symbol + group_name` 唯一约束。
- 订单、成交、回测、风控均按账户、用户、标的和时间建立查询索引。
- 真实组合和持仓使用 `active_name`、`active_symbol` 生成列实现软删除条件下的有效记录唯一约束。
- CSV 导入使用 `user_id + portfolio_id + idempotency_key` 唯一约束。
- 压力支撑使用 `symbol + period + quote_time + algorithm_version` 唯一约束，保留方法、来源和有效期。
- 模型配置使用 `user_id + active_name` 唯一约束，软删除后允许复用名称；仅通过测试的启用配置可设为当前模型。
- `encrypted_api_key` 只保存 AES-GCM 密文，`model_usage_records` 记录调用审计但不保存请求正文或密钥。
- `model_usage_records.report_id` 关联最终分析档案；`token_source` 区分供应商精确统计、估算和本地规则引擎。
- `agent_analysis_reports` 保存结构化章节、模型信息、Token、延迟、证据来源和完整度；原始上下文仅用于受控审计，不在普通前端展示。
- `candidate_agent_reports` 使用 `snapshot_no + symbol + report_type + user_id` 唯一约束，避免收盘任务重启后重复调用模型。
- `portfolio_analysis_records.advanced_json` 保存版本化风险指标、净值序列、归因、相关矩阵与压力情景，便于审计和算法升级。
- 板块分钟快照使用 `sector_code + snapshot_time` 唯一约束，按交易时间查询；保存覆盖率、来源和数据模式，避免把不完整成分行情包装为精确指数。
- `rag_vector_cleanup_tasks` 对 `document_id + status` 建索引；数据库删除先完成，Qdrant 失败任务由调度器重试，检索查询始终过滤软删除内容。
- `auto_strategy_evaluations` 按 `strategy_version_id + created_at` 查询，保存门禁结论和完整证据 JSON；策略晋级必须读取最近一次通过的评估。
- `auto_trade_scheduler_state` 使用账户与任务类型唯一键记录最后执行日/分钟，保证收盘评估和盘中调度幂等。
- `auto_trade_accounts.max_total_position_ratio` 与单票、单日亏损、最大持股数共同构成账户级风险预算；模拟订单仍由 paper-rpc 再次检查现金、T+1、冻结数量和盘口。

增量迁移：

- `010_depth_matching_and_auto_trader.sql`：五档部分成交、剩余数量、自动账户/策略/运行/决策表。
- `011_sector_realtime_research.sql`：板块分钟快照。
- `012_rag_vector_cleanup.sql`：Qdrant 清理补偿任务。
- `013_auto_strategy_governance.sql`：组合总仓位、真实回测评估和调度幂等状态。
