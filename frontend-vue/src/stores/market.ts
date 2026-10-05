import { defineStore } from 'pinia'
import { marketApi } from '@/api/market'
import type { RealtimeQuote, WatchlistItem } from '@/types/market'

export const useMarketStore = defineStore('market', {
  state: () => ({
    watchlist: [] as WatchlistItem[],
    realtimeQuotes: [] as RealtimeQuote[]
  }),
  actions: {
    async fetchWatchlist() {
      this.watchlist = await marketApi.watchlists()
    },
    async fetchRealtime(symbols?: string) {
      this.realtimeQuotes = await marketApi.realtime(symbols)
    }
  }
})
