import request, { requestList } from './request'
import type { CompanyProfile, IndicatorResult, KlineItem, MarketIndex, MoneyFlowItem, OrderBook, RealtimeQuote, Stock, StockAnnouncement, StockNews, SupportResistance, TimeshareItem, WatchlistItem } from '@/types/market'

interface PagedPayload<T> {
  items?: T[]
  total?: number
  count?: number
  page?: number
  page_size?: number
  source?: string
  data_mode?: string
}

const firstItem = <T>(payload: T | PagedPayload<T>): T => {
  if (payload && typeof payload === 'object' && Array.isArray((payload as PagedPayload<T>).items)) {
    return ((payload as PagedPayload<T>).items?.[0] || {}) as T
  }
  return payload as T
}

export const marketApi = {
  stocks: (keyword?: string, page = 1, page_size = 50) => requestList<Stock>({ url: '/api/stocks', method: 'GET', params: { keyword, page, page_size } }),
  stocksPayload: (params?: { keyword?: string; page?: number; page_size?: number }) =>
    request<PagedPayload<Stock>>({ url: '/api/stocks', method: 'GET', params }),
  searchStocks: (keyword: string, page = 1, page_size = 50) => requestList<Stock>({ url: '/api/stocks/search', method: 'GET', params: { keyword, page, page_size } }),
  stockDetail: (symbol: string) => request<Stock>({ url: `/api/stocks/${symbol}`, method: 'GET' }),
  realtime: (symbols?: string) => requestList<RealtimeQuote>({ url: '/api/market/realtime', method: 'GET', params: { symbols } }),
  realtimeBySymbol: async (symbol: string) => firstItem(await request<RealtimeQuote | PagedPayload<RealtimeQuote>>({ url: `/api/market/realtime/${symbol}`, method: 'GET' })),
  timeshare: (symbol: string) => requestList<TimeshareItem>({ url: `/api/market/timeshare/${symbol}`, method: 'GET' }),
  orderBook: (symbol: string) => request<OrderBook>({ url: `/api/market/orderbook/${symbol}`, method: 'GET' }),
  indices: () => requestList<MarketIndex>({ url: '/api/market/indices', method: 'GET' }),
  kline: (symbol: string, params?: Record<string, string | number>) => requestList<KlineItem>({ url: `/api/kline/${symbol}`, method: 'GET', params }),
  indicator: (symbol: string, params?: Record<string, string>) => request<IndicatorResult>({ url: `/api/indicator/${symbol}`, method: 'GET', params }),
  fullIndicator: (symbol: string, params?: Record<string, string | number>) => request<IndicatorResult>({ url: `/api/indicator/full/${symbol}`, method: 'GET', params }),
  supportResistance: (symbol: string, params?: Record<string, string | number>) => request<SupportResistance>({ url: `/api/stocks/${symbol}/support-resistance`, method: 'GET', params }),
  companyProfile: (symbol: string) => request<CompanyProfile>({ url: `/api/stocks/${symbol}/company`, method: 'GET' }),
  announcements: (symbol: string, limit = 20) => requestList<StockAnnouncement>({ url: `/api/stocks/${symbol}/announcements`, method: 'GET', params: { limit } }),
  news: (symbol: string, limit = 20) => requestList<StockNews>({ url: `/api/stocks/${symbol}/news`, method: 'GET', params: { limit } }),
  moneyFlow: (symbol: string, limit = 20) => requestList<MoneyFlowItem>({ url: `/api/stocks/${symbol}/money-flow`, method: 'GET', params: { limit } }),
  watchlists: () => requestList<WatchlistItem>({ url: '/api/watchlists', method: 'GET' }),
  addWatchlist: (data: { symbol: string; group_name?: string; remark?: string }) => request<WatchlistItem>({ url: '/api/watchlists', method: 'POST', data }),
  deleteWatchlist: (id: number) => request<{ deleted: boolean }>({ url: `/api/watchlists/${id}`, method: 'DELETE' })
}
