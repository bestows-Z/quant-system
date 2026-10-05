<template>
  <PageContainer title="回测工作台" description="配置策略与执行规则，逐笔核对结果和数据质量">
    <div class="workspace-grid">
      <section class="data-table-card setup-card" aria-labelledby="setup-title">
        <div class="section-heading"><div><span class="eyebrow">01 / 配置</span><h2 id="setup-title">新建回测</h2></div><span class="muted">A 股 · 日线</span></div>
        <el-form label-position="top" :model="form" class="backtest-form" @submit.prevent="run">
          <div class="form-grid">
            <el-form-item label="股票代码" required><el-input v-model.trim="form.symbol" maxlength="6" inputmode="numeric" placeholder="例如 600176" clearable /></el-form-item>
            <el-form-item label="策略" required><el-select v-model="form.strategy_code" aria-label="选择回测策略"><el-option v-for="item in strategies" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item>
            <el-form-item label="开始日期" required><el-date-picker v-model="form.start_date" type="date" value-format="YYYY-MM-DD" :disabled-date="disableFuture" /></el-form-item>
            <el-form-item label="结束日期" required><el-date-picker v-model="form.end_date" type="date" value-format="YYYY-MM-DD" :disabled-date="disableFuture" /></el-form-item>
            <el-form-item label="初始资金（元）" required><el-input-number v-model="form.initial_cash" :min="1000" :max="1000000000000" :step="1000" controls-position="right" /></el-form-item>
          </div>
          <p class="strategy-note">{{ activeStrategy.description }}</p>
          <div class="parameter-heading"><strong>策略参数</strong><span>参数会随报告保存，便于复现同一次实验</span></div>
          <div v-if="activeParameters.length" class="form-grid parameter-grid">
            <el-form-item v-for="item in activeParameters" :key="item.key" :label="item.label">
              <el-input-number v-model="strategyParams[item.key]" :min="parameterMinimum(item)" :max="item.max" :step="item.step || 1" :precision="item.precision || 0" controls-position="right" />
            </el-form-item>
          </div>
          <p v-else class="strategy-note">该策略使用固定评分规则，当前没有可调整参数。</p>
          <el-collapse class="advanced-settings"><el-collapse-item name="execution" title="交易成本与执行参数">
            <div class="form-grid">
              <el-form-item label="佣金率（%）"><el-input-number v-model="form.commission_percent" :min="0" :max="5" :precision="4" :step="0.01" controls-position="right" /></el-form-item>
              <el-form-item label="最低佣金（元）"><el-input-number v-model="form.min_commission" :min="0" :max="1000" :precision="2" controls-position="right" /></el-form-item>
              <el-form-item label="卖出印花税（%）"><el-input-number v-model="form.stamp_duty_percent" :min="0" :max="5" :precision="4" :step="0.01" controls-position="right" /></el-form-item>
              <el-form-item label="过户费率（%）"><el-input-number v-model="form.transfer_fee_percent" :min="0" :max="1" :precision="4" :step="0.001" controls-position="right" /></el-form-item>
              <el-form-item label="滑点率（%）"><el-input-number v-model="form.slippage_percent" :min="0" :max="10" :precision="4" :step="0.01" controls-position="right" /></el-form-item>
              <el-form-item label="最大成交量参与率（%）"><el-input-number v-model="form.max_volume_percent" :min="0.01" :max="100" :precision="2" :step="0.01" controls-position="right" /></el-form-item>
              <el-form-item label="目标资金使用率（%）"><el-input-number v-model="form.target_position_percent" :min="0.01" :max="100" :precision="2" :step="0.01" controls-position="right" /></el-form-item>
              <el-form-item label="买入单位（股）"><el-input-number v-model="form.lot_size" :min="100" :max="10000" :step="100" controls-position="right" /></el-form-item>
            </div>
          </el-collapse-item></el-collapse>
          <div class="submit-row"><el-button type="primary" native-type="submit" :loading="running" :disabled="running">{{ running ? '正在回测…' : '运行回测' }}</el-button><span>前复权；收盘产生信号，下一交易日开盘尝试成交</span></div>
        </el-form>
      </section>
      <aside class="data-table-card rules-card" aria-labelledby="rules-title">
        <span class="eyebrow">02 / 规则</span><h2 id="rules-title">执行口径</h2>
        <ol class="rule-list">
          <li><strong>信号时间</strong><span>仅用当日及以前数据，在收盘后产生委托。</span></li>
          <li><strong>成交时间</strong><span>下一交易日按开盘价、滑点及价格限制计算。</span></li>
          <li><strong>交易约束</strong><span>停牌、涨跌停、成交量与资金不足均保留原因。</span></li>
          <li><strong>成本口径</strong><span>佣金、卖出印花税和过户费分别入账。</span></li>
        </ol>
        <p class="rules-foot">历史证券状态或涨跌停价缺少逐日证据时，报告会标明缺口，策略不能晋级。</p>
      </aside>
    </div>
    <section class="data-table-card history-card" aria-labelledby="history-title">
      <div class="section-heading history-heading"><div><span class="eyebrow">03 / 记录</span><h2 id="history-title">历史回测</h2></div><div class="history-actions"><el-button :loading="loading" @click="load">刷新</el-button><el-button :disabled="!selected.length" type="danger" plain @click="removeSelected">删除选中</el-button><el-button @click="clearFailed">清理失败</el-button></div></div>
      <div class="table-scroll-x"><el-table :data="tasks" v-loading="loading" empty-text="暂无回测记录，先在上方配置并运行策略" @selection-change="selected = $event">
        <el-table-column type="selection" width="48" /><el-table-column prop="symbol" label="股票" width="100" />
        <el-table-column label="策略" min-width="140"><template #default="{ row }">{{ strategyLabel(row.strategy_code) }}</template></el-table-column>
        <el-table-column label="样本区间" min-width="185"><template #default="{ row }"><span class="mono">{{ row.start_date || '—' }} ~ {{ row.end_date || '—' }}</span></template></el-table-column>
        <el-table-column label="数据质量" min-width="160"><template #default="{ row }"><el-tag v-if="row.quality_status==='PASS'" type="success" effect="plain">已核验</el-tag><el-tag v-else-if="row.quality_status==='PARTIAL'" type="warning" effect="plain">规则待核验</el-tag><span v-else class="muted">{{ row.status==='SUCCESS' ? '旧版结果' : '待生成' }}</span><small v-if="row.data_source" class="source-label">{{ row.data_source }}</small></template></el-table-column>
        <el-table-column label="状态" width="100"><template #default="{ row }"><TaskStatusTag :value="row.status" /></template></el-table-column>
        <el-table-column label="创建时间" min-width="165"><template #default="{ row }">{{ formatTime(row.created_at) }}</template></el-table-column>
        <el-table-column label="操作" width="160" fixed="right"><template #default="{ row }"><el-button link type="primary" @click="router.push(`/backtest/${row.id}`)">查看报告</el-button><el-button link type="danger" :disabled="['RUNNING','PENDING'].includes(row.status || '')" @click="removeOne(row)">删除</el-button></template></el-table-column>
      </el-table></div>
    </section>
  </PageContainer>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import dayjs from 'dayjs'
