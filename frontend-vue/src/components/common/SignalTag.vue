<template>
  <el-tag :type="tagType">{{ signalLabel(value) }}</el-tag>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { signalLabel } from '@/utils/labels'
const props = defineProps<{ value?: string }>()
const tagType = computed(() => {
  if (props.value === 'BUY') return 'danger'
  if (props.value === 'SELL') return 'success'
  if (props.value === 'WATCH') return 'warning'
  return 'info'
})
</script>
