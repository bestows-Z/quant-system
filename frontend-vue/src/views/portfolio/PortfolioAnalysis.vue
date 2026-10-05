<template>
  <PageContainer title="持仓分析" description="组合收益、风险预算、相关性、归因与压力测试">
    <template #actions>
      <el-button @click="$router.push(`/portfolio/${id}`)">管理持仓</el-button>
      <el-button type="primary" :loading="loading" @click="load(true)">重新分析</el-button>
    </template>

    <div class="status-strip">
      <div><span>分析版本</span><strong>{{ analysis?.analysis_version || '--' }}</strong></div>
      <div><span>行情时点</span><strong>{{ formatTime(analysis?.quote_time) }}</strong></div>
      <div><span>行情覆盖</span><strong>{{ percentage(analysis?.coverage_rate) }}</strong></div>
      <div><span>历史样本</span><strong>{{ advanced.observation_count || 0 }} 个交易日</strong></div>
      <el-tag :type="riskTag(analysis?.risk_level)" effect="dark">{{ riskText(analysis?.risk_level) }}</el-tag>
    </div>

    <el-alert
      v-for="warning in analysis?.warnings || []"
      :key="warning"
      :title="warning"
      type="warning"
      show-icon
      class="alert"
    />

    <div class="section-grid summary-grid">
      <DataCard class="span-3" label="当前市值" :value="money(analysis?.market_value)" />
      <DataCard class="span-3" label="总盈亏" :value="money(analysis?.total_profit)" />
      <DataCard class="span-3" label="持仓收益率" :value="percentage(analysis?.total_profit_rate)" />
      <DataCard class="span-3" label="最大单票权重" :value="percentage(analysis?.concentration)" />
    </div>

    <section class="analysis-workbench" v-loading="loading">
      <el-tabs v-model="activeTab">
        <el-tab-pane label="组合概览" name="overview">
          <div class="overview-grid">
            <div class="chart-panel">
              <div class="panel-heading">
                <div><strong>历史净值曲线</strong><span>{{ historyRange }}</span></div>
                <el-tag size="small" type="info">{{ historySource }}</el-tag>
              </div>
              <EquityCurveChart :data="advanced.equity_curve || []" />
            </div>
            <div class="metric-panel">
              <MetricRow label="年化收益率" :value="percentage(advanced.annual_return)" />
              <MetricRow label="年化波动率" :value="percentage(advanced.annual_volatility)" />
              <MetricRow label="最大回撤" :value="percentage(advanced.max_drawdown)" danger />
              <MetricRow label="Sharpe" :value="ratio(advanced.sharpe_ratio)" />
              <MetricRow label="有效持仓数" :value="ratio(advanced.effective_positions)" />
              <MetricRow label="历史覆盖率" :value="percentage(advanced.history_coverage)" />
            </div>
          </div>
          <div class="section-grid exposure-grid">
            <section class="data-table-card span-5">
              <div class="toolbar"><strong>行业暴露</strong></div>
              <el-table :data="analysis?.industry_exposures || []" empty-text="暂无行业数据">
                <el-table-column prop="industry" label="行业" min-width="110" />
                <el-table-column label="市值" min-width="110"><template #default="{ row }">{{ compactNumber(row.market_value) }}</template></el-table-column>
                <el-table-column label="权重" width="100"><template #default="{ row }">{{ percentage(row.weight) }}</template></el-table-column>
              </el-table>
            </section>
            <section class="data-table-card span-7">
              <div class="toolbar"><strong>组合结构</strong><span class="muted">权重越集中，单一风险暴露越高</span></div>
              <div class="structure-list">
                <div><span>HHI 集中度</span><strong>{{ ratio(advanced.hhi, 4) }}</strong><el-progress :percentage="percent(advanced.hhi)" :show-text="false" /></div>
                <div><span>有效持仓数</span><strong>{{ ratio(advanced.effective_positions) }}</strong><el-progress :percentage="effectivePositionPercent" :show-text="false" color="#0ca88c" /></div>
                <div><span>最大单票权重</span><strong>{{ percentage(analysis?.concentration) }}</strong><el-progress :percentage="percent(analysis?.concentration)" :show-text="false" color="#d97706" /></div>
              </div>
            </section>
          </div>
        </el-tab-pane>

        <el-tab-pane label="风险指标" name="risk">
          <div class="risk-grid">
            <article><span>年化波动率</span><strong>{{ percentage(advanced.annual_volatility) }}</strong><small>日收益标准差 × √252</small></article>
            <article><span>最大回撤</span><strong>{{ percentage(advanced.max_drawdown) }}</strong><small>历史净值峰谷最大跌幅</small></article>
            <article><span>VaR 95%</span><strong>{{ percentage(advanced.var_95) }}</strong><small>历史模拟单日损失阈值</small></article>
            <article><span>CVaR 95%</span><strong>{{ percentage(advanced.cvar_95) }}</strong><small>超过 VaR 后的平均损失</small></article>
            <article><span>Sharpe Ratio</span><strong>{{ ratio(advanced.sharpe_ratio) }}</strong><small>单位总风险的超额收益</small></article>
            <article><span>Sortino Ratio</span><strong>{{ ratio(advanced.sortino_ratio) }}</strong><small>单位下行风险的收益</small></article>
          </div>
          <el-alert title="VaR 与 CVaR 采用历史模拟法，不能覆盖停牌、流动性枯竭和极端跳空等全部风险。" type="info" :closable="false" show-icon />
        </el-tab-pane>

        <el-tab-pane label="归因分析" name="attribution">
          <div class="table-scroll-x">
            <el-table :data="advanced.contributions || []" class="wide-table" empty-text="历史数据不足，暂无归因结果">
              <el-table-column prop="symbol" label="代码" width="100" />
              <el-table-column prop="name" label="名称" min-width="120" />
              <el-table-column label="当前权重" width="120"><template #default="{ row }">{{ percentage(row.weight) }}</template></el-table-column>
              <el-table-column label="区间收益" width="120"><template #default="{ row }"><span :class="numberClass(row.period_return)">{{ percentage(row.period_return) }}</span></template></el-table-column>
              <el-table-column label="收益贡献" width="120"><template #default="{ row }"><span :class="numberClass(row.return_contribution)">{{ percentage(row.return_contribution) }}</span></template></el-table-column>
              <el-table-column label="风险贡献" width="120"><template #default="{ row }">{{ percentage(row.risk_contribution) }}</template></el-table-column>
              <el-table-column label="风险贡献占比" min-width="180"><template #default="{ row }"><el-progress :percentage="percent(row.risk_contribution)" :show-text="false" /></template></el-table-column>
            </el-table>
          </div>
        </el-tab-pane>

        <el-tab-pane label="相关性" name="correlation">
          <div class="matrix-note">相关系数接近 1 表示同向波动较强，接近 -1 表示反向波动较强。</div>
          <div class="table-scroll-x">
            <table v-if="correlationSymbols.length" class="correlation-table">
              <thead><tr><th>代码</th><th v-for="symbol in correlationSymbols" :key="symbol">{{ symbol }}</th></tr></thead>
              <tbody>
                <tr v-for="row in advanced.correlations" :key="row.symbol">
                  <th>{{ row.symbol }}</th>
                  <td v-for="symbol in correlationSymbols" :key="symbol" :style="correlationStyle(row.values[symbol])" :aria-label="correlationAria(row.symbol, symbol, row.values[symbol])">{{ correlationLabel(row.values[symbol]) }}</td>
                </tr>
              </tbody>
            </table>
            <el-empty v-else description="至少需要两只具有重叠历史数据的持仓" />
          </div>
        </el-tab-pane>

        <el-tab-pane label="压力测试" name="scenario">
          <div class="scenario-grid">
            <article v-for="item in advanced.scenarios || []" :key="item.code">
              <div><strong>{{ item.name }}</strong><el-tag :type="riskTag(item.level)" size="small">{{ riskText(item.level) }}</el-tag></div>
              <p>{{ item.description }}</p>
              <dl><dt>预计损失率</dt><dd>{{ percentage(item.loss_rate) }}</dd><dt>预计损失额</dt><dd>{{ money(item.loss_amount) }}</dd></dl>
            </article>
          </div>
        </el-tab-pane>

        <el-tab-pane label="逐票持仓" name="positions">
          <div class="toolbar"><span class="muted">{{ analysis?.data_source || '--' }} / {{ analysis?.data_mode || '--' }}</span></div>
          <div class="table-scroll-x">
            <el-table :data="analysis?.positions || []" class="position-table">
              <el-table-column prop="symbol" label="代码" width="95" />
              <el-table-column prop="name" label="名称" width="120" />
              <el-table-column prop="industry" label="行业" width="120" />
              <el-table-column label="持仓" width="100"><template #default="{ row }">{{ compactNumber(row.quantity) }}</template></el-table-column>
              <el-table-column label="现价" width="100"><template #default="{ row }">{{ money(row.market_price) }}</template></el-table-column>
              <el-table-column label="市值" width="120"><template #default="{ row }">{{ compactNumber(row.market_value) }}</template></el-table-column>
              <el-table-column label="盈亏" width="110"><template #default="{ row }"><span :class="numberClass(row.profit)">{{ money(row.profit) }}</span></template></el-table-column>
              <el-table-column label="收益率" width="105"><template #default="{ row }"><span :class="numberClass(row.profit_rate)">{{ percentage(row.profit_rate) }}</span></template></el-table-column>
              <el-table-column label="权重" width="95"><template #default="{ row }">{{ percentage(row.weight) }}</template></el-table-column>
              <el-table-column label="风险" width="90"><template #default="{ row }"><el-tag :type="riskTag(row.risk_level)" size="small">{{ riskText(row.risk_level) }}</el-tag></template></el-table-column>
              <el-table-column label="操作" width="100" fixed="right"><template #default="{ row }"><el-button size="small" @click="$router.push(`/market/stocks/${row.symbol}`)">研判</el-button></template></el-table-column>
            </el-table>
          </div>
        </el-tab-pane>
      </el-tabs>
    </section>

    <el-alert :title="analysis?.disclaimer || '仅供研究，不构成投资建议'" type="info" show-icon :closable="false" class="disclaimer" />
  </PageContainer>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import PageContainer from '@/components/common/PageContainer.vue'
