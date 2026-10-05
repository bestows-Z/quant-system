<template>
  <PageContainer :title="`${stock?.name || quote?.name || symbol} 个股详情`" description="行情、K线、指标、策略信号与风险分析">
    <template #actions>
      <el-select v-model="range" style="width: 120px" @change="loadMarketData">
        <el-option label="近1月" value="1m" />
        <el-option label="近3月" value="3m" />
        <el-option label="近6月" value="6m" />
        <el-option label="近1年" value="1y" />
        <el-option label="近3年" value="3y" />
      </el-select>
      <el-switch v-model="autoRefresh" active-text="自动刷新" />
      <el-button :loading="loading" @click="load">刷新</el-button>
      <el-button @click="router.push(`/market/timeshare/${symbol}`)">分时盘口</el-button>
      <el-button @click="analyzeRisk">风险分析</el-button>
      <el-button type="primary" :loading="agentRunning" @click="runAgent">智能分析</el-button>
      <el-button type="danger" @click="openOrder('BUY')">模拟买入</el-button>
      <el-button type="success" @click="openOrder('SELL')">模拟卖出</el-button>
    </template>

    <el-alert v-if="errorMessage" :title="errorMessage" type="error" show-icon class="status-alert" />
    <el-alert v-if="dataMode === 'mock'" title="当前为 mock 数据，仅用于离线开发调试" type="warning" show-icon class="status-alert" />
    <el-alert v-if="sourceTip" :title="sourceTip" :type="sourceTipType" show-icon class="status-alert" />

    <div class="section-grid quote-overview">
      <DataCard class="span-3" label="当前价格" :value="money(quote?.price ?? quote?.current)" />
      <DataCard class="span-3" label="涨跌幅"><PriceChange :value="quotePct" percent-mode /></DataCard>
      <DataCard class="span-3" label="成交量" :value="compactNumber(quote?.volume)" />
      <DataCard class="span-3" label="成交额" :value="compactNumber(quote?.amount)" />
      <DataCard class="span-3" label="换手率" :value="percent(quote?.turnover_rate)" sub="当日成交占流通股本比例" />
      <DataCard class="span-3" label="量比" :value="metric(quote?.volume_ratio)" sub="当前成交活跃度与近期均值之比" />
      <DataCard class="span-3" label="昨收" :value="money(quote?.pre_close)" />
      <DataCard class="span-3" label="市盈率（动态）" :value="metric(quote?.pe_ratio)" sub="亏损或数据源未提供时显示暂无" />
      <DataCard class="span-3 limit-up-card" label="涨停价" :value="money(quote?.limit_up)" />
      <DataCard class="span-3 limit-down-card" label="跌停价" :value="money(quote?.limit_down)" />
      <DataCard class="span-3" label="当日最高" :value="money(quote?.high)" />
      <DataCard class="span-3" label="当日最低" :value="money(quote?.low)" />
    </div>

    <el-tabs v-model="workspaceTab" class="workbench-tabs">
      <el-tab-pane label="实时行情" name="market">
        <div class="section-grid">
          <section class="data-table-card span-8"><div class="toolbar"><strong>实时分时 / 成交量</strong><el-tag size="small">{{ timeshare.length }} 分钟</el-tag></div><TimeshareChart :data="timeshare" :pre-close="quote?.pre_close" :loading="loading" /></section>
          <section class="data-table-card span-4"><div class="toolbar"><strong>五档盘口</strong><el-tag size="small" type="info">{{ orderBook?.source || '--' }}</el-tag></div><OrderBook :book="orderBook" /></section>
        </div>
        <div class="section-grid analysis-band">
          <section class="data-table-card span-7"><div class="toolbar"><strong>压力支撑</strong><el-tag>{{ supportResistance?.algorithm_version || '--' }}</el-tag></div><div class="level-grid mono"><div><span>压力位</span><strong class="up">{{ money(supportResistance?.resistance_price) }}</strong><small>强度 {{ percentage(supportResistance?.resistance_strength) }}</small></div><div><span>枢轴位</span><strong>{{ money(supportResistance?.pivot_price) }}</strong><small>ATR {{ money(supportResistance?.atr14) }}</small></div><div><span>支撑位</span><strong class="down">{{ money(supportResistance?.support_price) }}</strong><small>强度 {{ percentage(supportResistance?.support_strength) }}</small></div></div></section>
          <section class="data-table-card span-5"><div class="toolbar"><strong>规则研判</strong><el-tag type="info">{{ supportResistance?.data_source || '--' }}</el-tag></div><p class="analysis-copy">{{ structuralSummary }}</p><div class="quick-actions"><el-button type="danger" @click="openOrder('BUY')">模拟买入</el-button><el-button type="success" @click="openOrder('SELL')">模拟卖出</el-button></div></section>
        </div>
        <section class="data-table-card capital-panel">
          <div class="toolbar"><strong>资金流向</strong><el-tag size="small" type="info">东方财富统计口径</el-tag></div>
          <div v-if="latestFlow" class="capital-summary">
            <div><span>主力净流入</span><strong :class="flowClass(latestFlow.main_net_inflow)">{{ compactNumber(latestFlow.main_net_inflow) }}</strong><small>{{ percent(latestFlow.main_net_ratio) }}</small></div>
            <div><span>超大单净流入</span><strong :class="flowClass(latestFlow.super_large_net_inflow)">{{ compactNumber(latestFlow.super_large_net_inflow) }}</strong><small>{{ percent(latestFlow.super_large_net_ratio) }}</small></div>
            <div><span>大单净流入</span><strong :class="flowClass(latestFlow.large_net_inflow)">{{ compactNumber(latestFlow.large_net_inflow) }}</strong><small>{{ percent(latestFlow.large_net_ratio) }}</small></div>
            <div><span>散户净流入</span><strong :class="flowClass(latestFlow.retail_net_inflow)">{{ compactNumber(latestFlow.retail_net_inflow) }}</strong><small>{{ percent(latestFlow.retail_net_ratio) }}</small></div>
          </div>
          <el-table :data="moneyFlow" max-height="300" empty-text="真实资金流数据暂不可用">
            <el-table-column prop="trade_date" label="交易日" width="110" />
            <el-table-column label="主力净流入" min-width="130"><template #default="{row}"><span :class="flowClass(row.main_net_inflow)">{{compactNumber(row.main_net_inflow)}}</span></template></el-table-column>
            <el-table-column label="超大单" min-width="120"><template #default="{row}">{{compactNumber(row.super_large_net_inflow)}}</template></el-table-column>
            <el-table-column label="大单" min-width="120"><template #default="{row}">{{compactNumber(row.large_net_inflow)}}</template></el-table-column>
            <el-table-column label="中单" min-width="120"><template #default="{row}">{{compactNumber(row.medium_net_inflow)}}</template></el-table-column>
            <el-table-column label="散户" min-width="120"><template #default="{row}">{{compactNumber(row.retail_net_inflow)}}</template></el-table-column>
          </el-table>
        </section>
      </el-tab-pane>
      <el-tab-pane label="K线与指标" name="technical">
        <section class="data-table-card"><div class="toolbar"><strong>K线 / 成交量 / MA / BOLL</strong><el-tag>{{ klines.length }} 条</el-tag></div><KLineChart :data="klines" :ma="indicators.ma" :boll="indicators.boll" :loading="loading" /></section>
        <section class="data-table-card indicator-card"><div class="toolbar"><strong>技术指标</strong><el-radio-group v-model="activeIndicator" size="small"><el-radio-button value="MACD">MACD</el-radio-button><el-radio-button value="RSI">RSI</el-radio-button><el-radio-button value="KDJ">KDJ</el-radio-button></el-radio-group></div><MACDChart v-if="activeIndicator === 'MACD'" :data="indicators.macd" :loading="loading" /><RSIChart v-else-if="activeIndicator === 'RSI'" :data="indicators.rsi" :loading="loading" /><el-table v-else :data="indicators.kdj" height="260" empty-text="暂无 KDJ 数据"><el-table-column prop="trade_date" label="日期" /><el-table-column prop="k" label="K" /><el-table-column prop="d" label="D" /><el-table-column prop="j" label="J" /></el-table></section>
      </el-tab-pane>
      <el-tab-pane label="策略研判" name="signals"><section class="data-table-card"><div class="toolbar"><strong>策略信号</strong><span class="muted">没有历史信号时，可运行默认趋势评分策略</span><el-button type="primary" :loading="strategyRunning" @click="runDefaultStrategy">立即运行策略分析</el-button></div><el-table :data="signals" max-height="560" empty-text="暂无策略信号，请点击“立即运行策略分析”"><el-table-column label="信号" width="90"><template #default="{ row }"><SignalTag :value="row.signal_type || row.action" /></template></el-table-column><el-table-column label="风险" width="110"><template #default="{ row }"><RiskTag :value="row.risk_level" /></template></el-table-column><el-table-column label="强度" width="110"><template #default="{row}">{{Number(row.score||0).toFixed(0)}} / 100</template></el-table-column><el-table-column prop="reason" label="形成原因" min-width="260" /></el-table></section></el-tab-pane>
      <el-tab-pane label="公司与资讯" name="research">
        <section class="data-table-card company-panel">
          <div class="toolbar"><strong>公司资料</strong><el-tag size="small">{{ company?.source || '--' }}</el-tag></div>
          <el-descriptions :column="2" border>
            <el-descriptions-item label="公司全称">{{ company?.company_name || '--' }}</el-descriptions-item>
            <el-descriptions-item label="所属行业">{{ company?.industry || stock?.industry || '--' }}</el-descriptions-item>
            <el-descriptions-item label="上市日期">{{ company?.listing_date || stock?.list_date || '--' }}</el-descriptions-item>
            <el-descriptions-item label="实际控制人">{{ company?.actual_controller || '--' }}</el-descriptions-item>
            <el-descriptions-item label="董事长">{{ company?.chairman || '--' }}</el-descriptions-item>
            <el-descriptions-item label="员工人数">{{ compactNumber(company?.employees) }}</el-descriptions-item>
            <el-descriptions-item label="主营业务" :span="2">{{ company?.main_business || '--' }}</el-descriptions-item>
            <el-descriptions-item label="公司地址" :span="2">{{ company?.address || '--' }}</el-descriptions-item>
          </el-descriptions>
          <p class="company-profile">{{ company?.company_profile || '真实公司简介暂不可用。' }}</p>
        </section>
        <div class="research-grid">
          <section class="data-table-card"><div class="toolbar"><strong>最新公告</strong><el-tag size="small">{{ announcements.length }} 条</el-tag></div><div class="research-list"><a v-for="item in announcements" :key="item.id" :href="item.url" target="_blank" rel="noopener noreferrer"><strong>{{item.title}}</strong><span>{{item.category || '公司公告'}} · {{item.published_at}}</span></a><el-empty v-if="!announcements.length" description="暂无可验证公告" /></div></section>
          <section class="data-table-card"><div class="toolbar"><strong>财经资讯</strong><el-tag size="small">{{ news.length }} 条</el-tag></div><div class="research-list"><a v-for="item in news" :key="item.id" :href="item.url" target="_blank" rel="noopener noreferrer"><strong>{{item.title}}</strong><span>{{item.media || '财经媒体'}} · {{item.published_at}}</span><small>{{item.summary}}</small></a><el-empty v-if="!news.length" description="暂无相关财经资讯" /></div></section>
        </div>
      </el-tab-pane>
      <el-tab-pane label="智能分析" name="agent">
        <section class="data-table-card agent-panel">
          <div class="toolbar">
            <div class="agent-heading"><strong>量化智能体报告</strong><span class="muted">综合行情、技术指标、策略信号与风险数据</span></div>
            <div class="agent-toolbar-actions">
              <RiskTag v-if="agentReport?.risk_level" :value="String(agentReport.risk_level)" />
              <el-tag v-if="agentReport?.confidence != null">置信度 {{ agentConfidence }}</el-tag>
              <el-button @click="router.push({ path: '/agent', query: { symbol } })">打开分析中心</el-button>
              <el-button type="primary" :loading="agentRunning" @click="runAgent">{{ agentReport ? '重新分析' : '开始分析' }}</el-button>
            </div>
          </div>
          <div v-if="agentReport" class="agent-report-grid">
            <article v-for="item in agentSections" :key="item.label" class="agent-report-block">
              <span>{{ item.label }}</span>
              <p>{{ item.value }}</p>
            </article>
          </div>
          <div v-else class="empty-action">
            <strong>尚未生成这只股票的智能分析报告</strong>
            <p class="muted">点击“开始分析”，系统会调用真实行情、指标、策略、回测、风控和当前启用的 AI 模型。</p>
            <el-button type="primary" :loading="agentRunning" @click="runAgent">开始分析</el-button>
          </div>
          <div v-if="agentReportId" class="report-link-row">
            <el-button type="primary" plain @click="router.push(`/agent/reports/${agentReportId}`)">查看完整报告页面</el-button>
          </div>
        </section>
      </el-tab-pane>
      <el-tab-pane label="持仓影响" name="portfolio"><section class="data-table-card empty-action"><strong>从真实持仓评估该股票对组合的影响</strong><p class="muted">进入持仓分析后可查看单票集中度、行业暴露、盈亏和风险警告。</p><el-button type="primary" @click="router.push('/portfolio')">打开真实持仓</el-button></section></el-tab-pane>
    </el-tabs>
    <DisclaimerAlert />
    <QuickPaperOrder v-model="orderVisible" :symbol="symbol" :name="stock?.name || quote?.name" :price="Number(quote?.price ?? quote?.current ?? 0)" :initial-side="orderSide" :order-book="orderBook" :quote-time="quote?.quote_time" :data-mode="quote?.data_mode" />
  </PageContainer>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import dayjs from 'dayjs'
