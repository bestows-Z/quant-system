const labels: Record<string, string> = {
  BUY: '买入', SELL: '卖出', HOLD: '观望', WATCH: '关注',
  LOW: '低风险', MEDIUM: '中风险', HIGH: '高风险', EXTREME: '极高风险', UNKNOWN: '未知',
  SUCCESS: '成功', COMPLETED: '已完成', FAILED: '失败', RUNNING: '运行中', PENDING: '等待中', CANCELLED: '已取消',
  ENABLED: '已启用', DISABLED: '已停用', ACTIVE: '正常', INFO: '普通通知', WARNING: '风险提示', ERROR: '错误通知',
  MARKET: '市价单', LIMIT: '限价单', FILLED: '已成交', PARTIAL_FILLED: '部分成交', NEW: '待成交', CANCELING: '撤单中'
}

export const chineseLabel = (value?: string, fallback = '未知') => labels[String(value || '').toUpperCase()] || value || fallback
export const signalLabel = (value?: string) => chineseLabel(value, '观望')
export const riskLabel = (value?: string) => chineseLabel(value, '未知风险')
export const statusLabel = (value?: string) => chineseLabel(value, '未知状态')
export const notificationLevelLabel = (value?: string) => chineseLabel(value, '普通通知')
