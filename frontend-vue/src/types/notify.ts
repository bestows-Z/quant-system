export interface Notification {
  id: number
  notify_type?: string
  level?: string
  title: string
  content?: string
  source_type?: string
  source_id?: string
  link?: string
  payload_json?: string
  is_read?: boolean
  created_at?: string
}

export interface WsServerMessage<T = unknown> {
  type: string
  channel: string
  trace_id?: string
  event_id?: string
  data: T
  created_at?: string
	quote_time?: string
	server_time?: string
	latency_ms?: number
	sequence?: number
	source?: string
	data_mode?: string
}
