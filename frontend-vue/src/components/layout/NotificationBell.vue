<template>
  <el-badge :value="notify.unreadCount" :hidden="notify.unreadCount === 0">
    <el-button :icon="Bell" circle @click="$router.push('/notifications')" aria-label="通知中心" />
  </el-badge>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { Bell } from '@element-plus/icons-vue'
import { useNotifyStore } from '@/stores/notify'

const notify = useNotifyStore()
onMounted(() => notify.fetchUnreadCount().catch(() => undefined))
</script>
