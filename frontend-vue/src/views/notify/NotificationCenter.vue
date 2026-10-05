<template>
  <PageContainer title="通知中心" description="系统通知、风险提示和任务消息">
    <template #actions>
      <el-button :loading="loading" @click="load">刷新</el-button>
      <el-button type="primary" :disabled="notify.unreadCount === 0" :loading="markingAll" @click="markAllRead">全部已读</el-button>
	  <el-popconfirm title="确认删除全部已读通知？" @confirm="clearRead"><template #reference><el-button type="danger" plain :disabled="!notify.notifications.some(item=>item.is_read)" :loading="clearingRead">清理已读</el-button></template></el-popconfirm>
	  <el-popconfirm title="确认清空当前账号的全部通知？此操作不可恢复。" @confirm="clearAll"><template #reference><el-button type="danger" :disabled="!notify.notifications.length" :loading="clearingAll">清空全部</el-button></template></el-popconfirm>
    </template>
    <section class="data-table-card">
      <div class="toolbar filter-toolbar">
        <el-radio-group v-model="statusFilter" size="small">
          <el-radio-button value="all">全部 {{ notify.notifications.length }}</el-radio-button>
          <el-radio-button value="unread">未读 {{ notify.unreadCount }}</el-radio-button>
          <el-radio-button value="read">已读</el-radio-button>
        </el-radio-group>
        <el-select v-model="levelFilter" size="small" style="width: 120px" aria-label="通知等级">
          <el-option label="全部等级" value="all" />
          <el-option label="风险提示" value="WARNING" />
          <el-option label="错误" value="ERROR" />
          <el-option label="普通" value="INFO" />
        </el-select>
      </div>
      <div class="table-scroll-x">
      <el-table :data="filteredNotifications" max-height="620" class="notify-table" empty-text="当前分类暂无通知">
        <el-table-column label="状态" width="80"><template #default="{ row }"><el-tag :type="row.is_read ? 'info' : 'success'">{{ row.is_read ? '已读' : '未读' }}</el-tag></template></el-table-column>
        <el-table-column prop="title" label="标题" />
        <el-table-column prop="content" label="内容" />
        <el-table-column label="等级" width="110"><template #default="{row}">{{notificationLevelLabel(row.level)}}</template></el-table-column>
        <el-table-column label="时间" width="180"><template #default="{ row }">{{ formatTime(row.created_at) }}</template></el-table-column>
        <el-table-column label="操作" width="260" fixed="right">
          <template #default="{ row }">
            <el-button v-if="notificationLink(row)" size="small" type="primary" @click="openNotification(row)">查看详情</el-button>
            <el-button size="small" :disabled="row.is_read" :loading="busyIds.has(row.id)" @click="markRead(row)">{{ row.is_read ? '已读' : '标为已读' }}</el-button>
            <el-popconfirm title="确认删除这条通知？" @confirm="remove(row.id)">
              <template #reference><el-button size="small" type="danger" :loading="busyIds.has(row.id)">删除</el-button></template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
      </div>
    </section>
  </PageContainer>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import PageContainer from '@/components/common/PageContainer.vue'
import { useNotifyStore } from '@/stores/notify'
import type { Notification } from '@/types/notify'
import { notificationLevelLabel } from '@/utils/labels'
import { formatTime } from '@/utils/time'

const notify = useNotifyStore()
const router = useRouter()
const loading = ref(false)
const markingAll = ref(false)
const clearingRead=ref(false)
const clearingAll=ref(false)
const busyIds = ref(new Set<number>())
const statusFilter = ref<'all' | 'unread' | 'read'>('all')
const levelFilter = ref('all')
const filteredNotifications = computed(() => notify.notifications.filter((item) => {
  const statusMatches = statusFilter.value === 'all' || (statusFilter.value === 'unread' ? !item.is_read : item.is_read)
  const levelMatches = levelFilter.value === 'all' || String(item.level || 'INFO').toUpperCase() === levelFilter.value
  return statusMatches && levelMatches
}))
const load = async () => {
  loading.value = true
  try { await Promise.all([notify.fetchNotifications({ limit: 100 }), notify.fetchUnreadCount()]) } finally { loading.value = false }
}
const markRead = async (row: Notification) => {
  if (row.is_read || busyIds.value.has(row.id)) return
  busyIds.value = new Set(busyIds.value).add(row.id)
  try { await notify.markRead(row.id) } finally { const next = new Set(busyIds.value); next.delete(row.id); busyIds.value = next }
}
const markAllRead = async () => {
  if (!notify.unreadCount) return
  markingAll.value = true
  try { await notify.markAllRead() } finally { markingAll.value = false }
}
const remove = async (id: number) => {
  if (busyIds.value.has(id)) return
  busyIds.value = new Set(busyIds.value).add(id)
  try { await notify.remove(id) } finally { const next = new Set(busyIds.value); next.delete(id); busyIds.value = next }
}
const clearRead=async()=>{
	clearingRead.value=true
	try{await notify.clear('read')}finally{clearingRead.value=false}
}
const clearAll=async()=>{
	clearingAll.value=true
	try{await notify.clear('all')}finally{clearingAll.value=false}
}
const sourceRoutes: Record<string,string> = { STRATEGY_SIGNAL:'/signals',RISK_ALERT:'/risk',RISK_WARNING:'/risk',TASK:'/tasks',BACKTEST:'/backtest',PAPER_ORDER:'/paper/orders',PAPER_TRADE:'/paper/trades',AGENT:'/agent',AGENT_REPORT:'/agent' }
const notificationLink = (row: Notification) => {
  const direct = String(row.link || '').trim()
  if (direct.startsWith('/') && !direct.startsWith('//')) return direct
  return sourceRoutes[String(row.notify_type || row.source_type || '').toUpperCase()] || ''
}
const openNotification = async (row: Notification) => {
  if (!row.is_read) await notify.markRead(row.id)
  const target = notificationLink(row)
  if (target) await router.push(target)
}
onMounted(load)
</script>

<style scoped>
.filter-toolbar { justify-content: space-between; }
.table-scroll-x { width: 100%; overflow-x: auto; overscroll-behavior-x: contain; }
.notify-table { min-width: 940px; }
</style>
