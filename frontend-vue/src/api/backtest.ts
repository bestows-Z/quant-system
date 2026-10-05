import request, { requestList } from './request'
import type { BacktestOrder, BacktestResult, BacktestRunResponse, BacktestTask, BacktestTrade, EquityPoint } from '@/types/backtest'

export const backtestApi = {
  run: (data: Record<string, unknown>) => request<BacktestRunResponse>({ url: '/api/backtest/run', method: 'POST', data, timeout: 90000 }),
  tasks: (params?: Record<string, unknown>) => requestList<BacktestTask>({ url: '/api/backtest/tasks', method: 'GET', params }),
  task: (id: number) => request<BacktestTask>({ url: `/api/backtest/${id}`, method: 'GET' }),
  result: (id: number) => request<BacktestResult>({ url: `/api/backtest/${id}/result`, method: 'GET' }),
  trades: (id: number) => requestList<BacktestTrade>({ url: `/api/backtest/${id}/trades`, method: 'GET' }),
  orders: (id: number) => requestList<BacktestOrder>({ url: `/api/backtest/${id}/orders`, method: 'GET' }),
  equityCurve: (id: number) => requestList<EquityPoint>({ url: `/api/backtest/${id}/equity-curve`, method: 'GET' }),
  remove: (id: number) => request<{ affected: number; already_deleted: boolean }>({ url: `/api/backtest/${id}`, method: 'DELETE' }),
  removeBatch: (taskIds: number[]) => request<{ affected: number; already_deleted: boolean }>({ url: '/api/backtest', method: 'DELETE', data: { task_ids: taskIds } }),
  clearFailed: () => request<{ affected: number; already_deleted: boolean }>({ url: '/api/backtest?scope=failed', method: 'DELETE' })
}
