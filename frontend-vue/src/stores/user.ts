import { defineStore } from 'pinia'
import type { UserInfo } from '@/types/api'

export const useUserStore = defineStore('user', {
  state: () => ({ profile: null as UserInfo | null }),
  actions: {
    setProfile(profile: UserInfo | null) {
      this.profile = profile
    }
  }
})
