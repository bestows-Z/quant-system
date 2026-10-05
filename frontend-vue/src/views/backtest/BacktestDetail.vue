<template>
  <PageContainer title="回测报告" :description="task ? `${task.symbol} · ${strategyName(task.strategy_code)} · ${task.start_date} 至 ${task.end_date}` : '绩效、成交与数据证据'">
    <template #actions><el-button :loading="loading" @click="load">刷新报告</el-button><el-button :disabled="!result" @click="analyzeRisk">风险分析</el-button><el-button :disabled="!result" @click="reviewAgent">研究复盘</el-button></template>
    <el-alert v-if="task?.status==='FAILED'" :title="task.error_message || '回测未完成'" type="error" show-icon :closable="false" class="report-alert" />
    <el-alert v-if="isLegacy" title="旧版回测缺少执行模型或数据质量证据，请重新运行后再用于策略判断。" type="warning" show-icon :closable="false" class="report-alert" />
    <el-alert v-else-if="quality.status==='PARTIAL'" title="历史交易规则证据不完整。本次结果可供复盘，不能用于策略晋级。" type="warning" show-icon :closable="false" class="report-alert" />
    <div v-if="result" class="report-meta"><span class="mono">#{{ task?.task_no || id }}</span><span>{{ result.data_source || task?.data_source || '来源待核验' }}</span><span>{{ result.adjust === 'qfq' ? '前复权' : '复权未核验' }}</span><span>{{ result.execution_model === 'NEXT_OPEN_V2' ? '次日开盘执行' : '执行模型未核验' }}</span><el-tag v-if="quality.status==='PASS'" type="success" effect="plain">数据质量通过</el-tag><el-tag v-else-if="quality.status==='PARTIAL'" type="warning" effect="plain">规则待核验</el-tag></div>
    <div v-if="result" class="metric-grid">
      <div class="metric-card"><span>总收益率</span><strong class="mono" :class="returnClass(result.total_return)">{{ percent(result.total_return) }}</strong><small>期末资产 {{ amount(result.final_asset) }} 元</small></div>
      <div class="metric-card"><span>年化收益率</span><strong class="mono" :class="returnClass(result.annual_return)">{{ percent(result.annual_return) }}</strong><small>按回测区间折算</small></div>
      <div class="metric-card"><span>最大回撤</span><strong class="mono">{{ percent(result.max_drawdown) }}</strong><small>权益曲线峰值至谷值</small></div>
      <div class="metric-card"><span>胜率</span><strong class="mono">{{ percent(result.win_rate) }}</strong><small>{{ result.total_trades || trades.length }} 笔成交</small></div>
    </div>
    <section class="data-table-card chart-card" aria-labelledby="equity-title"><div class="section-heading"><div><span class="eyebrow">绩效</span><h2 id="equity-title">权益曲线</h2></div><span class="muted">{{ curve.length }} 个交易日</span></div><EquityCurveChart v-if="curve.length >= 2" :data="curve" /><el-empty v-else description="暂无可绘制的权益曲线" /></section>
    <section class="data-table-card detail-card" aria-label="回测证据明细">
      <el-tabs v-model="activeTab">
        <el-tab-pane label="执行与数据" name="evidence">
          <div class="evidence-grid">
            <div class="evidence-block">
              <h3>数据质量</h3>
              <dl>
                <div><dt>状态</dt><dd>{{ qualityLabel }}</dd></div>
                <div><dt>有效 K 线</dt><dd>{{ quality.valid_rows ?? '—' }}</dd></div>
                <div><dt>数据来源</dt><dd>{{ quality.source || result?.data_source || '—' }}</dd></div>
                <div><dt>规则覆盖</dt><dd>{{ ruleCoverageLabel }}</dd></div>
                <div v-if="quality.verified_status_rows !== undefined"><dt>状态核验</dt><dd>{{ quality.verified_status_rows }} / {{ quality.valid_rows }} 日</dd></div>
                <div v-if="quality.verified_limit_rows !== undefined"><dt>限价核验</dt><dd>{{ quality.verified_limit_rows }} / {{ quality.valid_rows }} 日</dd></div>
                <div><dt>引擎版本</dt><dd class="mono">{{ result?.engine_version || '未记录' }}</dd></div>
              </dl>
              <ul v-if="quality.issues?.length" class="issue-list"><li v-for="issue in quality.issues" :key="issue">{{ issue }}</li></ul>
              <p v-else class="muted">{{ isLegacy ? '旧版结果没有保存质量证据。' : '未发现已记录的数据质量问题。' }}</p>
            </div>
            <div class="evidence-block"><h3>执行与成本</h3><dl><div><dt>执行模型</dt><dd>{{ result?.execution_model === 'NEXT_OPEN_V2' ? '收盘信号 → 下一交易日开盘' : '未记录' }}</dd></div><div><dt>复权方式</dt><dd>{{ result?.adjust === 'qfq' ? '前复权' : '未记录' }}</dd></div><div><dt>总费用</dt><dd>{{ amount(result?.total_fees) }} 元</dd></div><div><dt>已实现盈亏</dt><dd>{{ amount(result?.realized_pnl) }} 元</dd></div><div><dt>未实现盈亏</dt><dd>{{ amount(result?.unrealized_pnl) }} 元</dd></div><div><dt>换手率</dt><dd>{{ percent(result?.turnover_rate) }}</dd></div><div><dt>平均仓位</dt><dd>{{ percent(result?.exposure_rate) }}</dd></div></dl></div>
            <div class="evidence-block snapshot-block"><h3>参数快照</h3><div class="snapshot-grid"><div><h4>策略参数</h4><dl v-if="parameterRows.length"><div v-for="item in parameterRows" :key="item.key"><dt>{{item.label}}</dt><dd class="mono">{{item.value}}</dd></div></dl><p v-else class="muted">无可调参数或旧版未记录</p></div><div><h4>成交假设</h4><dl v-if="configRows.length"><div v-for="item in configRows" :key="item.key"><dt>{{item.label}}</dt><dd class="mono">{{item.value}}</dd></div></dl><p v-else class="muted">旧版未记录成交参数</p></div></div></div>
          </div>
        </el-tab-pane>
        <el-tab-pane :label="`委托记录 ${orders.length}`" name="orders"><div class="table-scroll-x"><el-table :data="orders" empty-text="没有委托记录"><el-table-column prop="signal_date" label="信号日" min-width="110"/><el-table-column prop="trade_date" label="计划成交日" min-width="115"><template #default="{row}">{{row.trade_date||'无下一交易日'}}</template></el-table-column><el-table-column label="方向" width="80"><template #default="{row}">{{sideLabel(row.side)}}</template></el-table-column><el-table-column prop="requested_quantity" label="委托股数" min-width="100"/><el-table-column prop="filled_quantity" label="成交股数" min-width="100"/><el-table-column label="结果" min-width="100"><template #default="{row}"><el-tag :type="orderTag(row.status)" effect="plain">{{orderStatus(row.status)}}</el-tag></template></el-table-column><el-table-column label="原因" min-width="250"><template #default="{row}">{{row.reason || row.reason_code || '—'}}</template></el-table-column></el-table></div></el-tab-pane>
        <el-tab-pane :label="`成交明细 ${trades.length}`" name="trades"><div class="table-scroll-x"><el-table :data="trades" empty-text="该区间没有实际成交"><el-table-column prop="signal_date" label="信号日" min-width="105"/><el-table-column prop="trade_time" label="成交日" min-width="105"/><el-table-column label="方向" width="75"><template #default="{row}">{{sideLabel(row.side)}}</template></el-table-column><el-table-column label="成交价" min-width="100"><template #default="{row}">{{amount(row.price,4)}}</template></el-table-column><el-table-column prop="volume" label="股数" min-width="90"/><el-table-column label="金额" min-width="110"><template #default="{row}">{{amount(row.amount)}}</template></el-table-column><el-table-column label="总费用" min-width="95"><template #default="{row}">{{amount(row.fee)}}</template></el-table-column><el-table-column label="佣金 / 印花税 / 过户费" min-width="205"><template #default="{row}"><span class="mono">{{amount(row.commission)}} / {{amount(row.stamp_duty)}} / {{amount(row.transfer_fee)}}</span></template></el-table-column></el-table></div></el-tab-pane>
      </el-tabs>
    </section>
  </PageContainer>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import PageContainer from '@/components/common/PageContainer.vue'