import PageContainer from '@/components/common/PageContainer.vue'
import DataCard from '@/components/common/DataCard.vue'
import PriceChange from '@/components/common/PriceChange.vue'
import SignalTag from '@/components/common/SignalTag.vue'
import RiskTag from '@/components/common/RiskTag.vue'
import DisclaimerAlert from '@/components/common/DisclaimerAlert.vue'
import KLineChart from '@/components/charts/KLineChart.vue'
import MACDChart from '@/components/charts/MACDChart.vue'
import RSIChart from '@/components/charts/RSIChart.vue'
import TimeshareChart from '@/components/charts/TimeshareChart.vue'
import OrderBook from '@/components/market/OrderBook.vue'
import QuickPaperOrder from '@/components/paper/QuickPaperOrder.vue'
import { marketApi } from '@/api/market'
import { riskApi } from '@/api/risk'
import { agentApi } from '@/api/agent'
import { strategyApi } from '@/api/strategy'
import { compactNumber, money, percent } from '@/utils/format'
import type { CompanyProfile, IndicatorFullResult, KlineItem, MoneyFlowItem, OrderBook as OrderBookData, RealtimeQuote, Stock, StockAnnouncement, StockNews, SupportResistance, TimeshareItem } from '@/types/market'
import type { StrategySignal } from '@/types/strategy'
import type { AgentReport } from '@/types/agent'
import type { WsServerMessage } from '@/types/notify'
import { useWebSocket } from '@/composables/useWebSocket'
import { useWebsocketStore } from '@/stores/websocket'

