<template>
  <PageContainer title="智能体分析报告" description="结构化证据、策略研判与风险提示">
    <template #actions><el-button @click="$router.push('/agent/reports')">返回分析档案</el-button></template>
    <el-alert title="报告由当前模型或本地结构化引擎生成，量化工具数据来自实际后端服务；仅供研究，不构成投资建议" type="info" show-icon :closable="false" class="notice"/>
    <div class="section-grid summary-grid">
      <DataCard class="span-3" label="报告编号" :value="report.id||id"/>
      <DataCard class="span-3" label="证据完整度" :value="evidenceText" sub="成功取得数据的工具占比"/>
      <DataCard class="span-3" label="分析引擎" :value="engineText" :sub="tokenText"/>
      <section class="data-card span-3"><span>风险等级</span><div><RiskTag :value="String(report.risk_level||'UNKNOWN')"/></div></section>
    </div>
    <section class="report-list"><article v-for="item in sections" :key="item.key" class="data-table-card report-section"><h2>{{item.label}}</h2><p>{{text(report[item.key],item.empty)}}</p></article></section>
    <section v-if="references.length" class="data-table-card"><h2>引用与证据</h2><div class="reference-list"><div v-for="(item,index) in references" :key="index"><b>{{String(item.tool||item.document_title||`证据 ${index+1}`)}}</b><span>{{item.success===false?'获取失败':'已用于本报告'}}<template v-if="item.latency_ms"> · {{item.latency_ms}} 毫秒</template></span></div></div></section>
    <DisclaimerAlert/>
  </PageContainer>
</template>
<script setup lang="ts">
import{computed,onMounted,ref}from'vue';import{useRoute}from'vue-router';import PageContainer from'@/components/common/PageContainer.vue';import DataCard from'@/components/common/DataCard.vue';import RiskTag from'@/components/common/RiskTag.vue';import DisclaimerAlert from'@/components/common/DisclaimerAlert.vue';import{agentApi}from'@/api/agent';import type{AgentReport}from'@/types/agent'
const id=Number(useRoute().params.id),report=ref<AgentReport>({})
const sections=[{key:'summary',label:'摘要',empty:'暂无摘要。'},{key:'market_analysis',label:'行情与盘面',empty:'行情数据不足，暂未形成盘面结论。'},{key:'technical_analysis',label:'技术指标',empty:'技术指标数据不足。'},{key:'strategy_analysis',label:'策略信号',empty:'尚未运行策略或暂无有效策略信号。'},{key:'rag_analysis',label:'知识库与消息面',empty:'知识检索暂未返回有效片段。'},{key:'paper_trading_analysis',label:'账户与持仓影响',empty:'本次分析未关联模拟账户。'},{key:'risk_analysis',label:'风险提示',empty:'暂无独立风险记录，仍需关注市场波动。'},{key:'final_view',label:'综合研判',empty:'当前证据不足，不形成方向性结论。'}]as const
const evidenceText=computed(()=>`${Math.round((Number(report.value.evidence_completeness)||0)*100)}%`),engineText=computed(()=>report.value.provider_code==='local-rules'?'本地量化引擎':`${report.value.provider_code||'本地'} / ${report.value.model||'规则模型'}`),tokenText=computed(()=>report.value.token_source==='LOCAL'?'未调用外部模型':`${Number(report.value.total_tokens)||0} Token · ${report.value.token_source==='ESTIMATED'?'系统估算':'供应商返回'}`),references=computed<Record<string,unknown>[]>(()=>{if(Array.isArray(report.value.references))return report.value.references as Record<string,unknown>[];try{return JSON.parse(String(report.value.references_json||'[]'))}catch{return[]}})
const text=(value:unknown,fallback:string)=>String(value||fallback);onMounted(async()=>{report.value=await agentApi.report(id)})
</script>
<style scoped>.notice,.summary-grid,.report-list{margin-bottom:16px}.data-card{min-height:104px;padding:16px;background:#fff;border:1px solid var(--qs-border);border-radius:8px}.data-card>span{display:block;color:var(--qs-muted);margin-bottom:14px}.report-list{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:16px}.report-section h2,.data-table-card h2{font-size:15px;margin:0 0 10px}.report-section p{white-space:pre-wrap;overflow-wrap:anywhere;line-height:1.75;margin:0;color:#374151}.reference-list{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:8px}.reference-list div{display:flex;justify-content:space-between;gap:12px;padding:10px;background:#f6f8fb}.reference-list span{color:var(--qs-muted)}@media(max-width:760px){.report-list,.reference-list{grid-template-columns:1fr}}</style>
