import axios, { AxiosError, type AxiosRequestConfig } from 'axios'
import { ElMessage } from 'element-plus'
import router from '@/router'
import { storage } from '@/utils/storage'
import type { ApiResponse } from '@/types/api'

const service = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '',
  timeout: 15000
})

export const httpClient = service

let lastAuthNoticeAt = 0
const handleExpiredSession = () => {
  storage.clearToken()
  if (router.currentRoute.value.path === '/login') return
  void router.replace('/login')
  const now = Date.now()
  if (now - lastAuthNoticeAt > 5000) {
    lastAuthNoticeAt = now
    ElMessage.warning('登录已过期，请重新登录')
  }
}

service.interceptors.request.use((config) => {
  const token = storage.getToken()
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

service.interceptors.response.use(
  (response) => response,
  (error: AxiosError<ApiResponse>) => {
    const status = error.response?.status
    const trace = error.response?.data?.trace_id ? ` trace_id=${error.response.data.trace_id}` : ''
    if (status === 401) {
      handleExpiredSession()
    } else if (error.code === 'ECONNABORTED') {
      ElMessage.error('请求超时，请稍后重试')
    } else {
      ElMessage.error(`${error.response?.data?.message || error.message || '网络请求失败'}${trace}`)
    }
    return Promise.reject(error)
  }
)

export const request = async <T = unknown>(config: AxiosRequestConfig): Promise<T> => {
  const response = await service.request<ApiResponse<T>>(config)
  const body = response.data
  if (typeof body?.code === 'number' && body.code !== 0) {
    if (body.code === 10002) {
      handleExpiredSession()
      throw new Error('登录已过期')
    }
    const suffix = body.trace_id ? ` trace_id=${body.trace_id}` : ''
    const detail = sourceErrorDetail(body.data)
    ElMessage.error(`${body.message || '请求失败'}${detail}${suffix}`)
    throw new Error(`${body.message || '请求失败'}${detail}`)
  }
  if (typeof body?.code === 'number') return unwrapEnvelope(body.data) as T
  return unwrapEnvelope(body) as T
}

const sourceErrorDetail = (data: unknown): string => {
  if (!data || typeof data !== 'object') return ''
  const errors = (data as { errors?: unknown }).errors
  if (!Array.isArray(errors) || errors.length === 0) return ''
  const text = errors
    .slice(0, 3)
    .map((item) => {
      if (!item || typeof item !== 'object') return ''
      const row = item as { source?: unknown; error?: unknown }
      return `${row.source || 'unknown'}: ${row.error || ''}`
    })
    .filter(Boolean)
    .join('；')
  return text ? `（${text}）` : ''
}

export const requestList = async <T = unknown>(config: AxiosRequestConfig): Promise<T[]> => {
  const data = unwrapEnvelope(await request<unknown>(config))
  if (Array.isArray(data)) return data as T[]
  if (data && typeof data === 'object') {
    const payload = data as { items?: unknown; list?: unknown; rows?: unknown; data?: unknown }
    if (Array.isArray(payload.items)) return payload.items as T[]
    if (Array.isArray(payload.list)) return payload.list as T[]
    if (Array.isArray(payload.rows)) return payload.rows as T[]
    const nested = unwrapEnvelope(payload.data)
    if (Array.isArray(nested)) return nested as T[]
    if (nested && typeof nested === 'object') {
      const nestedPayload = nested as { items?: unknown; list?: unknown; rows?: unknown }
      if (Array.isArray(nestedPayload.items)) return nestedPayload.items as T[]
      if (Array.isArray(nestedPayload.list)) return nestedPayload.list as T[]
      if (Array.isArray(nestedPayload.rows)) return nestedPayload.rows as T[]
    }
  }
  return []
}

const unwrapEnvelope = (value: unknown): unknown => {
  let current = value
  for (let i = 0; i < 3; i += 1) {
    if (!current || typeof current !== 'object') return current
    const payload = current as { code?: unknown; data?: unknown; message?: unknown }
    if (typeof payload.code === 'number' && 'data' in payload) {
      current = payload.data
      continue
    }
    return current
  }
  return current
}
export default request