const route = useRoute()
const router = useRouter()
const symbol = String(route.params.symbol)
const stock = ref<Stock>()
const quote = ref<RealtimeQuote>()
const klines = ref<KlineItem[]>([])
const indicator = ref<IndicatorFullResult>({})
const signals = ref<StrategySignal[]>([])
const timeshare = ref<TimeshareItem[]>([])
const orderBook = ref<OrderBookData>()
const supportResistance = ref<SupportResistance>()
const company = ref<CompanyProfile>()
const announcements = ref<StockAnnouncement[]>([])
const news = ref<StockNews[]>([])
const moneyFlow = ref<MoneyFlowItem[]>([])
const loading = ref(false)
const errorMessage = ref('')
const range = ref('6m')
const activeIndicator = ref('MACD')
const workspaceTab = ref('market')
const orderVisible = ref(false)
const orderSide = ref<'BUY' | 'SELL'>('BUY')
const autoRefresh = ref(true)
const strategyRunning = ref(false)
const agentRunning = ref(false)
const agentReport = ref<AgentReport>()
let timer: ReturnType<typeof window.setInterval> | undefined
const { subscribe, unsubscribe, onMessage } = useWebSocket()
let removeWsListener: (() => void) | undefined

const rawIndicators = computed(() => indicator.value.indicators || indicator.value || {})
const indicators = computed(() => ({
  ma: rawIndicators.value.ma || [],
  macd: rawIndicators.value.macd || [],
  rsi: rawIndicators.value.rsi || [],
  boll: rawIndicators.value.boll || [],
  kdj: rawIndicators.value.kdj || []
}))
const dataMode = computed(() => quote.value?.data_mode || stock.value?.data_mode || indicator.value.data_mode || klines.value[0]?.data_mode)
const sourceState = computed(() => quote.value?.source || stock.value?.source || indicator.value.source || klines.value[0]?.source || '')
const sourceTip = computed(() => dataSourceTip(sourceState.value))
const sourceTipType = computed(() => (sourceState.value === 'stale-cache' || sourceState.value === 'partial_database' ? 'warning' : 'info'))
const quotePct = computed(() => quote.value?.change_pct ?? quote.value?.pct_change ?? quote.value?.change_percent)
const latestFlow = computed(() => moneyFlow.value.length ? moneyFlow.value[moneyFlow.value.length - 1] : undefined)
const agentReportId = computed(() => Number(agentReport.value?.report_id || agentReport.value?.id || 0))
const agentConfidence = computed(() => {
  const value = Number(agentReport.value?.confidence || 0)
  return `${(value <= 1 ? value * 100 : value).toFixed(0)}%`
})
const agentSections = computed(() => [
  { label: '摘要', value: String(agentReport.value?.summary || '暂无摘要') },
  { label: '行情与盘面', value: String(agentReport.value?.market_analysis || '行情证据不足') },
  { label: '技术指标', value: String(agentReport.value?.technical_analysis || '指标证据不足') },
  { label: '策略信号', value: String(agentReport.value?.strategy_analysis || '暂无有效策略信号') },
  { label: '知识库与消息面', value: String(agentReport.value?.rag_analysis || '本次未关联知识库资料') },
  { label: '风险提示', value: String(agentReport.value?.risk_analysis || '暂无独立风险结论') },
  { label: '综合研判', value: String(agentReport.value?.final_view || '当前证据不足，不形成方向性结论') }
])
const percentage = (value?: number) => value === undefined ? '--' : `${(Number(value) * 100).toFixed(0)}%`
const metric = (value?: number) => value == null || !Number.isFinite(Number(value)) || Number(value) === 0 ? '--' : Number(value).toFixed(2)
const flowClass = (value?: number) => Number(value || 0) > 0 ? 'up' : Number(value || 0) < 0 ? 'down' : ''
const structuralSummary = computed(() => {
  const current = Number(quote.value?.price ?? quote.value?.current ?? supportResistance.value?.current_price ?? 0)
  const support = Number(supportResistance.value?.support_price || 0)
  const resistance = Number(supportResistance.value?.resistance_price || 0)
  if (!current || !support || !resistance) return '数据尚未完整，暂不生成结构研判。'
  const position = (current - support) / Math.max(resistance - support, 0.01)
  if (position >= 0.8) return '价格接近统计压力区，需关注突破有效性、成交量配合和回落风险。'
  if (position <= 0.2) return '价格接近统计支撑区，需结合趋势、市场情绪和风险承受能力观察。'
  return '价格位于支撑与压力区间中部，当前结构没有明确方向性优势。'
})

