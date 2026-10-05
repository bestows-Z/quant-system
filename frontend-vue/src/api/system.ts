import { requestList } from './request'

export interface SystemHealth {
  name: string
  status: 'UP' | 'DOWN' | 'UNKNOWN'
  message?: string
  latency_ms?: number
  checked_at?: string
  capability?: string
  last_success_at?: string
  last_error?: string
  enabled?: boolean
}

const fallback = async <T>(promise: Promise<T>, data: T) => {
  try {
    return await promise
  } catch {
    return data
  }
}

export const systemApi = {
  health: () =>
    fallback(
      requestList<SystemHealth>({ url: '/api/system/health', method: 'GET' }),
      [
        { name: 'gateway-api', status: 'UNKNOWN', message: '后端健康接口未启用或暂不可用' },
        { name: 'quant-python', status: 'UNKNOWN', message: '待接入健康检查' },
        { name: 'MySQL', status: 'UNKNOWN', message: '待接入健康检查' },
        { name: 'Redis', status: 'UNKNOWN', message: '待接入健康检查' },
        { name: 'RabbitMQ', status: 'UNKNOWN', message: '待接入健康检查' }
      ] as SystemHealth[]
    ),
  datasource: () =>
    fallback(requestList<SystemHealth>({ url: '/api/system/datasource', method: 'GET' }), [
      { name: 'AKShare', status: 'UNKNOWN', message: '行情源状态由 quant-python 提供' },
      { name: 'Tushare Pro', status: 'UNKNOWN', message: 'Token 预留' },
      { name: 'BaoStock', status: 'UNKNOWN', message: '数据源预留' }
    ] as SystemHealth[])
}
