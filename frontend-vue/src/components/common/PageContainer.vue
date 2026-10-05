<template>
  <main class="page-container">
    <div class="page-header">
      <div class="title-group">
        <el-tooltip v-if="showBack" content="返回上一页" placement="bottom"><el-button class="back-button" :icon="ArrowLeft" circle aria-label="返回上一页" @click="goBack"/></el-tooltip>
        <div>
        <h1>{{ title }}</h1>
        <p v-if="description">{{ description }}</p>
        </div>
      </div>
      <div v-if="$slots.actions" class="page-actions"><slot name="actions" /></div>
    </div>
    <slot />
  </main>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft } from '@element-plus/icons-vue'

defineProps<{ title: string; description?: string }>()
const route=useRoute(),router=useRouter()
const showBack=computed(()=>!['/dashboard','/login','/register'].includes(route.path))
const goBack=()=>{
  const parentRoutes:Array<[RegExp,string]>=[[/^\/market\/timeshare\//,'/market'],[/^\/market\/stocks\//,'/market'],[/^\/market\/sectors\/[^/]+$/,'/market/sectors'],[/^\/portfolio\/\d+\/analysis$/,'/portfolio'],[/^\/portfolio\/\d+$/,'/portfolio'],[/^\/backtest\/\d+$/,'/backtest'],[/^\/agent\/reports\/\d+$/,'/agent/reports']]
  const parent=parentRoutes.find(([pattern])=>pattern.test(route.path))?.[1]
  if(parent){router.push(parent);return}
  if(window.history.state?.back){router.back();return}
  router.push('/dashboard')
}
</script>

<style scoped>
.page-container {
  width: 100%;
  max-width: 1680px;
  margin: 0 auto;
  padding: 20px;
}
.page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
}
.title-group{display:flex;align-items:flex-start;gap:10px;min-width:0}.back-button{flex:0 0 auto;margin-top:1px}.title-group>div{min-width:0}
.page-actions{display:flex;align-items:center;justify-content:flex-end;flex-wrap:wrap;gap:8px;min-width:0}
h1 {
  margin: 0;
  font-size: 22px;
  font-weight: 650;
}
p {
  margin: 6px 0 0;
  color: var(--qs-muted);
}
@media (max-width: 720px) {
  .page-container { padding: 12px; }
  .page-header { display: grid; grid-template-columns: minmax(0, 1fr); gap: 12px; }
  .page-actions { justify-content:flex-start; }
  h1 { font-size: 20px; }
}
</style>
