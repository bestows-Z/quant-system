export interface CryptoSpotKline {
  open_time: number
  close_time: number
  open: string
  high: string
  low: string
  close: string
  base_volume: string
  quote_volume: string
}

export interface CryptoSpotMarket {
  symbol: string
  base_asset: string
  quote_asset: string
  trading_status: string
  interval: string
  quote: {
    last_price: string
    change_percent: string
    high_price: string
    low_price: string
    base_volume: string
    quote_volume: string
    bid_price: string
    ask_price: string
    event_time: number
  }
  rules: { tick_size: string; min_qty: string; step_size: string; min_notional: string }
  klines: CryptoSpotKline[]
  source: 'binance'
  data_mode: 'real'
  market: 'spot'
  kline_timezone: 'UTC'
  fetched_at: string
}
