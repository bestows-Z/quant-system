<template>
  <el-tag :type="type">{{ statusLabel(value) }}</el-tag>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { statusLabel } from '@/utils/labels'
const props = defineProps<{ value?: string }>()
const type = computed(() => {
  if (props.value === 'SUCCESS' || props.value === 'COMPLETED') return 'success'
  if (props.value === 'FAILED') return 'danger'
  if (props.value === 'RUNNING') return 'warning'
  return 'info'
})
</script>
