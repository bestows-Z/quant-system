<template>
  <div class="auth-page">
    <el-card class="auth-card">
      <h1>注册账户</h1>
      <p>创建个人/团队量化分析账号</p>
      <el-form :model="form" label-position="top">
        <el-form-item label="用户名"><el-input v-model="form.username" /></el-form-item>
        <el-form-item label="邮箱"><el-input v-model="form.email" type="email" /></el-form-item>
        <el-form-item label="密码"><el-input v-model="form.password" type="password" show-password /></el-form-item>
        <el-form-item label="确认密码"><el-input v-model="confirm" type="password" show-password /></el-form-item>
        <el-button type="primary" class="full" :loading="loading" @click="submit">注册</el-button>
      </el-form>
      <div class="link">已有账户？<router-link to="/login">登录</router-link></div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { authApi } from '@/api/auth'

const router = useRouter()
const loading = ref(false)
const confirm = ref('')
const form = reactive({ username: '', email: '', password: '' })
const submit = async () => {
  if (!form.username || !form.email || !form.password) return ElMessage.warning('请完整填写注册信息')
  if (form.password !== confirm.value) return ElMessage.warning('两次密码不一致')
  loading.value = true
  try {
    await authApi.register(form)
    ElMessage.success('注册成功，请登录')
    router.replace('/login')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.auth-page {
  display: grid;
  place-items: center;
  min-height: 100vh;
  background: #f5f7fb;
}
.auth-card {
  width: min(440px, calc(100vw - 32px));
}
h1 {
  margin: 0;
  font-size: 23px;
}
p {
  color: var(--qs-muted);
}
.full {
  width: 100%;
}
.link {
  margin-top: 18px;
  text-align: center;
}
</style>