import EquityCurveChart from '@/components/charts/EquityCurveChart.vue'
import { backtestApi } from '@/api/backtest'
import { riskApi } from '@/api/risk'
import { agentApi } from '@/api/agent'
import type { BacktestOrder, BacktestQuality, BacktestResult, BacktestTask, BacktestTrade, EquityPoint } from '@/types/backtest'

const id = Number(useRoute().params.id)
const task = ref<BacktestTask>()
const result = ref<BacktestResult>()
const orders = ref<BacktestOrder[]>([])
const trades = ref<BacktestTrade[]>([])
const curve = ref<EquityPoint[]>([])
const loading = ref(false)
const activeTab = ref('evidence')
const quality = computed<BacktestQuality>(() => { try { return JSON.parse(result.value?.quality_json || '{}') as BacktestQuality } catch { return {} } })
const parseRecord = (value?: string): Record<string, unknown> => { try { const parsed: unknown = JSON.parse(value || '{}'); return parsed && typeof parsed === 'object' && !Array.isArray(parsed) ? parsed as Record<string, unknown> : {} } catch { return {} } }
const parameterLabels: Record<string, string> = {short_window:'短期均线',long_window:'长期均线',fast:'快线周期',slow:'慢线周期',signal:'信号线周期',period:'统计周期',low:'超卖阈值',high:'超买阈值',std:'标准差倍数',volume_window:'成交量观察期',volume_multiplier:'放量倍数'}
const parameterOrder = ['short_window','long_window','fast','slow','signal','period','low','high','std','volume_window','volume_multiplier']
const parameterRows = computed(() => Object.entries(parseRecord(result.value?.params_json)).sort(([a],[b]) => parameterOrder.indexOf(a) - parameterOrder.indexOf(b)).map(([key, value]) => ({key, label: key === 'period' && task.value?.strategy_code === 'rsi_reversal' ? 'RSI 周期' : parameterLabels[key] || key, value: String(value)})))
const ratePercent = (value: unknown) => `${Number((Number(value) * 100).toFixed(4))}%`
const configRows = computed(() => { const config = parseRecord(result.value?.config_json); return [
  {key:'commission_rate',label:'佣金率',rate:true},{key:'min_commission',label:'最低佣金',unit:'元'},
  {key:'stamp_duty_rate',label:'卖出印花税',rate:true},{key:'transfer_fee_rate',label:'过户费率',rate:true},
  {key:'slippage_rate',label:'滑点率',rate:true},{key:'max_volume_ratio',label:'最大成交量参与率',rate:true},
  {key:'target_position_ratio',label:'目标资金使用率',rate:true},{key:'lot_size',label:'买入单位',unit:'股'}
].filter(item => config[item.key] !== undefined).map(item => ({key:item.key,label:item.label,value:item.rate ? ratePercent(config[item.key]) : `${config[item.key]}${item.unit || ''}`})) })
const qualityLabel = computed(() => quality.value.status === 'PASS' ? '通过' : quality.value.status === 'PARTIAL' ? '规则证据不完整' : '未记录')
const ruleCoverageLabel = computed(() => ({
  FULL: '逐日限价与证券状态已核验',
  STATUS_VERIFIED_LIMIT_INFERRED: '证券状态已核验，限价仍为推断',
  INFERRED: '证券状态与限价均未逐日核验',
}[quality.value.rule_coverage || ''] || '未记录'))
const isLegacy = computed(() => Boolean(result.value && (!result.value.engine_version || !result.value.execution_model || !result.value.quality_status)))
const strategyName = (value?: string) => ({ma_cross:'均线交叉',macd_cross:'MACD 交叉',rsi_reversal:'RSI 反转',boll_breakout:'布林带突破',volume_breakout:'放量突破',trend_score:'趋势综合评分'}[value || ''] || value || '策略')
const sideLabel = (value?: string) => value === 'BUY' ? '买入' : value === 'SELL' ? '卖出' : value || '—'
const orderStatus = (value?: string) => ({FILLED:'已成交',PARTIALLY_FILLED:'部分成交',REJECTED:'未成交',CANCELLED:'已取消'}[value || ''] || value || '—')
const orderTag = (value?: string) => value === 'FILLED' ? 'success' : value === 'PARTIALLY_FILLED' ? 'warning' : 'info'
const percent = (value?: number) => Number.isFinite(Number(value)) ? `${(Number(value) * 100).toFixed(2)}%` : '—'
const amount = (value?: number | string, digits = 2) => value !== undefined && value !== null && value !== '' && Number.isFinite(Number(value)) ? Number(value).toLocaleString('zh-CN', { minimumFractionDigits: digits, maximumFractionDigits: digits }) : '—'
const returnClass = (value?: number) => Number(value) > 0 ? 'positive' : Number(value) < 0 ? 'negative' : ''
const load = async () => {
  loading.value = true
  try {
    const responses = await Promise.allSettled([backtestApi.task(id), backtestApi.result(id), backtestApi.orders(id), backtestApi.trades(id), backtestApi.equityCurve(id)])
    if (responses[0].status === 'fulfilled') task.value = responses[0].value
    if (responses[1].status === 'fulfilled') result.value = responses[1].value
    if (responses[2].status === 'fulfilled') orders.value = responses[2].value
    if (responses[3].status === 'fulfilled') trades.value = responses[3].value
    if (responses[4].status === 'fulfilled') curve.value = responses[4].value
    if (responses[0].status === 'rejected') ElMessage.error('任务信息加载失败，请重试')
  } finally { loading.value = false }
}
const analyzeRisk = async () => { await riskApi.analyzeBacktest(id); ElMessage.success('风险分析已提交') }
const reviewAgent = async () => { await agentApi.reviewBacktest({ backtest_id: id, question: '请核对该回测的数据质量、收益、回撤、交易成本和未成交原因' }); ElMessage.success('研究复盘已提交') }
onMounted(load)
</script>

