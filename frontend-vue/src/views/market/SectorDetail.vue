<template>
  <PageContainer :title="sector?.name || '板块详情'" description="完整行业成分、真实行情覆盖与板块分时研究">
    <template #actions>
      <el-select v-model="sortBy" style="width:130px" @change="loadConstituents"><el-option label="按涨跌幅" value="change_percent"/><el-option label="按成交额" value="amount"/><el-option label="按换手率" value="turnover_rate"/></el-select>
      <el-button :loading="loading" @click="load">刷新</el-button>
    </template>
    <el-alert v-if="sector && sector.status !== 'AVAILABLE'" title="当前板块实时行情覆盖不足，强度和分时仅反映已取得真实行情的成分股" type="warning" show-icon class="status-alert" />
    <div v-if="sector" class="sector-strip">
      <div><span>平均涨跌</span><PriceChange :value="sector.average_change" percent-mode /></div>
      <div><span>上涨 / 下跌</span><strong>{{sector.rise_count}} / {{sector.fall_count}}</strong></div>
      <div><span>行情覆盖</span><strong>{{sector.quoted_count}} / {{sector.stock_count}}</strong><small>{{percentValue(sector.coverage_rate)}}</small></div>
      <div><span>成交额</span><strong>{{compactNumber(sector.total_amount)}}</strong></div>
      <div><span>领涨成分</span><el-link v-if="sector.leader_symbol" type="primary" @click="router.push(`/market/stocks/${sector.leader_symbol}`)">{{sector.leader_name}}</el-link><strong v-else>暂无</strong></div>
    </div>
    <section class="data-table-card chart-card">
      <div class="toolbar"><strong>板块分时</strong><span class="muted">由真实成分股快照聚合，分钟级更新</span></div>
      <div ref="chartRef" class="sector-chart" />
      <el-empty v-if="!timeshare.length" description="今日尚无满足覆盖要求的板块分钟快照" />
    </section>
    <section class="data-table-card">
      <div class="toolbar"><strong>全部成分股</strong><span class="muted">当前页通过 WebSocket 增量更新</span><el-tag>{{quotedCount}} / {{total}} 有行情</el-tag></div>
      <div class="table-scroll-x"><el-table :data="pagedItems" v-loading="loading" class="wide-table" empty-text="暂无成分股">
        <el-table-column prop="symbol" label="代码" width="100" />
        <el-table-column label="股票" min-width="140"><template #default="{row}"><el-link type="primary" @click="router.push(`/market/stocks/${row.symbol}`)">{{row.name}}</el-link></template></el-table-column>
        <el-table-column label="最新价" width="110"><template #default="{row}">{{row.price ? money(row.price) : '暂无'}}</template></el-table-column>
        <el-table-column label="涨跌幅" width="110"><template #default="{row}"><PriceChange v-if="row.price" :value="row.change_percent" percent-mode/><span v-else class="muted">暂无</span></template></el-table-column>
        <el-table-column label="成交额" width="130"><template #default="{row}">{{row.amount ? compactNumber(row.amount) : '暂无'}}</template></el-table-column>
        <el-table-column label="换手率" width="110"><template #default="{row}">{{row.price ? percentValue(row.turnover_rate) : '暂无'}}</template></el-table-column>
        <el-table-column label="行情时间" min-width="170"><template #default="{row}">{{formatTime(row.quote_time)}}</template></el-table-column>
      </el-table></div>
      <el-pagination v-model:current-page="page" :page-size="pageSize" :total="items.length" layout="prev, pager, next, total" class="pager" />
    </section>
    <DisclaimerAlert />
  </PageContainer>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import * as echarts from 'echarts'
import PageContainer from '@/components/common/PageContainer.vue'
import DisclaimerAlert from '@/components/common/DisclaimerAlert.vue'
import PriceChange from '@/components/common/PriceChange.vue'
import { featureApi } from '@/api/feature'
import { useWebSocket } from '@/composables/useWebSocket'
import { compactNumber, money } from '@/utils/format'
import { formatTime } from '@/utils/time'
import type { SectorConstituent, SectorStrength, SectorTimeshareItem } from '@/types/feature'
import type { RealtimeQuote } from '@/types/market'
import type { WsServerMessage } from '@/types/notify'

