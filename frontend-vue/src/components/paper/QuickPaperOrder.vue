<template>
  <el-drawer v-model="visible" title="模拟交易" :size="drawerSize" append-to-body>
    <el-alert title="仅执行 paper trading，不连接真实交易" type="info" show-icon :closable="false" />
	<el-alert :title="safety.message" :type="safety.safe ? 'success' : 'warning'" show-icon :closable="false" class="market-safety" />
    <el-form label-position="top" class="order-form">
      <el-form-item label="模拟账户"><el-select v-model="form.account_id" placeholder="选择账户"><el-option v-for="item in accounts" :key="item.id" :label="item.account_name || item.name" :value="item.id" /></el-select></el-form-item>
      <el-form-item label="方向"><el-segmented v-model="form.side" :options="sideOptions" /></el-form-item>
      <el-form-item label="股票"><el-input :model-value="`${symbol} ${name || ''}`" disabled /></el-form-item>
      <el-form-item label="订单类型"><el-radio-group v-model="form.order_type"><el-radio-button value="MARKET">市价</el-radio-button><el-radio-button value="LIMIT">限价</el-radio-button></el-radio-group></el-form-item>
      <el-form-item v-if="form.order_type === 'LIMIT'" label="委托价"><el-input-number v-model="form.price" :min="0.01" :precision="2" :step="0.01" /></el-form-item>
      <el-form-item label="数量（股）"><el-input-number v-model="form.volume" :min="100" :step="100" /></el-form-item>
      <div class="order-summary mono"><span>{{ form.side === 'BUY' ? '卖一盘口价' : '买一盘口价' }}</span><strong>{{ money(bookPrice) }}</strong><span>委托参考价</span><strong>{{ money(referencePrice) }}</strong><span>预估金额</span><strong>{{ compactNumber(referencePrice * form.volume) }}</strong></div>
      <el-button class="submit" :type="form.side === 'BUY' ? 'danger' : 'success'" :loading="submitting" :disabled="!safety.safe" @click="submit">提交模拟{{ form.side === 'BUY' ? '买入' : '卖出' }}</el-button>
    </el-form>
  </el-drawer>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { paperApi } from '@/api/paper'
import { compactNumber, money } from '@/utils/format'
import { quoteSafety } from '@/utils/marketSession'
import type { OrderBook } from '@/types/market'
import type { PaperAccount } from '@/types/paper'

const props = defineProps<{ modelValue: boolean; symbol: string; name?: string; price?: number; initialSide?: 'BUY' | 'SELL'; orderBook?: OrderBook; quoteTime?: string; dataMode?: string }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; completed: [] }>()
const accounts = ref<PaperAccount[]>([])
const submitting = ref(false)
const visible = computed({ get: () => props.modelValue, set: (value) => emit('update:modelValue', value) })
const drawerSize = computed(() => window.innerWidth <= 720 ? '100%' : '420px')
const bookPrice = computed(() => Number(form.side === 'BUY' ? props.orderBook?.asks?.[0]?.price : props.orderBook?.bids?.[0]?.price) || 0)
const referencePrice = computed(() => Number(form.order_type === 'LIMIT' ? form.price : bookPrice.value || props.price || 0))
const safety = computed(() => quoteSafety(props.orderBook?.quote_time || props.quoteTime, props.orderBook?.data_mode || props.dataMode, Boolean(bookPrice.value)))
const sideOptions = [{ label: '买入', value: 'BUY' }, { label: '卖出', value: 'SELL' }]
const form = reactive({ account_id: 0, side: props.initialSide || 'BUY', order_type: 'MARKET', price: Number(props.price || 0), volume: 100 })

const submit = async () => {
  if (!form.account_id) { ElMessage.warning('请选择模拟账户'); return }
	if (!safety.value.safe) { ElMessage.warning(safety.value.message); return }
  submitting.value = true
  try {
    await paperApi.createOrder({ account_id: form.account_id, client_order_id: `WEB-${Date.now()}`, symbol: props.symbol, name: props.name, side: form.side, order_type: form.order_type, price: form.order_type === 'LIMIT' ? form.price : 0, volume: form.volume })
    ElMessage.success('模拟委托已提交')
    visible.value = false
    emit('completed')
  } finally { submitting.value = false }
}
onMounted(async () => { accounts.value = await paperApi.accounts(); if (accounts.value.length) form.account_id = accounts.value[0].id })
watch(() => props.initialSide, (value) => { if (value) form.side = value })
watch(() => props.price, (value) => { if (value && form.order_type === 'LIMIT') form.price = Number(value) })
</script>

<style scoped>
.order-form { margin-top: 18px; }
.market-safety { margin-top: 12px; }
.order-summary { display:grid;grid-template-columns:1fr auto;gap:10px;padding:12px;background:#f5f7fa;margin-bottom:16px; }
.submit { width:100%;min-height:44px; }
</style>
