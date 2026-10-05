<template>
  <PageContainer title="数字资产现货" description="币安公开现货行情 · 按需刷新 · 只读">
    <section class="terminal" aria-label="币安现货行情终端">
      <header class="terminal-header">
        <div class="market-id"><span class="market-mark">✦</span><div><strong>SPOT MARKET</strong><small>BINANCE PUBLIC DATA</small></div></div>
        <div class="market-status"><span class="status-dot" :class="{ idle: !market }" />{{ market ? '公开行情已获取' : loading ? '正在获取行情' : '等待数据' }}<span class="status-separator">/</span>只读行情</div>
      </header>

      <div class="symbol-strip" role="group" aria-label="选择交易对">
        <button v-for="pair in pairs" :key="pair.symbol" class="symbol-tab" :class="{ active: symbol === pair.symbol }" type="button" :aria-pressed="symbol === pair.symbol" @click="symbol = pair.symbol">
          <span class="symbol-name">{{ pair.base }} <span>/ USDT</span></span>
          <small>{{ pair.name }}</small>
        </button>
      </div>

      <div v-if="error && !market" class="state-panel" role="alert"><strong>行情暂时不可用</strong><p>{{ error }}</p><button type="button" class="terminal-button" @click="load">重新获取</button></div>
      <div v-else-if="loading && !market" class="state-panel loading-panel" role="status"><span class="loading-bar" /><span class="loading-bar short" /><p>正在连接币安公开市场数据…</p></div>
      <template v-else-if="market">
        <div class="quote-band">
          <div class="quote-primary"><div class="eyebrow">{{ market.base_asset }} / {{ market.quote_asset }} <span>现货</span></div><div class="price-line"><strong :class="directionClass">{{ formatPrice(market.quote.last_price) }}</strong><span>USDT</span></div><div class="price-change" :class="directionClass">{{ signedPercent(market.quote.change_percent) }} <span>过去 24 小时</span></div></div>
          <dl class="quote-stats"><div><dt>24h 最高</dt><dd>{{ formatPrice(market.quote.high_price) }}</dd></div><div><dt>24h 最低</dt><dd>{{ formatPrice(market.quote.low_price) }}</dd></div><div><dt>24h 成交量 ({{ market.base_asset }})</dt><dd>{{ formatDecimal(market.quote.base_volume, 2) }}</dd></div><div><dt>24h 成交额 (USDT)</dt><dd>{{ compact(market.quote.quote_volume) }}</dd></div></dl>
        </div>

        <div class="terminal-body">
          <section class="chart-panel" aria-labelledby="chart-title">
            <div class="panel-heading"><div><span class="eyebrow">PRICE ACTION</span><h2 id="chart-title">K 线与成交量</h2></div><div class="intervals" role="group" aria-label="K 线周期"><button v-for="option in intervals" :key="option.value" type="button" :class="{ active: interval === option.value }" :aria-pressed="interval === option.value" @click="interval = option.value">{{ option.label }}</button></div></div>
            <CryptoSpotChart :klines="market.klines" :base-asset="market.base_asset" />
            <div class="chart-footer"><span>{{ market.klines.length }} 根已收盘 K 线 · UTC 时间</span><span v-if="loading" role="status">正在更新…</span><span v-else>最近更新 {{ formatUtc(market.fetched_at) }}</span></div>
          </section>
          <aside class="details-panel" aria-label="交易对规则与行情说明">
            <section class="detail-section"><span class="eyebrow">MARKET MICROSTRUCTURE</span><h2>现货盘口</h2><dl class="detail-list"><div><dt>最优买价</dt><dd class="positive">{{ formatPrice(market.quote.bid_price) }}</dd></div><div><dt>最优卖价</dt><dd class="negative">{{ formatPrice(market.quote.ask_price) }}</dd></div><div><dt>买卖价差</dt><dd>{{ spread }} USDT</dd></div><div><dt>交易状态</dt><dd><span class="trading-tag">{{ market.trading_status === 'TRADING' ? '交易中' : market.trading_status }}</span></dd></div></dl></section>
            <section class="detail-section"><span class="eyebrow">EXCHANGE FILTERS</span><h2>交易规则</h2><dl class="detail-list"><div><dt>最小价格变动</dt><dd>{{ market.rules.tick_size }}</dd></div><div><dt>最小下单数量</dt><dd>{{ market.rules.min_qty }} {{ market.base_asset }}</dd></div><div><dt>数量步进</dt><dd>{{ market.rules.step_size }}</dd></div><div><dt>最低名义价值</dt><dd>{{ market.rules.min_notional }} USDT</dd></div></dl></section>
            <div class="source-note"><strong>数据说明</strong><p>来自 Binance Spot 公开市场接口，无需 API Key。报价、K 线和交易规则均为真实交易所数据；当前页面不提供下单或账户连接。</p><span>行情时间 {{ formatUtc(market.quote.event_time) }} UTC</span></div>
          </aside>
        </div>
        <div v-if="error" class="inline-error" role="alert">刷新失败，已清除旧行情：{{ error }}</div>
      </template>
      <footer class="terminal-footer"><span>PUBLIC MARKET DATA</span><span>不构成投资建议 · 交易所行情可能延迟</span><button type="button" class="refresh-button" :disabled="loading" @click="load">{{ loading ? '更新中…' : '刷新行情 ↗' }}</button></footer>
    </section>
  </PageContainer>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import PageContainer from '@/components/common/PageContainer.vue'