const rangeParams = () => {
  const months = range.value === '1m' ? 1 : range.value === '3m' ? 3 : range.value === '6m' ? 6 : range.value === '1y' ? 12 : 36
  return { period: 'daily', start_date: dayjs().subtract(months, 'month').format('YYYY-MM-DD'), end_date: dayjs().format('YYYY-MM-DD'), adjust: 'qfq' }
}

const dataSourceTip = (value: string) => {
  if (value === 'local_csv') return '本地初始化数据，非实时行情'
  if (['efinance', 'baostock', 'sina', 'tencent', 'eastmoney', 'akshare'].includes(value)) return `真实数据源：${value}`
  if (value === 'cache') return '缓存数据，实时性约 5 秒内'
  if (value === 'database') return '本地数据库数据'
  if (value === 'stale-cache') return '行情源不可用，当前为过期缓存'
  if (value === 'partial_database') return '行情源不可用，当前为本地部分历史数据'
  if (value === 'source_unavailable' || value === 'unavailable') return '真实行情源暂时不可用，请稍后重试'
  return ''
}

const loadQuote = async () => {
  try {
    quote.value = await marketApi.realtimeBySymbol(symbol)
  } catch (error) {
    autoRefresh.value = false
    throw error
  }
}

const handleRealtimeMessage = (message: WsServerMessage) => {
	if (message.channel !== 'market') return
	if (message.type === 'market.quote.updated') {
		const data = message.data as RealtimeQuote
		if (data?.symbol !== symbol) return
		quote.value = { ...(quote.value || { symbol }), ...data, quote_time: message.quote_time || data.quote_time, source: message.source || data.source, data_mode: message.data_mode || data.data_mode }
		return
	}
	if (message.type === 'market.orderbook.updated') {
		const data = message.data as OrderBookData
		if (data?.symbol !== symbol) return
		orderBook.value = { ...data, quote_time: message.quote_time || data.quote_time, source: message.source || data.source, data_mode: message.data_mode || data.data_mode }
		return
	}
	if (message.type === 'market.timeshare.updated') {
		const data = message.data as { symbol: string; item?: TimeshareItem }
		if (data?.symbol !== symbol || !data.item) return
		const index = timeshare.value.findIndex((item) => item.time === data.item?.time)
		if (index >= 0) timeshare.value.splice(index, 1, data.item)
		else timeshare.value.push(data.item)
	}
}

