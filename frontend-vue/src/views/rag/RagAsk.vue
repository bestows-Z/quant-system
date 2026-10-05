<template>
  <PageContainer title="知识库问答" description="基于 Qdrant 向量检索与可追溯知识片段生成摘要">
    <section class="data-table-card">
      <el-form label-position="top">
        <el-form-item label="知识库"><el-select v-model="kbId" style="width: 260px"><el-option v-for="kb in kbs" :key="kb.id" :label="kb.name" :value="kb.id" /></el-select></el-form-item>
        <el-form-item label="问题"><el-input v-model="question" type="textarea" :rows="4" /></el-form-item>
        <el-button type="primary" @click="ask">提交</el-button>
      </el-form>
      <el-divider />
      <h3>答案</h3>
      <p class="answer-text">{{ answer.answer || '暂无回答' }}</p>
      <div class="toolbar" v-if="answer.provider || answer.model">
        <el-tag v-if="answer.provider">{{ answer.provider }}</el-tag>
        <el-tag v-if="answer.model" type="info">{{ answer.model }}</el-tag>
      </div>
      <el-table :data="answer.references || answer.chunks || []" height="240">
        <el-table-column prop="content" label="引用片段" />
        <el-table-column prop="score" label="分数" width="120" />
      </el-table>
    </section>
  </PageContainer>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import PageContainer from '@/components/common/PageContainer.vue'
import { ragApi } from '@/api/rag'
import type { KnowledgeBase, RagAskResult } from '@/types/rag'

const kbs = ref<KnowledgeBase[]>([])
const kbId = ref(0)
const question = ref('')
const answer = ref<RagAskResult>({})
const ask = async () => (answer.value = await ragApi.ask({ knowledge_base_id: kbId.value, question: question.value, top_k: 5 }))
onMounted(async () => {
  kbs.value = await ragApi.knowledgeBases().catch(() => [])
  kbId.value = kbs.value[0]?.id || 0
})
</script>

<style scoped>
.answer-text {
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  line-height: 1.7;
}
</style>
