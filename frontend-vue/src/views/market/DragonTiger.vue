<template>
  <PageContainer title="龙虎榜" description="真实公开榜单、个股买卖金额与上榜席位明细">
    <template #actions>
      <el-date-picker v-model="tradeDate" value-format="YYYY-MM-DD" placeholder="选择交易日" clearable />
      <el-button :loading="loading" @click="load">查询</el-button>
    </template>
    <el-alert :title="result.message || '正在读取龙虎榜数据'" :type="result.status === 'AVAILABLE' ? 'success' : 'info'" show-icon :closable="false" class="notice" />
    <section class="data-table-card">
      <div class="table-scroll-x">
        <el-table :data="result.items" v-loading="loading" class="wide-table" empty-text="当前没有可验证的龙虎榜记录">
          <el-table-column prop="trade_date" label="交易日" width="110" />
          <el-table-column prop="symbol" label="代码" width="100" />
          <el-table-column prop="name" label="名称" width="120" />
          <el-table-column prop="reason" label="上榜原因" min-width="220" show-overflow-tooltip />
          <el-table-column label="涨跌幅" width="100"><template #default="{ row }"><PriceChange :value="row.change_percent" percent-mode /></template></el-table-column>
          <el-table-column label="净买入额" width="120"><template #default="{ row }"><span :class="amountClass(row.net_buy_amount)">{{ compactNumber(row.net_buy_amount) }}</span></template></el-table-column>
          <el-table-column label="买入额" width="110"><template #default="{ row }">{{ compactNumber(row.buy_amount) }}</template></el-table-column>
          <el-table-column label="卖出额" width="110"><template #default="{ row }">{{ compactNumber(row.sell_amount) }}</template></el-table-column>
          <el-table-column label="操作" width="170" fixed="right"><template #default="{ row }"><el-button link type="primary" @click="openSeats(row)">席位明细</el-button><el-button link type="primary" @click="$router.push(`/market/stocks/${row.symbol}`)">个股</el-button></template></el-table-column>
        </el-table>
      </div>
    </section>
    <el-drawer v-model="seatVisible" :title="`${selected?.name || selected?.symbol || ''} 席位明细`" size="min(760px, 96vw)">
      <div class="seat-summary">
        <div><span>交易日</span><strong>{{ selected?.trade_date }}</strong></div>
        <div><span>榜单买入</span><strong>{{ compactNumber(selected?.buy_amount) }}</strong></div>
        <div><span>榜单卖出</span><strong>{{ compactNumber(selected?.sell_amount) }}</strong></div>
        <div><span>净买入</span><strong :class="amountClass(selected?.net_buy_amount)">{{ compactNumber(selected?.net_buy_amount) }}</strong></div>
      </div>
      <el-alert :title="seats.message || '席位数据加载中'" type="info" show-icon :closable="false" class="notice" />
      <div class="table-scroll-x"><el-table :data="seats.items" v-loading="seatLoading" class="seat-table" empty-text="暂无可验证席位明细">
        <el-table-column label="买入排名" width="90"><template #default="{row}">{{ row.buy_rank ? `买${row.buy_rank}` : '--' }}</template></el-table-column>
        <el-table-column label="卖出排名" width="90"><template #default="{row}">{{ row.sell_rank ? `卖${row.sell_rank}` : '--' }}</template></el-table-column>
        <el-table-column prop="seat_name" label="营业部 / 席位" min-width="260" />
        <el-table-column label="类型" width="100"><template #default="{row}">{{ seatType(row.seat_type) }}</template></el-table-column>
        <el-table-column label="买入额" width="120"><template #default="{row}">{{ compactNumber(row.buy_amount) }}</template></el-table-column>
        <el-table-column label="卖出额" width="120"><template #default="{row}">{{ compactNumber(row.sell_amount) }}</template></el-table-column>
        <el-table-column label="净额" width="120"><template #default="{row}"><span :class="amountClass(row.net_amount)">{{ compactNumber(row.net_amount) }}</span></template></el-table-column>
      </el-table></div>
    </el-drawer>
    <DisclaimerAlert />
  </PageContainer>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import PageContainer from '@/components/common/PageContainer.vue'
import DisclaimerAlert from '@/components/common/DisclaimerAlert.vue'
import PriceChange from '@/components/common/PriceChange.vue'
import { featureApi } from '@/api/feature'
import { compactNumber } from '@/utils/format'
import type { DragonTigerRecord, DragonTigerResult, DragonTigerSeatResult } from '@/types/feature'

const tradeDate = ref('')
const loading = ref(false)
const seatLoading = ref(false)
const seatVisible = ref(false)
const selected = ref<DragonTigerRecord>()
const result = ref<DragonTigerResult>({ items: [], count: 0, status: '', message: '', data_mode: 'real', disclaimer: '' })
const seats = ref<DragonTigerSeatResult>({ symbol: '', trade_date: '', items: [], count: 0, status: '', message: '', source: '', data_mode: 'real', disclaimer: '' })

const load = async () => {
  loading.value = true
  try { result.value = await featureApi.dragonTiger({ trade_date: tradeDate.value, limit: 100 }) } finally { loading.value = false }
}
const openSeats = async (row: DragonTigerRecord) => {
  selected.value = row
  seatVisible.value = true
  seatLoading.value = true
  seats.value = { symbol: row.symbol, trade_date: row.trade_date, items: [], count: 0, status: '', message: '', source: '', data_mode: 'real', disclaimer: '' }
  try { seats.value = await featureApi.dragonTigerSeats(row.symbol, row.trade_date) } finally { seatLoading.value = false }
}
const amountClass = (value?: number) => Number(value || 0) > 0 ? 'up' : Number(value || 0) < 0 ? 'down' : ''
const seatType = (value: string) => ({ INSTITUTION: '机构专用', NORTHBOUND: '沪深股通', BROKER_BRANCH: '营业部' }[value] || '其他席位')
onMounted(load)
</script>

<style scoped>
.notice { margin-bottom: 16px; }
.wide-table { min-width: 1180px; }
.seat-table { min-width: 900px; }
.seat-summary { display: flex; flex-wrap: wrap; gap: 1px; margin-bottom: 14px; overflow: hidden; border: 1px solid var(--qs-border); border-radius: 8px; background: var(--qs-border); }
.seat-summary > div { flex: 1 1 150px; min-height: 66px; padding: 10px 14px; display: flex; flex-direction: column; justify-content: center; background: var(--qs-surface); }
.seat-summary span { color: var(--qs-muted); font-size: 12px; }
.seat-summary strong { margin-top: 5px; }
</style>
