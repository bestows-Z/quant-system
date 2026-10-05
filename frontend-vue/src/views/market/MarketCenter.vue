<template>
  <PageContainer title="行情中心" description="股票列表、搜索与实时行情">
    <section class="market-controls">
      <div class="search-controls">
        <el-input v-model="keyword" placeholder="搜索代码/名称" clearable @keyup.enter="loadStocks" />
        <el-button type="primary" :loading="loadingStocks" @click="loadStocks">搜索</el-button>
      </div>
      <div class="refresh-controls">
        <el-button :loading="loadingQuotes" @click="loadQuotes">刷新行情</el-button>
        <el-switch v-model="autoRefresh" active-text="自动刷新" />
        <label class="interval-control">
          <span>刷新频率</span>
          <el-select v-model="refreshInterval">
            <el-option label="3 秒" :value="3000" />
            <el-option label="5 秒" :value="5000" />
            <el-option label="10 秒" :value="10000" />
            <el-option label="30 秒" :value="30000" />
          </el-select>
        </label>
      </div>
    </section>

    <el-alert v-if="errorMessage" :title="errorMessage" type="error" show-icon class="status-alert" />
    <el-alert v-if="dataMode === 'mock'" title="当前为 mock 数据，仅用于离线开发调试" type="warning" show-icon class="status-alert" />
    <el-alert v-if="sourceTip" :title="sourceTip" :type="sourceTipType" show-icon class="status-alert" />

    <section class="data-table-card">
      <div class="toolbar">
        <strong>股票列表</strong>
        <el-tag>{{ source || '--' }}</el-tag>
        <el-tag :type="dataMode === 'mock' ? 'warning' : 'success'">{{ dataMode || '--' }}</el-tag>
        <span class="muted">共 {{ total }} 条，实时行情 {{ quotes.length }} 条</span>
      </div>
      <div class="table-scroll-x">
      <el-table :data="stocks" max-height="620" v-loading="loadingStocks || loadingQuotes" empty-text="暂无行情数据" class="market-table">
        <el-table-column prop="symbol" label="代码" width="110" />
        <el-table-column prop="name" label="名称" width="140" />
        <el-table-column prop="market" label="市场" width="90" />
        <el-table-column prop="industry" label="行业" min-width="120" />
        <el-table-column label="最新价" width="110"><template #default="{ row }">{{ money(quoteMap[row.symbol]?.price ?? quoteMap[row.symbol]?.current) }}</template></el-table-column>
        <el-table-column label="涨跌幅" width="110"><template #default="{ row }"><PriceChange :value="quotePct(quoteMap[row.symbol])" percent-mode /></template></el-table-column>
        <el-table-column label="成交量" width="120"><template #default="{ row }">{{ compactNumber(quoteMap[row.symbol]?.volume) }}</template></el-table-column>
        <el-table-column label="成交额" width="120"><template #default="{ row }">{{ compactNumber(quoteMap[row.symbol]?.amount) }}</template></el-table-column>
        <el-table-column label="数据源" width="120"><template #default="{ row }">{{ quoteMap[row.symbol]?.source || row.source || '--' }}</template></el-table-column>
        <el-table-column label="模式" width="90"><template #default="{ row }"><el-tag size="small" :type="(quoteMap[row.symbol]?.data_mode || row.data_mode) === 'mock' ? 'warning' : 'success'">{{ quoteMap[row.symbol]?.data_mode || row.data_mode || '--' }}</el-tag></template></el-table-column>
        <el-table-column label="操作" width="220" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="$router.push(`/market/stocks/${row.symbol}`)">详情</el-button>
            <el-button size="small" type="primary" :disabled="watchlistSymbols.has(row.symbol)" :loading="addingSymbols.has(row.symbol)" @click="addWatch(row.symbol)">
              {{ watchlistSymbols.has(row.symbol) ? '已自选' : '加自选' }}
            </el-button>
          </template>
        </el-table-column>
      </el-table>
      </div>
      <div class="pager">
        <el-pagination v-model:current-page="page" v-model:page-size="pageSize" layout="total, sizes, prev, pager, next" :page-sizes="[20, 50, 100, 200]" :total="total" @change="loadStocks" />
      </div>
    </section>
  </PageContainer>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import PageContainer from '@/components/common/PageContainer.vue'
import PriceChange from '@/components/common/PriceChange.vue'
import { marketApi } from '@/api/market'
import { compactNumber, money } from '@/utils/format'
import type { RealtimeQuote, Stock } from '@/types/market'
import type { WsServerMessage } from '@/types/notify'
import { useWebSocket } from '@/composables/useWebSocket'

