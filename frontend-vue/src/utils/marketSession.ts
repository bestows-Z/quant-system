const parseQuoteTime = (value?: string) => {
  if (!value) return undefined
  const compact = value.match(/^(\d{4})(\d{2})(\d{2})(\d{2})(\d{2})(\d{2})$/)
  const normalized = compact
    ? `${compact[1]}-${compact[2]}-${compact[3]}T${compact[4]}:${compact[5]}:${compact[6]}+08:00`
    : value.includes('T') ? value : `${value.replace(' ', 'T')}+08:00`
  const timestamp = new Date(normalized).getTime()
  return Number.isFinite(timestamp) ? timestamp : undefined
}

const shanghaiParts = (now = new Date()) => {
  const parts = new Intl.DateTimeFormat('en-CA', {
    timeZone: 'Asia/Shanghai', weekday: 'short', hour: '2-digit', minute: '2-digit', second: '2-digit', hourCycle: 'h23'
  }).formatToParts(now)
  const part = (type: Intl.DateTimeFormatPartTypes) => parts.find((item) => item.type === type)?.value || ''
  return { weekday: part('weekday'), seconds: Number(part('hour')) * 3600 + Number(part('minute')) * 60 + Number(part('second')) }
}

export const isContinuousAuction = (now = new Date()) => {
  const { weekday, seconds } = shanghaiParts(now)
  if (weekday === 'Sat' || weekday === 'Sun') return false
  return (seconds >= 9 * 3600 + 30 * 60 && seconds <= 11 * 3600 + 30 * 60)
    || (seconds >= 13 * 3600 && seconds <= 15 * 3600)
}

export const quoteSafety = (quoteTime?: string, dataMode?: string, hasOrderBook = false, now = new Date()) => {
  if (dataMode !== 'real') return { safe: false, message: '当前不是真实行情，禁止提交模拟委托' }
  if (!hasOrderBook) return { safe: false, message: '尚未取得真实五档盘口，暂不能提交委托' }
  if (!isContinuousAuction(now)) return { safe: false, message: '当前不在连续竞价时段，仅展示最近行情' }
  const timestamp = parseQuoteTime(quoteTime)
  if (!timestamp) return { safe: false, message: '行情时间不可验证，暂不能提交委托' }
  const age = now.getTime() - timestamp
  if (age < -1000 || age > 3000) return { safe: false, message: `盘口已延迟 ${Math.max(0, age / 1000).toFixed(1)} 秒，请等待实时更新` }
  return { safe: true, message: '真实盘口有效，提交时服务端会再次校验' }
}