import DataCard from '@/components/common/DataCard.vue'
import EquityCurveChart from '@/components/charts/EquityCurveChart.vue'
import { portfolioApi } from '@/api/portfolio'
import { compactNumber, money } from '@/utils/format'
import type { PortfolioAdvancedAnalysis, PortfolioAnalysis } from '@/types/portfolio'
import { formatTime } from '@/utils/time'

const emptyAdvanced: PortfolioAdvancedAnalysis = {
  observation_count: 0, start_date: '', end_date: '', history_coverage: 0,
  annual_volatility: 0, annual_return: 0, sharpe_ratio: 0, sortino_ratio: 0,
  max_drawdown: 0, var_95: 0, cvar_95: 0, hhi: 0, effective_positions: 0,
  equity_curve: [], contributions: [], correlations: [], scenarios: [], history_sources: [], history_modes: []
}

const MetricRow = defineComponent({
  props: { label: String, value: String, danger: Boolean },
  setup: (props) => () => h('div', { class: 'metric-row' }, [h('span', props.label), h('strong', { class: props.danger ? 'is-danger' : '' }, props.value)])
})

const route = useRoute()
const id = computed(() => Number(route.params.id))
const analysis = ref<PortfolioAnalysis>()
const loading = ref(false)
const activeTab = ref('overview')
const advanced = computed<PortfolioAdvancedAnalysis>(() => {
  try { return analysis.value?.advanced_json ? { ...emptyAdvanced, ...JSON.parse(analysis.value.advanced_json) } : emptyAdvanced }
  catch { return emptyAdvanced }
})
const correlationSymbols = computed(() => advanced.value.correlations.map((item) => item.symbol))
const historyRange = computed(() => advanced.value.start_date ? `${advanced.value.start_date} 至 ${advanced.value.end_date}` : '历史数据不足')
const historySource = computed(() => [...advanced.value.history_sources, ...advanced.value.history_modes].filter(Boolean).join(' / ') || '暂无来源')
const effectivePositionPercent = computed(() => {
  const count = analysis.value?.positions?.length || 0
  return count ? Math.min(100, Math.max(0, advanced.value.effective_positions / count * 100)) : 0
})

