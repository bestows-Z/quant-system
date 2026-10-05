<template>
  <div class="chart-wrap">
    <el-empty v-if="!loading && normalized.length === 0" description="暂无 K 线数据" />
    <div v-show="normalized.length > 0" ref="chartRef" v-loading="loading" class="chart" />
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import * as echarts from 'echarts'
import type { BollItem, KlineItem, MaItem } from '@/types/market'

const props = defineProps<{ data: KlineItem[]; ma?: MaItem[]; boll?: BollItem[]; loading?: boolean }>()
const chartRef = ref<HTMLElement>()
let chart: echarts.ECharts | null = null
let observer: ResizeObserver | null = null

const normalized = computed(() =>
  (props.data || []).map((item) => ({
    date: normalizeDate(item.date || item.trade_date || item.time || ''),
    open: Number(item.open),
    close: Number(item.close),
    low: Number(item.low),
    high: Number(item.high),
    volume: Number(item.volume || 0),
    amount: item.amount == null || !Number.isFinite(Number(item.amount)) || (Number(item.amount) <= 0 && Number(item.volume || 0) > 0)
      ? undefined
      : Number(item.amount),
    preClose: Number(item.pre_close || 0),
    changePct: item.change_pct == null ? undefined : Number(item.change_pct),
    isRealtime: Boolean(item.is_realtime_bar),
    barStatus: item.bar_status
  }))
)

const maMap = computed(() => toMap(props.ma || []))
const bollMap = computed(() => toMap(props.boll || []))

const render = async () => {
  await nextTick()
  if (!chartRef.value || normalized.value.length === 0) return
  chart ||= echarts.init(chartRef.value)
  const dates = normalized.value.map((item) => item.date)
  const values = normalized.value.map((item) => [item.open, item.close, item.low, item.high])
  const volumes = normalized.value.map((item, index) => [index, item.volume, item.close >= item.open ? 1 : -1])
  const zoomStart = normalized.value.length > 120 ? Math.max(0, 100 - (120 / normalized.value.length) * 100) : 0
  const series = [
    {
      name: 'K线',
      type: 'candlestick',
      data: values,
      itemStyle: { color: '#d93025', color0: '#059669', borderColor: '#d93025', borderColor0: '#059669' }
    },
    line('MA5', dates.map((date) => maMap.value.get(date)?.ma5), MA_COLORS.MA5),
    line('MA10', dates.map((date) => maMap.value.get(date)?.ma10), MA_COLORS.MA10),
    line('MA20', dates.map((date) => maMap.value.get(date)?.ma20), MA_COLORS.MA20),
    line('MA60', dates.map((date) => maMap.value.get(date)?.ma60), MA_COLORS.MA60),
    line('BOLL上轨', dates.map((date) => bollMap.value.get(date)?.upper), '#ef4444', true),
    line('BOLL中轨', dates.map((date) => bollMap.value.get(date)?.middle), '#64748b', true),
    line('BOLL下轨', dates.map((date) => bollMap.value.get(date)?.lower), '#22c55e', true),
    {
      name: '成交量',
      type: 'bar',
      xAxisIndex: 1,
      yAxisIndex: 1,
      data: volumes,
      itemStyle: { color: (params: { data: [number, number, number] }) => (params.data[2] > 0 ? '#d93025' : '#059669') }
    }
  ]
  chart.setOption({
    animation: false,
    legend: { top: 0, left: 8, type: 'scroll' },
    tooltip: {
      trigger: 'axis', confine: true, appendToBody: false,
      axisPointer: { type: 'cross' },
      position: (point: number[], _params: unknown, _dom: HTMLElement, _rect: unknown, size: { contentSize: number[]; viewSize: number[] }) => {
        const left = point[0] + size.contentSize[0] + 24 > size.viewSize[0] ? point[0] - size.contentSize[0] - 18 : point[0] + 18
        const top = Math.min(Math.max(8, point[1] - size.contentSize[1] / 2), size.viewSize[1] - size.contentSize[1] - 8)
        return [left, top]
      },
      formatter: (params: Array<{ dataIndex: number }>) => klineTooltip(params[0]?.dataIndex ?? 0)
    },
    axisPointer: { link: [{ xAxisIndex: 'all' }] },
    grid: [
      { left: 58, right: 18, top: 42, height: '56%' },
      { left: 58, right: 18, top: '72%', height: '14%' }
    ],
    xAxis: [
      { type: 'category', data: dates, boundaryGap: true, axisLine: { onZero: false }, axisLabel: { hideOverlap: true, formatter: (value: string) => value.slice(5) }, splitLine: { show: false } },
      { type: 'category', gridIndex: 1, data: dates, boundaryGap: true, axisLabel: { hideOverlap: true, formatter: (value: string) => value.slice(5) }, splitLine: { show: false } }
    ],
    yAxis: [
      { scale: true, splitArea: { show: true } },
      { scale: true, gridIndex: 1, splitNumber: 2, axisLabel: { formatter: compactAxisNumber } }
    ],
    dataZoom: [
      { type: 'inside', xAxisIndex: [0, 1], start: zoomStart, end: 100, minValueSpan: 10 },
      { type: 'slider', xAxisIndex: [0, 1], bottom: 4, height: 18, start: zoomStart, end: 100, brushSelect: false }
    ],
    series
  })
}

