<template>
  <div class="app-shell">
    <div v-if="ui.mobileMenuOpen" class="sidebar-mask" @click="ui.closeMobileMenu" />
    <SidebarMenu />
    <div class="main">
      <TopNavbar />
      <router-view />
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import SidebarMenu from './SidebarMenu.vue'
import TopNavbar from './TopNavbar.vue'
import { useAuthStore } from '@/stores/auth'
import { useWebSocket } from '@/composables/useWebSocket'
import { useUiStore } from '@/stores/ui'

const auth = useAuthStore()
const { connect } = useWebSocket()
const ui = useUiStore()

onMounted(async () => {
  await auth.fetchProfile().catch(() => undefined)
  connect('notify')
})
</script>

<style scoped>
.app-shell {
  display: flex;
  height: 100dvh;
  overflow: hidden;
}
.main {
  flex: 1;
  min-width: 0;
  height: 100dvh;
  overflow-y: auto;
  overscroll-behavior-y: contain;
  background: var(--qs-bg);
}
@media (max-width: 900px) {
  .app-shell {
    display: block;
    height: 100dvh;
  }
  .sidebar-mask { position: fixed; inset: 0; z-index: 39; background: rgba(15, 23, 42, 0.48); }
}
</style>