const keyword = ref('')
const stocks = ref<Stock[]>([])
const quotes = ref<RealtimeQuote[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(50)
const source = ref('')
const dataMode = ref('')
const loadingStocks = ref(false)
const loadingQuotes = ref(false)
const errorMessage = ref('')
const autoRefresh = ref(true)
const refreshInterval = ref(5000)
const watchlistSymbols = ref(new Set<string>())
const addingSymbols = ref(new Set<string>())
let timer: ReturnType<typeof window.setInterval> | undefined
const { subscribe, unsubscribe, onMessage } = useWebSocket()
let removeWsListener: (() => void) | undefined

const quoteMap = computed(() => Object.fromEntries(quotes.value.map((item) => [item.symbol, item])))
const currentSymbols = computed(() => stocks.value.map((item) => item.symbol).filter(Boolean).join(','))

const quotePct = (quote?: RealtimeQuote) => quote?.change_pct ?? quote?.pct_change ?? quote?.change_percent
const sourceTip = computed(() => dataSourceTip(source.value))
const sourceTipType = computed(() => (source.value === 'stale-cache' || source.value === 'partial_database' ? 'warning' : 'info'))

const loadStocks = async () => {
  loadingStocks.value = true
  errorMessage.value = ''
  try {
    const payload = await marketApi.stocksPayload({ keyword: keyword.value || undefined, page: page.value, page_size: pageSize.value })
    stocks.value = payload.items || []
    total.value = payload.total || payload.count || stocks.value.length
    source.value = payload.source || stocks.value[0]?.source || source.value
    dataMode.value = payload.data_mode || stocks.value[0]?.data_mode || dataMode.value
    await loadQuotes()
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '股票列表加载失败'
    stocks.value = []
    autoRefresh.value = false
  } finally {
    loadingStocks.value = false
  }
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

const loadQuotes = async () => {
  loadingQuotes.value = true
  try {
    quotes.value = await marketApi.realtime(currentSymbols.value || undefined)
    source.value = quotes.value[0]?.source || source.value
    dataMode.value = quotes.value[0]?.data_mode || dataMode.value
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '实时行情加载失败'
    autoRefresh.value = false
  } finally {
    loadingQuotes.value = false
  }
}

const addWatch = async (symbol: string) => {
  if (watchlistSymbols.value.has(symbol) || addingSymbols.value.has(symbol)) return
  addingSymbols.value = new Set(addingSymbols.value).add(symbol)
  try {
    await marketApi.addWatchlist({ symbol })
    watchlistSymbols.value = new Set(watchlistSymbols.value).add(symbol)
    ElMessage.success('已添加自选')
  } finally {
    const next = new Set(addingSymbols.value)
    next.delete(symbol)
    addingSymbols.value = next
  }
}

const loadWatchlistState = async () => {
  const rows = await marketApi.watchlists().catch(() => [])
  watchlistSymbols.value = new Set(rows.map((item) => item.symbol))
}

const replaceSubscriptions = (previous: string[], next: string[]) => {
  if (previous.length) unsubscribe('market', previous)
  if (next.length) subscribe('market', next)
}

const handleRealtimeMessage = (message: WsServerMessage) => {
  if (message.type !== 'market.quote.updated' || message.channel !== 'market') return
  const data = message.data as RealtimeQuote
  if (!data?.symbol || !stocks.value.some((item) => item.symbol === data.symbol)) return
  const index = quotes.value.findIndex((item) => item.symbol === data.symbol)
  if (index >= 0) quotes.value.splice(index, 1, { ...quotes.value[index], ...data })
  else quotes.value.push(data)
}

const startPolling = () => {
  stopPolling()
  if (!autoRefresh.value) return
  timer = window.setInterval(() => {
    if (!document.hidden) loadQuotes()
  }, refreshInterval.value)
}
const stopPolling = () => {
  if (timer) window.clearInterval(timer)
  timer = undefined
}
const onVisibility = () => {
  if (!document.hidden && autoRefresh.value) loadQuotes()
}

watch([autoRefresh, refreshInterval], startPolling)
watch(() => stocks.value.map((item) => item.symbol), replaceSubscriptions)
onMounted(() => {
  loadWatchlistState()
  loadStocks()
  startPolling()
  document.addEventListener('visibilitychange', onVisibility)
  removeWsListener = onMessage(handleRealtimeMessage)
})
onBeforeUnmount(() => {
  stopPolling()
  document.removeEventListener('visibilitychange', onVisibility)
  unsubscribe('market', stocks.value.map((item) => item.symbol))
  removeWsListener?.()
})
</script>

<style scoped>
.status-alert {
  margin-bottom: 14px;
}

.market-controls {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  margin-bottom: 14px;
  padding: 12px;
  border: 1px solid var(--qs-border);
  border-radius: 8px;
  background: var(--qs-surface);
}
.search-controls { display: grid; grid-template-columns: minmax(220px, 360px) auto; gap: 8px; flex: 1; }
.refresh-controls { display: flex; align-items: center; gap: 12px; }
.interval-control { display: grid; grid-template-columns: auto 100px; align-items: center; gap: 8px; color: var(--qs-muted); white-space: nowrap; }

.pager {
  display: flex;
  justify-content: flex-end;
  padding-top: 14px;
}
.table-scroll-x { width: 100%; overflow-x: auto; overscroll-behavior-x: contain; }
.market-table { min-width: 1180px; }
@media (max-width: 900px) {
  .market-controls { align-items: stretch; flex-direction: column; }
  .refresh-controls { justify-content: space-between; flex-wrap: wrap; }
}
@media (max-width: 600px) {
  .search-controls { grid-template-columns: 1fr; }
  .refresh-controls { display: grid; grid-template-columns: 1fr 1fr; }
  .refresh-controls > :first-child { grid-column: 1 / -1; }
  .interval-control { grid-column: 1 / -1; grid-template-columns: 1fr 120px; }
}
</style>
