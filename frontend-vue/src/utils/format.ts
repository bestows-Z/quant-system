type NumericValue = number | string | null | undefined

export const money = (value?: NumericValue, digits = 2) => {
  if (value === undefined || value === null || value === '') return '--'
  const n = Number(value)
  return Number.isFinite(n) ? n.toLocaleString('zh-CN', { minimumFractionDigits: digits, maximumFractionDigits: digits }) : '--'
}

export const percent = (value?: NumericValue, digits = 2) => {
  if (value === undefined || value === null || value === '') return '--'
  const n = Number(value)
  return Number.isFinite(n) ? `${(n * 100).toFixed(digits)}%` : '--'
}

export const signed = (value?: NumericValue, digits = 2) => {
  if (value === undefined || value === null || value === '') return '--'
  const n = Number(value)
  if (!Number.isFinite(n)) return '--'
  return `${n > 0 ? '+' : ''}${n.toFixed(digits)}`
}

export const compactNumber = (value?: NumericValue, digits = 2) => {
  if (value === undefined || value === null || value === '') return '--'
  const n = Number(value)
  if (!Number.isFinite(n)) return '--'
  const abs = Math.abs(n)
  if (abs >= 1000000000000) return `${trimFixed(n / 1000000000000, digits)}万亿`
  if (abs >= 100000000) return `${trimFixed(n / 100000000, digits)}亿`
  if (abs >= 10000) return `${trimFixed(n / 10000, digits)}万`
  return n.toLocaleString('zh-CN', { maximumFractionDigits: digits })
}

const trimFixed = (value: number, digits: number) => value.toFixed(digits).replace(/\.?0+$/, '')

export const pickArray = <T>(payload: unknown): T[] => {
  if (Array.isArray(payload)) return payload as T[]
  if (payload && typeof payload === 'object') {
    const obj = payload as Record<string, unknown>
    for (const key of ['items', 'list', 'data', 'records']) {
      if (Array.isArray(obj[key])) return obj[key] as T[]
    }
  }
  return []
}

export const pickObject = <T extends Record<string, unknown>>(payload: unknown): T => {
  return payload && typeof payload === 'object' ? (payload as T) : ({} as T)
}
