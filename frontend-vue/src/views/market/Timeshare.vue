<template>
  <PageContainer title="分时盘口" description="当日分时、分钟成交量与买卖五档实时监控">
    <template #actions>
      <el-autocomplete
        v-model="searchText"
        :fetch-suggestions="searchStocks"
        value-key="label"
        placeholder="输入股票代码或名称"
        clearable
        class="symbol-search"
        @select="selectStock"
        @keyup.enter="openSymbol"
      />
      <el-button :icon="Refresh" :loading="loading" @click="load">刷新</el-button>
    </template>

	<el-alert v-if="errorMessage" :title="errorMessage" type="error" show-icon class="status-alert" />
	<div class="compare-toolbar">
	  <strong>分时对比</strong>
	  <el-segmented v-model="compareKind" :options="[{label:'股票',value:'stock'},{label:'板块',value:'sector'}]" />
	  <el-autocomplete v-if="compareKind==='stock'" v-model="compareText" :fetch-suggestions="searchStocks" value-key="label" placeholder="搜索对比股票" class="compare-picker" @select="addCompareStock" />
	  <el-select v-else v-model="compareSector" filterable placeholder="选择对比板块" class="compare-picker" @change="addCompareSector"><el-option v-for="item in sectors" :key="item.code" :label="item.name" :value="item.code" /></el-select>
	  <span class="muted">最多 3 个，统一按涨跌幅比较</span>
	  <div class="compare-tags"><el-tag v-for="item in comparisons" :key="item.key" closable @close="removeComparison(item)">{{item.name}}</el-tag></div>
	</div>
    <div class="quote-strip">
      <div><span>{{ quote?.name || orderBook?.name || symbol }}</span><strong class="mono">{{ symbol }}</strong></div>
      <div><small>最新</small><strong class="mono" :class="priceClass">{{ money(quote?.price || orderBook?.price) }}</strong></div>
      <div><small>涨跌幅</small><PriceChange :value="quotePct" percent-mode /></div>
      <div><small>成交量</small><strong class="mono">{{ compactNumber(quote?.volume) }}</strong></div>
      <div><small>成交额</small><strong class="mono">{{ compactNumber(quote?.amount) }}</strong></div>
      <div><small>连接</small><strong :class="ws.connected ? 'online' : 'muted'">{{ ws.connected ? '实时推送' : 'HTTP 兜底' }}</strong></div>
    </div>

    <div class="market-workspace">
      <section class="market-panel chart-panel">
        <div class="panel-title"><strong>当日分时</strong><span>{{ lastUpdate }}</span></div>
		<TimeshareChart :data="timeshare" :pre-close="quote?.pre_close || orderBook?.pre_close" :loading="loading" :comparisons="comparisons" />
      </section>
      <section class="market-panel book-panel">
        <div class="panel-title"><strong>五档盘口</strong><span>{{ orderBook?.source || '--' }}</span></div>
        <OrderBook :book="orderBook" />
      </section>
    </div>
    <DisclaimerAlert />
  </PageContainer>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Refresh } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import PageContainer from '@/components/common/PageContainer.vue'
import DisclaimerAlert from '@/components/common/DisclaimerAlert.vue'
import PriceChange from '@/components/common/PriceChange.vue'
import TimeshareChart from '@/components/charts/TimeshareChart.vue'
import OrderBook from '@/components/market/OrderBook.vue'
import { marketApi } from '@/api/market'
import { featureApi } from '@/api/feature'
import { useWebSocket } from '@/composables/useWebSocket'
import { useWebsocketStore } from '@/stores/websocket'
import { compactNumber, money } from '@/utils/format'
import type { OrderBook as OrderBookData, RealtimeQuote, Stock, TimeshareItem } from '@/types/market'
import type { WsServerMessage } from '@/types/notify'
import type { SectorStrength } from '@/types/feature'
import { formatTime } from '@/utils/time'

