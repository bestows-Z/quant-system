<template>
  <el-tag :type="tagType" effect="light">{{ riskLabel(value) }}</el-tag>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { riskLabel } from '@/utils/labels'
const props = defineProps<{ value?: string }>()
const tagType = computed(() => {
  if (props.value === 'LOW') return 'success'
  if (props.value === 'MEDIUM') return 'warning'
  if (props.value === 'HIGH' || props.value === 'EXTREME') return 'danger'
  return 'info'
})
</script>