const loadIntraday = async () => {
  const results = await Promise.allSettled([
    marketApi.timeshare(symbol).then((res) => (timeshare.value = res)),
    marketApi.orderBook(symbol).then((res) => (orderBook.value = res))
  ])
  if (results.every((item) => item.status === 'rejected')) throw (results[0] as PromiseRejectedResult).reason
}

const loadMarketData = async () => {
  loading.value = true
  errorMessage.value = ''
  try {
    const params = rangeParams()
    const results = await Promise.allSettled([
      marketApi.stockDetail(symbol).then((res) => (stock.value = res)),
      marketApi.kline(symbol, params).then((res) => (klines.value = res)),
      marketApi.fullIndicator(symbol, params).then((res) => (indicator.value = res)),
      marketApi.supportResistance(symbol, params).then((res) => (supportResistance.value = res)),
      strategyApi.signalsBySymbol(symbol).then((res) => (signals.value = res))
    ])
    const failed = results.find((item) => item.status === 'rejected')
    if (failed?.status === 'rejected') {
      autoRefresh.value = false
      const reason = failed.reason
      errorMessage.value = reason instanceof Error ? reason.message : '部分行情数据加载失败'
    }
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '行情数据加载失败'
    autoRefresh.value = false
  } finally {
    loading.value = false
  }
}

