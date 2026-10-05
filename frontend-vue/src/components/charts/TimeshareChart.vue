<template>
  <div class="chart-wrap">
    <el-empty v-if="!loading && !data.length" description="暂无分时数据" />
    <div v-show="data.length" ref="chartRef" v-loading="loading" class="chart" />
  </div>
</template>

<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import * as echarts from 'echarts'
import type { TimeshareItem } from '@/types/market'

interface ComparisonLine { key:string;name:string;data:Array<{time:string;value:number}> }
const props = defineProps<{ data: TimeshareItem[]; preClose?: number; loading?: boolean; comparisons?:ComparisonLine[] }>()
const chartRef = ref<HTMLElement>()
let chart: echarts.ECharts | null = null

const render = async () => {
  await nextTick()
  if (!chartRef.value || !props.data.length) return
  chart ||= echarts.init(chartRef.value)
  const times = tradingMinutes()
  const rows = new Map(props.data.map((item) => [item.time.slice(11, 16), item]))
  const prices = times.map((time) => rows.has(time) ? Number(rows.get(time)?.price) : null)
  const averages = times.map((time) => rows.has(time) ? Number(rows.get(time)?.average_price) : null)
  const volumes = times.map((time) => rows.has(time) ? Number(rows.get(time)?.volume) : null)
  const preClose = Number(props.preClose || prices.find((value) => value !== null) || 0)
  const percentages = prices.map((value) => value === null || preClose <= 0 ? null : (value / preClose - 1) * 100)
  const averagePercentages = averages.map((value) => value === null || preClose <= 0 ? null : (value / preClose - 1) * 100)
  const comparisonValues = (props.comparisons || []).flatMap((line) => line.data.map((item) => item.value)).filter(Number.isFinite)
  const percentageBounds = [...percentages, ...averagePercentages, ...comparisonValues].filter((value): value is number => value !== null && Number.isFinite(value))
  let axisMin = Math.min(0, ...percentageBounds)
  let axisMax = Math.max(0, ...percentageBounds)
  const rawRange = Math.max(0.5, axisMax - axisMin)
  const padding = Math.max(0.08, rawRange * 0.12)
  axisMin = Math.min(-0.05, axisMin - padding)
  axisMax = Math.max(0.05, axisMax + padding)
  const minuteAmounts = times.map((time, index) => {
    const current = Number(rows.get(time)?.amount || 0)
    if (!current) return null
    const previousTime = index > 0 ? times[index - 1] : ''
    const previous = Number(rows.get(previousTime)?.amount || 0)
    return Math.max(0, current - previous)
  })
	const comparisonSeries=(props.comparisons||[]).map((line,index)=>{
		const values=new Map(line.data.map(item=>[item.time.slice(11,16),item.value]))
		return {name:line.name,type:'line',data:times.map(time=>values.get(time)??null),showSymbol:false,lineStyle:{width:1.2,color:['#7c3aed','#0891b2','#be123c'][index]},connectNulls:true}
	})
  chart.setOption({
    animation: false,
    tooltip: {
      trigger: 'axis',
      confine: true,
      axisPointer: { type: 'cross' },
	  formatter: (params: Array<{ dataIndex: number }>) => timeshareTooltip(params, times, rows, percentages, minuteAmounts,props.comparisons||[])
    },
	legend: { top: 0, data: ['价格', '均价',...(props.comparisons||[]).map(item=>item.name)] },
    grid: [
      { left: 52, right: 22, top: 34, height: 210 },
      { left: 52, right: 22, top: 268, height: 72 }
    ],
    xAxis: [
      { type: 'category', boundaryGap: false, data: times, axisLabel: { interval: (_index: number, value: string) => ['09:30', '10:30', '11:30', '14:00', '15:00'].includes(value), formatter: (value: string) => value === '11:30' ? '11:30/13:00' : value } },
      { type: 'category', gridIndex: 1, boundaryGap: false, data: times, axisLabel: { show: false } }
    ],
    yAxis: [
      {
        type: 'value', min: axisMin, max: axisMax, interval: (axisMax - axisMin) / 4,
        axisPointer: { label: { formatter: ({ value }: { value: number }) => `${Number(value) > 0 ? '+' : ''}${Number(value).toFixed(2)}%` } },
        axisLabel: {
          formatter: (value: number) => value > 0 ? `{up|+${value.toFixed(2)}%}` : value < 0 ? `{down|${value.toFixed(2)}%}` : '{flat|0.00%}',
          rich: { up: { color: '#d93025' }, down: { color: '#059669' }, flat: { color: '#64748b' } }
        },
        splitLine: { lineStyle: { color: '#e5e7eb', type: 'dashed' } }
      },
      { type: 'value', gridIndex: 1, scale: true, splitNumber: 2, axisLabel: { formatter: (value: number) => `${Math.round(value / 10000)}万` } }
    ],
    series: [
      { name: '价格', type: 'line', data: percentages, showSymbol: false, lineStyle: { width: 1.4, color: '#1677ff' }, areaStyle: { color: 'rgba(22,119,255,0.08)' }, markLine: { silent: true, symbol: 'none', data: [{ yAxis: 0 }], lineStyle: { color: '#94a3b8', width: 1 } } },
      { name: '均价', type: 'line', data: averagePercentages, showSymbol: false, lineStyle: { width: 1.1, color: '#d97706' } },
	  ...comparisonSeries,
      { name: '成交量', type: 'bar', xAxisIndex: 1, yAxisIndex: 1, data: volumes, barMaxWidth: 5, itemStyle: { color: (params: { dataIndex: number }) => {
        const current = prices[params.dataIndex]
        const previous = params.dataIndex > 0 ? prices[params.dataIndex - 1] : Number(props.preClose || current)
        return current !== null && previous !== null && current >= previous ? '#d93025' : '#059669'
      } } }
    ]
  }, true)
}

