export function utf8ByteLength(value: string) {
  return new TextEncoder().encode(value).length
}

export function toLocalReferenceTime(date: Date) {
  const offsetMinutes = -date.getTimezoneOffset()
  const localISO = new Date(date.getTime() + offsetMinutes * 60_000).toISOString().slice(0, -1)
  const sign = offsetMinutes >= 0 ? '+' : '-'
  const absoluteOffset = Math.abs(offsetMinutes)
  const hours = String(Math.floor(absoluteOffset / 60)).padStart(2, '0')
  const minutes = String(absoluteOffset % 60).padStart(2, '0')
  return `${localISO}${sign}${hours}:${minutes}`
}
