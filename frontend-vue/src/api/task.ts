import request, { requestList } from './request'
import type { SystemTask, TaskLog } from '@/types/task'

export const taskApi = {
  create: (data: Partial<SystemTask>) => request<SystemTask>({ url: '/api/tasks', method: 'POST', data }),
  list: (params?: Record<string, unknown>) => requestList<SystemTask>({ url: '/api/tasks', method: 'GET', params }),
  detail: (id: number) => request<SystemTask>({ url: `/api/tasks/${id}`, method: 'GET' }),
  updateStatus: (id: number, data: Partial<SystemTask>) => request<SystemTask>({ url: `/api/tasks/${id}/status`, method: 'PUT', data }),
  cancel: (id: number) => request<SystemTask>({ url: `/api/tasks/${id}/cancel`, method: 'POST' }),
  addLog: (id: number, data: Partial<TaskLog>) => request<TaskLog>({ url: `/api/tasks/${id}/logs`, method: 'POST', data }),
  logs: (id: number, limit = 100) => requestList<TaskLog>({ url: `/api/tasks/${id}/logs`, method: 'GET', params: { limit } })
}
