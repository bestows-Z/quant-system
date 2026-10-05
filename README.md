# Quant System

量化研究与策略执行工作台。系统使用真实行情、可追溯回测和风控规则；当前版本的账户执行仅连接模拟账户，尚未接入券商实盘接口。

## 当前能力

- go-zero API Gateway 与模块化 RPC/WS 微服务：用户、行情、市场特征、策略、回测、模拟交易、真实持仓、模型网关、风控、RAG、Agent、任务、通知、WebSocket。
- Python FastAPI 行情量化服务：多数据源、日 K、MA/MACD/KDJ/RSI/BOLL、策略与回测。
- 回测 v2：前复权日线、收盘信号次日开盘执行、A 股交易约束、费用拆分、数据质量和未成交委托证据。
- MySQL、Redis、RabbitMQ、etcd、Qdrant 与 Vue 3 管理端。
- 腾讯/新浪真实实时行情；efinance、BaoStock、EastMoney、AKShare fallback；显式 local CSV bootstrap。
- 自动化 smoke 与完整端到端验收。
- 大盘广度、板块强度、涨跌幅排序选股、可解释智能候选池和真实源龙虎榜契约。
- 数字资产现货行情终端：接入币安公开市场数据，支持 BTC/USDT、ETH/USDT、BNB/USDT 的 1h/4h/1d 已收盘 K 线、24h 行情、盘口最优价与交易规则；页面路径 `/market/crypto`。这一版是只读真实行情，不连接币安账户、不执行真实或模拟订单。

## 环境准备

```bash
cd /Users/Admin/develop/thesis/quant-system
cp .env.example .env
```

至少修改 `.env` 中的 `JWT_SECRET`、MySQL 密码。若宿主机使用 Clash Fake-IP/TUN，在本地 `.env` 配置：

```dotenv
HTTP_PROXY=http://host.docker.internal:7897
HTTPS_PROXY=http://host.docker.internal:7897
NO_PROXY=localhost,127.0.0.1,mysql,redis,etcd,rabbitmq,qdrant,quant-python,gateway-api,user-rpc,market-rpc,feature-rpc,strategy-rpc,backtest-rpc,paper-rpc,portfolio-rpc,model-gateway-rpc,risk-rpc,rag-rpc,agent-rpc,task-rpc,notify-rpc,ws-api
```

代理地址不写死在代码中。Compose 会显式读取项目根目录 `.env`。

## 启动与停止

已有 MySQL 数据卷升级到回测 v2 前，先备份数据库，再执行 `backend-go/migrations/015_backtest_integrity_v2.sql`。Compose 的初始化 SQL 只会在全新数据卷首次启动时执行，不会自动修改已有数据库。

```bash
docker compose -f deploy/docker-compose.yml up -d --build
docker compose -f deploy/docker-compose.yml ps
```

停止：

```bash
docker compose -f deploy/docker-compose.yml down
```

查看日志：

```bash
docker compose -f deploy/docker-compose.yml logs -f --tail=200 gateway-api market-rpc quant-python frontend-vue
```

访问地址：

- 前端：http://127.0.0.1:8088
- Gateway API：http://127.0.0.1:8080
- Python API 文档：http://127.0.0.1:8000/docs（仅绑定本机回环地址；对外访问统一通过网关）
- WebSocket：ws://127.0.0.1:8081/ws/market
- RabbitMQ 管理台：http://127.0.0.1:15672
- Qdrant：http://127.0.0.1:6333

## 验收测试

```bash
bash scripts/smoke-test.sh
bash scripts/full-acceptance-test.sh
```

脚本会动态注册一次性 `quant_test_<timestamp>` 用户，不依赖固定账号。完整验收覆盖认证、行情、市场洞察、智能候选池、自选股、非零策略信号、回测、模拟交易、风控、RAG、Agent、模型供应商、任务、通知与 WebSocket。

分别验证工程构建：

```bash
cd quant-python
PYTHONPYCACHEPREFIX=.pycache python3 -m compileall app
PYTHONPYCACHEPREFIX=.pycache python3 -m unittest discover -s tests -v

cd ../backend-go
GOCACHE=$PWD/.gocache go mod tidy
GOCACHE=$PWD/.gocache go test ./...
GOCACHE=$PWD/.gocache go build ./...

cd ../frontend-vue
npm install
npm run build
```

## 数据来源说明

- `data_mode=real, source=tencent/sina/...`：真实公网行情。
- `source=cache/database/stale-cache`：Redis/MySQL 缓存或数据库数据。
- `data_mode=bootstrap, source=local_csv`：明确标注的本地初始化历史数据，不是实时行情，也不是 mock。
- `DATA_MODE=mock`：仅显式开发调试时启用；real 模式不会静默返回 mock。
- 数字资产行情使用 Binance Spot 的 `data-api.binance.vision` 公开接口，无需 API Key；若交易所或网络不可用，返回明确错误，不回退为虚构价格。K 线仅使用已收盘周期，显示时间为 UTC。当前仅支持现货，不含合约或杠杆。
- RAG 默认使用 384 维本地特征哈希 embedding，并以 Qdrant 为主检索、MySQL 为显式灾备检索。
- Agent 未配置可用模型时使用本地规则化报告引擎。登录后进入“系统状态 → AI 模型设置”，可配置 OpenAI、DeepSeek、Gemini、豆包、通义千问和 Kimi；连接测试成功并设为当前模型后，Agent 自动通过 model-gateway-rpc 调用该模型。
- `.env` 中必须设置至少 16 字符的 `MODEL_ENCRYPTION_KEY`；生产环境应使用随机高强度值并保持稳定，否则已加密的 API Key 将无法解密。

详细接口与架构见 [docs/api.md](docs/api.md) 和 [docs/architecture.md](docs/architecture.md)。

## 下一步演进

数字资产交易执行需要独立的资产账本、订单状态机、交易所过滤器校验、限频、幂等、断线恢复和风控开关，不能直接复用 A 股模拟订单逻辑。当前先提供可核验的真实行情底座；开源系统的代码、页面与 AI 使用方式对照见 [docs/open-source-quant-review.md](docs/open-source-quant-review.md)。