const MA_COLORS = { MA5: '#f59e0b', MA10: '#0ea5e9', MA20: '#8b5cf6', MA60: '#64748b' } as const

const klineTooltip = (index: number) => {
  const item = normalized.value[index]
  if (!item) return ''
  const previousClose = item.preClose || normalized.value[index - 1]?.close || item.open
  const pct = item.changePct ?? (previousClose ? item.close / previousClose - 1 : 0)
  const color = pct > 0 ? '#d93025' : pct < 0 ? '#059669' : '#64748b'
  const ma = maMap.value.get(item.date)
  const rows = [
    `<strong>${item.date}${item.isRealtime || item.barStatus === 'INTRADAY' ? '（盘中）' : ''}</strong>`,
    `涨跌幅：<b style="color:${color}">${pct > 0 ? '+' : ''}${(pct * 100).toFixed(2)}%</b>`,
    `开 / 高 / 低 / 收：${item.open.toFixed(2)} / ${item.high.toFixed(2)} / ${item.low.toFixed(2)} / ${item.close.toFixed(2)}`,
    `成交量：${compactAxisNumber(item.volume)}　成交额：${compactAmount(item.amount)}`
  ]
  ;(['MA5', 'MA10', 'MA20', 'MA60'] as const).forEach((name) => {
    const value = ma?.[name.toLowerCase() as keyof MaItem]
    if (typeof value === 'number') rows.push(`<span style="color:${MA_COLORS[name]}">${name}：${value.toFixed(2)}</span>`)
  })
  return `<div style="min-width:270px;line-height:1.7">${rows.join('<br/>')}</div>`
}

const line = (name: string, data: Array<number | undefined>, color: string, dashed = false) => ({
  name,
  type: 'line',
  data,
  smooth: true,
  showSymbol: false,
  connectNulls: false,
  lineStyle: { width: 1.2, color, type: dashed ? 'dashed' : 'solid' },
  itemStyle: { color },
  emphasis: { lineStyle: { width: 2, color }, itemStyle: { color } }
})

const toMap = <T extends { date?: string; trade_date?: string }>(rows: T[]) => {
  const map = new Map<string, T>()
  rows.forEach((row) => {
    const key = normalizeDate(row.date || row.trade_date || '')
    if (key) map.set(key, row)
  })
  return map
}

const normalizeDate = (value: string) => String(value || '').slice(0, 10)
const compactAxisNumber = (value: number) => {
  const absolute = Math.abs(value)
  if (absolute >= 100000000) return `${Number((value / 100000000).toFixed(1))}亿`
  if (absolute >= 10000) return `${Math.round(value / 10000)}万`
  return String(Math.round(value))
}
const compactAmount = (value?: number) => {
  if (value == null || !Number.isFinite(value)) return '暂无（数据源未提供）'
  const absolute = Math.abs(value)
  if (absolute >= 100000000) return `${(value / 100000000).toFixed(2)}亿`
  if (absolute >= 10000) return `${(value / 10000).toFixed(2)}万`
  return value.toFixed(2)
}

const resize = () => chart?.resize()

onMounted(() => {
  render()
  window.addEventListener('resize', resize)
  if (chartRef.value) { observer = new ResizeObserver(resize); observer.observe(chartRef.value) }
})
watch(() => [props.data, props.ma, props.boll, props.loading], render, { deep: true })
onBeforeUnmount(() => {
  window.removeEventListener('resize', resize)
  observer?.disconnect()
  chart?.dispose()
  chart = null
})
</script>

<style scoped>
.chart-wrap {
  min-height: 500px;
}

.chart {
  width: 100%;
  height: 500px;
}
@media (max-width: 720px) {
  .chart-wrap, .chart { min-height: 420px; height: 420px; }
}
</style>