const loadResearch = async () => {
  await Promise.allSettled([
    marketApi.companyProfile(symbol).then((res) => (company.value = res)),
    marketApi.announcements(symbol).then((res) => (announcements.value = res)),
    marketApi.news(symbol).then((res) => (news.value = res)),
    marketApi.moneyFlow(symbol).then((res) => (moneyFlow.value = res))
  ])
}

const load = async () => {
  try {
    await Promise.all([loadQuote(), loadIntraday(), loadMarketData(), loadResearch()])
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '行情数据加载失败'
  }
}

const startPolling = () => {
  stopPolling()
  if (!autoRefresh.value) return
  timer = window.setInterval(() => {
		if (!document.hidden && !useWebsocketStore().connected) Promise.allSettled([loadQuote(), loadIntraday()])
  }, 5000)
}
const stopPolling = () => {
  if (timer) window.clearInterval(timer)
  timer = undefined
}
const onVisibility = () => {
  if (!document.hidden && autoRefresh.value) Promise.allSettled([loadQuote(), loadIntraday()])
}

const analyzeRisk = async () => {
  await riskApi.analyzeStock(symbol)
  ElMessage.success('风险分析任务已提交')
}
const runDefaultStrategy = async () => {
  try {
    const strategies = await strategyApi.list()
    const selected = strategies.find((item) => item.code === 'trend_score') || strategies[0]
    if (!selected) return ElMessage.warning('当前没有可运行策略，请先到策略中心创建或启用策略')
    await strategyApi.run(selected.id, { symbol, strategy_code: selected.code })
    signals.value = await strategyApi.signalsBySymbol(symbol)
    ElMessage.success('策略分析完成，已生成最新信号')
  } finally { strategyRunning.value = false }
}
const runAgent = async () => {
  agentRunning.value = true
  workspaceTab.value = 'agent'
  try {
    agentReport.value = await agentApi.analyzeStock({ symbol, question: '请从行情、技术指标、策略信号和风险角度进行量化分析' })
    ElMessage.success('智能分析完成，报告已显示在当前页面')
  } finally {
    agentRunning.value = false
  }
}
const openOrder = (side: 'BUY' | 'SELL') => { orderSide.value = side; orderVisible.value = true }

