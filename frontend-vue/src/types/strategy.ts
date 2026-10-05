export type SignalType = 'BUY' | 'SELL' | 'HOLD' | 'WATCH' | string
export type RiskLevel = 'LOW' | 'MEDIUM' | 'HIGH' | 'EXTREME' | string

export interface Strategy {
  id: number
  name: string
  code?: string
  description?: string
  params_json?: string
  status?: string
  is_system?: boolean
  created_at?: string
}

export interface StrategySignal {
  id?: number
  symbol: string
  name?: string
  strategy_code?: string
  signal_type?: SignalType
  action?: SignalType
  score?: number
  reason?: string
  risk_level?: RiskLevel
  trigger_price?: number | string
  trigger_time?: string
  created_at?: string
}

export interface RunStrategyResult {
  run_id: number
  signals: StrategySignal[]
  summary?: string
  disclaimer?: string
}
