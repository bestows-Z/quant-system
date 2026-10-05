export interface RiskRecord {
  id: number
  risk_type?: string
  source_type?: string
  source_id?: string
  symbol?: string
  title?: string
  message?: string
  risk_level?: string
  score?: number
  status?: 'OPEN' | 'RESOLVED' | 'IGNORED' | string
  created_at?: string
}
