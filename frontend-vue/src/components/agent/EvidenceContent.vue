<template>
  <div class="evidence-content">
    <template v-if="message&&message.role==='assistant'">
      <h3>回答依据</h3><dl><dt>模型</dt><dd>{{message.provider||'本地量化引擎'}} {{message.model}}</dd><dt>响应时间</dt><dd>{{message.latency_ms||0}} ms</dd><dt>输入 / 输出</dt><dd>{{message.input_tokens||0}} / {{message.output_tokens||0}} Token</dd></dl>
      <h3>引用来源</h3>
      <div v-if="citations.length" class="evidence-list">
        <div v-for="(item,index) in citations" :key="`${item.url||item.tool}-${index}`" class="evidence-item">
          <a v-if="item.url" :href="item.url" target="_blank" rel="noopener noreferrer">{{item.title||sourceLabel(item)}}</a><strong v-else>{{sourceLabel(item)}}</strong>
          <span>{{[item.source,item.published_at,latencyText(item.latency_ms)].filter(Boolean).join(' · ')}}</span>
        </div>
      </div><el-empty v-else description="本次回答没有外部引用" :image-size="48"/>
      <h3>只读工具调用</h3>
      <div v-if="toolCalls.length" class="tool-list"><div v-for="(item,index) in toolCalls" :key="`${item.name}-${index}`"><el-tag size="small" type="success">只读</el-tag><span>{{toolLabel(item.name)}}</span><small v-if="item.report_id">报告 #{{item.report_id}}</small></div></div>
      <el-empty v-else description="本次回答没有工具调用记录" :image-size="48"/>
    </template><el-empty v-else description="点击一条助手回答查看证据" :image-size="64"/>
  </div>
</template>
<script setup lang="ts">
import { computed } from 'vue'
import type { AgentChatMessage } from '@/types/agent'

type EvidenceItem={url?:string;title?:string;source?:string;published_at?:string;tool?:string;latency_ms?:number;success?:boolean;evidence_type?:string}
type ToolItem={name?:string;mode?:string;report_id?:number}
const props=defineProps<{ message?: AgentChatMessage }>()
const parseList = <T,>(value?:string):T[] => {
  try {
    const parsed=JSON.parse(value||'[]')
    return Array.isArray(parsed)?parsed:[]
  } catch {
    return []
  }
}
const citations=computed(()=>parseList<EvidenceItem>(props.message?.citations_json))
const toolCalls=computed(()=>parseList<ToolItem>(props.message?.tool_calls_json))
const sourceLabel=(item:EvidenceItem)=>item.source||({market_tool:'行情与公告',indicator_tool:'技术指标',strategy_tool:'量化策略',risk_tool:'风险分析',backtest_tool:'历史回测',rag_search_tool:'知识库检索'}[item.tool||'']||'研究数据')
const toolLabel=(name?:string)=>({quant_research_pipeline:'量化研究工具链'}[name||'']||name||'研究工具')
const latencyText=(value?:number)=>value===undefined?'':`${value} ms`
</script>
<style scoped>.evidence-content h3{margin:0 0 10px;font-size:15px}.evidence-content h3:not(:first-child){margin-top:20px}.evidence-content dl{display:grid;grid-template-columns:88px minmax(0,1fr);gap:8px;margin:0;font-size:13px}.evidence-content dt{color:var(--qs-muted)}.evidence-content dd{margin:0;overflow-wrap:anywhere}.evidence-list{max-height:300px;overflow:auto;border-block:1px solid var(--qs-border)}.evidence-item{display:flex;flex-direction:column;gap:4px;padding:10px 2px;border-bottom:1px solid var(--qs-border)}.evidence-item:last-child{border-bottom:0}.evidence-item a,.evidence-item strong{color:var(--qs-text);font-size:13px;line-height:1.45}.evidence-item a:hover{color:var(--qs-primary)}.evidence-item span,.tool-list small{color:var(--qs-muted);font-size:12px}.tool-list>div{display:flex;align-items:center;gap:8px;padding:7px 0;font-size:13px}.tool-list small{margin-left:auto}</style>
