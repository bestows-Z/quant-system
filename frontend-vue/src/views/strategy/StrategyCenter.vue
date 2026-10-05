<template>
  <PageContainer title="策略中心" description="配置、检查并运行基于真实 K 线的可解释量化策略">
    <template #actions><el-button :loading="loading" @click="load">刷新策略</el-button><el-button type="primary" @click="openCreate">添加策略</el-button></template>
    <el-alert title="策略运行使用真实历史 K 线，并保存触发价格、样本数量与数据源。策略信号仅供研究，不构成投资建议。" type="info" show-icon :closable="false" class="notice" />
    <section class="data-table-card"><div class="table-scroll-x"><el-table :data="items" v-loading="loading" class="strategy-table" empty-text="暂无可用策略">
      <el-table-column prop="name" label="策略名称" min-width="170" />
      <el-table-column prop="code" label="策略编码" min-width="170" />
      <el-table-column prop="description" label="适用说明" min-width="260" show-overflow-tooltip />
      <el-table-column label="类型" width="100"><template #default="{row}"><el-tag :type="row.is_system ? 'info' : 'success'">{{row.is_system ? '系统策略' : '自定义'}}</el-tag></template></el-table-column>
      <el-table-column label="状态" width="100"><template #default="{row}"><TaskStatusTag :value="row.status" /></template></el-table-column>
      <el-table-column label="操作" width="450" fixed="right"><template #default="{row}"><div class="row-actions"><el-button @click="openDetail(row)">详情</el-button><el-input v-model="runSymbol[row.id]" maxlength="6" placeholder="股票代码" /><el-button type="primary" :loading="runningId===row.id" @click="run(row)">运行分析</el-button><el-button v-if="!row.is_system" type="danger" plain @click="removeStrategy(row)">删除</el-button></div></template></el-table-column>
    </el-table></div></section>
    <section v-if="lastResult" class="data-table-card result-panel"><div class="toolbar"><strong>最近一次策略分析</strong><el-tag>运行编号 {{lastResult.run_id}}</el-tag><el-button link type="primary" @click="$router.push('/signals')">查看全部信号</el-button></div><div v-for="signal in lastResult.signals" :key="`${signal.symbol}-${signal.strategy_code}`" class="signal-result"><div><span>股票</span><strong>{{signal.symbol}}</strong></div><div><span>策略信号</span><SignalTag :value="signal.signal_type || signal.action" /></div><div><span>信号强度</span><strong>{{Number(signal.score || 0).toFixed(0)}} / 100</strong></div><div><span>触发价格</span><strong>{{Number(signal.trigger_price || 0).toFixed(2)}}</strong></div><div><span>风险等级</span><RiskTag :value="signal.risk_level" /></div><div class="reason"><span>形成原因</span><p>{{signal.reason || lastResult.summary}}</p></div></div></section>

    <el-dialog v-model="createVisible" title="添加量化策略" width="min(720px, 95vw)">
      <el-form label-position="top" class="strategy-form">
        <el-form-item label="策略名称"><el-input v-model="form.name" placeholder="例如：稳健均线趋势" /></el-form-item>
        <el-form-item label="基础算法"><el-select v-model="form.base_strategy" @change="resetParams"><el-option v-for="item in algorithms" :key="item.value" :label="item.label" :value="item.value" /></el-select><small>{{algorithmDescription}}</small></el-form-item>
        <el-form-item label="策略说明"><el-input v-model="form.description" type="textarea" :rows="3" placeholder="说明适用行情、观察周期和风险边界" /></el-form-item>
        <div class="parameter-row" v-for="field in activeFields" :key="field.key"><div><strong>{{field.label}}</strong><small>{{field.help}}</small></div><el-input-number v-model="form.params[field.key]" :min="field.min" :max="field.max" :step="field.step" /></div>
      </el-form>
      <template #footer><el-button @click="createVisible=false">取消</el-button><el-button type="primary" :loading="saving" @click="create">校验并创建</el-button></template>
    </el-dialog>

    <el-drawer v-model="detailVisible" title="策略详情" size="min(620px, 96vw)"><template v-if="detail"><el-descriptions :column="1" border><el-descriptions-item label="名称">{{detail.name}}</el-descriptions-item><el-descriptions-item label="编码">{{detail.code}}</el-descriptions-item><el-descriptions-item label="说明">{{detail.description || '--'}}</el-descriptions-item><el-descriptions-item label="状态">{{detail.status}}</el-descriptions-item><el-descriptions-item label="创建时间">{{detail.created_at || '--'}}</el-descriptions-item></el-descriptions><section class="parameter-detail"><strong>参数配置</strong><div v-for="(value,key) in detailParams" :key="key"><span>{{parameterLabel(String(key))}}</span><code>{{value}}</code></div></section></template></el-drawer>
    <DisclaimerAlert />
  </PageContainer>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import PageContainer from '@/components/common/PageContainer.vue'
