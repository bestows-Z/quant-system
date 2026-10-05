import { defineStore } from 'pinia'
import { notifyApi } from '@/api/notify'
import type { Notification } from '@/types/notify'

export const useNotifyStore = defineStore('notify', {
  state: () => ({
    unreadCount: 0,
    notifications: [] as Notification[]
  }),
  actions: {
    async fetchUnreadCount() {
      const res = await notifyApi.unreadCount()
      this.unreadCount = Number(res.count || 0)
    },
    async fetchNotifications(params?: Record<string, unknown>) {
      this.notifications = await notifyApi.list(params)
    },
    async markRead(id: number) {
      await notifyApi.markRead(id)
      await Promise.all([this.fetchNotifications(), this.fetchUnreadCount()])
    },
    async markAllRead() {
      await notifyApi.markAllRead()
      await Promise.all([this.fetchNotifications(), this.fetchUnreadCount()])
    },
    async remove(id: number) {
      await notifyApi.remove(id)
      await Promise.all([this.fetchNotifications(), this.fetchUnreadCount()])
    },
    async clear(scope: 'read' | 'all') {
      await notifyApi.clear(scope)
      await Promise.all([this.fetchNotifications(), this.fetchUnreadCount()])
    }
  }
})
