<template>
  <section class="data-card">
    <div class="label">{{ label }}</div>
    <div class="value mono"><slot>{{ value ?? '--' }}</slot></div>
    <div v-if="sub" class="sub">{{ sub }}</div>
  </section>
</template>

<script setup lang="ts">
defineProps<{ label: string; value?: string | number; sub?: string }>()
</script>

<style scoped>
.data-card {
  min-height: 104px;
  padding: 16px;
  background: var(--qs-surface);
  border: 1px solid var(--qs-border);
  border-radius: 8px;
}
.label {
  color: var(--qs-muted);
}
.value {
  margin-top: 12px;
  font-size: 24px;
  font-weight: 700;
}
.sub {
  margin-top: 8px;
  color: var(--qs-muted);
  font-size: 12px;
}
</style>
