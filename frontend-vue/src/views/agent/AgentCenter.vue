<template>
  <PageContainer title="智能体分析" description="量化智能体编排分析与自然语言问答">
    <template #actions><el-button @click="$router.push('/agent/reports')">分析档案</el-button><el-button @click="$router.push('/tasks')">任务中心</el-button></template>
    <el-alert title="智能体优先使用“AI 模型配置”中的当前模型；未配置时自动使用本地结构化引擎。行情、指标、策略、风控和知识检索均来自后端工具调用。" type="info" show-icon :closable="false" class="provider-alert" />
    <div class="section-grid agent-layout">
      <section class="data-table-card span-5 agent-form-card">
        <el-form label-position="top" class="agent-form">
          <el-form-item label="股票代码"><el-input v-model="form.symbol" placeholder="000001" /></el-form-item>
          <el-form-item label="知识依据"><el-select v-model="form.knowledge_base_id"><el-option label="系统量化研究库（推荐）" :value="0"/></el-select><small class="field-help">系统会自动检索方法、回测与 A 股执行规范，无需填写编号。</small></el-form-item>
          <el-form-item label="模拟账户 ID"><el-input-number v-model="form.account_id" :min="0" /></el-form-item>
          <el-form-item label="分析问题"><el-input v-model="form.question" type="textarea" :rows="5" /></el-form-item>
          <div class="agent-actions">
            <el-button type="primary" :loading="loading" @click="analyze">发起个股分析</el-button>
            <el-button :loading="loading" @click="ask">自然语言问答</el-button>
          </div>
        </el-form>
      </section>
      <section class="data-table-card span-7 agent-report-card">
        <div class="toolbar report-toolbar">
          <strong>结构化报告</strong>
          <el-tag type="info">模型编排报告</el-tag>
          <RiskTag v-if="report.risk_level" :value="report.risk_level as string" />
          <el-tag>置信度 {{ confidenceText }}</el-tag>
        </div>
        <div class="report-grid">
          <article v-for="item in reportSections" :key="item.label" class="report-block">
            <div class="report-label">{{ item.label }}</div>
            <div class="report-content">{{ item.value }}</div>
          </article>
        </div>
      </section>
    </div>
    <DisclaimerAlert />
  </PageContainer>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import PageContainer from '@/components/common/PageContainer.vue'
import RiskTag from '@/components/common/RiskTag.vue'
import DisclaimerAlert from '@/components/common/DisclaimerAlert.vue'
import { agentApi } from '@/api/agent'
import type { AgentReport } from '@/types/agent'

const loading = ref(false)
const report = ref<AgentReport>({})
const route=useRoute()
const form = reactive({ symbol: String(route.query.symbol || '000001'), question: '请进行量化分析，覆盖行情、技术指标、策略信号、知识库资料和风险提示', knowledge_base_id: 0, account_id: 0 })
const confidenceText=computed(()=>report.value.confidence===undefined?'--':`${(Number(report.value.confidence)<=1?Number(report.value.confidence)*100:Number(report.value.confidence)).toFixed(0)}%`)
const reportSections = computed(() => [
  { label: '摘要', value: report.value.summary || '暂无分析结果，请先发起个股分析或自然语言问答。' },
  { label: '行情分析', value: report.value.market_analysis || '暂无' },
  { label: '技术分析', value: report.value.technical_analysis || '暂无' },
  { label: '策略分析', value: report.value.strategy_analysis || '暂无' },
  { label: '知识库与消息面', value: report.value.rag_analysis || '系统知识检索暂未返回有效片段。' },
  { label: '模拟账户影响', value: report.value.paper_trading_analysis || '未选择模拟账户，当前报告不包含账户持仓影响。' },
  { label: '风险分析', value: report.value.risk_analysis || '暂无' },
  { label: '最终观点', value: report.value.final_view || '仅供研究，不构成投资建议' }
])
const analyze = async () => {
  loading.value = true
  try {
    report.value = await agentApi.analyzeStock(form)
  } finally {
    loading.value = false
  }
}
const ask = async () => {
  loading.value = true
  try {
    report.value = await agentApi.ask(form)
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.agent-layout {
  align-items: start;
}

.provider-alert {
  margin-bottom: 14px;
}

.agent-form-card {
  min-width: 0;
}

.agent-form :deep(.el-input-number) {
  width: 100%;
}
.field-help{display:block;margin-top:6px;color:var(--qs-muted);line-height:1.5}

.agent-actions {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.agent-actions :deep(.el-button) {
  width: 100%;
  margin-left: 0;
}

.agent-report-card {
  min-width: 0;
}

.report-toolbar {
  justify-content: space-between;
}

.report-grid {
  display: grid;
  gap: 12px;
}

.report-block {
  border: 1px solid var(--qs-border);
  border-radius: 8px;
  padding: 12px;
  background: #fbfcfe;
}

.report-label {
  margin-bottom: 8px;
  color: var(--qs-muted);
  font-size: 13px;
  font-weight: 600;
}

.report-content {
  min-height: 22px;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  line-height: 1.65;
}

@media (max-width: 720px) {
  .agent-actions {
    grid-template-columns: 1fr;
  }
}
</style>
