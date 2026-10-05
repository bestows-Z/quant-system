<template>
  <PageContainer title="研究总览" description="从市场数据到策略验证，查看当前最重要的状态">
    <section class="welcome-panel" aria-label="常用研究入口"><div><span class="eyebrow">研究工作流</span><h2>从行情观察，到策略验证</h2><p>使用真实市场数据检查信号，再用可复现的回测报告核对风险和交易成本。</p></div><div class="quick-actions"><el-button type="primary" @click="$router.push('/backtest')">新建回测</el-button><el-button @click="$router.push('/market/overview')">查看市场</el-button></div></section>
    <div class="section-grid">
      <DataCard class="span-3" label="未读通知" :value="notify.unreadCount" sub="实时通知中心" />
      <DataCard class="span-3" label="运行任务" :value="task.runningTasks.length" sub="异步任务中心" />
      <DataCard class="span-3" label="行情连接" :value="ws.connected ? '已连接' : '未连接'" sub="实时推送状态" />
      <DataCard class="span-3" label="模拟账户" :value="accounts.length" sub="策略执行账户" />
    </div>
    <div class="section-grid">
      <section class="data-table-card span-8">
        <div class="toolbar">
          <strong>实时行情</strong>
          <el-button size="small" @click="loadMarket">刷新</el-button>
        </div>
        <el-table :data="quotes" height="320" empty-text="当前暂无行情数据，请检查数据源状态">
          <el-table-column prop="symbol" label="代码" width="110" />
          <el-table-column prop="name" label="名称" />
          <el-table-column label="最新价"><template #default="{ row }">{{ money(row.price ?? row.current) }}</template></el-table-column>
          <el-table-column label="涨跌幅"><template #default="{ row }"><PriceChange :value="row.pct_change ?? row.change_percent" percent-mode /></template></el-table-column>
          <el-table-column label="成交量"><template #default="{ row }">{{ compactNumber(row.volume) }}</template></el-table-column>
        </el-table>
      </section>
      <section class="data-table-card span-4">
        <div class="toolbar"><strong>最新任务</strong><el-button link type="primary" @click="$router.push('/tasks')">查看全部</el-button></div>
        <el-empty v-if="!task.latestTasks.length" description="暂无任务" :image-size="72" />
        <el-timeline>
          <el-timeline-item v-for="item in task.latestTasks.slice(0, 6)" :key="item.id" :timestamp="formatTime(item.created_at)">
            <TaskStatusTag :value="item.status" /> {{ item.title || item.task_type }}
          </el-timeline-item>
        </el-timeline>
      </section>
    </div>
    <div class="section-grid">
      <section class="data-table-card span-6">
        <div class="toolbar"><strong>最新通知</strong><el-button link type="primary" @click="$router.push('/notifications')">查看全部</el-button></div>
        <el-table :data="notify.notifications.slice(0, 5)" height="240" empty-text="暂无通知">
          <el-table-column prop="title" label="标题" />
          <el-table-column prop="level" label="等级" width="100" />
        </el-table>
      </section>
      <section class="data-table-card span-6">
        <div class="toolbar"><strong>最新策略信号</strong><el-button link type="primary" @click="$router.push('/signals')">查看全部</el-button></div>
        <el-table :data="signals" height="240" empty-text="暂无策略信号">
          <el-table-column prop="symbol" label="代码" width="100" />
          <el-table-column label="信号" width="110"><template #default="{ row }"><SignalTag :value="row.signal_type || row.action" /></template></el-table-column>
          <el-table-column prop="reason" label="原因" />
        </el-table>
      </section>
    </div>
    <DisclaimerAlert />
  </PageContainer>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import PageContainer from '@/components/common/PageContainer.vue'
import DataCard from '@/components/common/DataCard.vue'
import DisclaimerAlert from '@/components/common/DisclaimerAlert.vue'
import PriceChange from '@/components/common/PriceChange.vue'
import SignalTag from '@/components/common/SignalTag.vue'
import TaskStatusTag from '@/components/common/TaskStatusTag.vue'
import { marketApi } from '@/api/market'
import { paperApi } from '@/api/paper'
import { strategyApi } from '@/api/strategy'
import { useNotifyStore } from '@/stores/notify'
import { useTaskStore } from '@/stores/task'
import { useWebsocketStore } from '@/stores/websocket'
import { compactNumber, money } from '@/utils/format'
import { formatTime } from '@/utils/time'
import type { PaperAccount } from '@/types/paper'
import type { RealtimeQuote } from '@/types/market'
import type { StrategySignal } from '@/types/strategy'

const notify = useNotifyStore()
const task = useTaskStore()
const ws = useWebsocketStore()
const quotes = ref<RealtimeQuote[]>([])
const accounts = ref<PaperAccount[]>([])
const signals = ref<StrategySignal[]>([])
const loadMarket = async () => {
  quotes.value = await marketApi.realtime().catch(() => [])
}
onMounted(async () => {
  await Promise.allSettled([
    loadMarket(),
    notify.fetchUnreadCount(),
    notify.fetchNotifications({ limit: 5 }),
    task.fetchLatest(),
    paperApi.accounts().then((res) => (accounts.value = res)),
    strategyApi.signals({ limit: 6 }).then((res) => (signals.value = res))
  ])
})
</script>

<style scoped>
.welcome-panel{display:flex;justify-content:space-between;align-items:center;gap:24px;background:var(--qs-sidebar);color:#f5f9fb;border-radius:12px;padding:24px 28px;margin-bottom:16px}.welcome-panel h2{font-size:22px;margin:4px 0 8px;line-height:1.35}.welcome-panel p{margin:0;color:#c2d0da;line-height:1.6}.eyebrow{font-size:11px;letter-spacing:.1em;color:#7ce0cc;font-weight:700}.quick-actions{display:flex;gap:8px;flex-shrink:0}.quick-actions .el-button{min-height:42px}.section-grid{margin-bottom:16px}.toolbar{justify-content:space-between}@media(max-width:760px){.welcome-panel{align-items:stretch;flex-direction:column;padding:20px}.quick-actions{flex-wrap:wrap}.quick-actions .el-button{margin:0}}
</style>
