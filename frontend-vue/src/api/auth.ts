import request from './request'
import type { LoginPayload, LoginResult, RegisterPayload } from '@/types/api'

export const authApi = {
  login: (data: LoginPayload) => request<LoginResult>({ url: '/api/auth/login', method: 'POST', data }),
  register: (data: RegisterPayload) => request<unknown>({ url: '/api/auth/register', method: 'POST', data })
}
