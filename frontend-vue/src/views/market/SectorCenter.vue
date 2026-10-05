<template>
  <PageContainer title="板块强度" description="按行业统计涨跌广度、成交额与领涨股票">
    <template #actions><el-button :loading="loading" @click="load">刷新数据</el-button></template>
    <section class="data-table-card">
      <div class="toolbar"><strong>行业强弱排序</strong><span class="muted">强度分综合平均涨跌幅与上涨家数，不代表未来收益</span></div>
      <div class="table-scroll-x"><el-table :data="items" v-loading="loading" class="wide-table" empty-text="暂无板块数据">
        <el-table-column type="index" label="排名" width="70" />
		<el-table-column label="行业" min-width="150"><template #default="{row}"><el-link type="primary" @click="$router.push(`/market/sectors/${encodeURIComponent(row.code)}`)">{{row.name}}</el-link></template></el-table-column>
        <el-table-column label="强度分" width="150"><template #default="{row}">
          <el-progress v-if="hasValidStrength(row.status)" :percentage="row.strength_score" :stroke-width="10" />
          <span v-else class="muted">{{ sectorStatusLabel(row.status) }}</span>
          <small v-if="hasValidStrength(row.status)" class="muted status-label">{{ sectorStatusLabel(row.status) }}</small>
        </template></el-table-column>
        <el-table-column label="平均涨跌" width="110"><template #default="{row}"><PriceChange :value="row.average_change" percent-mode /></template></el-table-column>
        <el-table-column label="上涨 / 下跌" width="120"><template #default="{row}">{{row.rise_count}} / {{row.fall_count}}</template></el-table-column>
        <el-table-column label="成交额" width="120"><template #default="{row}">{{compactNumber(row.total_amount)}}</template></el-table-column>
		<el-table-column label="行情覆盖" width="110"><template #default="{row}">{{row.quoted_count}} / {{row.stock_count}}<br><small class="muted">{{(Number(row.coverage_rate||0)*100).toFixed(0)}}%</small></template></el-table-column>
        <el-table-column label="领涨股票" min-width="180"><template #default="{row}"><el-link type="primary" @click="$router.push(`/market/stocks/${row.leader_symbol}`)">{{row.leader_name}} {{row.leader_symbol}}</el-link></template></el-table-column>
        <el-table-column label="领涨幅" width="100"><template #default="{row}"><PriceChange :value="row.leader_change" percent-mode /></template></el-table-column>
      </el-table></div>
    </section>
    <DisclaimerAlert />
  </PageContainer>
</template>
<script setup lang="ts">
import { onMounted, ref } from 'vue'
import PageContainer from '@/components/common/PageContainer.vue';import DisclaimerAlert from '@/components/common/DisclaimerAlert.vue';import PriceChange from '@/components/common/PriceChange.vue';import { featureApi } from '@/api/feature';import { compactNumber } from '@/utils/format';import type { SectorStrength } from '@/types/feature'
const items=ref<SectorStrength[]>([]),loading=ref(false);const load=async()=>{loading.value=true;try{items.value=await featureApi.sectors()}finally{loading.value=false}};onMounted(load)
const hasValidStrength=(status:string)=>status==='AVAILABLE'||status==='CLOSED_SNAPSHOT'
const sectorStatusLabel=(status:string)=>({AVAILABLE:'盘中实时',CLOSED_SNAPSHOT:'收盘快照',STALE_SNAPSHOT:'数据陈旧',INSUFFICIENT_COVERAGE:'覆盖不足',SOURCE_PENDING:'数据源待接入'}[status]||'状态未知')
</script>
<style scoped>.wide-table{min-width:940px}.status-label{display:block;margin-top:4px}</style>
