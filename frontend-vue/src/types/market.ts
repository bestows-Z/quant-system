export interface Stock {
  id?: number
  symbol: string
  code?: string
  name: string
  market?: string
  industry?: string
  list_date?: string
  exchange?: string
  source?: string
  message?: string
  data_mode?: 'real' | 'mock' | string
}

export interface RealtimeQuote {
  symbol: string
  name?: string
  price?: number
  current?: number
  open?: number
  high?: number
  low?: number
  pre_close?: number
  change?: number
  pct_change?: number
  change_pct?: number
  change_percent?: number
  volume?: number
  amount?: number
  turnover_rate?: number
  volume_ratio?: number
  pe_ratio?: number
  limit_up?: number
  limit_down?: number
  amplitude?: number
  quote_time?: string
  updated_at?: string
  source?: string
  message?: string
  data_mode?: 'real' | 'mock' | string
}

export interface TimeshareItem {
  symbol: string
  time: string
  price: number
  volume: number
  cumulative_volume: number
  amount: number
  average_price: number
  source?: string
  data_mode?: string
}

export interface OrderBookLevel {
  level: number
  price: number
  volume: number
}

export interface OrderBook {
  symbol: string
  name?: string
  price?: number
  pre_close?: number
  bids: OrderBookLevel[]
  asks: OrderBookLevel[]
  quote_time?: string
  source?: string
  data_mode?: string
}

export interface KlineItem {
  date?: string
  trade_date?: string
  time?: string
  open: number
  high: number
  low: number
  close: number
  volume?: number
  amount?: number
  pre_close?: number
  change_pct?: number
  amplitude?: number
  turnover_rate?: number
  source?: string
  message?: string
  data_mode?: 'real' | 'mock' | string
  is_realtime_bar?: boolean
  bar_status?: 'INTRADAY' | 'CLOSED' | string
  quote_time?: string
}

export interface MaItem {
  date?: string
  trade_date?: string
  ma5?: number
  ma10?: number
  ma20?: number
  ma60?: number
}

export interface MacdItem {
  date?: string
  trade_date?: string
  dif?: number
  dea?: number
  macd?: number
}

export interface RsiItem {
  date?: string
  trade_date?: string
  rsi6?: number
  rsi12?: number
  rsi24?: number
}

export interface BollItem {
  date?: string
  trade_date?: string
  upper?: number
  middle?: number
  lower?: number
}

export interface KdjItem {
  date?: string
  trade_date?: string
  k?: number
  d?: number
  j?: number
}

export interface IndicatorFullResult {
  symbol?: string
  source?: string
  message?: string
  data_mode?: 'real' | 'mock' | string
  indicators?: {
    ma?: MaItem[]
    ema?: Array<Record<string, number | string | undefined>>
    macd?: MacdItem[]
    rsi?: RsiItem[]
    boll?: BollItem[]
    kdj?: KdjItem[]
  }
  ma?: MaItem[]
  macd?: MacdItem[]
  rsi?: RsiItem[]
  boll?: BollItem[]
  kdj?: KdjItem[]
}

export type IndicatorResult = IndicatorFullResult

export interface SupportResistance {
  symbol: string
  period: string
  quote_time: string
  current_price: number
  support_price: number
  pivot_price: number
  resistance_price: number
  support_strength: number
  resistance_strength: number
  atr14: number
  methods?: { support?: string[]; resistance?: string[]; components?: string[] }
  algorithm_version: string
  data_source: string
  data_mode?: string
  valid_until: string
  disclaimer?: string
}

export interface WatchlistItem {
  id: number
  symbol: string
  name?: string
  group_name?: string
  remark?: string
  quote?: RealtimeQuote
  created_at?: string
}

export interface MarketIndex {
  symbol: string
  market_code: string
  name: string
  board: string
  price: number
  pre_close: number
  change: number
  change_pct: number
  high: number
  low: number
  volume: number
  amount: number
  quote_time: string
  source: string
}

export interface CompanyProfile {
  symbol: string
  name?: string
  company_name?: string
  company_name_en?: string
  former_name?: string
  security_type?: string
  exchange?: string
  industry?: string
  chairman?: string
  president?: string
  legal_person?: string
  secretary?: string
  phone?: string
  email?: string
  website?: string
  address?: string
  registered_address?: string
  province?: string
  registered_capital_wan?: number
  employees?: number
  listing_date?: string
  founded_date?: string
  actual_controller?: string
  main_business?: string
  business_scope?: string
  company_profile?: string
  source?: string
}

export interface StockAnnouncement {
  id: string
  symbol: string
  title: string
  category?: string
  published_at: string
  url: string
  source: string
}

export interface StockNews {
  id: string
  symbol: string
  title: string
  summary?: string
  published_at: string
  media?: string
  url: string
  source: string
}

export interface MoneyFlowItem {
  trade_date: string
  main_net_inflow: number
  retail_net_inflow: number
  medium_net_inflow: number
  large_net_inflow: number
  super_large_net_inflow: number
  main_net_ratio: number | null
  retail_net_ratio: number | null
  medium_net_ratio: number | null
  large_net_ratio: number | null
  super_large_net_ratio: number | null
  close: number | null
  change_pct: number | null
}
