export interface PaperAccount {
  id: number
  account_name?: string
  name?: string
  initial_cash?: number
  cash?: number
  available_cash?: number
  frozen_cash?: number
  market_value?: number
  total_asset?: number
  total_profit?: number
  total_profit_rate?: number
  return_rate?: number
  status?: string
}

export interface PaperPosition {
  id?: number
  account_id?: number
  symbol: string
  name?: string
  volume?: number
  available_volume?: number
  today_buy_volume?: number
  last_buy_date?: string
  cost_price?: number
  avg_cost?: number
  current_price?: number
  market_price?: number
  market_value?: number
  profit?: number
  profit_rate?: number
  position_ratio?: number
}

export interface PaperOrder {
  id?: number
  order_no?: string
  account_id: number
  client_order_id?: string
  symbol: string
  name?: string
  side: 'BUY' | 'SELL' | string
  order_type: 'MARKET' | 'LIMIT' | string
  price?: number
  volume: number
  filled_volume?: number
  status?: string
  created_at?: string
}

export interface PaperTrade {
  id?: number
  trade_no?: string
  order_id?: number
  account_id?: number
  symbol?: string
  name?: string
  side?: string
  price?: number
  volume?: number
  amount?: number
  fee?: number
  trade_time?: string
}
