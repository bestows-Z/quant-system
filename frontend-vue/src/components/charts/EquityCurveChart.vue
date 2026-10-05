<template>
  <div ref="chartRef" class="chart" />
</template>

<script setup lang="ts">
import * as echarts from 'echarts'
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import type { EquityPoint } from '@/types/backtest'

const props = defineProps<{ data: EquityPoint[]; title?: string }>()
const chartRef = ref<HTMLElement>()
let chart: echarts.ECharts | null = null
let observer: ResizeObserver | null = null
const points = computed(() => props.data
  .map((item) => ({
    date: item.trade_date || item.date || item.time || '',
    value: Number(item.total_asset ?? item.equity ?? item.asset ?? item.value)
  }))
  .filter((item) => item.date && Number.isFinite(item.value)))

const draw = () => {
  if (!chartRef.value) return
  chart ||= echarts.init(chartRef.value)
  const x = points.value.map((item) => item.date)
  const y = points.value.map((item) => item.value)
  chart.setOption({
    title: { text: props.title || '', left: 8, top: 6, textStyle: { fontSize: 13, fontWeight: 500 } },
    tooltip: { trigger: 'axis', valueFormatter: (value: unknown) => Number(value).toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) },
    grid: { left: 12, right: 12, top: 36, bottom: 28, containLabel: true },
    xAxis: { type: 'category', data: x, boundaryGap: false },
    yAxis: {
      type: 'value',
      scale: true,
      axisLabel: {
        formatter: (value: number) => Math.abs(value) >= 10000
          ? `${(value / 10000).toFixed(1)}万`
          : value.toLocaleString('zh-CN')
      }
    },
    series: [{ type: 'line', data: y, smooth: true, symbol: 'none', lineStyle: { color: '#0ca88c', width: 2 }, areaStyle: { color: 'rgba(12,168,140,0.12)' } }]
  })
}

onMounted(() => {
  nextTick(draw)
  if (chartRef.value) {
    observer = new ResizeObserver(() => chart?.resize())
    observer.observe(chartRef.value)
  }
})
watch(() => props.data, draw, { deep: true })
onBeforeUnmount(() => { observer?.disconnect(); chart?.dispose() })
</script>

<style scoped>
.chart {
  width: 100%;
  height: 320px;
}
</style>
