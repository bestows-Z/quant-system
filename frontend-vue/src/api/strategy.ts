import request, { requestList } from './request'
import type { RunStrategyResult, Strategy, StrategySignal } from '@/types/strategy'

export const strategyApi = {
  list: () => requestList<Strategy>({ url: '/api/strategies', method: 'GET' }),
  detail: (id: number) => request<Strategy>({ url: `/api/strategies/${id}`, method: 'GET' }),
  create: (data: Partial<Strategy>) => request<Strategy>({ url: '/api/strategies', method: 'POST', data }),
  update: (id: number, data: Partial<Strategy>) => request<Strategy>({ url: `/api/strategies/${id}`, method: 'PUT', data }),
  remove: (id: number) => request<{ deleted_count:number }>({ url: `/api/strategies/${id}`, method: 'DELETE' }),
  run: (id: number, data: { symbol: string; strategy_code?: string; params_json?: string }) => request<RunStrategyResult>({ url: `/api/strategies/${id}/run`, method: 'POST', data }),
  signals: (params?: Record<string, unknown>) => requestList<StrategySignal>({ url: '/api/signals', method: 'GET', params }),
  signalsBySymbol: (symbol: string, limit = 20) => requestList<StrategySignal>({ url: `/api/signals/${symbol}`, method: 'GET', params: { limit } }),
  latestSignal: (symbol: string) => request<StrategySignal>({ url: `/api/signals/${symbol}/latest`, method: 'GET' })
}
