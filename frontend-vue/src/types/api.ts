export interface ApiResponse<T = unknown> {
  code: number
  message: string
  data: T
  trace_id?: string
}

export interface PageResult<T> {
  list: T[]
  total: number
  page?: number
  page_size?: number
}

export interface UserInfo {
  id: number
  username: string
  email?: string
  role?: string
  roles?: string[]
  created_at?: string
}

export interface LoginPayload {
  username: string
  password: string
}

export interface RegisterPayload {
  username: string
  email: string
  password: string
}

export interface LoginResult {
  token: string
  user?: UserInfo
  expires_at?: number
}

export type ID = number | string