import DisclaimerAlert from '@/components/common/DisclaimerAlert.vue'
import SignalTag from '@/components/common/SignalTag.vue'
import RiskTag from '@/components/common/RiskTag.vue'
import TaskStatusTag from '@/components/common/TaskStatusTag.vue'
import { strategyApi } from '@/api/strategy'
import type { RunStrategyResult, Strategy } from '@/types/strategy'

type ParamField = { key:string;label:string;help:string;min:number;max:number;step:number;default:number }
const algorithms = [
  {value:'ma_cross',label:'均线交叉',description:'比较短周期与长周期均线交叉。',fields:[{key:'short_window',label:'短均线周期',help:'短期趋势窗口，常用 5。',min:2,max:60,step:1,default:5},{key:'long_window',label:'长均线周期',help:'中期趋势窗口，必须大于短周期。',min:3,max:250,step:1,default:20}]},
  {value:'macd_cross',label:'MACD 交叉',description:'使用 DIF 与 DEA 的交叉判断趋势变化。',fields:[{key:'fast',label:'快速周期',help:'常用 12。',min:2,max:60,step:1,default:12},{key:'slow',label:'慢速周期',help:'常用 26。',min:3,max:120,step:1,default:26},{key:'signal',label:'信号周期',help:'常用 9。',min:2,max:60,step:1,default:9}]},
  {value:'rsi_reversal',label:'RSI 超买超卖',description:'根据 RSI 上下阈值识别动能极端区间。',fields:[{key:'period',label:'RSI 周期',help:'常用 14。',min:2,max:60,step:1,default:14},{key:'low',label:'超卖阈值',help:'低于该值进入超卖观察区。',min:1,max:49,step:1,default:30},{key:'high',label:'超买阈值',help:'高于该值进入超买风险区。',min:51,max:99,step:1,default:70}]},
  {value:'boll_breakout',label:'BOLL 突破',description:'根据布林带上下轨识别价格突破。',fields:[{key:'period',label:'中轨周期',help:'常用 20。',min:5,max:120,step:1,default:20},{key:'std',label:'标准差倍数',help:'常用 2。',min:.5,max:5,step:.1,default:2}]},
  {value:'volume_breakout',label:'放量突破',description:'成交量放大且价格突破阶段高点时触发。',fields:[{key:'volume_window',label:'成交量窗口',help:'用于计算平均成交量。',min:5,max:120,step:1,default:20},{key:'volume_multiplier',label:'放量倍数',help:'当前成交量相对均量的最低倍数。',min:1.05,max:10,step:.05,default:1.8}]},
  {value:'trend_score',label:'趋势综合评分',description:'综合均线、动量和波动率形成趋势评分。',fields:[] as ParamField[]}
]
const items=ref<Strategy[]>([]),runSymbol=reactive<Record<number,string>>({}),lastResult=ref<RunStrategyResult>(),runningId=ref(0),loading=ref(false),saving=ref(false),createVisible=ref(false),detailVisible=ref(false),detail=ref<Strategy>()
const form=reactive<{name:string;description:string;base_strategy:string;params:Record<string,number>}>({name:'',description:'',base_strategy:'ma_cross',params:{}})
const activeAlgorithm=computed(()=>algorithms.find(v=>v.value===form.base_strategy) || algorithms[0])
const activeFields=computed(()=>activeAlgorithm.value.fields)
const algorithmDescription=computed(()=>activeAlgorithm.value.description)
const detailParams=computed(()=>{try{return JSON.parse(detail.value?.params_json||'{}')}catch{return {}}})
const resetParams=()=>{form.params={};activeFields.value.forEach(v=>form.params[v.key]=v.default)}
const openCreate=()=>{form.name='';form.description='';form.base_strategy='ma_cross';resetParams();createVisible.value=true}
const load=async()=>{loading.value=true;try{items.value=await strategyApi.list();items.value.forEach(v=>{runSymbol[v.id] ||= '600176'})}finally{loading.value=false}}
const create=async()=>{if(!form.name.trim())return ElMessage.warning('请输入策略名称');if(form.base_strategy==='ma_cross'&&Number(form.params.long_window)<=Number(form.params.short_window))return ElMessage.warning('长均线周期必须大于短均线周期');saving.value=true;try{const stamp=Date.now();await strategyApi.create({name:form.name.trim(),code:`custom_${stamp}`,description:form.description.trim()||algorithmDescription.value,params_json:JSON.stringify({base_strategy:form.base_strategy,...form.params})});ElMessage.success('策略校验通过并创建成功');createVisible.value=false;await load()}finally{saving.value=false}}
const openDetail=async(row:Strategy)=>{detailVisible.value=true;detail.value=await strategyApi.detail(row.id)}
const run=async(row:Strategy)=>{const symbol=String(runSymbol[row.id]||'').trim();if(!/^\d{6}$/.test(symbol))return ElMessage.warning('请输入 6 位股票代码，例如 600176');runningId.value=row.id;try{lastResult.value=await strategyApi.run(row.id,{symbol,strategy_code:row.code});ElMessage.success('已使用真实 K 线完成策略分析')}finally{runningId.value=0}}
const removeStrategy=async(row:Strategy)=>{await ElMessageBox.confirm(`确认删除个人策略“${row.name}”？历史信号和运行记录仍会保留。`,'删除个人策略',{type:'warning'});await strategyApi.remove(row.id);ElMessage.success('个人策略已删除');await load()}
const parameterLabel=(key:string)=>({base_strategy:'基础算法',short_window:'短均线周期',long_window:'长均线周期',fast:'快速周期',slow:'慢速周期',signal:'信号周期',period:'计算周期',low:'超卖阈值',high:'超买阈值',std:'标准差倍数',volume_window:'成交量窗口',volume_multiplier:'放量倍数'}[key]||key)
onMounted(load)
</script>

