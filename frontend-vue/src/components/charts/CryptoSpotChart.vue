<template>
  <div ref="chartRef" class="crypto-chart" role="img" :aria-label="chartDescription" />
</template>

<script setup lang="ts">
import * as echarts from 'echarts'
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import type { CryptoSpotKline } from '@/types/crypto'

const props = defineProps<{ klines: CryptoSpotKline[]; baseAsset: string }>()
const chartRef = ref<HTMLElement>()
let chart: echarts.ECharts | null = null
let observer: ResizeObserver | null = null
const number = (value: string) => Number(value)
const dateLabel = (stamp: number) => new Intl.DateTimeFormat('zh-CN', { timeZone: 'UTC', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }).format(stamp)
const chartDescription = computed(() => `${props.baseAsset}/USDT 的 ${props.klines.length} 根已收盘 K 线，时间为 UTC`)
const render = async () => {
  await nextTick()
  if (!chartRef.value || !props.klines.length) return
  chart ||= echarts.init(chartRef.value)
  const dates = props.klines.map(item => dateLabel(item.open_time))
  const candles = props.klines.map(item => [number(item.open), number(item.close), number(item.low), number(item.high)])
  const volumes = props.klines.map((item, index) => [index, number(item.base_volume), number(item.close) >= number(item.open) ? 1 : -1])
  chart.setOption({
    animation: false,
    backgroundColor: 'transparent',
    textStyle: { color: '#aeb4bd', fontFamily: 'Inter, system-ui, sans-serif' },
    tooltip: {
      trigger: 'axis', confine: true, axisPointer: { type: 'cross' },
      backgroundColor: '#252931', borderColor: '#414752', textStyle: { color: '#f2f3f5' },
      formatter: (points: Array<{ dataIndex: number }>) => {
        const item = props.klines[points[0]?.dataIndex ?? 0]
        if (!item) return ''
        return `<div style="line-height:1.65"><strong>${dateLabel(item.open_time)} UTC</strong><br/>开 ${item.open}　高 ${item.high}<br/>低 ${item.low}　收 ${item.close}<br/>成交量 ${Number(item.base_volume).toLocaleString('zh-CN')} ${props.baseAsset}</div>`
      }
    },
    axisPointer: { link: [{ xAxisIndex: 'all' }] },
    grid: [{ left: 12, right: 76, top: 18, height: '61%' }, { left: 12, right: 76, top: '75%', height: '13%' }],
    xAxis: [
      { type: 'category', data: dates, boundaryGap: true, axisLine: { lineStyle: { color: '#404650' } }, axisLabel: { hideOverlap: true, color: '#8d949e' }, splitLine: { show: false } },
      { type: 'category', gridIndex: 1, data: dates, boundaryGap: true, axisLine: { lineStyle: { color: '#404650' } }, axisLabel: { show: false }, splitLine: { show: false } }
    ],
    yAxis: [
      { type: 'value', scale: true, position: 'right', splitLine: { lineStyle: { color: '#30353d' } }, axisLabel: { color: '#8d949e', formatter: (value: number) => value.toLocaleString('zh-CN', { maximumFractionDigits: 2 }) } },
      { type: 'value', gridIndex: 1, position: 'right', scale: true, splitLine: { show: false }, axisLabel: { color: '#8d949e', formatter: (value: number) => value >= 1000 ? `${(value / 1000).toFixed(0)}k` : String(Math.round(value)) } }
    ],
    dataZoom: [{ type: 'inside', xAxisIndex: [0, 1], start: 30, end: 100, minValueSpan: 8 }],
    series: [
      { name: '价格', type: 'candlestick', data: candles, itemStyle: { color: '#0ecb81', color0: '#f6465d', borderColor: '#0ecb81', borderColor0: '#f6465d' } },
      { name: '成交量', type: 'bar', xAxisIndex: 1, yAxisIndex: 1, data: volumes, itemStyle: { color: (item: { data: number[] }) => item.data[2] > 0 ? '#0d7652' : '#8e3443' } }
    ]
  }, true)
}

onMounted(() => {
  void render()
  if (chartRef.value) {
    observer = new ResizeObserver(() => chart?.resize())
    observer.observe(chartRef.value)
  }
})
watch(() => props.klines, () => void render())
onBeforeUnmount(() => { observer?.disconnect(); chart?.dispose() })
</script>

<style scoped>
.crypto-chart { width: 100%; height: 470px; }
@media (max-width: 640px) { .crypto-chart { height: 360px; } }
</style>
