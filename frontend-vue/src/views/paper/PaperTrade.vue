<template>
  <PageContainer title="模拟下单" description="仅做虚拟撮合，不接入真实交易">
    <template #actions><el-button @click="$router.push('/paper/account')">账户与持仓</el-button><el-button @click="$router.push('/paper/orders')">委托记录</el-button></template>
      <section class="data-table-card order-panel">
        <div class="toolbar"><strong>模拟下单</strong></div>
        <el-form label-position="top" :model="order" class="order-form">
          <el-form-item label="账户"><el-select v-model="order.account_id" style="width: 180px"><el-option v-for="a in accounts" :key="a.id" :label="a.account_name || a.name || a.id" :value="a.id" /></el-select></el-form-item>
          <el-form-item label="股票"><el-input v-model="order.symbol" placeholder="000001" /></el-form-item>
          <el-form-item label="方向"><el-select v-model="order.side" style="width: 110px"><el-option label="买入" value="BUY" /><el-option label="卖出" value="SELL" /></el-select></el-form-item>
          <el-form-item label="类型"><el-select v-model="order.order_type" style="width: 110px"><el-option label="市价" value="MARKET" /><el-option label="限价" value="LIMIT" /></el-select></el-form-item>
          <el-form-item label="价格"><el-input-number v-model="order.price" :disabled="order.order_type === 'MARKET'" /></el-form-item>
          <el-form-item label="数量"><el-input-number v-model="order.volume" :min="100" :step="100" /></el-form-item>
          <el-form-item><el-button type="primary" @click="submitOrder">提交委托</el-button></el-form-item>
        </el-form>
      </section>
    <DisclaimerAlert />
  </PageContainer>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import PageContainer from '@/components/common/PageContainer.vue'
import DisclaimerAlert from '@/components/common/DisclaimerAlert.vue'
import { paperApi } from '@/api/paper'
import type { PaperAccount } from '@/types/paper'

const accounts = ref<PaperAccount[]>([])
const order = reactive({ account_id: 0, symbol: '', name: '', side: 'BUY', order_type: 'MARKET', price: 0, volume: 100, client_order_id: '' })
const load = async () => {
  accounts.value = await paperApi.accounts()
  order.account_id ||= accounts.value[0]?.id || 0
}
const submitOrder = async () => {
  order.client_order_id = `WEB-${Date.now()}`
  await paperApi.createOrder(order)
  ElMessage.success('模拟委托已提交')
}
onMounted(load)
</script>

<style scoped>
.order-panel { max-width: 1120px; }
.order-form { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 0 16px; }
.order-form :deep(.el-select), .order-form :deep(.el-input-number) { width: 100% !important; }
@media (max-width: 900px) { .order-form { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
@media (max-width: 600px) { .order-form { grid-template-columns: 1fr; } }
</style>