import CryptoSpotChart from '@/components/charts/CryptoSpotChart.vue'
import { cryptoApi } from '@/api/crypto'
import type { CryptoSpotMarket } from '@/types/crypto'

const pairs = [{ symbol: 'BTCUSDT', base: 'BTC', name: 'Bitcoin' }, { symbol: 'ETHUSDT', base: 'ETH', name: 'Ethereum' }, { symbol: 'BNBUSDT', base: 'BNB', name: 'BNB' }] as const
const intervals = [{ value: '1h', label: '1 小时' }, { value: '4h', label: '4 小时' }, { value: '1d', label: '1 天' }] as const
const route = useRoute()
const router = useRouter()
const initialSymbol = String(route.query.symbol || '').toUpperCase()
const initialInterval = String(route.query.interval || '')
const symbol = ref(pairs.some(item => item.symbol === initialSymbol) ? initialSymbol : 'BTCUSDT')
const interval = ref(intervals.some(item => item.value === initialInterval) ? initialInterval : '1h')
const market = ref<CryptoSpotMarket | null>(null)
const loading = ref(false)
const error = ref('')
let sequence = 0
const formatDecimal = (value: string, digits: number) => Number(value).toLocaleString('en-US', { minimumFractionDigits: digits, maximumFractionDigits: digits })
const priceDigits = computed(() => {
  const tick = market.value?.rules.tick_size || '0.01'
  return tick.includes('.') ? tick.split('.')[1].replace(/0+$/, '').length : 0
})
const formatPrice = (value: string) => formatDecimal(value, priceDigits.value)
const compact = (value: string) => new Intl.NumberFormat('en-US', { notation: 'compact', maximumFractionDigits: 2 }).format(Number(value))
const signedPercent = (value: string) => `${Number(value) >= 0 ? '+' : ''}${Number(value).toFixed(2)}%`
const formatUtc = (value: string | number) => new Intl.DateTimeFormat('zh-CN', { timeZone: 'UTC', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false }).format(new Date(value))
const directionClass = computed(() => Number(market.value?.quote.change_percent || 0) >= 0 ? 'positive' : 'negative')
const spread = computed(() => market.value ? formatPrice(String(Math.max(Number(market.value.quote.ask_price) - Number(market.value.quote.bid_price), 0))) : '--')
const load = async () => {
  const requestId = ++sequence
  loading.value = true
  error.value = ''
  market.value = null
  try {
    const result = await cryptoApi.spotMarket(symbol.value, interval.value)
    if (requestId === sequence) market.value = result
  } catch (cause) {
    if (requestId === sequence) error.value = cause instanceof Error ? cause.message : '请检查网络与数据源状态后重试'
  } finally {
    if (requestId === sequence) loading.value = false
  }
}
watch([symbol, interval], () => { void router.replace({ query: { ...route.query, symbol: symbol.value, interval: interval.value } }); void load() })
onMounted(() => void load())
</script>

