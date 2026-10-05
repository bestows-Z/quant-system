<template>
  <PageContainer title="市场全景" description="全市场涨跌结构、成交活跃度与板块强弱">
    <template #actions><el-button :loading="loading" @click="load">刷新数据</el-button></template>
    <el-alert title="市场温度是涨跌家数与平均涨跌幅的合成观察值，仅用于描述盘面状态" type="info" show-icon :closable="false" class="notice" />
    <section class="data-table-card index-panel">
      <div class="toolbar"><strong>核心指数</strong><span class="muted">沪、深、科创、创业、沪深核心与北交所</span></div>
      <div class="index-grid">
        <article v-for="item in indices" :key="item.market_code">
          <div><strong>{{item.name}}</strong><span>{{item.board}}</span></div>
          <b class="mono">{{money(item.price)}}</b>
          <PriceChange :value="item.change_pct" percent-mode />
          <small>成交额 {{compactNumber(item.amount)}}</small>
        </article>
      </div>
      <el-empty v-if="!indices.length" description="核心指数实时数据暂不可用" />
    </section>
    <div class="section-grid">
      <DataCard class="span-3" label="市场温度" :value="`${money(data.overview?.market_temperature, 0)} / 100`" :sub="temperatureText" />
      <DataCard class="span-3" label="上涨 / 下跌" :value="`${data.overview?.rise_count || 0} / ${data.overview?.fall_count || 0}`" :sub="`平盘 ${data.overview?.flat_count || 0} 家`" />
      <DataCard class="span-3" label="涨停 / 跌停" :value="`${data.overview?.limit_up_count || 0} / ${data.overview?.limit_down_count || 0}`" sub="按当前涨跌幅近似统计" />
      <DataCard class="span-3" label="全市场成交额" :value="compactNumber(data.overview?.total_amount)" :sub="`覆盖 ${data.overview?.total || 0} 只交易中证券`" />
    </div>
    <div class="section-grid content-band">
      <section class="data-table-card span-7">
        <div class="toolbar"><strong>涨跌分布</strong><span class="muted">更新时间 {{ formatTime(data.overview?.quote_time) }}</span></div>
        <div v-if="data.distribution?.length" class="distribution">
          <div v-for="item in data.distribution" :key="item.label" class="distribution-row">
            <span>{{ item.label }}</span><el-progress :percentage="distributionPct(item.count)" :stroke-width="14" :show-text="false" /><strong class="mono">{{ item.count }} 家</strong>
          </div>
        </div>
        <el-empty v-else description="暂无全市场快照，请先刷新行情数据" />
      </section>
      <section class="data-table-card span-5">
        <div class="toolbar"><strong>盘面解读</strong></div>
        <el-descriptions :column="1" border>
          <el-descriptions-item label="上涨占比">{{ percent(data.overview?.rise_ratio) }}</el-descriptions-item>
          <el-descriptions-item label="平均涨跌幅">{{ percent(data.overview?.average_change) }}</el-descriptions-item>
          <el-descriptions-item label="涨跌幅中位数">{{ percent(data.overview?.median_change) }}</el-descriptions-item>
          <el-descriptions-item label="统计范围">{{ data.overview?.scope === 'all_a_shares' ? '全 A 股快照' : '数据待确认' }}</el-descriptions-item>
          <el-descriptions-item label="数据来源">{{ data.overview?.source || '--' }}</el-descriptions-item>
        </el-descriptions>
        <p class="summary">{{ summary }}</p>
      </section>
    </div>
    <DisclaimerAlert />
  </PageContainer>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import PageContainer from '@/components/common/PageContainer.vue'
import DataCard from '@/components/common/DataCard.vue'
import DisclaimerAlert from '@/components/common/DisclaimerAlert.vue'
import PriceChange from '@/components/common/PriceChange.vue'
import { featureApi } from '@/api/feature'
import { marketApi } from '@/api/market'
import { compactNumber, money, percent } from '@/utils/format'
import type { MarketOverviewResult } from '@/types/feature'
import type { MarketIndex } from '@/types/market'
import { formatTime } from '@/utils/time'

const data = ref<MarketOverviewResult>({ overview: {} as MarketOverviewResult['overview'], distribution: [] })
const loading = ref(false)
const indices = ref<MarketIndex[]>([])
const load = async () => { loading.value = true; try { const [overview, indexRows] = await Promise.allSettled([featureApi.overview(), marketApi.indices()]); if (overview.status === 'fulfilled') data.value = overview.value; if (indexRows.status === 'fulfilled') indices.value = indexRows.value } finally { loading.value = false } }
const distributionPct = (count: number) => Math.round(count / Math.max(data.value.overview?.total || 1, 1) * 100)
const temperatureText = computed(() => { const v = data.value.overview?.market_temperature || 0; return v >= 70 ? '盘面偏热' : v >= 45 ? '盘面中性' : '盘面偏冷' })
const summary = computed(() => { const o = data.value.overview; if (!o?.total) return '当前快照覆盖不足，暂不生成盘面结论。'; if (o.rise_ratio >= .6) return '上涨家数占优，市场广度偏强；仍需结合成交额、板块持续性和风险水平观察。'; if (o.rise_ratio <= .4) return '下跌家数占优，市场广度偏弱，应重点关注波动和仓位集中风险。'; return '涨跌家数相对均衡，盘面方向性有限，可继续观察成交量与领涨板块变化。' })
onMounted(load)
</script>

<style scoped>.notice{margin-bottom:16px}.index-panel{margin-bottom:16px}.index-grid{display:grid;grid-template-columns:repeat(6,minmax(0,1fr));gap:1px;background:var(--qs-border);border:1px solid var(--qs-border)}.index-grid article{display:grid;gap:7px;min-width:0;padding:12px;background:var(--qs-surface)}.index-grid article>div{display:flex;justify-content:space-between;gap:6px}.index-grid span,.index-grid small{color:var(--qs-muted);font-size:12px}.index-grid b{font-size:20px}.content-band{margin-top:16px}.distribution{display:grid;gap:14px}.distribution-row{display:grid;grid-template-columns:110px minmax(100px,1fr) 64px;align-items:center;gap:12px}.distribution-row strong{text-align:right}.summary{margin:16px 0 0;line-height:1.8;color:var(--qs-muted)}@media(max-width:1100px){.index-grid{grid-template-columns:repeat(3,minmax(0,1fr))}}@media(max-width:560px){.index-grid{grid-template-columns:repeat(2,minmax(0,1fr))}.index-grid b{font-size:17px}.distribution-row{grid-template-columns:92px minmax(70px,1fr) 58px;font-size:12px}}</style>
