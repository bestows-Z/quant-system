import dayjs from 'dayjs'
import utc from 'dayjs/plugin/utc'
import timezone from 'dayjs/plugin/timezone'

dayjs.extend(utc)
dayjs.extend(timezone)

const SHANGHAI_TIMEZONE = 'Asia/Shanghai'

export const formatTime = (value?: string | number | Date) => (value ? dayjs(value).tz(SHANGHAI_TIMEZONE).format('YYYY-MM-DD HH:mm:ss') : '--')
export const formatDate = (value?: string | number | Date) => (value ? dayjs(value).tz(SHANGHAI_TIMEZONE).format('YYYY-MM-DD') : '--')
