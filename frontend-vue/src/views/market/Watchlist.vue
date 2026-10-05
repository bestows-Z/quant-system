<template>
  <PageContainer title="自选股" description="自选股行情、备注与实时刷新">
    <template #actions>
      <el-input v-model="symbol" placeholder="股票代码" style="width: 160px" />
      <el-input v-model="remark" placeholder="备注" style="width: 200px" />
      <el-button type="primary" @click="add">添加</el-button>
      <el-button :loading="loadingQuotes" @click="loadQuotes">刷新行情</el-button>
      <el-switch v-model="autoRefresh" active-text="自动刷新" />
    </template>
    <el-alert v-if="errorMessage" :title="errorMessage" type="error" show-icon class="status-alert" />
    <el-alert v-if="dataMode === 'mock'" title="当前为 mock 数据，仅用于离线开发调试" type="warning" show-icon class="status-alert" />
    <el-alert v-if="sourceTip" :title="sourceTip" :type="sourceTipType" show-icon class="status-alert" />
    <section class="data-table-card">
      <el-table :data="items" height="620" v-loading="loading || loadingQuotes" empty-text="暂无自选股">
        <el-table-column prop="symbol" label="代码" width="110" />
        <el-table-column label="名称" width="140"><template #default="{ row }">{{ quoteMap[row.symbol]?.name || row.name || '--' }}</template></el-table-column>
        <el-table-column label="最新价" width="110"><template #default="{ row }">{{ money(quoteMap[row.symbol]?.price) }}</template></el-table-column>
        <el-table-column label="涨跌幅" width="110"><template #default="{ row }"><PriceChange :value="quotePct(quoteMap[row.symbol])" percent-mode /></template></el-table-column>
        <el-table-column prop="group_name" label="分组" width="120" />
        <el-table-column prop="remark" label="备注" />
        <el-table-column label="成交量" width="120"><template #default="{ row }">{{ compactNumber(quoteMap[row.symbol]?.volume) }}</template></el-table-column>
        <el-table-column label="成交额" width="120"><template #default="{ row }">{{ compactNumber(quoteMap[row.symbol]?.amount) }}</template></el-table-column>
        <el-table-column label="数据源" width="120"><template #default="{ row }">{{ quoteMap[row.symbol]?.source || '--' }}</template></el-table-column>
        <el-table-column label="模式" width="90"><template #default="{ row }"><el-tag size="small" :type="quoteMap[row.symbol]?.data_mode === 'mock' ? 'warning' : 'success'">{{ quoteMap[row.symbol]?.data_mode || '--' }}</el-tag></template></el-table-column>
        <el-table-column label="操作" width="170" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="$router.push(`/market/stocks/${row.symbol}`)">详情</el-button>
            <el-button size="small" type="danger" @click="remove(row.id)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
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
import type { RealtimeQuote, WatchlistItem } from '@/types/market'

const items = ref<WatchlistItem[]>([])
const quotes = ref<RealtimeQuote[]>([])
const symbol = ref('')
const remark = ref('')
const loading = ref(false)
const loadingQuotes = ref(false)
const errorMessage = ref('')
const autoRefresh = ref(true)
let timer: ReturnType<typeof window.setInterval> | undefined

const quoteMap = computed(() => Object.fromEntries(quotes.value.map((item) => [item.symbol, item])))
const dataMode = computed(() => quotes.value[0]?.data_mode)
const sourceState = computed(() => quotes.value[0]?.source || '')
const sourceTip = computed(() => dataSourceTip(sourceState.value))
const sourceTipType = computed(() => (sourceState.value === 'stale-cache' ? 'warning' : 'info'))
const quotePct = (quote?: RealtimeQuote) => quote?.change_pct ?? quote?.pct_change ?? quote?.change_percent
const symbols = computed(() => items.value.map((item) => item.symbol).filter(Boolean).join(','))

const load = async () => {
  loading.value = true
  try {
    items.value = await marketApi.watchlists()
    await loadQuotes()
  } finally {
    loading.value = false
  }
}
const loadQuotes = async () => {
  if (!symbols.value) {
    quotes.value = []
    return
  }
  loadingQuotes.value = true
  try {
    quotes.value = await marketApi.realtime(symbols.value)
    errorMessage.value = ''
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '自选股行情刷新失败'
    autoRefresh.value = false
  } finally {
    loadingQuotes.value = false
  }
}

const dataSourceTip = (value: string) => {
  if (value === 'cache') return '缓存数据，实时性约 5 秒内'
  if (value === 'database') return '本地数据库数据'
  if (value === 'stale-cache') return '行情源不可用，当前为过期缓存'
  if (value === 'partial_database') return '行情源不可用，当前为本地部分历史数据'
  if (value === 'source_unavailable' || value === 'unavailable') return '真实行情源暂时不可用，请稍后重试'
  return ''
}
const add = async () => {
  const normalized = symbol.value.trim()
  if (!normalized) return ElMessage.warning('请输入股票代码')
  if (items.value.some((item) => item.symbol === normalized)) return ElMessage.info('该股票已在自选列表')
  await marketApi.addWatchlist({ symbol: normalized, remark: remark.value })
  symbol.value = ''
  remark.value = ''
  ElMessage.success('已添加自选')
  await load()
}
const remove = async (id: number) => {
  await marketApi.deleteWatchlist(id)
  await load()
}
const startPolling = () => {
  stopPolling()
  if (!autoRefresh.value) return
  timer = window.setInterval(() => {
    if (!document.hidden) loadQuotes()
  }, 5000)
}
const stopPolling = () => {
  if (timer) window.clearInterval(timer)
  timer = undefined
}
const onVisibility = () => {
  if (!document.hidden && autoRefresh.value) loadQuotes()
}

watch(autoRefresh, startPolling)
onMounted(() => {
  load()
  startPolling()
  document.addEventListener('visibilitychange', onVisibility)
})
onBeforeUnmount(() => {
  stopPolling()
  document.removeEventListener('visibilitychange', onVisibility)
})
</script>

<style scoped>
.status-alert {
  margin-bottom: 14px;
}
</style>
