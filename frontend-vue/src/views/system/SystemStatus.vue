<template>
  <PageContainer title="系统状态" description="Gateway、RPC、存储和实时链路状态">
    <template #actions><el-button @click="load">刷新</el-button></template>
    <div class="section-grid">
      <DataCard class="span-3" label="WebSocket" :value="ws.connected ? 'UP' : 'UNKNOWN'" sub="浏览器实时连接" />
      <DataCard class="span-3" label="重连状态" :value="ws.reconnecting ? '重连中' : '正常'" />
      <DataCard class="span-3" label="订阅数" :value="ws.subscriptions.length" />
      <DataCard class="span-3" label="最后消息" :value="ws.lastMessage?.type || '--'" />
    </div>
    <section class="data-table-card">
      <el-table :data="items" height="520">
        <el-table-column prop="name" label="服务" />
        <el-table-column prop="status" label="状态" width="140">
          <template #default="{ row }"><el-tag :type="row.status === 'UP' ? 'success' : row.status === 'DOWN' ? 'danger' : 'info'">{{ row.status }}</el-tag></template>
        </el-table-column>
        <el-table-column label="延迟" width="110"><template #default="{ row }">{{ row.latency_ms ?? '--' }} ms</template></el-table-column>
        <el-table-column prop="message" label="说明" />
        <el-table-column label="检查时间" min-width="190"><template #default="{ row }">{{ formatTime(row.checked_at) }}</template></el-table-column>
      </el-table>
    </section>
  </PageContainer>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import PageContainer from '@/components/common/PageContainer.vue'
import DataCard from '@/components/common/DataCard.vue'
import { systemApi, type SystemHealth } from '@/api/system'
import { useWebsocketStore } from '@/stores/websocket'
import { formatTime } from '@/utils/time'

const ws = useWebsocketStore()
const items = ref<SystemHealth[]>([])
const load = async () => (items.value = await systemApi.health())
onMounted(load)
</script>
