<template>
  <PageContainer title="委托记录" description="模拟交易委托和撤单">
    <template #actions><el-button @click="load">刷新</el-button></template>
    <section class="data-table-card">
      <div class="table-scroll-x"><el-table :data="items" max-height="620" style="min-width:900px" empty-text="暂无模拟委托记录">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="symbol" label="股票" />
        <el-table-column label="方向"><template #default="{row}">{{chineseLabel(row.side)}}</template></el-table-column>
        <el-table-column label="类型"><template #default="{row}">{{chineseLabel(row.order_type)}}</template></el-table-column>
        <el-table-column prop="price" label="价格" />
        <el-table-column prop="volume" label="数量" />
        <el-table-column prop="filled_volume" label="成交" />
        <el-table-column label="状态"><template #default="{row}"><TaskStatusTag :value="row.status"/></template></el-table-column>
        <el-table-column label="操作" width="100"><template #default="{ row }"><el-button size="small" type="danger" :disabled="!canCancel(row.status)" @click="cancel(row.id)">撤单</el-button></template></el-table-column>
      </el-table></div>
    </section>
  </PageContainer>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import PageContainer from '@/components/common/PageContainer.vue'
import { paperApi } from '@/api/paper'
import type { PaperOrder } from '@/types/paper'
import TaskStatusTag from '@/components/common/TaskStatusTag.vue'
import { chineseLabel } from '@/utils/labels'

const items = ref<PaperOrder[]>([])
const load = async () => (items.value = await paperApi.orders())
const canCancel=(status?:string)=>['NEW','PENDING','PARTIAL_FILLED'].includes(String(status||'').toUpperCase())
const cancel = async (id: number) => {
  await paperApi.cancelOrder(id)
  await load()
}
onMounted(load)
</script>
