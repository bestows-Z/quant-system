export interface RealPortfolio {
  id: number
  name: string
  description?: string
  base_currency?: string
  status?: string
  version?: number
  created_at?: string
  updated_at?: string
}

export interface RealPosition {
  id: number
  portfolio_id: number
  symbol: string
  name?: string
  quantity: number
  available_quantity: number
  avg_cost: string | number
  opened_on?: string
  note?: string
  version: number
  created_at?: string
  updated_at?: string
}

export interface PositionImportError { row_number: number; field: string; message: string }
export interface PositionImportPreview {
  task_id: number
  file_hash: string
  status: string
  total_rows?: number
  valid_rows?: number
  invalid_rows?: number
  rows?: Array<Record<string, unknown>>
  errors?: PositionImportError[]
}

export interface PositionAnalysis {
  position_id: number
  symbol: string
  name?: string
  quantity: number
  avg_cost: string
  market_price?: string
  cost_value: string
  market_value?: string
  profit?: string
  profit_rate?: string
  weight?: string
  industry?: string
  risk_level?: string
  warnings?: string[]
}

export interface PortfolioAnalysis {
  portfolio_id: number
  analysis_version: string
  quote_time?: string
  market_value: string
  cost_value: string
  total_profit: string
  total_profit_rate: string
  concentration: string
  volatility: string
  max_drawdown: string
  risk_level: string
  coverage_rate: string
  data_source?: string
  data_mode?: string
  positions: PositionAnalysis[]
  industry_exposures: Array<{ industry: string; market_value: string; weight: string }>
  warnings?: string[]
  disclaimer?: string
  advanced_json?: string
}

export interface PortfolioEquityPoint { date: string; value: number }
export interface PortfolioContribution {
  symbol: string
  name?: string
  weight: number
  risk_contribution: number
  return_contribution: number
  period_return: number
}
export interface PortfolioCorrelation { symbol: string; values: Record<string, number | null> }
export interface PortfolioScenario {
  code: string
  name: string
  description: string
  loss_rate: number
  loss_amount: number
  level: 'LOW' | 'MEDIUM' | 'HIGH' | 'EXTREME'
}
export interface PortfolioAdvancedAnalysis {
  observation_count: number
  start_date: string
  end_date: string
  history_coverage: number
  annual_volatility: number
  annual_return: number
  sharpe_ratio: number
  sortino_ratio: number
  max_drawdown: number
  var_95: number
  cvar_95: number
  hhi: number
  effective_positions: number
  equity_curve: PortfolioEquityPoint[]
  contributions: PortfolioContribution[]
  correlations: PortfolioCorrelation[]
  scenarios: PortfolioScenario[]
  history_sources: string[]
  history_modes: string[]
}