const route = useRoute()
const router = useRouter()
const ws = useWebsocketStore()
const symbol = computed(() => String(route.params.symbol || '600176'))
const searchText = ref('')
const quote = ref<RealtimeQuote>()
const timeshare = ref<TimeshareItem[]>([])
const orderBook = ref<OrderBookData>()
const loading = ref(false)
const errorMessage = ref('')
type ComparisonLine={key:string;name:string;kind:'stock'|'sector';symbol?:string;code?:string;preClose?:number;data:Array<{time:string;value:number}>}
const comparisons=ref<ComparisonLine[]>([])
const compareKind=ref<'stock'|'sector'>('stock')
const compareText=ref('')
const compareSector=ref('')
const sectors=ref<SectorStrength[]>([])
const { subscribe, unsubscribe, onMessage } = useWebSocket()
let removeListener: (() => void) | undefined
let fallbackTimer: number | undefined

const quotePct = computed(() => quote.value?.change_pct ?? quote.value?.pct_change ?? quote.value?.change_percent)
const priceClass = computed(() => Number(quote.value?.price || 0) >= Number(quote.value?.pre_close || 0) ? 'up' : 'down')
const lastUpdate = computed(() => formatTime(quote.value?.quote_time || timeshare.value[timeshare.value.length - 1]?.time))

const load = async () => {
  loading.value = true
  errorMessage.value = ''
  const results = await Promise.allSettled([
    marketApi.realtimeBySymbol(symbol.value).then((data) => (quote.value = data)),
    marketApi.timeshare(symbol.value).then((data) => (timeshare.value = data)),
    marketApi.orderBook(symbol.value).then((data) => (orderBook.value = data))
  ])
  const failed = results.filter((item) => item.status === 'rejected')
  if (failed.length === results.length) errorMessage.value = '分时行情暂时不可用，请稍后重试'
  loading.value = false
}

const searchStocks = async (query: string, callback: (items: Array<{ label: string; value: string }>) => void) => {
  if (!query.trim()) return callback([])
  const rows = await marketApi.searchStocks(query.trim(), 1, 10).catch(() => [] as Stock[])
  callback(rows.map((item) => ({ label: `${item.symbol} ${item.name}`, value: item.symbol })))
}
const selectStock = (item: { value: string }) => router.push(`/market/timeshare/${item.value}`)
const openSymbol = () => {
  const matched = searchText.value.match(/\d{6}/)?.[0]
  if (matched) router.push(`/market/timeshare/${matched}`)
}
const ensureComparisonRoom=(key:string)=>{
	if(comparisons.value.some(item=>item.key===key)){ElMessage.info('该标的已在对比中');return false}
	if(comparisons.value.length>=3){ElMessage.warning('最多添加 3 个对比标的');return false}
	return true
}
const addCompareStock=async(item:{value:string;label:string})=>{
	const target=item.value
	if(target===symbol.value){ElMessage.info('主标的无需重复添加');return}
	if(!ensureComparisonRoom(`stock:${target}`))return
	const [targetQuote,rows]=await Promise.all([marketApi.realtimeBySymbol(target),marketApi.timeshare(target)])
	const preClose=Number(targetQuote.pre_close||0)
	comparisons.value.push({key:`stock:${target}`,name:`${targetQuote.name||item.label} ${target}`,kind:'stock',symbol:target,preClose,data:rows.map(row=>({time:row.time,value:preClose>0?(Number(row.price)/preClose-1)*100:0}))})
	subscribe('market',[target]);compareText.value=''
}
const addCompareSector=async(code:string)=>{
	const target=sectors.value.find(item=>item.code===code)
	if(!target||!ensureComparisonRoom(`sector:${code}`))return
	const result=await featureApi.sectorTimeshare(code)
	comparisons.value.push({key:`sector:${code}`,name:target.name,kind:'sector',code,data:(result.items||[]).filter(item=>item.change_percent!=null).map(item=>({time:item.time,value:Number(item.change_percent)*100}))})
	compareSector.value=''
}
const removeComparison=(item:ComparisonLine)=>{
	comparisons.value=comparisons.value.filter(line=>line.key!==item.key)
	if(item.symbol)unsubscribe('market',[item.symbol])
}
const refreshSectorComparisons=async()=>{
	await Promise.all(comparisons.value.filter(item=>item.kind==='sector'&&item.code).map(async item=>{
		const result=await featureApi.sectorTimeshare(String(item.code))
		item.data=(result.items||[]).filter(point=>point.change_percent!=null).map(point=>({time:point.time,value:Number(point.change_percent)*100}))
	}))
}
const handleMessage = (message: WsServerMessage) => {
	if (message.channel !== 'market') return
	if (message.type === 'market.quote.updated') {
		const data = message.data as RealtimeQuote
		if (data.symbol !== symbol.value) return
		quote.value = { ...(quote.value || { symbol: symbol.value }), ...data, quote_time: message.quote_time || data.quote_time, source: message.source || data.source, data_mode: message.data_mode || data.data_mode }
		return
	}
	if (message.type === 'market.orderbook.updated') {
		const data = message.data as OrderBookData
		if (data.symbol !== symbol.value) return
		orderBook.value = { ...data, quote_time: message.quote_time || data.quote_time, source: message.source || data.source, data_mode: message.data_mode || data.data_mode }
		return
	}
	if (message.type === 'market.timeshare.updated') {
		const data = message.data as { symbol: string; item?: TimeshareItem }
		if (!data.item) return
		if(data.symbol===symbol.value){const index = timeshare.value.findIndex((item) => item.time === data.item?.time);if (index >= 0) timeshare.value.splice(index, 1, data.item);else timeshare.value.push(data.item);return}
		const line=comparisons.value.find(item=>item.symbol===data.symbol)
		if(!line||!line.preClose)return
		const point={time:data.item.time,value:(Number(data.item.price)/line.preClose-1)*100}
		const index=line.data.findIndex(item=>item.time===point.time)
		if(index>=0)line.data.splice(index,1,point);else line.data.push(point)
	}
}
const bindSymbol = (next: string, previous?: string) => {
  if (previous) unsubscribe('market', [previous])
  subscribe('market', [next])
  searchText.value = next
  load()
}

