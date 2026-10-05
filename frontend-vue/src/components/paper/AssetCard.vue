<template>
  <DataCard :label="label" :sub="sub">
    {{ money(value) }}
  </DataCard>
</template>

<script setup lang="ts">
import DataCard from '@/components/common/DataCard.vue'
import { money } from '@/utils/format'
defineProps<{ label: string; value?: number; sub?: string }>()
</script>
