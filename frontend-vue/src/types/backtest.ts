export interface BacktestTask {
  id: number
  task_no?: string
  symbol: string
  strategy_code?: string
  status?: string
  error_message?: string
  progress?: number
  start_date?: string
  end_date?: string
  initial_cash?: number
  created_at?: string
  data_source?: string
  data_mode?: string
  params_json?: string
  adjust?: string
  engine_version?: string
  execution_model?: string
  quality_status?: string
  config_json?: string
}

export interface BacktestResult {
  id?: number
  task_id?: number
  total_return?: number
  annual_return?: number
  max_drawdown?: number
  win_rate?: number
  profit_loss_ratio?: number
  trade_count?: number
  total_trades?: number
  final_asset?: number
  engine_version?: string
  execution_model?: string
  adjust?: string
  quality_status?: string
  params_json?: string
  config_json?: string
  quality_json?: string
  realized_pnl?: number
  unrealized_pnl?: number
  total_fees?: number
  turnover_rate?: number
  exposure_rate?: number
  data_source?: string
  data_mode?: string
}

export interface BacktestTrade {
  id?: number
  symbol?: string
  side?: string
  price?: number
  volume?: number
  amount?: number
  trade_time?: string
  signal_date?: string
  base_price?: number | string
  fee?: number | string
  commission?: number | string
  stamp_duty?: number | string
  transfer_fee?: number | string
  profit?: number | string
  reason?: string
}

export interface BacktestOrder {
  id: number
  signal_date: string
  trade_date?: string
  symbol: string
  side: string
  requested_quantity: number
  filled_quantity: number
  cancelled_quantity: number
  base_price?: number | string
  fill_price?: number | string
  status: string
  reason_code?: string
  reason?: string
  signal_reason?: string
}

export interface BacktestRunResponse {
  task_id: number
  task_no: string
  status: string
  result: BacktestResult
  orders: BacktestOrder[]
}

export interface BacktestQuality {
  status?: string
  issues?: string[]
  valid_rows?: number
  start_date?: string
  end_date?: string
  source?: string
  data_mode?: string
  adjust?: string
  rule_coverage?: string
  verified_status_rows?: number
  verified_limit_rows?: number
}

export interface EquityPoint {
  trade_date?: string
  total_asset?: number | string
  cash?: number | string
  market_value?: number | string
  drawdown?: number
  date?: string
  time?: string
  equity?: number
  asset?: number
  value?: number
}
