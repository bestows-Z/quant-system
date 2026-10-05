import request, { requestList } from './request'
import type { EquityPoint } from '@/types/backtest'
import type { PaperAccount, PaperOrder, PaperPosition, PaperTrade } from '@/types/paper'

export const paperApi = {
  createAccount: (data: { account_name: string; initial_cash: number }) => request<PaperAccount>({ url: '/api/paper/accounts', method: 'POST', data }),
  accounts: () => requestList<PaperAccount>({ url: '/api/paper/accounts', method: 'GET' }),
  account: (id: number) => request<PaperAccount>({ url: `/api/paper/accounts/${id}`, method: 'GET' }),
  assets: (id: number) => request<PaperAccount>({ url: `/api/paper/accounts/${id}/assets`, method: 'GET' }),
  positions: (id: number) => requestList<PaperPosition>({ url: `/api/paper/accounts/${id}/positions`, method: 'GET' }),
  createOrder: (data: Partial<PaperOrder>) => request<PaperOrder>({ url: '/api/paper/orders', method: 'POST', data }),
  orders: (params?: Record<string, unknown>) => requestList<PaperOrder>({ url: '/api/paper/orders', method: 'GET', params }),
  order: (id: number) => request<PaperOrder>({ url: `/api/paper/orders/${id}`, method: 'GET' }),
  cancelOrder: (id: number) => request<PaperOrder>({ url: `/api/paper/orders/${id}`, method: 'DELETE' }),
  trades: (params?: Record<string, unknown>) => requestList<PaperTrade>({ url: '/api/paper/trades', method: 'GET', params }),
  equityCurve: (account_id: number) => requestList<EquityPoint>({ url: '/api/paper/equity-curve', method: 'GET', params: { account_id } })
}