import { ElMessage, ElMessageBox } from 'element-plus'
import PageContainer from '@/components/common/PageContainer.vue'
import TaskStatusTag from '@/components/common/TaskStatusTag.vue'
import { backtestApi } from '@/api/backtest'
import type { BacktestTask } from '@/types/backtest'
import { formatTime } from '@/utils/time'

const router = useRouter()
const strategies = [
  { value: 'ma_cross', label: '均线交叉', description: '短期均线上穿或下穿长期均线时生成信号，适合检验趋势跟随假设。' },
  { value: 'macd_cross', label: 'MACD 交叉', description: '根据 DIF 与 DEA 的交叉测试动量变化。' },
  { value: 'rsi_reversal', label: 'RSI 反转', description: '测试动量进入超买或超卖区后的反转行为。' },
  { value: 'boll_breakout', label: '布林带突破', description: '用价格对波动区间的突破生成信号。' },
  { value: 'volume_breakout', label: '放量突破', description: '同时观察价格突破和成交量放大。' },
  { value: 'trend_score', label: '趋势综合评分', description: '综合均线、动量与波动率，用统一评分判断趋势。' }
] as const
type StrategyParameter = { key: string; label: string; min: number; max: number; step?: number; precision?: number; payloadKey?: string }
const parameterDefinitions: Record<string, StrategyParameter[]> = {
  ma_cross: [{ key: 'short_window', label: '短期均线（交易日）', min: 2, max: 60 }, { key: 'long_window', label: '长期均线（交易日）', min: 3, max: 250 }],
  macd_cross: [{ key: 'fast', label: '快线周期', min: 2, max: 60 }, { key: 'slow', label: '慢线周期', min: 3, max: 120 }, { key: 'signal', label: '信号线周期', min: 2, max: 60 }],
  rsi_reversal: [{ key: 'rsi_period', payloadKey: 'period', label: 'RSI 周期', min: 2, max: 60 }, { key: 'low', label: '超卖阈值', min: 1, max: 49 }, { key: 'high', label: '超买阈值', min: 51, max: 99 }],
  boll_breakout: [{ key: 'boll_period', payloadKey: 'period', label: '统计周期', min: 5, max: 120 }, { key: 'std', label: '标准差倍数', min: 0.5, max: 5, step: 0.1, precision: 1 }],
  volume_breakout: [{ key: 'volume_window', label: '成交量观察期', min: 5, max: 120 }, { key: 'volume_multiplier', label: '放量倍数', min: 1.05, max: 10, step: 0.05, precision: 2 }]
}
const strategyParams = reactive<Record<string, number>>({ short_window: 5, long_window: 20, fast: 12, slow: 26, signal: 9, rsi_period: 14, boll_period: 20, low: 30, high: 70, std: 2, volume_window: 20, volume_multiplier: 1.8 })
const tasks = ref<BacktestTask[]>([])
const selected = ref<BacktestTask[]>([])
const loading = ref(false)
const running = ref(false)
const form = reactive({ symbol: '', strategy_code: 'ma_cross', start_date: dayjs().subtract(2, 'year').format('YYYY-MM-DD'), end_date: dayjs().subtract(1, 'day').format('YYYY-MM-DD'), initial_cash: 100000, commission_percent: 0.03, min_commission: 5, stamp_duty_percent: 0.05, transfer_fee_percent: 0.001, slippage_percent: 0.01, max_volume_percent: 10, target_position_percent: 95, lot_size: 100 })
const activeStrategy = computed(() => strategies.find(item => item.value === form.strategy_code) || strategies[0])
const activeParameters = computed(() => parameterDefinitions[form.strategy_code] || [])
const parameterMinimum = (item: StrategyParameter) => item.key === 'long_window' ? Math.max(item.min, strategyParams.short_window + 1) : item.key === 'slow' ? Math.max(item.min, strategyParams.fast + 1) : item.min
const strategyLabel = (value?: string) => strategies.find(item => item.value === value)?.label || value || '—'
const disableFuture = (date: Date) => date.getTime() >= dayjs().startOf('day').valueOf()
const load = async () => { loading.value = true; try { tasks.value = await backtestApi.tasks({ limit: 100 }) } finally { loading.value = false } }
const run = async () => {
  if (!/^\d{6}$/.test(form.symbol)) return ElMessage.warning('请输入 6 位股票代码')
  if (!form.start_date || !form.end_date || form.start_date > form.end_date) return ElMessage.warning('请检查回测日期区间')
  if (activeParameters.value.some(item => !Number.isFinite(strategyParams[item.key]) || strategyParams[item.key] < parameterMinimum(item) || strategyParams[item.key] > item.max)) return ElMessage.warning('请检查策略参数范围')
  if (form.strategy_code === 'ma_cross' && strategyParams.long_window <= strategyParams.short_window) return ElMessage.warning('长期均线必须大于短期均线')
  if (form.strategy_code === 'macd_cross' && strategyParams.slow <= strategyParams.fast) return ElMessage.warning('MACD 慢线周期必须大于快线周期')
  const params = Object.fromEntries(activeParameters.value.map(item => [item.payloadKey || item.key, strategyParams[item.key]]))
  running.value = true
  try {
    const response = await backtestApi.run({ symbol: form.symbol, strategy_code: form.strategy_code, start_date: form.start_date, end_date: form.end_date, initial_cash: form.initial_cash, adjust: 'qfq', fee_rate: form.commission_percent / 100, min_commission: form.min_commission, stamp_duty_rate: form.stamp_duty_percent / 100, transfer_fee_rate: form.transfer_fee_percent / 100, slippage_rate: form.slippage_percent / 100, max_volume_ratio: form.max_volume_percent / 100, target_position_ratio: form.target_position_percent / 100, lot_size: form.lot_size, tick_size: 0.01, params_json: JSON.stringify(params) })
    ElMessage.success('回测完成，正在打开报告')
    await router.push(`/backtest/${response.task_id}`)
  } finally { running.value = false }
}
const confirmDelete = (count: number) => ElMessageBox.confirm(`将删除 ${count} 条回测记录及其报告、委托和成交明细，是否继续？`, '删除回测记录', { type: 'warning', confirmButtonText: '确认删除', cancelButtonText: '取消' })
const removeOne = async (row: BacktestTask) => { await confirmDelete(1); await backtestApi.remove(row.id); ElMessage.success('回测记录已删除'); await load() }
const removeSelected = async () => { const ids = selected.value.map(item => item.id); await confirmDelete(ids.length); const result = await backtestApi.removeBatch(ids); ElMessage.success(`已删除 ${result.affected} 条回测记录`); selected.value = []; await load() }
const clearFailed = async () => { await ElMessageBox.confirm('删除全部失败回测记录？', '清理失败记录', { type: 'warning' }); const result = await backtestApi.clearFailed(); ElMessage.success(result.affected ? `已清理 ${result.affected} 条失败记录` : '没有需要清理的失败记录'); await load() }
onMounted(load)
</script>

