export interface MarketOverview {
  quote_time: string
  total: number
  listed_total: number
  coverage_rate: number
  rise_count: number
  fall_count: number
  flat_count: number
  limit_up_count: number
  limit_down_count: number
  rise_ratio: number
  median_change: number
  average_change: number
  total_amount: number
  market_temperature: number
  source: string
  scope: string
  data_mode: string
  disclaimer: string
}

export interface MarketDistribution { label: string; count: number }
export interface MarketOverviewResult { overview: MarketOverview; distribution: MarketDistribution[] }

export interface SectorStrength {
  code: string
  name: string
  stock_count: number
  rise_count: number
  fall_count: number
  average_change: number
  total_amount: number
  leader_symbol: string
  leader_name: string
  leader_change: number
  strength_score: number
	quoted_count: number
	coverage_rate: number
	quote_time?: string
	status?: 'AVAILABLE' | 'INSUFFICIENT_COVERAGE' | string
	source?: string
	data_mode?: string
}

export interface SectorConstituent extends ScreenerStock {}
export interface SectorDetailResult { sector: SectorStrength; disclaimer: string }
export interface SectorConstituentResult { sector_code:string;sector_name:string;items:SectorConstituent[];count:number;quoted_count:number;coverage_rate:number;source:string;data_mode:string }
export interface SectorTimeshareItem { time:string;change_percent?:number;rise_count:number;fall_count:number;quoted_count:number;stock_count:number;coverage_rate:number;total_amount:number;source:string;data_mode:string }
export interface SectorTimeshareResult { sector_code:string;sector_name:string;items:SectorTimeshareItem[];count:number;source:string;data_mode:string }

export interface ScreenerStock {
  symbol: string
  name: string
  industry: string
  price: number
  change_percent: number
  volume: number
  amount: number
  turnover_rate: number
  amplitude: number
  quote_time: string
  source: string
  data_mode: string
}

export interface CandidateStock extends ScreenerStock {
  total_score: number
  technical_score: number
  capital_score: number
  sentiment_score: number
  sector_score: number
  risk_score: number
  confidence: number
  reasons: string[]
  negative_factors: string[]
  evidence: Record<string, number>
  factor_version: string
  scoring_version: string
  data_source: string
  disclaimer: string
}

export interface CandidateResult { items:CandidateStock[];count:number;snapshot_no?:string;snapshot_time?:string;snapshot_mode?:'CLOSE_SNAPSHOT'|'REFRESHED'|'LIVE_FALLBACK'|string;factor_version?:string;scoring_version?:string;disclaimer?:string }

export interface DragonTigerRecord {
  trade_date: string
  symbol: string
  name: string
  reason: string
  close_price: number
  change_percent: number
  net_buy_amount: number
  buy_amount: number
  sell_amount: number
  turnover_amount: number
  source: string
}

export interface DragonTigerResult {
  items: DragonTigerRecord[]
  count: number
  status: 'AVAILABLE' | 'SOURCE_PENDING' | string
  message: string
  data_mode: string
  disclaimer: string
}

export interface DragonTigerSeat {
  seat_code: string
  seat_name: string
  seat_type: 'INSTITUTION' | 'NORTHBOUND' | 'BROKER_BRANCH' | string
  buy_amount: number
  sell_amount: number
  net_amount: number
  buy_ratio?: number
  sell_ratio?: number
  buy_rank?: number
  sell_rank?: number
}

export interface DragonTigerSeatResult {
  symbol: string
  trade_date: string
  items: DragonTigerSeat[]
  count: number
  status: string
  message: string
  source: string
  data_mode: string
  disclaimer: string
}
