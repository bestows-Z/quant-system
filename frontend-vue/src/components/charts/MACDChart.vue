<template>
  <div class="chart-card">
    <el-empty v-if="!loading && rows.length === 0" description="暂无 MACD 数据" />
    <div v-show="rows.length > 0" ref="chartRef" v-loading="loading" class="chart" />
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import * as echarts from 'echarts'
import type { MacdItem } from '@/types/market'

const props = defineProps<{ data: MacdItem[]; loading?: boolean }>()
const chartRef = ref<HTMLElement>()
let chart: echarts.ECharts | null = null
let observer: ResizeObserver | null = null
const rows = computed(() => props.data || [])

const render = async () => {
  await nextTick()
  if (!chartRef.value || rows.value.length === 0) return
  chart ||= echarts.init(chartRef.value)
  const dates = rows.value.map((item) => item.date || item.trade_date || '')
  chart.setOption({
    animation: false,
    tooltip: { trigger: 'axis' },
    legend: { top: 0 },
    grid: { left: 44, right: 20, top: 34, bottom: 32 },
    xAxis: { type: 'category', data: dates, boundaryGap: false },
    yAxis: { type: 'value', scale: true },
    series: [
      { name: 'MACD', type: 'bar', data: rows.value.map((item) => item.macd), itemStyle: { color: (p: { value: number }) => (p.value >= 0 ? '#d93025' : '#059669') } },
      { name: 'DIF', type: 'line', showSymbol: false, data: rows.value.map((item) => item.dif), color: '#f59e0b' },
      { name: 'DEA', type: 'line', showSymbol: false, data: rows.value.map((item) => item.dea), color: '#0ea5e9' }
    ]
  })
}

const resize = () => chart?.resize()
onMounted(() => {
  render()
  window.addEventListener('resize', resize)
  if (chartRef.value) { observer = new ResizeObserver(resize); observer.observe(chartRef.value) }
})
watch(() => [props.data, props.loading], render, { deep: true })
onBeforeUnmount(() => {
  window.removeEventListener('resize', resize)
  observer?.disconnect()
  chart?.dispose()
})
</script>

<style scoped>
.chart {
  width: 100%;
  height: 260px;
}
</style>
