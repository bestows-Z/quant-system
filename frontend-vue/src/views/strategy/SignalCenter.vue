<template>
  <PageContainer title="策略信号" description="策略信号筛选与风险等级展示">
    <template #actions>
      <el-input v-model="symbol" placeholder="股票代码" clearable style="width: 160px" />
      <el-select v-model="signalType" clearable placeholder="信号类型" style="width: 140px">
        <el-option label="买入" value="BUY" /><el-option label="卖出" value="SELL" /><el-option label="观望" value="HOLD" /><el-option label="关注" value="WATCH" />
      </el-select>
      <el-button type="primary" @click="load">筛选</el-button>
    </template>
    <section class="data-table-card">
      <div class="table-scroll-x"><el-table :data="items" max-height="620" style="min-width:920px" empty-text="暂无策略信号，可先到策略中心运行分析">
        <el-table-column prop="symbol" label="代码" width="110" />
        <el-table-column prop="strategy_code" label="策略" width="170" />
        <el-table-column label="信号" width="100"><template #default="{ row }"><SignalTag :value="row.signal_type || row.action" /></template></el-table-column>
        <el-table-column label="强度" width="110"><template #default="{row}">{{Number(row.score||0).toFixed(0)}} / 100</template></el-table-column>
        <el-table-column label="风险" width="110"><template #default="{ row }"><RiskTag :value="row.risk_level" /></template></el-table-column>
        <el-table-column prop="reason" label="原因" />
        <el-table-column label="操作" width="100"><template #default="{ row }"><el-button size="small" @click="$router.push(`/market/stocks/${row.symbol}`)">详情</el-button></template></el-table-column>
      </el-table></div>
    </section>
    <DisclaimerAlert />
  </PageContainer>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import PageContainer from '@/components/common/PageContainer.vue'
import SignalTag from '@/components/common/SignalTag.vue'
import RiskTag from '@/components/common/RiskTag.vue'
import DisclaimerAlert from '@/components/common/DisclaimerAlert.vue'
import { strategyApi } from '@/api/strategy'
import type { StrategySignal } from '@/types/strategy'

const items = ref<StrategySignal[]>([])
const symbol = ref('')
const signalType = ref('')
const load = async () => {
  items.value = symbol.value ? await strategyApi.signalsBySymbol(symbol.value) : await strategyApi.signals({ signal_type: signalType.value, limit: 100 })
}
onMounted(load)
</script>
