import request from './request'
import type { CryptoSpotMarket } from '@/types/crypto'

export const cryptoApi = {
  spotMarket: (symbol: string, interval: string) => request<CryptoSpotMarket>({
    url: '/api/crypto/spot/market', method: 'GET', params: { symbol, interval }, timeout: 20000
  })
}