<style scoped>
.notice{margin-bottom:16px}.strategy-table{min-width:1100px}.row-actions{display:flex;align-items:center;gap:8px}.row-actions .el-input{width:120px}.result-panel{margin-top:16px}.signal-result{display:flex;flex-wrap:wrap;gap:18px}.signal-result>div{flex:1 1 110px;display:flex;flex-direction:column;gap:7px}.signal-result span,.strategy-form small,.parameter-row small{color:var(--qs-muted);font-size:12px}.signal-result .reason{flex:3 1 280px}.reason p{margin:0;line-height:1.65}.strategy-form{display:flex;flex-direction:column;gap:2px}.strategy-form :deep(.el-select){width:100%}.parameter-row{display:flex;align-items:center;justify-content:space-between;gap:18px;padding:12px 0;border-top:1px solid var(--qs-border)}.parameter-row>div{display:flex;flex-direction:column;gap:4px}.parameter-detail{margin-top:18px;display:flex;flex-direction:column;gap:1px;background:var(--qs-border);border:1px solid var(--qs-border);border-radius:8px;overflow:hidden}.parameter-detail>strong{padding:12px;background:var(--qs-surface)}.parameter-detail>div{display:flex;justify-content:space-between;gap:20px;padding:10px 12px;background:var(--qs-surface)}.parameter-detail span{color:var(--qs-muted)}@media(max-width:640px){.parameter-row{align-items:flex-start;flex-direction:column}.parameter-row :deep(.el-input-number){width:100%}}
</style>
