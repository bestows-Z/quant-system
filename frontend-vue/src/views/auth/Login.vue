<template>
  <div class="auth-page">
    <div class="auth-intro">
      <div class="brand-mark">Q<span>·</span>S</div>
      <div class="eyebrow">QUANT SYSTEM / WORKSPACE</div>
      <h1>从行情到策略，<br />每一步都有依据。</h1>
      <p>统一查看数据、回测、组合和风控。策略表现与成交假设可以追溯，不用猜结果从哪里来。</p>
      <div class="intro-foot">行情 · 策略 · 回测 · 风控</div>
    </div>
    <el-card class="auth-card">
      <div class="form-eyebrow">欢迎回来</div>
      <h2>登录量化工作台</h2>
      <p>继续查看你的策略和研究记录</p>
      <el-form :model="form" label-position="top" @submit.prevent="submit">
        <el-form-item label="用户名 / 邮箱"><el-input v-model="form.username" autocomplete="username" /></el-form-item>
        <el-form-item label="密码"><el-input v-model="form.password" type="password" show-password autocomplete="current-password" /></el-form-item>
        <el-button type="primary" class="full" :loading="loading" @click="submit">登录工作台</el-button>
      </el-form>
      <div class="link">没有账户？<router-link to="/register">注册</router-link></div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()
const loading = ref(false)
const form = reactive({ username: '', password: '' })
const submit = async () => {
  if (!form.username || !form.password) return ElMessage.warning('请输入用户名和密码')
  loading.value = true
  try {
    await auth.login(form)
    router.replace(String(route.query.redirect || '/dashboard'))
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.auth-page {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(380px, 42%);
  min-height: 100vh;
  background: #f3f6f8;
}
.auth-intro {
  display: flex;
  flex-direction: column;
  justify-content: center;
  padding: clamp(32px, 8vw, 120px);
  color: #f4f8f8;
  background: #142631;
}
.brand-mark {
  display: flex;
  gap: 1px;
  place-items: center;
  width: 52px;
  height: 52px;
  margin-bottom: 64px;
  border: 1px solid #498b83;
  border-radius: 10px;
  color: #fff;
  font-size: 19px;
  font-weight: 700;
  letter-spacing: -.08em;
}
.brand-mark span { color: #45b7a9; }
.eyebrow, .form-eyebrow {
  font-size: 11px;
  font-weight: 700;
  letter-spacing: .17em;
}
.eyebrow { color: #71bdb2; }
.auth-intro h1 {
  margin: 20px 0 18px;
  max-width: 650px;
  font-size: clamp(36px, 4.4vw, 64px);
  line-height: 1.2;
  letter-spacing: -.04em;
}
.auth-intro p {
  max-width: 480px;
  margin: 0;
  color: #bed0d5;
  font-size: 16px;
  line-height: 1.8;
}
.intro-foot {
  margin-top: 72px;
  padding-top: 20px;
  border-top: 1px solid #38515a;
  color: #9db3b8;
  font-size: 12px;
  letter-spacing: .1em;
}
.auth-card {
  align-self: center;
  justify-self: center;
  width: min(420px, calc(100% - 48px));
  border: 1px solid var(--qs-border);
  border-radius: 12px;
  box-shadow: 0 18px 50px rgba(18, 39, 51, .08);
}
.auth-card :deep(.el-card__body) { padding: 34px; }
.form-eyebrow { color: var(--qs-primary); }
h2 {
  margin: 10px 0 4px;
  font-size: 26px;
  letter-spacing: -.03em;
}
.auth-card p {
  margin: 0 0 28px;
  color: var(--qs-muted);
}
.full {
  width: 100%;
  margin-top: 8px;
  min-height: 44px;
}
.link {
  margin-top: 18px;
  text-align: center;
}
@media (max-width: 760px) {
  .auth-page { display: block; }
  .auth-intro { min-height: 250px; padding: 28px 24px 52px; }
  .brand-mark { width: 40px; height: 40px; margin-bottom: 26px; font-size: 15px; }
  .auth-intro h1 { margin: 10px 0; font-size: 30px; }
  .auth-intro p, .intro-foot { display: none; }
  .auth-card { position: relative; width: min(420px, calc(100% - 32px)); margin: -24px auto 28px; }
  .auth-card :deep(.el-card__body) { padding: 26px; }
}
</style>
