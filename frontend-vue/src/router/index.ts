import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { storage } from '@/utils/storage'

const AppLayout = () => import('@/components/layout/AppLayout.vue')

const routes: RouteRecordRaw[] = [
  { path: '/', redirect: '/dashboard' },
  { path: '/login', name: 'login', component: () => import('@/views/auth/Login.vue'), meta: { public: true } },
  { path: '/register', name: 'register', component: () => import('@/views/auth/Register.vue'), meta: { public: true } },
  {
    path: '/',
    component: AppLayout,
    children: [
      { path: 'dashboard', name: 'dashboard', component: () => import('@/views/dashboard/Dashboard.vue'), meta: { title: '首页' } },
      { path: 'market', name: 'market', component: () => import('@/views/market/MarketCenter.vue'), meta: { title: '行情中心' } },
      { path: 'market/crypto', name: 'crypto-spot', component: () => import('@/views/market/CryptoSpot.vue'), meta: { title: '数字资产现货' } },
      { path: 'market/overview', name: 'market-overview', component: () => import('@/views/market/MarketOverview.vue'), meta: { title: '市场全景' } },
      { path: 'market/sectors', name: 'market-sectors', component: () => import('@/views/market/SectorCenter.vue'), meta: { title: '板块强度' } },
	  { path: 'market/sectors/:code', name: 'market-sector-detail', component: () => import('@/views/market/SectorDetail.vue'), meta: { title: '板块详情' } },
      { path: 'market/screener', name: 'market-screener', component: () => import('@/views/market/MarketScreener.vue'), meta: { title: '行情选股器' } },
      { path: 'market/candidates', name: 'candidate-pool', component: () => import('@/views/market/CandidatePool.vue'), meta: { title: '研究候选池' } },
      { path: 'market/dragon-tiger', name: 'dragon-tiger', component: () => import('@/views/market/DragonTiger.vue'), meta: { title: '龙虎榜' } },
      { path: 'market/stocks/:symbol', name: 'stock-detail', component: () => import('@/views/market/StockDetail.vue'), meta: { title: '个股详情' } },
      { path: 'market/timeshare/:symbol', name: 'market-timeshare', component: () => import('@/views/market/Timeshare.vue'), meta: { title: '分时盘口' } },
      { path: 'watchlist', name: 'watchlist', component: () => import('@/views/market/Watchlist.vue'), meta: { title: '自选股' } },
      { path: 'portfolio', name: 'portfolio-list', component: () => import('@/views/portfolio/PortfolioList.vue'), meta: { title: '真实持仓' } },
      { path: 'portfolio/:id', name: 'portfolio-detail', component: () => import('@/views/portfolio/PortfolioDetail.vue'), meta: { title: '持仓管理' } },
      { path: 'portfolio/:id/analysis', name: 'portfolio-analysis', component: () => import('@/views/portfolio/PortfolioAnalysis.vue'), meta: { title: '持仓分析' } },
      { path: 'strategy', name: 'strategy', component: () => import('@/views/strategy/StrategyCenter.vue'), meta: { title: '策略中心' } },
      { path: 'signals', name: 'signals', component: () => import('@/views/strategy/SignalCenter.vue'), meta: { title: '策略信号' } },
      { path: 'backtest', name: 'backtest', component: () => import('@/views/backtest/BacktestCenter.vue'), meta: { title: '回测中心' } },
      { path: 'backtest/:id', name: 'backtest-detail', component: () => import('@/views/backtest/BacktestDetail.vue'), meta: { title: '回测详情' } },
      { path: 'paper/account', name: 'paper-account', component: () => import('@/views/paper/PaperAccount.vue'), meta: { title: '模拟账户' } },
      { path: 'paper/trade', name: 'paper-trade', component: () => import('@/views/paper/PaperTrade.vue'), meta: { title: '模拟下单' } },
      { path: 'paper/orders', name: 'paper-orders', component: () => import('@/views/paper/PaperOrders.vue'), meta: { title: '委托记录' } },
      { path: 'paper/trades', name: 'paper-trades', component: () => import('@/views/paper/PaperTrades.vue'), meta: { title: '成交记录' } },
      { path: 'paper/auto-trader', name: 'auto-trader', component: () => import('@/views/paper/AutoTrader.vue'), meta: { title: '自动策略' } },
      { path: 'risk', name: 'risk', component: () => import('@/views/risk/RiskCenter.vue'), meta: { title: '风控中心' } },
      { path: 'rag/knowledge', name: 'rag-knowledge', component: () => import('@/views/rag/KnowledgeBase.vue'), meta: { title: 'RAG 知识库' } },
      { path: 'rag/ask', name: 'rag-ask', component: () => import('@/views/rag/RagAsk.vue'), meta: { title: 'RAG Ask' } },
      { path: 'agent', name: 'agent', component: () => import('@/views/agent/AgentCenter.vue'), meta: { title: '研究分析' } },
      { path: 'agent/chat', name: 'agent-chat', component: () => import('@/views/agent/AgentChat.vue'), meta: { title: '研究问答' } },
      { path: 'agent/reports', name: 'agent-reports', component: () => import('@/views/agent/AgentArchive.vue'), meta: { title: '分析档案' } },
      { path: 'agent/reports/:id', name: 'agent-report', component: () => import('@/views/agent/AgentReport.vue'), meta: { title: '智能体报告' } },
      { path: 'tasks', name: 'tasks', component: () => import('@/views/task/TaskCenter.vue'), meta: { title: '任务中心' } },
      { path: 'notifications', name: 'notifications', component: () => import('@/views/notify/NotificationCenter.vue'), meta: { title: '通知中心' } },
      { path: 'system/status', name: 'system-status', component: () => import('@/views/system/SystemStatus.vue'), meta: { title: '系统状态' } },
      { path: 'system/datasource', name: 'datasource-status', component: () => import('@/views/system/DataSourceStatus.vue'), meta: { title: '数据源状态' } },
      { path: 'system/models', name: 'model-settings', component: () => import('@/views/system/ModelSettings.vue'), meta: { title: '模型设置' } }
    ]
  },
  { path: '/:pathMatch(.*)*', name: 'not-found', component: () => import('@/views/error/NotFound.vue'), meta: { public: true } }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach((to) => {
  const token = storage.getToken()
  if (!to.meta.public && !token) return `/login?redirect=${encodeURIComponent(to.fullPath)}`
  if ((to.path === '/login' || to.path === '/register') && token) return '/dashboard'
  return true
})

export default router
