<template>
  <PageContainer title="任务中心" description="异步任务状态、日志和取消操作">
    <template #actions>
      <el-select v-model="status" clearable placeholder="状态" style="width: 150px"><el-option label="运行中" value="RUNNING" /><el-option label="成功" value="SUCCESS" /><el-option label="失败" value="FAILED" /></el-select>
      <el-button type="primary" @click="load">筛选</el-button>
    </template>
    <div class="section-grid">
      <section class="data-table-card span-8">
        <el-table :data="items" height="620" @row-click="select">
          <el-table-column prop="id" label="ID" width="80" />
          <el-table-column prop="title" label="标题" />
          <el-table-column prop="task_type" label="类型" width="140" />
          <el-table-column label="进度" width="180"><template #default="{ row }"><el-progress :percentage="Number(row.progress || 0)" /></template></el-table-column>
          <el-table-column label="状态" width="110"><template #default="{ row }"><TaskStatusTag :value="row.status" /></template></el-table-column>
          <el-table-column label="操作" width="100"><template #default="{ row }"><el-button size="small" @click.stop="cancel(row.id)">取消</el-button></template></el-table-column>
        </el-table>
      </section>
      <section class="data-table-card span-4">
        <div class="toolbar"><strong>任务日志</strong></div>
        <el-timeline>
          <el-timeline-item v-for="log in logs" :key="log.id" :timestamp="formatTime(log.created_at)">{{ notificationLevelLabel(log.level) }}：{{ log.message }}</el-timeline-item>
        </el-timeline>
      </section>
    </div>
  </PageContainer>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import PageContainer from '@/components/common/PageContainer.vue'
import TaskStatusTag from '@/components/common/TaskStatusTag.vue'
import { taskApi } from '@/api/task'
import type { SystemTask, TaskLog } from '@/types/task'
import { notificationLevelLabel } from '@/utils/labels'
import { formatTime } from '@/utils/time'

const items = ref<SystemTask[]>([])
const logs = ref<TaskLog[]>([])
const status = ref('')
const load = async () => (items.value = await taskApi.list({ status: status.value, limit: 100 }))
const select = async (row: SystemTask) => (logs.value = await taskApi.logs(row.id))
const cancel = async (id: number) => {
  await taskApi.cancel(id)
  await load()
}
onMounted(load)
</script>