<style scoped>
.workspace-grid{display:grid;grid-template-columns:minmax(0,1.55fr) minmax(280px,.8fr);gap:16px;margin-bottom:16px}.setup-card,.rules-card{padding:22px}.section-heading{display:flex;justify-content:space-between;align-items:start;gap:12px;margin-bottom:16px}.section-heading h2,.rules-card h2{margin:4px 0 0;font-size:18px;line-height:1.4}.eyebrow{color:var(--qs-primary-dark);font-size:11px;font-weight:700;letter-spacing:.08em}.form-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:0 16px}.backtest-form :deep(.el-form-item){margin-bottom:13px}.backtest-form :deep(.el-select),.backtest-form :deep(.el-date-editor),.backtest-form :deep(.el-input-number){width:100%}.strategy-note{margin:0 0 8px;color:var(--qs-muted);line-height:1.6}.advanced-settings{border-top:1px solid var(--qs-border);border-bottom:0}.advanced-settings :deep(.el-collapse-item__header){font-weight:600}.advanced-settings :deep(.el-collapse-item__content){padding:14px 0 2px}.submit-row{display:flex;align-items:center;gap:14px;margin-top:18px}.submit-row .el-button{min-width:126px;min-height:44px}.submit-row span{color:var(--qs-muted);font-size:12px;line-height:1.5}.rule-list{padding:0;margin:18px 0;list-style:none;counter-reset:rule}.rule-list li{position:relative;padding:0 0 18px 36px;counter-increment:rule;border-left:1px solid var(--qs-border);margin-left:12px}.rule-list li:last-child{border-left:0;padding-bottom:0}.rule-list li::before{content:counter(rule);position:absolute;left:-13px;top:0;width:25px;height:25px;display:grid;place-items:center;border-radius:50%;background:var(--qs-bg);border:1px solid var(--qs-border);font:600 12px/1 sans-serif}.rule-list strong,.rule-list span{display:block}.rule-list strong{margin-bottom:4px}.rule-list span,.rules-foot{color:var(--qs-muted);font-size:13px;line-height:1.6}.rules-foot{padding:14px;background:var(--qs-bg);border-radius:8px;margin:20px 0 0}.history-card{padding:20px}.history-heading{align-items:center}.history-actions{display:flex;flex-wrap:wrap;gap:8px}.source-label{display:block;color:var(--qs-muted);margin-top:3px}.table-scroll-x :deep(.el-table){min-width:850px}@media(max-width:1050px){.workspace-grid{grid-template-columns:1fr}.rules-card{padding:18px}.rule-list{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:12px}.rule-list li{border:0;padding-bottom:0}}@media(max-width:640px){.setup-card,.history-card{padding:16px}.form-grid,.rule-list{grid-template-columns:1fr}.submit-row{align-items:stretch;flex-direction:column}.history-heading{align-items:flex-start;flex-direction:column}.history-actions{width:100%}.history-actions .el-button{margin:0}}
.parameter-heading{display:flex;align-items:baseline;flex-wrap:wrap;gap:6px 12px;margin:16px 0 10px}.parameter-heading strong{font-size:14px}.parameter-heading span{color:var(--qs-muted);font-size:12px}.parameter-grid{padding:14px 14px 0;margin-bottom:14px;background:var(--qs-bg);border:1px solid var(--qs-border);border-radius:8px}
</style>
