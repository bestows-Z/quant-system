# 量化工作台前端

Vue 3 + TypeScript + Vite + Pinia + Element Plus 前端后台，接入 `gateway-api`、`ws-api`、任务中心、通知中心、RAG 和量化智能体。

## 功能

- 登录 / 注册 / JWT 路由守卫
- Dashboard、行情中心、个股详情、K线图、技术指标、自选股
- 策略中心、策略信号、回测中心、回测详情
- 模拟账户、模拟下单、委托记录、成交记录
- 风控中心、RAG 知识库、RAG Ask、智能体分析
- 任务中心、通知中心、WebSocket 实时状态
- 系统状态、数据源状态

所有策略、Agent 和模拟交易页面均使用“仅供研究，不构成投资建议”文案。

## 环境变量

```bash
cp .env.example .env
```

默认值：

```bash
VITE_API_BASE_URL=http://127.0.0.1:8080
VITE_WS_BASE_URL=ws://127.0.0.1:8081
```

## 本地启动

```bash
npm install
npm run dev
```

访问 `http://127.0.0.1:5173`。

## 构建

```bash
npm run build
npm run preview
```

## Docker

```bash
docker build -t quant-system-frontend .
docker run --rm -p 8080:80 quant-system-frontend
```

`nginx.conf` 已支持 SPA history fallback、`/api` 反向代理到 `gateway-api:8888`、`/ws` 反向代理到 `ws-api:8081`。

## 接口封装

`src/api/request.ts` 统一处理 `baseURL`、JWT、响应拦截、`code != 0` 错误提示、`401` 跳转登录、`trace_id` 辅助提示、网络错误和超时。

## 当前限制

- 系统健康接口后端暂未全部开放时，系统状态页会显示明确的 `UNKNOWN` 占位。
- 行情源不可用时，页面显示接口错误提示，不伪造真实行情。
- WebSocket 默认连接 `/ws/notify`，业务页面可按频道调用订阅。
