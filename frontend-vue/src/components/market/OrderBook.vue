<template>
  <div class="order-book-wrap">
    <div class="book-toolbar">
      <span class="muted">委托量</span>
      <el-radio-group v-model="displayMode" size="small" aria-label="盘口委托量单位">
        <el-radio-button value="hands">手</el-radio-button>
        <el-radio-button value="amount">金额</el-radio-button>
      </el-radio-group>
    </div>
    <div class="order-book mono">
    <div v-for="item in displayAsks" :key="`ask-${item.level}`" class="level-row">
      <span class="label">卖{{ item.level }}</span><span class="down">{{ price(item.price) }}</span><span>{{ size(item.price, item.volume) }}</span>
    </div>
    <div class="latest"><span>最新</span><strong :class="latestClass">{{ price(book?.price || 0) }}</strong><small>{{ formatTime(book?.quote_time) }}</small></div>
    <div v-for="item in book?.bids || []" :key="`bid-${item.level}`" class="level-row">
      <span class="label">买{{ item.level }}</span><span class="up">{{ price(item.price) }}</span><span>{{ size(item.price, item.volume) }}</span>
    </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import type { OrderBook } from '@/types/market'
import { compactNumber } from '@/utils/format'
import { formatTime } from '@/utils/time'

const props = defineProps<{ book?: OrderBook }>()
const displayMode = ref<'hands' | 'amount'>('hands')
const displayAsks = computed(() => [...(props.book?.asks || [])].reverse())
const latestClass = computed(() => Number(props.book?.price || 0) >= Number(props.book?.pre_close || 0) ? 'up' : 'down')
const price = (value: number) => value ? Number(value).toFixed(2) : '--'
const size = (priceValue: number, shares: number) => displayMode.value === 'amount'
  ? compactNumber(priceValue * shares)
  : `${compactNumber(shares / 100)}手`
</script>

<style scoped>
.order-book-wrap { min-width: 0; }
.book-toolbar { min-height: 34px; display: flex; align-items: center; justify-content: space-between; gap: 10px; margin-bottom: 6px; }
.order-book { min-height: 298px; display: grid; align-content: start; }
.level-row { display: grid; grid-template-columns: 48px minmax(70px, 1fr) minmax(70px, 1fr); min-height: 28px; align-items: center; border-bottom: 1px solid #f0f2f5; text-align: right; }
.level-row .label { color: var(--qs-muted); text-align: left; }
.latest { display: grid; grid-template-columns: 48px minmax(70px, 1fr); align-items: center; min-height: 44px; border-block: 1px solid var(--qs-border); }
.latest strong { font-size: 18px; text-align: right; }
.latest small { grid-column: 1 / -1; color: var(--qs-muted); text-align: right; overflow: hidden; text-overflow: ellipsis; }
</style>
