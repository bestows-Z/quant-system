# 开源量化系统调研与本项目取舍

截至 2026-10-05。以下以官方仓库源码和文档为依据，不把开源项目的能力直接当作本系统已有能力。

| 项目 | 代码与核心流程 | 页面组织 | AI 用法 | 本项目借鉴 |
| --- | --- | --- | --- | --- |
| [Freqtrade](https://github.com/freqtrade/freqtrade) / [FreqUI](https://github.com/freqtrade/frequi) | 策略接口把指标计算、入场趋势、出场趋势分开；回测使用历史 OHLCV，结果可比较和导出 | 独立的交易、图表、策略与回测工作区，密集数据置于可切换面板 | [FreqAI](https://github.com/freqtrade/freqtrade/blob/develop/freqtrade/freqai/freqai_interface.py) 把特征、训练、预测接进策略流程，模型预测仍受回测和执行规则约束 | 现有回测与策略应继续明确数据来源、执行价、费率及样本外检验；AI 输出不越过风控 |
| [Qlib](https://github.com/microsoft/qlib) | [SigAnaRecord / PortAnaRecord](https://github.com/microsoft/qlib/blob/main/qlib/workflow/record_temp.py) 将信号分析（IC/RankIC）与组合执行分析分层记录 | 以实验记录和分析结果为中心，不是交易终端 | 机器学习模型主要用于特征预测与实验评估 | 增加实验版本、数据快照、指标诊断与可复现记录 |
| [Hummingbot Dashboard](https://github.com/hummingbot/dashboard) | 策略配置、连接器、回测、部署分阶段，交易所规则是下单约束 | 配置、回测、运行监控分开，参数与绩效图表并列 | AI 并非下单必需环节 | 数字资产从行情底座开始，订单和风控应单独建设 |
| [Jesse](https://github.com/jesse-ai/jesse) | 同一套策略关注历史回测与交易执行一致性，图表联动订单和仓位 | 图表、订单、绩效围绕交易对组织 | AI/MCP 更适合作为分析或操作辅助层 | 现货页先建立交易对中心的图表布局，后续才能叠加信号与订单 |

## 当前落地范围

新增的币安现货页面参考交易所的信息层次：交易对切换、价格与 24h 统计、K 线成交量、盘口与过滤器规则，使用石墨色面板与克制的金色强调。行情只访问[币安公开 market-data 域名](https://github.com/binance/binance-spot-api-docs/blob/master/faqs/market_data_only.md)，不接 API Key，也不暗示页面可以交易。币安的 [K 线接口](https://developers.binance.com/en/docs/catalog/core-trading-spot-trading/api/rest-api/market) 与[交易过滤器](https://developers.binance.com/en/docs/products/spot/filters)分别决定图表和订单参数，未来接入模拟订单也必须按价格精度、数量步进及最低名义价值做服务端校验。

## 后续优先级

1. 回测工作台：缩短现有表单，加入策略模板、参数区、运行记录和结果对比；明确训练/验证/样本外窗口，检查未来函数和幸存者偏差。
2. 数字资产模拟交易：独立账本、订单状态机、手续费/滑点/部分成交模型、交易所过滤器校验与可重放事件；保持与真实行情的时间对齐。
3. AI 辅助：用于策略假设解释、参数异常检测、回测报告归纳和研究检索。生成内容须附数据证据和模型版本，不让模型直接决定资金执行。
4. 只有在账户授权、风控、审计、故障恢复、合规边界明确后，再讨论真实交易所账户接入；当前没有实盘下单能力。
