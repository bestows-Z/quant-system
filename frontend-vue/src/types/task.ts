export interface SystemTask {
  id: number
  task_no?: string
  user_id?: number
  task_type?: string
  source_type?: string
  source_id?: string
  title?: string
  status?: string
  progress?: number
  payload_json?: string
  result_json?: string
  error_message?: string
  created_at?: string
  updated_at?: string
}

export interface TaskLog {
  id: number
  task_id?: number
  task_no?: string
  level?: string
  message?: string
  detail_json?: string
  created_at?: string
}
