import { defineStore } from 'pinia'
import type { WsServerMessage } from '@/types/notify'

export const useWebsocketStore = defineStore('websocket', {
  state: () => ({
    connected: false,
    reconnecting: false,
    lastMessage: null as WsServerMessage | null,
    subscriptions: [] as string[]
  }),
  actions: {
    setConnected(value: boolean) {
      this.connected = value
    },
    setReconnecting(value: boolean) {
      this.reconnecting = value
    },
    setLastMessage(message: WsServerMessage) {
      this.lastMessage = message
    },
    addSubscription(key: string) {
      if (!this.subscriptions.includes(key)) this.subscriptions.push(key)
    },
    removeSubscription(key: string) {
      this.subscriptions = this.subscriptions.filter((item) => item !== key)
    },
    reset() {
      this.connected = false
      this.reconnecting = false
      this.lastMessage = null
      this.subscriptions = []
    }
  }
})
