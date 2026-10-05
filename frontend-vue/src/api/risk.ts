import request, { requestList } from './request'
import type { RiskRecord } from '@/types/risk'

export const riskApi = {
  records: (params?: Record<string, unknown>) => requestList<RiskRecord>({ url: '/api/risk/records', method: 'GET', params }),
  record: (id: number) => request<RiskRecord>({ url: `/api/risk/records/${id}`, method: 'GET' }),
  analyzeStock: (symbol: string) => request<RiskRecord>({ url: `/api/risk/stock/${symbol}/analyze`, method: 'POST' }),
  analyzeAccount: (id: number) => request<RiskRecord>({ url: `/api/risk/account/${id}/analyze`, method: 'POST' }),
  analyzePositions: (id: number) => requestList<RiskRecord>({ url: `/api/risk/account/${id}/positions/analyze`, method: 'POST' }),
  analyzeBacktest: (id: number) => request<RiskRecord>({ url: `/api/risk/backtest/${id}/analyze`, method: 'POST' }),
  resolve: (id: number) => request<RiskRecord>({ url: `/api/risk/records/${id}/resolve`, method: 'PUT' }),
  ignore: (id: number) => request<RiskRecord>({ url: `/api/risk/records/${id}/ignore`, method: 'PUT' }),
  remove: (id: number) => request<{ deleted_count:number }>({ url: `/api/risk/records/${id}`, method: 'DELETE' }),
  removeBatch: (ids: number[]) => request<{ deleted_count:number }>({ url: '/api/risk/records', method: 'DELETE', data: { ids } })
}
