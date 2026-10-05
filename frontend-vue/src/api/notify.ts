import request, { requestList } from './request'
import type { Notification } from '@/types/notify'

export const notifyApi = {
  list: (params?: Record<string, unknown>) => requestList<Notification>({ url: '/api/notifications', method: 'GET', params }),
  unreadCount: () => request<{ count: number }>({ url: '/api/notifications/unread-count', method: 'GET' }),
  markRead: (id: number) => request<Notification>({ url: `/api/notifications/${id}/read`, method: 'PUT' }),
  markAllRead: () => request<{ affected: number }>({ url: '/api/notifications/read-all', method: 'PUT' }),
  remove: (id: number) => request<{ ok: boolean; affected: number }>({ url: `/api/notifications/${id}`, method: 'DELETE' }),
  clear: (scope: 'read' | 'all') => request<{ ok: boolean; affected: number }>({ url: '/api/notifications', method: 'DELETE', params: { scope } })
}
