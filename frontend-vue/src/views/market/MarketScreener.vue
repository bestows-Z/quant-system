<template>
  <PageContainer title="行情选股器" description="按涨跌幅、成交额、换手率和行业筛选当前行情">
    <template #actions><el-button type="primary" :loading="loading" @click="load">应用筛选</el-button></template>
    <section class="data-table-card filter-panel">
      <el-form :inline="true" label-position="top">
        <el-form-item label="排序指标"><el-select v-model="filters.sort_by"><el-option label="涨跌幅" value="change_percent"/><el-option label="成交额" value="amount"/><el-option label="换手率" value="turnover_rate"/><el-option label="成交量" value="volume"/></el-select></el-form-item>
        <el-form-item label="排序方向"><el-segmented v-model="filters.order" :options="[{label:'从高到低',value:'desc'},{label:'从低到高',value:'asc'}]" /></el-form-item>
        <el-form-item label="行业"><el-input v-model="filters.industry" clearable placeholder="留空表示全部"/></el-form-item>
        <el-form-item label="最低涨跌幅"><el-input-number v-model="filters.min_change" :step="0.01" :min="-1" :max="1"/><small>例如 0.03 表示 3%</small></el-form-item>
        <el-form-item label="最低成交额"><el-input-number v-model="filters.min_amount" :step="10000000" :min="0"/><small>单位：元</small></el-form-item>
      </el-form>
    </section>
    <section class="data-table-card result-card"><div class="toolbar"><strong>筛选结果</strong><el-tag>{{items.length}} 只</el-tag></div><div class="table-scroll-x"><el-table :data="items" v-loading="loading" class="wide-table" empty-text="暂无符合条件的股票">
      <el-table-column prop="symbol" label="代码" width="100"/><el-table-column prop="name" label="名称" min-width="120"/><el-table-column prop="industry" label="行业" min-width="110"/><el-table-column label="现价" width="100"><template #default="{row}">{{money(row.price)}}</template></el-table-column><el-table-column label="涨跌幅" width="100"><template #default="{row}"><PriceChange :value="row.change_percent" percent-mode/></template></el-table-column><el-table-column label="成交额" width="110"><template #default="{row}">{{compactNumber(row.amount)}}</template></el-table-column><el-table-column label="换手率" width="100"><template #default="{row}">{{percent(row.turnover_rate)}}</template></el-table-column><el-table-column label="操作" width="100" fixed="right"><template #default="{row}"><el-button link type="primary" @click="$router.push(`/market/stocks/${row.symbol}`)">查看</el-button></template></el-table-column>
    </el-table></div></section><DisclaimerAlert/>
  </PageContainer>
</template>
<script setup lang="ts">
import{onMounted,reactive,ref}from'vue';import PageContainer from'@/components/common/PageContainer.vue';import DisclaimerAlert from'@/components/common/DisclaimerAlert.vue';import PriceChange from'@/components/common/PriceChange.vue';import{featureApi}from'@/api/feature';import{compactNumber,money,percent}from'@/utils/format';import type{ScreenerStock}from'@/types/feature'
const items=ref<ScreenerStock[]>([]),loading=ref(false),filters=reactive({sort_by:'change_percent',order:'desc',industry:'',min_change:0,min_amount:0,limit:100});const load=async()=>{loading.value=true;try{items.value=await featureApi.screener(filters)}finally{loading.value=false}};onMounted(load)
</script>
<style scoped>.filter-panel small{display:block;color:var(--qs-muted);margin-top:5px}.result-card{margin-top:16px}.wide-table{min-width:900px}.el-form-item{min-width:180px}</style>
