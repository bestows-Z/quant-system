import request, { httpClient, requestList } from '@/api/request'
import type { AutoDecision, AutoStrategy, AutoTradeAccount, AutoTradeRun, DeliveryStatement } from '@/types/auto-trader'

export const autoTraderApi = {
  accounts: () => requestList<AutoTradeAccount>({ url: '/api/auto-trader/accounts' }),
  createAccount: (data: Record<string, unknown>) => request<AutoTradeAccount>({ url: '/api/auto-trader/accounts', method: 'POST', data }),
  archivedAccounts: () => requestList<AutoTradeAccount>({ url: '/api/auto-trader/accounts/archived' }),
  updateAccount: (id: number, data: Record<string, unknown>) => request<AutoTradeAccount>({ url: `/api/auto-trader/accounts/${id}`, method: 'PUT', data }),
  archiveAccount: (id: number) => request<{ id:number; status?:string; already_archived:boolean }>({ url: `/api/auto-trader/accounts/${id}`, method: 'DELETE' }),
  strategies: (accountId: number) => requestList<AutoStrategy>({ url: `/api/auto-trader/accounts/${accountId}/strategies` }),
  generateStrategy: (accountId: number, objective: string) => request<{ item: AutoStrategy; notice: string }>({ url: `/api/auto-trader/accounts/${accountId}/strategies/generate`, method: 'POST', data: { objective } }),
  strategy: (id: number) => request<AutoStrategy>({ url: `/api/auto-trader/strategies/${id}` }),
  evaluate: (id: number) => request<{ evaluation: AutoStrategy['evaluation']; notice: string }>({ url: `/api/auto-trader/strategies/${id}/evaluate`, method: 'POST' }),
  promote: (id: number) => request<{ item: AutoStrategy; notice: string }>({ url: `/api/auto-trader/strategies/${id}/promote`, method: 'POST' }),
  start: (accountId: number) => request<Record<string, unknown>>({ url: `/api/auto-trader/accounts/${accountId}/start`, method: 'POST' }),
  pause: (accountId: number) => request<Record<string, unknown>>({ url: `/api/auto-trader/accounts/${accountId}/pause`, method: 'POST' }),
  runs: (accountId: number) => requestList<AutoTradeRun>({ url: `/api/auto-trader/accounts/${accountId}/runs` }),
  decisions: (accountId: number) => requestList<AutoDecision>({ url: `/api/auto-trader/accounts/${accountId}/decisions` }),
  deliveries: (accountId: number) => requestList<DeliveryStatement>({ url: `/api/auto-trader/accounts/${accountId}/deliveries` }),
  exportDeliveries: async (accountId: number) => {
    const response = await httpClient.get<Blob>(`/api/auto-trader/accounts/${accountId}/deliveries/export`, { responseType: 'blob' })
    const disposition = String(response.headers['content-disposition'] || '')
    const match = disposition.match(/filename="?([^";]+)"?/i)
    const url = URL.createObjectURL(response.data)
    const link = document.createElement('a')
    link.href = url
    link.download = match?.[1] || `auto-trader-${accountId}-deliveries.csv`
    document.body.appendChild(link)
    link.click()
    link.remove()
    URL.revokeObjectURL(url)
  }
}