<style scoped>
.report-alert{margin-bottom:14px}.report-meta{display:flex;flex-wrap:wrap;align-items:center;gap:8px 14px;margin-bottom:16px;color:var(--qs-muted);font-size:13px}.report-meta span+span{border-left:1px solid var(--qs-border);padding-left:14px}.metric-grid{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:14px;margin-bottom:16px}.metric-card{padding:18px;background:var(--qs-surface);border:1px solid var(--qs-border);border-radius:10px;display:flex;flex-direction:column;gap:8px}.metric-card>span,.metric-card small{color:var(--qs-muted)}.metric-card strong{font-size:24px;font-weight:650}.metric-card small{font-size:12px}.positive{color:var(--qs-up)}.negative{color:var(--qs-down)}.chart-card,.detail-card{margin-bottom:16px}.section-heading{display:flex;justify-content:space-between;align-items:center;margin-bottom:12px}.section-heading h2{margin:3px 0 0;font-size:18px}.eyebrow{font-size:11px;font-weight:700;letter-spacing:.08em;color:var(--qs-primary-dark)}.evidence-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:16px;padding:6px 0 10px}.evidence-block{padding:18px;border:1px solid var(--qs-border);border-radius:8px}.evidence-block h3{margin:0 0 14px;font-size:15px}.evidence-block h4{margin:0 0 10px;font-size:13px}.evidence-block dl{margin:0}.evidence-block dl>div{display:flex;justify-content:space-between;gap:12px;border-top:1px solid var(--qs-border);padding:10px 0}.evidence-block dt{color:var(--qs-muted)}.evidence-block dd{margin:0;text-align:right}.snapshot-block{grid-column:1/-1}.snapshot-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:24px}.issue-list{margin:10px 0 0;padding-left:20px;color:var(--qs-warning);line-height:1.7}.table-scroll-x :deep(.el-table){min-width:760px}@media(max-width:1050px){.metric-grid{grid-template-columns:repeat(2,minmax(0,1fr))}}@media(max-width:700px){.evidence-grid,.snapshot-grid{grid-template-columns:1fr}.metric-grid{gap:8px}.metric-card{padding:12px}.metric-card strong{font-size:19px}.report-meta span+span{padding-left:0;border:0}}
.evidence-block dd{min-width:0;overflow-wrap:anywhere}
</style>
