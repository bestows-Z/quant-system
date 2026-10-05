<template>
  <PageContainer title="风控中心" description="风险记录、等级筛选与处理">
    <template #actions>
      <el-select v-model="level" clearable placeholder="风险等级" style="width: 140px"><el-option label="低风险" value="LOW" /><el-option label="中风险" value="MEDIUM" /><el-option label="高风险" value="HIGH" /><el-option label="极高风险" value="EXTREME" /></el-select>
      <el-button type="primary" @click="load">筛选</el-button>
      <el-button :disabled="!selected.length" type="danger" plain @click="removeSelected">删除所选</el-button>
    </template>
    <section class="data-table-card">
      <el-table :data="items" height="620" @selection-change="selected=$event">
        <el-table-column type="selection" width="48" />
        <el-table-column prop="risk_type" label="类型" />
        <el-table-column prop="symbol" label="标的" width="110" />
        <el-table-column prop="title" label="标题" />
        <el-table-column label="等级" width="110"><template #default="{ row }"><RiskTag :value="row.risk_level" /></template></el-table-column>
        <el-table-column label="状态" width="110"><template #default="{row}"><TaskStatusTag :value="row.status"/></template></el-table-column>
        <el-table-column label="时间"><template #default="{ row }">{{ formatTime(row.created_at) }}</template></el-table-column>
        <el-table-column label="操作" width="235"><template #default="{ row }"><el-button size="small" @click="resolve(row.id)">解决</el-button><el-button size="small" @click="ignore(row.id)">忽略</el-button><el-button size="small" type="danger" plain @click="removeOne(row.id)">删除</el-button></template></el-table-column>
      </el-table>
    </section>
    <DisclaimerAlert />
  </PageContainer>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import PageContainer from '@/components/common/PageContainer.vue'
import RiskTag from '@/components/common/RiskTag.vue'
import DisclaimerAlert from '@/components/common/DisclaimerAlert.vue'
import { riskApi } from '@/api/risk'
import type { RiskRecord } from '@/types/risk'
import TaskStatusTag from '@/components/common/TaskStatusTag.vue'
import { formatTime } from '@/utils/time'
import { ElMessage, ElMessageBox } from 'element-plus'

const items = ref<RiskRecord[]>([])
const selected = ref<RiskRecord[]>([])
const level = ref('')
const load = async () => (items.value = await riskApi.records({ level: level.value }))
const resolve = async (id: number) => {
  await riskApi.resolve(id)
  await load()
}
const ignore = async (id: number) => {
  await riskApi.ignore(id)
  await load()
}
const removeOne = async (id: number) => { await ElMessageBox.confirm('确认删除这条风控记录？删除后无法恢复。','删除风控记录',{type:'warning'});await riskApi.remove(id);ElMessage.success('风控记录已删除');await load() }
const removeSelected = async () => { if(!selected.value.length)return;await ElMessageBox.confirm(`确认删除所选 ${selected.value.length} 条风控记录？`,'批量删除',{type:'warning'});const result=await riskApi.removeBatch(selected.value.map(v=>v.id));ElMessage.success(`已删除 ${result.deleted_count} 条风控记录`);await load() }
onMounted(load)
</script>
