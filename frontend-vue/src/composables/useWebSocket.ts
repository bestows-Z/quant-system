import { ElMessage, ElNotification } from 'element-plus'
import { useNotifyStore } from '@/stores/notify'
import { useWebsocketStore } from '@/stores/websocket'
import { storage } from '@/utils/storage'
import type { WsServerMessage } from '@/types/notify'

let ws: WebSocket | null = null
let reconnectTimer: number | undefined
let manualClosed = false
const listeners = new Set<(message: WsServerMessage) => void>()
const activeSubscriptions = new Map<string, { channel: string; symbol?: string }>()
const latestSequences = new Map<string, number>()

const notifyByMessage = async (message: WsServerMessage) => {
  const notifyStore = useNotifyStore()
  if (message.type === 'notify.created') await notifyStore.fetchUnreadCount()
  const titleMap: Record<string, string> = {
    'risk.warning': '风险提示',
    'paper.order.filled': '模拟成交',
    'agent.task.completed': '智能体分析完成',
    'backtest.task.completed': '回测任务完成',
    'strategy.signal.generated': '策略信号'
  }
  if (titleMap[message.type]) {
    ElNotification({
      title: titleMap[message.type],
      message: '仅供研究，不构成投资建议',
      type: message.type.includes('risk') ? 'warning' : 'success',
      duration: 4500
    })
  }
}

export const useWebSocket = () => {
  const store = useWebsocketStore()

  const urlFor = (channel = 'notify') => {
    const base = import.meta.env.VITE_WS_BASE_URL || '/ws'
    const token = encodeURIComponent(storage.getToken())
    if (base.startsWith('/')) {
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      return `${protocol}//${window.location.host}${base.replace(/\/$/, '')}/${channel}?token=${token}`
    }
    return `${base.replace(/\/$/, '')}/ws/${channel}?token=${token}`
  }

  const close = () => {
    manualClosed = true
    if (reconnectTimer) window.clearTimeout(reconnectTimer)
    ws?.close()
    ws = null
    store.reset()
  }

  const send = (payload: Record<string, unknown>) => {
    if (ws?.readyState === WebSocket.OPEN) ws.send(JSON.stringify(payload))
  }

  const subscribe = (channel: string, symbols: string[] = []) => {
    const targets = symbols.length ? symbols : ['']
    targets.forEach((symbol) => {
      const key = `${channel}:${symbol}`
      activeSubscriptions.set(key, { channel, symbol: symbol || undefined })
      send({ type: 'subscribe', channel, ...(symbol ? { symbol } : {}) })
      store.addSubscription(key)
    })
  }

  const unsubscribe = (channel: string, symbols: string[] = []) => {
    const targets = symbols.length ? symbols : ['']
    targets.forEach((symbol) => {
      const key = `${channel}:${symbol}`
      activeSubscriptions.delete(key)
      send({ type: 'unsubscribe', channel, ...(symbol ? { symbol } : {}) })
      store.removeSubscription(key)
    })
  }

  const onMessage = (listener: (message: WsServerMessage) => void) => {
    listeners.add(listener)
    return () => listeners.delete(listener)
  }

  const connect = (channel = 'notify') => {
    const token = storage.getToken()
    if (!token || ws?.readyState === WebSocket.OPEN) return
    manualClosed = false
    store.setReconnecting(Boolean(ws))
    ws = new WebSocket(urlFor(channel))
    ws.onopen = () => {
      store.setConnected(true)
      store.setReconnecting(false)
      activeSubscriptions.forEach((item) => send({ type: 'subscribe', channel: item.channel, ...(item.symbol ? { symbol: item.symbol } : {}) }))
    }
    ws.onmessage = async (event) => {
      try {
        const message = JSON.parse(event.data) as WsServerMessage
		const symbol = String((message.data as { symbol?: string } | undefined)?.symbol || '')
		const sequenceKey = `${message.type}:${symbol}`
		if (message.sequence && message.sequence <= (latestSequences.get(sequenceKey) || 0)) return
		if (message.sequence) latestSequences.set(sequenceKey, message.sequence)
        store.setLastMessage(message)
        listeners.forEach((listener) => listener(message))
        await notifyByMessage(message)
      } catch {
        ElMessage.warning('收到无法解析的实时消息')
      }
    }
    ws.onclose = () => {
      store.setConnected(false)
      if (!manualClosed && storage.getToken()) {
        store.setReconnecting(true)
        reconnectTimer = window.setTimeout(() => connect(channel), 3000)
      }
    }
    ws.onerror = () => {
      store.setConnected(false)
    }
  }

  return { connect, close, send, subscribe, unsubscribe, onMessage }
}
