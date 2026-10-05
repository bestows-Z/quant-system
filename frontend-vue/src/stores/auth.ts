import { defineStore } from 'pinia'
import { authApi } from '@/api/auth'
import { userApi } from '@/api/user'
import { storage } from '@/utils/storage'
import type { LoginPayload, UserInfo } from '@/types/api'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    token: storage.getToken(),
    userInfo: null as UserInfo | null
  }),
  getters: {
    isLoggedIn: (state) => Boolean(state.token)
  },
  actions: {
    async login(payload: LoginPayload) {
      const res = await authApi.login(payload)
      this.token = res.token
      storage.setToken(res.token)
      this.userInfo = res.user || null
      await this.fetchProfile()
    },
    async fetchProfile() {
      if (!this.token) return
      this.userInfo = await userApi.profile()
    },
    logout() {
      this.token = ''
      this.userInfo = null
      storage.clearToken()
    }
  }
})
