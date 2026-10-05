import request, { requestList } from './request'
import type { PortfolioAnalysis, PositionImportPreview, RealPortfolio, RealPosition } from '@/types/portfolio'

export const portfolioApi = {
  create: (data: { name: string; description?: string }) => request<RealPortfolio>({ url: '/api/portfolios', method: 'POST', data }),
  list: () => requestList<RealPortfolio>({ url: '/api/portfolios', method: 'GET' }),
  get: (id: number) => request<RealPortfolio>({ url: `/api/portfolios/${id}`, method: 'GET' }),
  remove: (id: number) => request<{ deleted: boolean }>({ url: `/api/portfolios/${id}`, method: 'DELETE' }),
  positions: (id: number) => requestList<RealPosition>({ url: `/api/portfolios/${id}/positions`, method: 'GET' }),
  addPosition: (id: number, data: Partial<RealPosition>) => request<RealPosition>({ url: `/api/portfolios/${id}/positions`, method: 'POST', data }),
  updatePosition: (id: number, positionId: number, data: Partial<RealPosition>) => request<RealPosition>({ url: `/api/portfolios/${id}/positions/${positionId}`, method: 'PUT', data }),
  removePosition: (id: number, positionId: number) => request<{ deleted: boolean }>({ url: `/api/portfolios/${id}/positions/${positionId}`, method: 'DELETE' }),
  previewImport: (id: number, file: File, idempotencyKey: string) => {
    const data = new FormData()
    data.append('file', file)
    return request<PositionImportPreview>({ url: `/api/portfolios/${id}/import/preview`, method: 'POST', data, headers: { 'Idempotency-Key': idempotencyKey } })
  },
  confirmImport: (id: number, task_id: number) => request<{ task_id: number; status: string; imported_rows?: number; idempotent_replay?: boolean }>({ url: `/api/portfolios/${id}/import/confirm`, method: 'POST', data: { task_id } }),
  analysis: (id: number, refresh = false) => request<PortfolioAnalysis>({ url: refresh ? `/api/portfolios/${id}/analyze` : `/api/portfolios/${id}/analysis`, method: refresh ? 'POST' : 'GET', data: refresh ? {} : undefined })
}
