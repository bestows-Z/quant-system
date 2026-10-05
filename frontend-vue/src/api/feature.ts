import request, { requestList } from './request'
import type { CandidateResult, CandidateStock, DragonTigerResult, DragonTigerSeatResult, MarketOverviewResult, ScreenerStock, SectorConstituentResult, SectorDetailResult, SectorStrength, SectorTimeshareResult } from '@/types/feature'

export const featureApi = {
  overview: () => request<MarketOverviewResult>({ url: '/api/market/overview', method: 'GET' }),
  sectors: () => requestList<SectorStrength>({ url: '/api/market/sectors', method: 'GET' }),
	sectorDetail: (code: string) => request<SectorDetailResult>({ url: `/api/market/sectors/${encodeURIComponent(code)}`, method: 'GET' }),
	sectorConstituents: (code: string, params?: Record<string, unknown>) => request<SectorConstituentResult>({ url: `/api/market/sectors/${encodeURIComponent(code)}/constituents`, method: 'GET', params }),
	sectorTimeshare: (code: string) => request<SectorTimeshareResult>({ url: `/api/market/sectors/${encodeURIComponent(code)}/timeshare`, method: 'GET' }),
  screener: (params?: Record<string, unknown>) => requestList<ScreenerStock>({ url: '/api/market/screener', method: 'GET', params }),
  candidates: (params?: Record<string, unknown>) => requestList<CandidateStock>({ url: '/api/candidates', method: 'GET', params }),
  candidateResult: (params?: Record<string, unknown>) => request<CandidateResult>({ url: '/api/candidates', method: 'GET', params }),
  candidateEvidence: (symbol: string) => request<{ items: CandidateStock[] }>({ url: `/api/candidates/${symbol}/evidence`, method: 'GET' }),
  dragonTiger: (params?: Record<string, unknown>) => request<DragonTigerResult>({ url: '/api/market/dragon-tiger', method: 'GET', params })
  ,dragonTigerSeats: (symbol: string, tradeDate: string) => request<DragonTigerSeatResult>({ url: `/api/market/dragon-tiger/${symbol}/seats`, method: 'GET', params: { trade_date: tradeDate } })
}
