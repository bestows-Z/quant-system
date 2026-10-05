<template>
  <PageContainer title="RAG 知识库" description="文档切块、向量索引与 Qdrant collection 管理">
    <template #actions><el-button type="primary" @click="$router.push('/rag/ask')">进入检索问答</el-button></template>
    <div class="section-grid">
      <section class="data-table-card span-4">
        <div class="toolbar"><strong>知识库</strong><el-tag type="success">Qdrant</el-tag></div>
        <el-input v-model="kbName" placeholder="知识库名称" />
        <el-input v-model="kbDesc" placeholder="描述" class="field-gap" />
        <el-button type="primary" class="field-gap" :loading="creatingKb" @click="createKb">创建知识库</el-button>
        <el-divider />
        <el-table :data="kbs" height="390" highlight-current-row @row-click="selectKb">
          <el-table-column prop="name" label="名称" min-width="120" />
          <el-table-column prop="visibility" label="可见性" width="90" />
		  <el-table-column label="操作" width="78"><template #default="{row}"><el-button v-if="row.user_id" link type="danger" @click.stop="deleteKb(row)">删除</el-button><el-tag v-else size="small" type="info">系统</el-tag></template></el-table-column>
        </el-table>
      </section>

      <section class="data-table-card span-8">
        <div class="toolbar">
          <strong>{{ activeKb?.name || '文档与向量索引' }}</strong>
          <el-tag v-if="activeKb?.embedding_provider">{{ activeKb.embedding_provider }}</el-tag>
          <el-tag v-if="activeKb?.vector_store" type="success">{{ activeKb.vector_store }}</el-tag>
          <span v-if="activeKb?.collection_name" class="mono muted collection-name">{{ activeKb.collection_name }}</span>
        </div>

        <el-form label-position="top" class="document-form">
          <el-form-item label="文档标题"><el-input v-model="docTitle" placeholder="例如：风险管理制度" /></el-form-item>
          <el-form-item label="文档内容"><el-input v-model="docContent" type="textarea" :rows="4" placeholder="输入需要构建索引的研究资料、制度或说明文本" /></el-form-item>
          <el-button type="primary" :loading="creatingDoc" @click="createDoc">切块并写入 Qdrant</el-button>
        </el-form>

        <el-divider />
        <el-table :data="docs" height="360">
          <el-table-column prop="title" label="文档" min-width="160" />
          <el-table-column prop="doc_type" label="类型" width="90" />
          <el-table-column prop="chunk_count" label="切块" width="80" />
          <el-table-column prop="status" label="索引状态" width="110" />
		  <el-table-column label="操作" width="78"><template #default="{row}"><el-button v-if="row.user_id" link type="danger" @click="deleteDoc(row)">删除</el-button><span v-else class="muted">只读</span></template></el-table-column>
        </el-table>
      </section>
    </div>

  </PageContainer>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import PageContainer from '@/components/common/PageContainer.vue'
import { ragApi } from '@/api/rag'
import type { KnowledgeBase, RagDocument } from '@/types/rag'

const kbs = ref<KnowledgeBase[]>([])
const docs = ref<RagDocument[]>([])
const selectedKb = ref(0)
const kbName = ref('')
const kbDesc = ref('')
const docTitle = ref('')
const docContent = ref('')
const creatingKb = ref(false)
const creatingDoc = ref(false)
const activeKb = computed(() => kbs.value.find((item) => item.id === selectedKb.value))

const loadKbs = async () => {
  kbs.value = await ragApi.knowledgeBases()
  if (!selectedKb.value && kbs.value.length) await selectKb(kbs.value[0])
}
const selectKb = async (row: KnowledgeBase) => {
  selectedKb.value = row.id
  docs.value = await ragApi.documents(row.id)
}
const createKb = async () => {
  if (!kbName.value.trim()) return ElMessage.warning('请输入知识库名称')
  creatingKb.value = true
  try {
    const created = await ragApi.createKnowledgeBase({ name: kbName.value.trim(), description: kbDesc.value.trim(), visibility: 'PRIVATE' })
    kbName.value = ''
    kbDesc.value = ''
    await loadKbs()
    await selectKb(created)
    ElMessage.success('知识库与 Qdrant collection 已创建')
  } finally {
    creatingKb.value = false
  }
}
const createDoc = async () => {
  if (!selectedKb.value) return ElMessage.warning('请先选择知识库')
  if (!docTitle.value.trim() || !docContent.value.trim()) return ElMessage.warning('请填写文档标题和内容')
  creatingDoc.value = true
  try {
    await ragApi.createDocument({ knowledge_base_id: selectedKb.value, title: docTitle.value.trim(), doc_type: 'TEXT', content: docContent.value.trim() })
    docTitle.value = ''
    docContent.value = ''
    docs.value = await ragApi.documents(selectedKb.value)
    ElMessage.success('文档已切块并完成向量索引')
  } finally {
    creatingDoc.value = false
  }
}
const deleteDoc=async(row:RagDocument)=>{
	await ElMessageBox.confirm(`删除文档“${row.title}”？文档切块会立即停止参与检索。`,'删除文档',{type:'warning',confirmButtonText:'删除',cancelButtonText:'取消'})
	const result=await ragApi.deleteDocument(row.id)
	docs.value=await ragApi.documents(selectedKb.value)
	ElMessage.success(result.vector_cleanup_pending?'文档已删除，向量清理任务正在后台重试':'文档及向量已删除')
}
const deleteKb=async(row:KnowledgeBase)=>{
	await ElMessageBox.confirm(`删除知识库“${row.name}”及其中全部文档？此操作不可恢复。`,'删除知识库',{type:'warning',confirmButtonText:'确认删除',cancelButtonText:'取消'})
	const result=await ragApi.deleteKnowledgeBase(row.id)
	if(selectedKb.value===row.id){selectedKb.value=0;docs.value=[]}
	await loadKbs()
	ElMessage.success(result.vector_cleanup_pending?'知识库已删除，向量集合正在后台清理':'知识库及 Qdrant 集合已删除')
}
onMounted(loadKbs)
</script>

<style scoped>
.field-gap { margin-top: 10px; }
.collection-name { margin-left: auto; max-width: 280px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.document-form { max-width: 760px; }
@media (max-width: 720px) {
  .collection-name { width: 100%; margin-left: 0; }
}
</style>