const percentage = (value?: string | number) => value === undefined || value === null || !Number.isFinite(Number(value)) ? '--' : `${(Number(value) * 100).toFixed(2)}%`
const ratio = (value?: string | number, digits = 2) => value === undefined || value === null || !Number.isFinite(Number(value)) ? '--' : Number(value).toFixed(digits)
const percent = (value?: string | number) => Math.min(100, Math.max(0, Number(value || 0) * 100))
const numberClass = (value?: string | number) => Number(value || 0) > 0 ? 'is-up' : Number(value || 0) < 0 ? 'is-down' : ''
const riskText = (value?: string) => ({ LOW: '低风险', MEDIUM: '中风险', HIGH: '高风险', EXTREME: '极高风险' }[value || ''] || value || '--')
const riskTag = (value?: string) => value === 'LOW' ? 'success' : value === 'MEDIUM' ? 'warning' : value === 'HIGH' || value === 'EXTREME' ? 'danger' : 'info'
const correlationStyle = (value: number | null = 0) => {
	if (value === null) return { background: 'var(--el-fill-color-light)', color: 'var(--el-text-color-secondary)' }
  const alpha = Math.min(0.34, Math.abs(value) * 0.34)
  return { background: value >= 0 ? `rgba(202, 51, 54, ${alpha})` : `rgba(12, 168, 140, ${alpha})` }
}
const correlationLabel = (value: number | null) => value === null ? '样本不足' : `${value > 0 ? '+' : ''}${value.toFixed(2)}`
const correlationAria = (left: string, right: string, value: number | null) => value === null ? `${left} 与 ${right} 相关性样本不足` : `${left} 与 ${right} ${value >= 0 ? '正' : '负'}相关 ${Math.abs(value).toFixed(2)}`
const load = async (refresh = false) => {
  loading.value = true
  try { analysis.value = await portfolioApi.analysis(id.value, refresh) }
  catch (error) {
    if (!refresh) analysis.value = await portfolioApi.analysis(id.value, true)
    else throw error
  }
  finally { loading.value = false }
}
onMounted(() => load())
watch(id, (next, previous) => { if (next && next !== previous) load() })
</script>