watch(symbol, bindSymbol)
onMounted(() => {
  bindSymbol(symbol.value)
	featureApi.sectors().then(rows=>{sectors.value=rows}).catch(()=>undefined)
  removeListener = onMessage(handleMessage)
	fallbackTimer = window.setInterval(() => { if (!document.hidden) { if(!ws.connected) load(); refreshSectorComparisons() } }, 15000)
})
onBeforeUnmount(() => {
  unsubscribe('market', [symbol.value])
	comparisons.value.filter(item=>item.symbol).forEach(item=>unsubscribe('market',[String(item.symbol)]))
  removeListener?.()
  if (fallbackTimer) window.clearInterval(fallbackTimer)
})
</script>

<style scoped>
.status-alert { margin-bottom: 12px; }
.symbol-search { width: min(300px, 100%); }
.compare-toolbar{display:flex;align-items:center;gap:10px;flex-wrap:wrap;margin-bottom:12px;padding:10px 12px;border:1px solid var(--qs-border);background:var(--qs-surface)}.compare-picker{width:240px}.compare-tags{display:flex;gap:6px;flex-wrap:wrap}
.quote-strip { display: grid; grid-template-columns: 1.4fr repeat(5, minmax(110px, 1fr)); gap: 1px; margin-bottom: 12px; overflow: hidden; border: 1px solid var(--qs-border); border-radius: 8px; background: var(--qs-border); }
.quote-strip > div { min-height: 66px; padding: 10px 14px; display: flex; flex-direction: column; justify-content: center; background: var(--qs-surface); }
.quote-strip span, .quote-strip small { color: var(--qs-muted); }
.quote-strip strong { margin-top: 4px; font-size: 16px; }
.quote-strip > div:first-child span { color: var(--qs-text); font-weight: 650; font-size: 16px; }
.market-workspace { display: grid; grid-template-columns: minmax(0, 1fr) 300px; gap: 12px; }
.market-panel { min-width: 0; padding: 14px; border: 1px solid var(--qs-border); border-radius: 8px; background: var(--qs-surface); }
.panel-title { min-height: 28px; display: flex; justify-content: space-between; gap: 8px; color: var(--qs-muted); }
.panel-title strong { color: var(--qs-text); }
.online { color: var(--qs-down); }
@media (max-width: 1100px) { .quote-strip { grid-template-columns: repeat(3, 1fr); } .market-workspace { grid-template-columns: minmax(0, 1fr); } }
@media (max-width: 600px) { .quote-strip { grid-template-columns: repeat(2, 1fr); } .quote-strip > div { min-height: 58px; padding: 8px 10px; } .market-panel { padding: 10px; } }
@media (max-width:600px){.compare-picker{width:100%}.compare-toolbar>.muted{display:none}}
</style>