const route=useRoute(),router=useRouter(),code=computed(()=>String(route.params.code||''))
const sector=ref<SectorStrength>(),items=ref<SectorConstituent[]>([]),timeshare=ref<SectorTimeshareItem[]>([]),loading=ref(false),sortBy=ref('change_percent'),page=ref(1),pageSize=20
const chartRef=ref<HTMLElement>();let chart:echarts.ECharts|null=null;let removeListener:(()=>void)|undefined;let subscribed:string[]=[];let timer:number|undefined
const {subscribe,unsubscribe,onMessage}=useWebSocket()
const total=computed(()=>items.value.length),quotedCount=computed(()=>items.value.filter(item=>Number(item.price)>0).length),pagedItems=computed(()=>items.value.slice((page.value-1)*pageSize,page.value*pageSize))
const percentValue=(value?:number)=>value==null?'暂无':`${(Number(value)*100).toFixed(2)}%`
const bindPage=()=>{if(subscribed.length)unsubscribe('market',subscribed);subscribed=pagedItems.value.map(item=>item.symbol);if(subscribed.length)subscribe('market',subscribed)}
const loadConstituents=async()=>{const result=await featureApi.sectorConstituents(code.value,{sort_by:sortBy.value,order:'desc',limit:1000});items.value=result.items||[];page.value=1;bindPage()}
const load=async()=>{loading.value=true;try{const [detail,constituents,line]=await Promise.all([featureApi.sectorDetail(code.value),featureApi.sectorConstituents(code.value,{sort_by:sortBy.value,order:'desc',limit:1000}),featureApi.sectorTimeshare(code.value)]);sector.value=detail.sector;items.value=constituents.items||[];timeshare.value=line.items||[];await renderChart();bindPage()}finally{loading.value=false}}
const handleMessage=(message:WsServerMessage)=>{if(message.type!=='market.quote.updated')return;const data=message.data as RealtimeQuote;const index=items.value.findIndex(item=>item.symbol===data.symbol);if(index<0)return;items.value.splice(index,1,{...items.value[index],...data,change_percent:Number(data.change_pct??data.change_percent??0)} as SectorConstituent)}
const renderChart=async()=>{await nextTick();if(!chartRef.value||!timeshare.value.length)return;chart||=echarts.init(chartRef.value);chart.setOption({animation:false,tooltip:{trigger:'axis',confine:true,formatter:(params:Array<{dataIndex:number}>)=>{const item=timeshare.value[params[0]?.dataIndex||0];return `${item.time}<br/>板块涨跌：${Number(item.change_percent||0)>=0?'+':''}${(Number(item.change_percent||0)*100).toFixed(2)}%<br/>上涨/下跌：${item.rise_count}/${item.fall_count}<br/>覆盖率：${percentValue(item.coverage_rate)}`}},grid:{left:54,right:24,top:24,bottom:34},xAxis:{type:'category',data:timeshare.value.map(item=>item.time.slice(11,16)),boundaryGap:false},yAxis:{type:'value',axisLabel:{formatter:(value:number)=>`${value>0?'+':''}${(value*100).toFixed(2)}%`},splitLine:{lineStyle:{type:'dashed'}}},series:[{type:'line',name:'板块涨跌',data:timeshare.value.map(item=>item.change_percent),showSymbol:false,lineStyle:{color:'#1677ff',width:1.5},areaStyle:{color:'rgba(22,119,255,.08)'},markLine:{silent:true,symbol:'none',data:[{yAxis:0}]}}]},true)}
const resize=()=>chart?.resize()
watch(page,bindPage)
onMounted(()=>{load();removeListener=onMessage(handleMessage);timer=window.setInterval(()=>{if(!document.hidden)featureApi.sectorTimeshare(code.value).then(result=>{timeshare.value=result.items||[];renderChart()})},15000);window.addEventListener('resize',resize)})
onBeforeUnmount(()=>{if(subscribed.length)unsubscribe('market',subscribed);removeListener?.();if(timer)window.clearInterval(timer);window.removeEventListener('resize',resize);chart?.dispose()})
</script>

<style scoped>
.status-alert{margin-bottom:12px}.sector-strip{display:flex;gap:1px;margin-bottom:14px;overflow-x:auto;border:1px solid var(--qs-border);background:var(--qs-border)}.sector-strip>div{flex:1 0 150px;min-height:72px;padding:11px 14px;display:flex;flex-direction:column;gap:5px;background:var(--qs-surface)}.sector-strip span,.sector-strip small{color:var(--qs-muted);font-size:12px}.chart-card{margin-bottom:14px}.sector-chart{height:300px}.wide-table{min-width:850px}.pager{justify-content:flex-end;margin-top:14px}@media(max-width:720px){.sector-chart{height:260px}.sector-strip>div{flex-basis:135px}}
</style>
