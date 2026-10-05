export const toNumber = (value: unknown, fallback = 0) => {
  const n = Number(value)
  return Number.isFinite(n) ? n : fallback
}

export const ratioColorClass = (value?: number | string) => {
  const n = toNumber(value)
  if (n > 0) return 'up'
  if (n < 0) return 'down'
  return ''
}
