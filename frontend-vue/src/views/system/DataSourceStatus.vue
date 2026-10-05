<template>
  <PageContainer title="数据源状态" description="行情、历史数据和向量库接入状态">
    <template #actions><el-button @click="load">刷新</el-button></template>
    <section class="data-table-card">
      <el-table :data="items" height="520">
        <el-table-column prop="name" label="数据源" />
        <el-table-column prop="status" label="状态" width="140">
          <template #default="{ row }"><el-tag :type="row.status === 'UP' ? 'success' : row.status === 'DOWN' ? 'danger' : 'info'">{{ row.status }}</el-tag></template>
        </el-table-column>
        <el-table-column prop="capability" label="能力" min-width="180" />
        <el-table-column label="延迟" width="110"><template #default="{ row }">{{ row.latency_ms ?? '--' }} ms</template></el-table-column>
        <el-table-column prop="message" label="说明" />
        <el-table-column prop="last_error" label="最近错误" min-width="220" show-overflow-tooltip />
      </el-table>
    </section>
  </PageContainer>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import PageContainer from '@/components/common/PageContainer.vue'
import { systemApi, type SystemHealth } from '@/api/system'

const items = ref<SystemHealth[]>([])
const load = async () => (items.value = await systemApi.datasource())
onMounted(load)
</script>
