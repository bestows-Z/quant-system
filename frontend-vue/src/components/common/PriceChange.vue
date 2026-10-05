<template>
  <span :class="ratioColorClass(value)" class="mono">{{ text }}</span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { percent, signed } from '@/utils/format'
import { ratioColorClass } from '@/utils/decimal'

const props = defineProps<{ value?: number | string; percentMode?: boolean }>()
const text = computed(() => (props.percentMode ? percent(props.value) : signed(props.value)))
</script>