<style scoped>
.terminal{--surface:#181a20;--raised:#1e222b;--line:#2b3139;--muted:#8d949e;--gold:#f0b90b;--up:#0ecb81;--down:#f6465d;color:#eaecef;background:var(--surface);border:1px solid var(--line);border-radius:14px;overflow:hidden;box-shadow:0 18px 42px rgba(16,24,32,.11);font-variant-numeric:tabular-nums}
.terminal-header,.symbol-strip,.quote-band,.terminal-footer{display:flex;align-items:center}.terminal-header{justify-content:space-between;padding:16px 24px;border-bottom:1px solid var(--line)}.market-id{display:flex;align-items:center;gap:10px}.market-mark{display:grid;place-items:center;width:30px;height:30px;color:#181a20;background:var(--gold);border-radius:7px;font-size:20px}.market-id strong,.market-id small{display:block;line-height:1.25}.market-id strong{font-size:12px;letter-spacing:.12em}.market-id small{font-size:10px;letter-spacing:.12em;color:var(--muted);margin-top:3px}.market-status{display:flex;align-items:center;gap:8px;color:#aeb4bd;font-size:12px}.status-dot{width:7px;height:7px;border-radius:50%;background:var(--up);box-shadow:0 0 0 3px rgba(14,203,129,.13)}.status-dot.idle{background:#8d949e;box-shadow:none}.status-separator{color:#5e6671}.symbol-strip{gap:2px;padding:0 20px;border-bottom:1px solid var(--line);overflow-x:auto}.symbol-tab{position:relative;display:flex;flex-direction:column;align-items:flex-start;gap:3px;min-width:156px;padding:17px 14px 15px;background:none;border:0;color:#eaecef;cursor:pointer;text-align:left}.symbol-tab:hover{background:var(--raised)}.symbol-tab.active:after{content:"";position:absolute;left:14px;right:14px;bottom:0;height:2px;background:var(--gold)}.symbol-name{font-weight:700;font-size:14px}.symbol-name span,.symbol-tab small{color:var(--muted);font-size:11px;font-weight:400}.quote-band{gap:40px;padding:27px 30px;border-bottom:1px solid var(--line)}.quote-primary{min-width:270px}.eyebrow{color:var(--gold);font-size:10px;letter-spacing:.13em;font-weight:700}.quote-primary .eyebrow span{margin-left:6px;padding:2px 5px;border:1px solid #565023;border-radius:3px;color:#d6bf73;letter-spacing:0}.price-line{display:flex;align-items:baseline;gap:9px;margin-top:9px}.price-line strong{font-size:36px;line-height:1.1;font-weight:650;letter-spacing:-.03em}.price-line span{font-size:14px;color:var(--muted)}.positive{color:var(--up)!important}.negative{color:var(--down)!important}.price-change{margin-top:7px;font-size:15px;font-weight:650}.price-change span{color:var(--muted);font-size:11px;font-weight:400;margin-left:6px}.quote-stats{display:grid;grid-template-columns:repeat(4,minmax(110px,1fr));gap:15px;flex:1;margin:0}.quote-stats div{min-width:0}.quote-stats dt,.detail-list dt{color:var(--muted);font-size:11px}.quote-stats dd{margin:8px 0 0;font-size:15px;font-weight:600;white-space:nowrap}.terminal-body{display:grid;grid-template-columns:minmax(0,1fr) 300px}.chart-panel{padding:22px 23px 13px;min-width:0}.panel-heading{display:flex;justify-content:space-between;align-items:flex-end;gap:12px}.panel-heading h2,.details-panel h2{margin:5px 0 0;font-size:17px;font-weight:650}.intervals{display:flex;gap:3px}.intervals button,.refresh-button,.terminal-button{background:transparent;border:1px solid transparent;border-radius:5px;color:#aeb4bd;padding:6px 10px;cursor:pointer;font:inherit;font-size:12px}.intervals button:hover,.refresh-button:hover{color:#fff;background:#2b3139}.intervals button.active{color:#181a20;background:var(--gold);font-weight:700}.chart-footer{display:flex;justify-content:space-between;gap:8px;color:var(--muted);font-size:11px;padding:4px 7px}.details-panel{border-left:1px solid var(--line);background:#1b1e25}.detail-section{padding:23px 22px;border-bottom:1px solid var(--line)}.detail-list{display:grid;gap:16px;margin:21px 0 0}.detail-list div{display:flex;justify-content:space-between;gap:10px}.detail-list dd{margin:0;text-align:right;font-size:12px;font-weight:600;white-space:nowrap}.trading-tag{color:var(--up);border:1px solid #205c49;border-radius:3px;padding:3px 6px;font-size:10px}.source-note{margin:22px;border:1px solid #3d3c30;background:#27271f;border-radius:7px;padding:14px}.source-note strong{font-size:12px;color:#f0cf6b}.source-note p{margin:9px 0;line-height:1.65;color:#b6b5a8;font-size:11px}.source-note span{color:#85877f;font-size:10px}.terminal-footer{justify-content:space-between;gap:12px;padding:10px 24px;border-top:1px solid var(--line);color:var(--muted);font-size:11px}.terminal-footer>span:first-child{color:#737b86;letter-spacing:.13em;font-size:10px}.refresh-button{color:var(--gold);border-color:#61551b}.refresh-button:disabled{opacity:.5;cursor:wait}.state-panel{min-height:560px;display:flex;flex-direction:column;align-items:center;justify-content:center;gap:12px;text-align:center;padding:30px}.state-panel strong{font-size:17px}.state-panel p{color:var(--muted);font-size:13px}.terminal-button{background:var(--gold);color:#181a20;font-weight:700;padding:9px 15px}.loading-bar{width:230px;height:18px;border-radius:4px;background:#30353d;animation:pulse 1.2s ease-in-out infinite}.loading-bar.short{width:140px}.inline-error{padding:12px 24px;border-top:1px solid #6b3039;color:#f4939d;font-size:12px}@keyframes pulse{50%{opacity:.45}}button:focus-visible{outline:2px solid var(--gold);outline-offset:2px}@media(max-width:1200px){.quote-band{display:block}.quote-stats{margin-top:22px}.terminal-body{grid-template-columns:minmax(0,1fr) 260px}}@media(max-width:850px){.terminal-body{grid-template-columns:1fr}.details-panel{border-left:0;border-top:1px solid var(--line);display:grid;grid-template-columns:1fr 1fr}.source-note{grid-column:1/-1}.detail-section:nth-child(2){border-left:1px solid var(--line)}}@media(max-width:600px){.terminal-header{padding:13px 15px}.market-status{font-size:10px}.symbol-strip{padding:0 8px}.symbol-tab{min-width:116px;padding:14px 9px}.quote-band{padding:20px 16px}.price-line strong{font-size:30px}.quote-stats{grid-template-columns:1fr 1fr;gap:18px 8px}.quote-stats dd{font-size:13px}.chart-panel{padding:18px 10px 8px}.panel-heading{padding:0 6px}.details-panel{display:block}.detail-section:nth-child(2){border-left:0}.terminal-footer{padding:12px 15px;flex-wrap:wrap}.terminal-footer>span:nth-child(2){order:3;width:100%}}
</style>
