<template>
  <header class="topbar">
    <div class="navigation">
      <el-button class="menu-trigger" :icon="Menu" circle aria-label="打开导航" @click="ui.toggleMobileMenu" />
      <el-breadcrumb separator="/">
        <el-breadcrumb-item>量化工作台</el-breadcrumb-item>
        <el-breadcrumb-item>{{ title }}</el-breadcrumb-item>
      </el-breadcrumb>
    </div>
    <div class="actions">
      <WebSocketStatus />
      <NotificationBell />
      <el-dropdown>
        <el-button class="user-button" text aria-label="用户菜单">
          <span class="username">{{ auth.userInfo?.username || '用户' }}</span>
          <el-icon class="el-icon--right"><ArrowDown /></el-icon>
        </el-button>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item disabled>{{ auth.userInfo?.email || '个人账户' }}</el-dropdown-item>
            <el-dropdown-item divided @click="logout">退出登录</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </div>
  </header>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowDown, Menu } from '@element-plus/icons-vue'
import { useAuthStore } from '@/stores/auth'
import { useWebSocket } from '@/composables/useWebSocket'
import NotificationBell from './NotificationBell.vue'
import WebSocketStatus from './WebSocketStatus.vue'
import { useUiStore } from '@/stores/ui'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const { close } = useWebSocket()
const ui = useUiStore()
const title = computed(() => String(route.meta.title || '工作台'))
const logout = () => {
  close()
  auth.logout()
  router.replace('/login')
}
</script>

<style scoped>
.topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 64px;
  padding: 0 20px;
  background: #fff;
  border-bottom: 1px solid var(--qs-border);
  position: sticky;
  top: 0;
  z-index: 30;
}
.actions {
  display: flex;
  align-items: center;
  gap: 12px;
}
.navigation { display: flex; align-items: center; gap: 10px; min-width: 0; }
.menu-trigger { display: none; }
@media (max-width: 900px) {
  .topbar { height: 56px; padding: 0 12px; }
  .menu-trigger { display: inline-flex; }
  :deep(.el-breadcrumb__item:first-child) { display: none; }
  .actions { gap: 4px; }
}
@media (max-width: 600px) {
  .menu-trigger { width: 44px; height: 44px; }
  .username { display: none; }
  .user-button { width: 40px; padding-inline: 8px; }
  .user-button :deep(.el-icon--right) { margin-left: 0; }
  :deep(.el-breadcrumb__inner) { white-space: nowrap; font-size: 13px; }
}
</style>