watch(autoRefresh, startPolling)
onMounted(() => {
  load()
  startPolling()
  subscribe('market', [symbol])
  removeWsListener = onMessage(handleRealtimeMessage)
  document.addEventListener('visibilitychange', onVisibility)
})
onBeforeUnmount(() => {
  stopPolling()
  document.removeEventListener('visibilitychange', onVisibility)
  unsubscribe('market', [symbol])
  removeWsListener?.()
})
</script>

<style scoped>
.status-alert {
  margin-bottom: 14px;
}
.quote-overview { margin-bottom: 4px; }
.limit-up-card :deep(.value) { color:#d93025; }
.limit-down-card :deep(.value) { color:#059669; }
.workbench-tabs { margin-top: 16px; min-width:0; }
.workbench-tabs :deep(.el-tabs__content),.workbench-tabs :deep(.el-tab-pane){overflow:visible;min-width:0}
.workbench-tabs .data-table-card{min-width:0;overflow:hidden}
.analysis-band,.indicator-card { margin-top:16px; }
.capital-panel { margin-top:16px; }
.capital-summary { display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:10px;margin-bottom:14px; }
.capital-summary>div { display:grid;gap:6px;padding:12px;background:var(--el-fill-color-light);border:1px solid var(--el-border-color-lighter); }
.capital-summary span,.capital-summary small { color:var(--qs-muted);font-size:12px; }.capital-summary strong{font-size:18px}
.company-panel{margin-bottom:16px}.company-profile{max-height:180px;overflow:auto;line-height:1.8;color:var(--qs-muted)}
.research-grid{display:grid;grid-template-columns:1fr 1fr;gap:16px}.research-list{display:grid;max-height:520px;overflow:auto}.research-list a{display:grid;gap:6px;padding:13px 4px;color:inherit;text-decoration:none;border-bottom:1px solid var(--qs-border)}.research-list a:hover strong{color:var(--el-color-primary)}.research-list span,.research-list small{color:var(--qs-muted);line-height:1.5}
.level-grid { display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:10px; }
.level-grid>div { display:grid;gap:6px;padding:12px;background:#f6f8fb;border:1px solid var(--qs-border); }
.level-grid span,.level-grid small { color:var(--qs-muted); }.level-grid strong{font-size:20px}
.analysis-copy{min-height:52px;line-height:1.7}.quick-actions{display:flex;gap:10px}.quick-actions .el-button{flex:1}
.empty-action{display:grid;justify-items:start;gap:12px;padding-block:28px}
.agent-panel{min-height:360px}.agent-heading{display:grid;gap:4px}.agent-toolbar-actions{display:flex;align-items:center;justify-content:flex-end;gap:8px;flex-wrap:wrap}.agent-toolbar-actions .el-button{margin-left:0}.agent-report-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:12px;margin-top:16px}.agent-report-block{padding:14px;border:1px solid var(--qs-border);background:#fbfcfe;border-radius:8px}.agent-report-block span{display:block;margin-bottom:8px;color:var(--qs-muted);font-size:13px;font-weight:600}.agent-report-block p{margin:0;white-space:pre-wrap;overflow-wrap:anywhere;line-height:1.75}.agent-report-block:last-child{grid-column:1/-1}.report-link-row{display:flex;justify-content:flex-end;margin-top:14px}
@media(max-width:900px){.capital-summary{grid-template-columns:repeat(2,minmax(0,1fr))}.research-grid{grid-template-columns:1fr}}
@media(max-width:720px){.level-grid,.capital-summary,.agent-report-grid{grid-template-columns:1fr}.agent-report-block:last-child{grid-column:auto}.agent-toolbar-actions{justify-content:flex-start}.quick-actions{position:sticky;bottom:0;background:#fff;padding-top:10px}.company-panel :deep(.el-descriptions__body){overflow-x:auto}.company-panel :deep(table){min-width:620px}}
</style>
