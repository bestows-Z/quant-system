<template>
  <PageContainer title="成交记录" description="模拟撮合成交明细">
    <template #actions><el-button @click="load">刷新</el-button></template>
    <section class="data-table-card">
      <div class="table-scroll-x"><el-table :data="items" max-height="620" style="min-width:900px" empty-text="暂无模拟成交记录">
        <el-table-column prop="trade_no" label="成交编号" />
        <el-table-column prop="symbol" label="股票" />
        <el-table-column label="方向"><template #default="{row}">{{chineseLabel(row.side)}}</template></el-table-column>
        <el-table-column prop="price" label="成交价" />
        <el-table-column prop="volume" label="数量" />
        <el-table-column prop="amount" label="金额" />
        <el-table-column prop="fee" label="手续费" />
        <el-table-column label="时间"><template #default="{ row }">{{ formatTime(row.trade_time) }}</template></el-table-column>
      </el-table></div>
    </section>
  </PageContainer>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import PageContainer from '@/components/common/PageContainer.vue'
import { paperApi } from '@/api/paper'
import type { PaperTrade } from '@/types/paper'
import { chineseLabel } from '@/utils/labels'
import { formatTime } from '@/utils/time'

const items = ref<PaperTrade[]>([])
const load = async () => (items.value = await paperApi.trades())
onMounted(load)
</script>
