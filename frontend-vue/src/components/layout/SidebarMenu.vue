<template>
  <aside class="sidebar" :class="{ open: ui.mobileMenuOpen }">
    <div class="brand">
      <div class="mark">Q</div>
      <div>
        <strong>量化工作台</strong>
        <span>行情 · 策略 · 风控</span>
      </div>
    </div>
    <el-menu :default-active="$route.path" router background-color="#142631" text-color="#d3dce4" active-text-color="#7ce0cc" @select="ui.closeMobileMenu">
      <el-menu-item index="/dashboard"><el-icon><DataLine /></el-icon><span>首页</span></el-menu-item>
      <el-sub-menu index="/market-menu">
        <template #title><el-icon><TrendCharts /></el-icon><span>市场洞察</span></template>
        <el-menu-item index="/market/overview">市场全景</el-menu-item>
        <el-menu-item index="/market">股票行情</el-menu-item>
        <el-menu-item index="/market/crypto">数字资产现货</el-menu-item>
        <el-menu-item index="/market/sectors">板块强度</el-menu-item>
        <el-menu-item index="/market/screener">行情选股器</el-menu-item>
        <el-menu-item index="/market/candidates">研究候选池</el-menu-item>
        <el-menu-item index="/market/dragon-tiger">龙虎榜</el-menu-item>
        <el-menu-item index="/watchlist">自选股</el-menu-item>
      </el-sub-menu>
      <el-sub-menu index="/research">
        <template #title><el-icon><SetUp /></el-icon><span>量化研究</span></template>
        <el-menu-item index="/strategy">策略配置</el-menu-item>
        <el-menu-item index="/signals">策略信号</el-menu-item>
        <el-menu-item index="/backtest">回测中心</el-menu-item>
        <el-menu-item index="/risk">风控中心</el-menu-item>
      </el-sub-menu>
      <el-sub-menu index="/portfolio-trading">
        <template #title><el-icon><Wallet /></el-icon><span>组合与交易</span></template>
        <el-menu-item index="/portfolio">真实持仓</el-menu-item>
        <el-menu-item index="/paper/account">模拟账户</el-menu-item>
        <el-menu-item index="/paper/auto-trader">自动策略</el-menu-item>
        <el-menu-item index="/paper/trade">模拟下单</el-menu-item>
        <el-menu-item index="/paper/orders">委托记录</el-menu-item>
        <el-menu-item index="/paper/trades">成交记录</el-menu-item>
      </el-sub-menu>
      <el-sub-menu index="/intelligence">
        <template #title><el-icon><Cpu /></el-icon><span>研究工具</span></template>
        <el-menu-item index="/agent">研究分析</el-menu-item>
        <el-menu-item index="/agent/chat">研究问答</el-menu-item>
        <el-menu-item index="/agent/reports">分析档案</el-menu-item>
        <el-menu-item index="/rag/knowledge">知识库</el-menu-item>
        <el-menu-item index="/rag/ask">检索问答</el-menu-item>
      </el-sub-menu>
      <el-sub-menu index="/operations">
        <template #title><el-icon><Operation /></el-icon><span>运营中心</span></template>
        <el-menu-item index="/tasks">任务中心</el-menu-item>
        <el-menu-item index="/notifications">通知中心</el-menu-item>
      </el-sub-menu>
      <el-sub-menu index="/system">
        <template #title><el-icon><Monitor /></el-icon><span>系统状态</span></template>
        <el-menu-item index="/system/status">服务状态</el-menu-item>
        <el-menu-item index="/system/datasource">数据源状态</el-menu-item>
        <el-menu-item index="/system/models">模型配置</el-menu-item>
      </el-sub-menu>
    </el-menu>
  </aside>
</template>

<script setup lang="ts">
import { Cpu, DataLine, Monitor, Operation, SetUp, TrendCharts, Wallet } from '@element-plus/icons-vue'
import { useUiStore } from '@/stores/ui'
const ui = useUiStore()
</script>

<style scoped>
.sidebar {
  width: 248px;
  height: 100dvh;
  background: var(--qs-sidebar);
  color: #fff;
  flex: 0 0 248px;
  overflow-y: auto;
  overscroll-behavior-y: contain;
}
@media (max-width: 900px) {
  .sidebar { position: fixed; inset: 0 auto 0 0; z-index: 40; transform: translateX(-100%); transition: transform 200ms ease-out; box-shadow: 12px 0 32px rgba(15, 23, 42, 0.22); }
  .sidebar.open { transform: translateX(0); }
}
.brand {
  display: flex;
  align-items: center;
  gap: 12px;
  height: 64px;
  padding: 0 18px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
}
.mark {
  display: grid;
  place-items: center;
  width: 34px;
  height: 34px;
  border-radius: 8px;
  background: var(--qs-primary);
  font-weight: 800;
}
strong,
span {
  display: block;
}
span {
  margin-top: 2px;
  font-size: 12px;
  color: #91a3b5;
}
.el-menu {
  border-right: 0;
}
</style>