const timeshareTooltip = (
  params: Array<{ dataIndex: number }>,
  times: string[],
  rows: Map<string, TimeshareItem>,
  percentages: Array<number | null>,
  minuteAmounts: Array<number | null>,
	comparisons: ComparisonLine[]
) => {
  const index = params[0]?.dataIndex ?? 0
  const time = times[index]
  const row = rows.get(time)
  if (!row) return `${time}<br/>暂无成交`
  const pct = percentages[index] || 0
  const color = pct > 0 ? '#d93025' : pct < 0 ? '#059669' : '#64748b'
	const compared=comparisons.map(line=>{const point=line.data.find(item=>item.time.slice(11,16)===time);return point?`<br/>${line.name}：<b>${point.value>0?'+':''}${point.value.toFixed(2)}%</b>`:''}).join('')
  return `<div style="min-width:170px"><strong>${time}</strong><br/>主标的涨跌：<b style="color:${color}">${pct > 0 ? '+' : ''}${pct.toFixed(2)}%</b>${compared}<br/>价格：${Number(row.price).toFixed(2)}<br/>均价：${Number(row.average_price || 0).toFixed(2)}<br/>成交量：${compactNumber(Number(row.volume || 0))}<br/>成交额：${compactNumber(Number(minuteAmounts[index] || 0))}</div>`
}

const compactNumber = (value: number) => {
  const absolute = Math.abs(value)
  if (absolute >= 100000000) return `${(value / 100000000).toFixed(2)}亿`
  if (absolute >= 10000) return `${(value / 10000).toFixed(2)}万`
  return String(Math.round(value))
}

const tradingMinutes = () => {
  const result: string[] = []
  const append = (startHour: number, startMinute: number, count: number) => {
    for (let offset = 0; offset < count; offset += 1) {
      const total = startHour * 60 + startMinute + offset
      result.push(`${String(Math.floor(total / 60)).padStart(2, '0')}:${String(total % 60).padStart(2, '0')}`)
    }
  }
  append(9, 30, 121)
  append(13, 0, 121)
  return result
}

const resize = () => chart?.resize()
onMounted(() => { render(); window.addEventListener('resize', resize) })
watch(() => [props.data, props.preClose, props.loading,props.comparisons], render, { deep: true })
onBeforeUnmount(() => { window.removeEventListener('resize', resize); chart?.dispose(); chart = null })
</script>

<style scoped>
.chart-wrap { min-height: 360px; }
.chart { width: 100%; height: 360px; }
</style>
