export interface AgentTask {
  id: number
  task_no?: string
  task_type?: string
  symbol?: string
  status?: string
  progress?: number
  created_at?: string
}

export interface AgentReport {
  id?: number
	report_id?: number
  task_id?: number
	symbol?: string
  summary?: string
  market_analysis?: string
  technical_analysis?: string
  strategy_analysis?: string
  rag_analysis?: string
  paper_trading_analysis?: string
  risk_analysis?: string
  final_view?: string
  confidence?: number
  risk_level?: string
  references?: unknown[]
  references_json?: string
  data_sources_json?: string
  evidence_completeness?: number
  model_config_id?: number
  provider_code?: string
  model?: string
  prompt_tokens?: number
  completion_tokens?: number
  total_tokens?: number
  latency_ms?: number
  token_source?: string
  disclaimer?: string
  created_at?: string
  [key: string]: unknown
}

export interface AgentReportPage { items:AgentReport[];total:number }
export interface AgentChatSession { id:number;title:string;status:string;last_message_at?:string;created_at:string }
export interface AgentChatMessage { id:number;session_id:number;request_id:string;role:'user'|'assistant';content:string;status:'COMPLETED'|'FAILED'|'STREAMING';provider:string;model:string;input_tokens:number;output_tokens:number;latency_ms:number;citations_json:string;tool_calls_json:string;error_message:string;created_at:string }
