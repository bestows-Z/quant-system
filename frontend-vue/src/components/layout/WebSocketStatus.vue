<template>
  <el-tag :type="store.connected ? 'success' : store.reconnecting ? 'warning' : 'info'" effect="light">
    {{ store.connected ? '实时已连接' : store.reconnecting ? '重连中' : '实时未连接' }}
  </el-tag>
</template>

<script setup lang="ts">
import { useWebsocketStore } from '@/stores/websocket'
const store = useWebsocketStore()
</script>
