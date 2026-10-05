<template>
  <PageContainer title="模拟账户" description="管理虚拟资金、模拟持仓与账户收益，仅用于研究">
    <template #actions><el-select v-if="accounts.length" v-model="accountId" style="width:220px" @change="loadDetail"><el-option v-for="a in accounts" :key="a.id" :label="a.account_name||a.name||`账户 ${a.id}`" :value="a.id"/></el-select><el-button type="primary" @click="createVisible=true">创建模拟账户</el-button><el-button :disabled="!accountId" @click="analyze">账户风险分析</el-button></template>
    <el-alert title="模拟交易不连接券商，不会产生真实委托。A股当日模拟买入数量会记录为“今买”，次日才可模拟卖出。" type="info" show-icon :closable="false" class="notice"/>
    <section v-if="!accounts.length&&!loading" class="data-table-card empty-account"><el-empty description="还没有模拟账户"><el-button type="primary" @click="createVisible=true">创建第一个模拟账户</el-button></el-empty><p>建议初始资金填写 100,000 元，创建后可从股票详情或模拟下单页开始体验。</p></section>
    <template v-else>
      <div class="section-grid">
        <AssetCard class="span-3" label="总资产" :value="asset?.total_asset" />
        <AssetCard class="span-3" label="可用资金" :value="asset?.available_cash??asset?.cash" />
        <AssetCard class="span-3" label="持仓市值" :value="asset?.market_value" />
        <AssetCard class="span-3" label="总盈亏" :value="asset?.total_profit" />
      </div>
      <section class="data-table-card chart-card"><EquityCurveChart :data="curve" title="账户权益曲线" /></section>
      <section class="data-table-card positions-card">
        <div class="toolbar"><strong>模拟持仓</strong><el-tag>{{positions.length}} 只</el-tag><span class="muted">可用数量不包含当日买入数量</span><el-button type="primary" plain @click="router.push(`/paper/trade?account_id=${accountId}`)">模拟下单</el-button></div>
        <div class="table-scroll-x"><el-table :data="positions" v-loading="loading" class="positions-table" empty-text="当前没有模拟持仓，可点击“模拟下单”创建第一笔委托">
          <el-table-column prop="symbol" label="代码" width="100"/><el-table-column prop="name" label="名称" width="120"/><el-table-column prop="volume" label="总持仓" width="100"/><el-table-column prop="available_volume" label="可卖数量" width="110"/><el-table-column prop="today_buy_volume" label="今日买入(T+1)" width="130"/><el-table-column label="成本价" width="100"><template #default="{row}">{{money(row.avg_cost??row.cost_price)}}</template></el-table-column><el-table-column label="现价" width="100"><template #default="{row}">{{money(row.market_price??row.current_price)}}</template></el-table-column><el-table-column label="市值" width="120"><template #default="{row}">{{compactNumber(row.market_value)}}</template></el-table-column><el-table-column label="持仓盈亏" width="120"><template #default="{row}"><PriceChange :value="row.profit"/></template></el-table-column><el-table-column label="盈亏比例" width="110"><template #default="{row}"><PriceChange :value="row.profit_rate" percent-mode/></template></el-table-column><el-table-column label="操作" width="90" fixed="right"><template #default="{row}"><el-button link type="primary" @click="router.push(`/market/stocks/${row.symbol}`)">查看</el-button></template></el-table-column>
        </el-table></div>
      </section>
    </template>
    <DisclaimerAlert/>
    <el-dialog v-model="createVisible" title="创建模拟账户" width="min(460px,94vw)"><el-form label-position="top"><el-form-item label="账户名称"><el-input v-model="form.account_name" placeholder="例如：趋势策略测试账户"/></el-form-item><el-form-item label="初始虚拟资金"><el-input-number v-model="form.initial_cash" :min="10000" :max="100000000" :step="10000" style="width:100%"/><small>仅用于模拟计算，不涉及真实资金。新手建议 100,000 元。</small></el-form-item></el-form><template #footer><el-button @click="createVisible=false">取消</el-button><el-button type="primary" :loading="creating" @click="createAccount">确认创建</el-button></template></el-dialog>
  </PageContainer>
</template>
<script setup lang="ts">
import{onMounted,reactive,ref}from'vue';import{useRouter}from'vue-router';import{ElMessage}from'element-plus';import PageContainer from'@/components/common/PageContainer.vue';import DisclaimerAlert from'@/components/common/DisclaimerAlert.vue';import PriceChange from'@/components/common/PriceChange.vue';import AssetCard from'@/components/paper/AssetCard.vue';import EquityCurveChart from'@/components/charts/EquityCurveChart.vue';import{paperApi}from'@/api/paper';import{riskApi}from'@/api/risk';import{compactNumber,money}from'@/utils/format';import type{EquityPoint}from'@/types/backtest';import type{PaperAccount,PaperPosition}from'@/types/paper'
const router=useRouter(),accounts=ref<PaperAccount[]>([]),accountId=ref(0),asset=ref<PaperAccount>(),positions=ref<PaperPosition[]>([]),curve=ref<EquityPoint[]>([]),loading=ref(false),creating=ref(false),createVisible=ref(false),form=reactive({account_name:'我的模拟账户',initial_cash:100000})
const loadAccounts=async()=>{accounts.value=await paperApi.accounts();if(!accounts.value.some(v=>v.id===accountId.value))accountId.value=accounts.value[0]?.id||0}
const loadDetail=async()=>{if(!accountId.value){asset.value=undefined;positions.value=[];curve.value=[];return}loading.value=true;try{await Promise.all([paperApi.assets(accountId.value).then(r=>asset.value=r),paperApi.positions(accountId.value).then(r=>positions.value=r),paperApi.equityCurve(accountId.value).then(r=>curve.value=r)])}finally{loading.value=false}}
const createAccount=async()=>{if(!form.account_name.trim())return ElMessage.warning('请输入账户名称');creating.value=true;try{const created=await paperApi.createAccount({...form});ElMessage.success('模拟账户创建成功');createVisible.value=false;await loadAccounts();accountId.value=created.id||accounts.value[0]?.id||0;await loadDetail()}finally{creating.value=false}}
const analyze=async()=>{if(!accountId.value)return ElMessage.warning('请先创建或选择模拟账户');await riskApi.analyzeAccount(accountId.value);ElMessage.success('账户风险分析已完成，可在风控中心查看结果')}
onMounted(async()=>{loading.value=true;try{await loadAccounts();await loadDetail()}finally{loading.value=false}})
</script>
<style scoped>.notice{margin-bottom:16px}.empty-account{text-align:center;padding:32px}.empty-account p{color:var(--qs-muted)}.chart-card,.positions-card{margin-top:16px}.positions-table{min-width:1180px}.positions-card .toolbar .el-button{margin-left:auto}.el-form-item small{color:var(--qs-muted);margin-top:6px}</style>
