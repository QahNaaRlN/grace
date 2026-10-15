import type { FinancialEvent } from '@/shared/api'
import { APP_TIME_ZONE, APP_UTC_OFFSET } from '@/shared/config'

// Calendar day "YYYY-MM-DD" of a moment in the app time zone.
export function calendarDay(moment: Date | string) {
  return new Intl.DateTimeFormat('sv-SE', {
    timeZone: APP_TIME_ZONE,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).format(new Date(moment))
}
export function today() {
  return calendarDay(new Date())
}
// Current month "YYYY-MM" in the app time zone.
export function currentMonth() {
  return new Intl.DateTimeFormat('sv-SE', {
    timeZone: APP_TIME_ZONE,
    year: 'numeric',
    month: '2-digit',
  }).format(new Date())
}
// Start of a calendar day "YYYY-MM-DD" as an RFC 3339 moment.
export function dayStart(value: string) {
  return `${value}T00:00:00${APP_UTC_OFFSET}`
}
export function dateText(value: string) {
  return new Intl.DateTimeFormat('ru-RU', {
    timeZone: APP_TIME_ZONE,
    dateStyle: 'medium',
  }).format(new Date(value.length === 10 ? `${value}T12:00:00${APP_UTC_OFFSET}` : value))
}
export function dateTimeText(iso: string) {
  return new Date(iso).toLocaleString('ru-RU', { timeZone: APP_TIME_ZONE })
}
export function clockText(iso: string) {
  return new Intl.DateTimeFormat('ru-RU', {
    timeZone: APP_TIME_ZONE,
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(iso))
}
export function eventDateText(event: Pick<FinancialEvent, 'occurredAt' | 'occurredDate'>) {
  if (event.occurredDate) return `${dateText(event.occurredDate)} · время неизвестно`
  return event.occurredAt ? dateText(event.occurredAt) : 'Дата неизвестна'
}
export function eventTimeText(event: Pick<FinancialEvent, 'occurredAt' | 'occurredDate'>) {
  if (event.occurredDate) return eventDateText(event)
  return event.occurredAt ? dateTimeText(event.occurredAt) : 'Дата неизвестна'
}
// datetime-local text ↔ RFC 3339 in the app time zone.
export function localDateTimeInput(iso: string) {
  return new Intl.DateTimeFormat('sv-SE', {
    timeZone: APP_TIME_ZONE,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
    .format(new Date(iso))
    .replace(' ', 'T')
}
export function fromLocalDateTimeInput(value: string) {
  return `${value}:00${APP_UTC_OFFSET}`
}
export function momentText(iso: string) {
  return new Intl.DateTimeFormat('ru-RU', {
    timeZone: APP_TIME_ZONE,
    day: 'numeric',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(iso))
}
