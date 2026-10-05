import request from './request'
import type { UserInfo } from '@/types/api'

export const userApi = {
  profile: () => request<UserInfo>({ url: '/api/user/profile', method: 'GET' })
}