<style scoped>
.status-strip{display:flex;align-items:center;gap:24px;padding:12px 16px;margin-bottom:12px;background:var(--el-fill-color-light);border:1px solid var(--el-border-color-lighter);border-radius:6px;overflow-x:auto}.status-strip>div{display:grid;gap:3px;min-width:max-content}.status-strip span,.panel-heading span{font-size:12px;color:var(--el-text-color-secondary)}.status-strip strong{font-size:14px;font-variant-numeric:tabular-nums}.status-strip .el-tag{margin-left:auto}.alert{margin-bottom:10px}.summary-grid{margin-bottom:16px}.analysis-workbench{padding:0 16px 16px;background:var(--el-bg-color);border:1px solid var(--el-border-color-lighter);border-radius:6px;min-height:440px}.overview-grid{display:grid;grid-template-columns:minmax(0,2fr) minmax(260px,1fr);gap:16px}.chart-panel,.metric-panel{border:1px solid var(--el-border-color-lighter);border-radius:6px}.panel-heading{display:flex;justify-content:space-between;align-items:center;padding:14px 16px 0}.panel-heading>div{display:grid;gap:4px}.metric-panel{padding:4px 16px}.metric-row{display:flex;justify-content:space-between;align-items:center;min-height:52px;border-bottom:1px solid var(--el-border-color-lighter)}.metric-row:last-child{border-bottom:0}.metric-row strong{font-size:17px;font-variant-numeric:tabular-nums}.is-danger,.is-down{color:var(--el-color-danger)}.is-up{color:#ca3336}.exposure-grid{margin-top:16px}.structure-list{padding:8px 16px 16px}.structure-list>div{display:grid;grid-template-columns:1fr auto;gap:8px;padding:10px 0}.structure-list .el-progress{grid-column:1/-1}.risk-grid{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:12px;margin-bottom:16px}.risk-grid article,.scenario-grid article{padding:16px;border:1px solid var(--el-border-color-lighter);border-radius:6px;background:var(--el-fill-color-blank)}.risk-grid article{display:grid;gap:8px}.risk-grid strong{font-size:24px;font-variant-numeric:tabular-nums}.risk-grid small,.matrix-note,.scenario-grid p{color:var(--el-text-color-secondary);line-height:1.5}.wide-table{min-width:900px}.matrix-note{margin-bottom:12px}.correlation-table{border-collapse:collapse;min-width:max-content;width:100%;font-variant-numeric:tabular-nums}.correlation-table th,.correlation-table td{min-width:86px;padding:12px;text-align:center;border:1px solid var(--el-border-color-lighter)}.correlation-table th{background:var(--el-fill-color-light);font-weight:600}.scenario-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:12px}.scenario-grid article>div{display:flex;align-items:center;justify-content:space-between;gap:12px}.scenario-grid dl{display:grid;grid-template-columns:1fr auto;gap:8px;margin:16px 0 0}.scenario-grid dt{color:var(--el-text-color-secondary)}.scenario-grid dd{margin:0;font-weight:600;font-variant-numeric:tabular-nums}.position-table{min-width:1120px}.disclaimer{margin-top:14px}@media(max-width:900px){.overview-grid{grid-template-columns:1fr}.risk-grid{grid-template-columns:repeat(2,minmax(0,1fr))}}@media(max-width:640px){.status-strip{gap:16px}.status-strip .el-tag{margin-left:0}.summary-grid{grid-template-columns:repeat(2,minmax(0,1fr))}.summary-grid :deep(.span-3){grid-column:span 1}.analysis-workbench{padding:0 10px 10px}.risk-grid,.scenario-grid{grid-template-columns:1fr}.risk-grid strong{font-size:20px}.panel-heading{align-items:flex-start;gap:8px}.chart-panel :deep(.chart){height:260px}}
</style>
